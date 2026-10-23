// Package sqldump reads the tables of a MySQL dump (as published by the cmangos world database projects) without a
// database server: CREATE TABLE gives the column names, INSERT INTO ... VALUES gives the rows.
package sqldump

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Table is the column names and rows of one table; a value is int64, float64, string or nil.
type Table struct {
	Columns []string
	Rows    [][]any
	index   map[string]int
}

// Col is the position of a column, -1 when it does not exist.
func (t *Table) Col(name string) int {
	if t.index == nil {
		t.index = map[string]int{}
		for i, c := range t.Columns {
			t.index[c] = i
		}
	}
	if i, ok := t.index[name]; ok {
		return i
	}
	return -1
}

// Int reads column name of row r as an integer (0 for missing or NULL).
func (t *Table) Int(r []any, name string) int {
	i := t.Col(name)
	if i < 0 || i >= len(r) {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// Float reads column name of row r as a number.
func (t *Table) Float(r []any, name string) float64 {
	i := t.Col(name)
	if i < 0 || i >= len(r) {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	}
	return 0
}

// String reads column name of row r as text.
func (t *Table) String(r []any, name string) string {
	i := t.Col(name)
	if i < 0 || i >= len(r) {
		return ""
	}
	if s, ok := r[i].(string); ok {
		return s
	}
	return ""
}

var (
	createRe = regexp.MustCompile("^CREATE TABLE `(\\w+)`")
	colRe    = regexp.MustCompile("^\\s+`(\\w+)`")
	insertRe = regexp.MustCompile("^INSERT INTO `(\\w+)`")
)

// ReadFile reads the wanted tables from a dump file (plain or .gz).
func ReadFile(path string, want ...string) (map[string]*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	}
	return Read(r, want...)
}

// Read reads the wanted tables from a dump.
func Read(r io.Reader, want ...string) (map[string]*Table, error) {
	wanted := map[string]bool{}
	for _, w := range want {
		wanted[w] = true
	}
	out := map[string]*Table{}
	columns := map[string][]string{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<30)
	creating := ""
	for sc.Scan() {
		line := sc.Text()
		if m := createRe.FindStringSubmatch(line); m != nil {
			creating = m[1]
			columns[creating] = nil
			continue
		}
		if creating != "" {
			if m := colRe.FindStringSubmatch(line); m != nil {
				columns[creating] = append(columns[creating], m[1])
			}
			if strings.HasPrefix(line, ")") {
				creating = ""
			}
			continue
		}
		m := insertRe.FindStringSubmatch(line)
		if m == nil || !wanted[m[1]] {
			continue
		}
		t := out[m[1]]
		if t == nil {
			t = &Table{Columns: columns[m[1]]}
			out[m[1]] = t
		}
		i := strings.Index(line, "VALUES")
		if i < 0 {
			continue
		}
		rows, err := parseValues(line[i+6:])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m[1], err)
		}
		t.Rows = append(t.Rows, rows...)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	for _, w := range want {
		if out[w] == nil {
			out[w] = &Table{Columns: columns[w]}
		}
	}
	return out, nil
}

func parseValues(s string) ([][]any, error) {
	var rows [][]any
	i, n := 0, len(s)
	for i < n {
		for i < n && (s[i] == ' ' || s[i] == ',' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
			i++
		}
		if i >= n || s[i] == ';' {
			break
		}
		if s[i] != '(' {
			return nil, fmt.Errorf("expected ( at %d", i)
		}
		i++
		var row []any
		for {
			if i >= n {
				return nil, fmt.Errorf("unterminated row")
			}
			if s[i] == '\'' {
				i++
				var b strings.Builder
				for {
					if i >= n {
						return nil, fmt.Errorf("unterminated string")
					}
					c := s[i]
					if c == '\\' && i+1 < n {
						switch s[i+1] {
						case 'n':
							b.WriteByte('\n')
						case 'r':
							b.WriteByte('\r')
						case 't':
							b.WriteByte('\t')
						case '0':
							b.WriteByte(0)
						default:
							b.WriteByte(s[i+1])
						}
						i += 2
						continue
					}
					if c == '\'' {
						if i+1 < n && s[i+1] == '\'' {
							b.WriteByte('\'')
							i += 2
							continue
						}
						i++
						break
					}
					b.WriteByte(c)
					i++
				}
				row = append(row, b.String())
			} else {
				j := i
				for j < n && s[j] != ',' && s[j] != ')' {
					j++
				}
				tok := strings.TrimSpace(s[i:j])
				i = j
				switch {
				case tok == "NULL":
					row = append(row, nil)
				default:
					if v, err := strconv.ParseInt(tok, 10, 64); err == nil {
						row = append(row, v)
					} else if f, err := strconv.ParseFloat(tok, 64); err == nil {
						row = append(row, f)
					} else {
						row = append(row, tok)
					}
				}
			}
			if i < n && s[i] == ',' {
				i++
				continue
			}
			if i < n && s[i] == ')' {
				i++
				break
			}
			return nil, fmt.Errorf("bad value list near %d", i)
		}
		rows = append(rows, row)
	}
	return rows, nil
}
