package sqldump

import (
	"strings"
	"testing"
)

const dump = "CREATE TABLE `quest_template` (\n  `entry` mediumint NOT NULL,\n  `Title` text,\n  `RewXP` float\n) ENGINE=MyISAM;\n" +
	"INSERT INTO `quest_template` VALUES (1,'It\\'s ''fine''',1.5),(2,NULL,-3),(3,'a,b)c',0);\n" +
	"INSERT INTO `other` VALUES (9);\n"

func TestRead(t *testing.T) {
	tabs, err := Read(strings.NewReader(dump), "quest_template", "missing")
	if err != nil {
		t.Fatal(err)
	}
	q := tabs["quest_template"]
	if len(q.Rows) != 3 || q.Col("Title") != 1 {
		t.Fatalf("rows %v cols %v", q.Rows, q.Columns)
	}
	if q.String(q.Rows[0], "Title") != "It's 'fine'" || q.Float(q.Rows[0], "RewXP") != 1.5 {
		t.Errorf("row 1 %v", q.Rows[0])
	}
	if q.Rows[1][1] != nil || q.Int(q.Rows[1], "RewXP") != -3 {
		t.Errorf("row 2 %v", q.Rows[1])
	}
	if q.String(q.Rows[2], "Title") != "a,b)c" {
		t.Errorf("row 3 %v", q.Rows[2])
	}
	if len(tabs["missing"].Rows) != 0 || tabs["other"] != nil {
		t.Error("only wanted tables")
	}
}
