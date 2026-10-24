// Package lua reads Lua table literals as they appear in addon data files (QuestieDB, EverythingQuests): numbers,
// strings, booleans, nil and nested tables. It never runs Lua code; anything else is an error.
package lua

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Table is a Lua table with explicit keys ([k] = v). Keys are int or string. Tables with only positional values are
// returned as []any instead.
type Table map[any]any

type reader struct {
	s string
	i int
}

// Parse reads exactly one literal.
func Parse(s string) (any, error) {
	r := &reader{s: s}
	v, err := r.value()
	if err != nil {
		return nil, err
	}
	r.ws()
	if r.i != len(r.s) {
		return nil, fmt.Errorf("trailing tokens at %d", r.i)
	}
	return v, nil
}

func (r *reader) ws() {
	for r.i < len(r.s) && (r.s[r.i] == ' ' || r.s[r.i] == '\t' || r.s[r.i] == '\n' || r.s[r.i] == '\r') {
		r.i++
	}
}

func (r *reader) take(c byte) bool {
	r.ws()
	if r.i < len(r.s) && r.s[r.i] == c {
		r.i++
		return true
	}
	return false
}

var number = regexp.MustCompile(`^-?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?`)
var keyword = regexp.MustCompile(`^(?:true|false|nil)\b`)

func (r *reader) value() (any, error) {
	r.ws()
	if r.i >= len(r.s) {
		return nil, fmt.Errorf("unexpected end")
	}
	switch c := r.s[r.i]; c {
	case '{':
		return r.table()
	case '"', '\'':
		r.i++
		var b strings.Builder
		for r.i < len(r.s) {
			ch := r.s[r.i]
			r.i++
			if ch == c {
				return b.String(), nil
			}
			if ch == '\\' {
				if r.i == len(r.s) {
					return nil, fmt.Errorf("truncated escape")
				}
				esc := r.s[r.i]
				r.i++
				switch esc {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				default:
					b.WriteByte(esc)
				}
				continue
			}
			b.WriteByte(ch)
		}
		return nil, fmt.Errorf("unterminated string")
	}
	if m := number.FindString(r.s[r.i:]); m != "" {
		r.i += len(m)
		if strings.ContainsAny(m, ".eE") {
			return strconv.ParseFloat(m, 64)
		}
		return strconv.Atoi(m)
	}
	if m := keyword.FindString(r.s[r.i:]); m != "" {
		r.i += len(m)
		switch m {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		return nil, nil
	}
	end := min(r.i+30, len(r.s))
	return nil, fmt.Errorf("unsupported token at %d: %q", r.i, r.s[r.i:end])
}

func (r *reader) table() (any, error) {
	if !r.take('{') {
		return nil, fmt.Errorf("expected {")
	}
	var seq []any
	var idx Table
	for {
		r.ws()
		if r.take('}') {
			break
		}
		if r.take('[') {
			k, err := r.value()
			if err != nil {
				return nil, err
			}
			if !r.take(']') || !r.take('=') {
				return nil, fmt.Errorf("malformed key at %d", r.i)
			}
			v, err := r.value()
			if err != nil {
				return nil, err
			}
			if idx == nil {
				idx = Table{}
			}
			idx[k] = v
		} else {
			v, err := r.value()
			if err != nil {
				return nil, err
			}
			seq = append(seq, v)
		}
		if r.take('}') {
			break
		}
		if !r.take(',') && !r.take(';') {
			return nil, fmt.Errorf("expected , at %d", r.i)
		}
	}
	if idx != nil {
		for i, v := range seq {
			if _, ok := idx[i+1]; !ok {
				idx[i+1] = v
			}
		}
		return idx, nil
	}
	if seq == nil {
		seq = []any{}
	}
	return seq, nil
}

var rowLine = regexp.MustCompile(`^\[(\d+)\]\s*=\s*(\{.*\}),?\s*$`)

// Rows reads data files with one `[id] = {...},` row per line, as QuestieDB writes them. Rows must be positional
// tables.
func Rows(raw string) (map[int][]any, error) {
	out := map[int][]any{}
	for _, line := range strings.Split(raw, "\n") {
		m := rowLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		v, err := Parse(m[2])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", id, err)
		}
		row, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("row %d: expected a positional table", id)
		}
		out[id] = row
	}
	return out, nil
}

// At is row[i], or nil past the end.
func At(row []any, i int) any {
	if i < len(row) {
		return row[i]
	}
	return nil
}

// Int is v as a positive or zero int when it is one.
func Int(v any) (int, bool) {
	n, ok := v.(int)
	return n, ok
}

// IDs collects every positive int in v, depth first (lists and tables in key order for ints).
func IDs(v any) []int {
	switch t := v.(type) {
	case int:
		if t > 0 {
			return []int{t}
		}
	case []any:
		var out []int
		for _, e := range t {
			out = append(out, IDs(e)...)
		}
		return out
	case Table:
		var out []int
		for _, k := range sortedKeys(t) {
			out = append(out, IDs(t[k])...)
		}
		return out
	}
	return nil
}

// sortedKeys orders int keys ascending, then string keys; the Python original kept insertion order, which in these
// files is ascending.
func sortedKeys(t Table) []any {
	var ints []int
	var strs []string
	for k := range t {
		switch kk := k.(type) {
		case int:
			ints = append(ints, kk)
		case string:
			strs = append(strs, kk)
		}
	}
	sort.Ints(ints)
	sort.Strings(strs)
	out := make([]any, 0, len(t))
	for _, k := range ints {
		out = append(out, k)
	}
	for _, k := range strs {
		out = append(out, k)
	}
	return out
}

// SortedKeys exposes the key order used by IDs.
func SortedKeys(t Table) []any { return sortedKeys(t) }
