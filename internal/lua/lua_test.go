package lua

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	v, err := Parse(`{'Cutting Teeth',{{3143}},{{3143},nil},1,2,nil,"a\"b",{[12]={{42.1,68.4}},[14]={}},-3,1e2,true}`)
	if err != nil {
		t.Fatal(err)
	}
	row := v.([]any)
	if row[0] != "Cutting Teeth" || row[3] != 1 || row[5] != nil || row[6] != `a"b` || row[8] != -3 || row[9] != 100.0 || row[10] != true {
		t.Errorf("row %#v", row)
	}
	sp := row[7].(Table)
	if !reflect.DeepEqual(sp[12], []any{[]any{42.1, 68.4}}) {
		t.Errorf("spawns %#v", sp)
	}
	if got := IDs(row[1]); !reflect.DeepEqual(got, []int{3143}) {
		t.Errorf("ids %v", got)
	}
}

func TestParseRejectsCode(t *testing.T) {
	for _, s := range []string{`{os.execute("x")}`, `{1,2`, `{[1]}`, `{1} junk`} {
		if _, err := Parse(s); err == nil {
			t.Errorf("%q should fail", s)
		}
	}
}

func TestRows(t *testing.T) {
	rows, err := Rows("local x = {\n[788] = {'Cutting Teeth',{{3143}},nil},\n[789] = {'Sting'},\n-- comment\n}")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[788][0] != "Cutting Teeth" || At(rows[789], 5) != nil {
		t.Errorf("%#v", rows)
	}
}

func TestParseNameKeys(t *testing.T) {
	v, err := Parse(`{ map = 0, x = -8832.77, name = "Stormwind, Elwynn", { dock = 1, arrive = 66861 }, ok = true }`)
	if err != nil {
		t.Fatal(err)
	}
	tb := v.(Table)
	if tb["map"] != 0 || tb["x"] != -8832.77 || tb["name"] != "Stormwind, Elwynn" || tb["ok"] != true {
		t.Fatalf("%v", tb)
	}
	if stop := tb[1].(Table); stop["dock"] != 1 || stop["arrive"] != 66861 {
		t.Fatalf("%v", stop)
	}
}
