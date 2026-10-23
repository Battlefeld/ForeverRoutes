package xp

import "testing"

func TestQuestDecay(t *testing.T) {
	cases := []struct{ pl, ql, full, want int }{
		{10, 10, 1000, 1000}, {15, 10, 1000, 1000}, {16, 10, 1000, 800}, {17, 10, 1000, 600},
		{18, 10, 1000, 400}, {19, 10, 1000, 200}, {20, 10, 1000, 100}, {40, 10, 1000, 100}, {1, 10, 1000, 1000},
	}
	for _, c := range cases {
		if got := Quest(c.full, c.pl, c.ql); got != c.want {
			t.Errorf("Quest(%d, pl %d, ql %d) = %d, want %d", c.full, c.pl, c.ql, got, c.want)
		}
	}
	if QuestFromMoney(24) != 40 || QuestFromMoney(0) != 0 {
		t.Error("XP from money at max level")
	}
}

func TestKillXP(t *testing.T) {
	// level 10 vs level 10 mob: 10*5+45 = 95
	if got := Kill(10, 10, false); got != 95 {
		t.Errorf("same level %d", got)
	}
	if got := Kill(10, 12, false); got != 105 { // 95 * 1.10 = 104.5 -> 105
		t.Errorf("two levels above %d", got)
	}
	if got := Kill(10, 10, true); got != 190 {
		t.Errorf("elite %d", got)
	}
	if got := Kill(20, GrayLevel(20), false); got != 0 {
		t.Errorf("gray mob gives XP: %d", got)
	}
	if got := Kill(20, 18, false); got <= 0 || got >= Kill(20, 20, false) {
		t.Errorf("lower mob gives less: %d", got)
	}
	if GrayLevel(5) != 0 || GrayLevel(20) != 13 || GrayLevel(50) != 39 {
		t.Error("gray levels")
	}
}

func TestLevel(t *testing.T) {
	if l, into := Level(ClassicLevelXP, 0); l != 1 || into != 0 {
		t.Error("start")
	}
	if l, into := Level(ClassicLevelXP, 400); l != 2 || into != 0 {
		t.Error("level 2 at 400")
	}
	if l, _ := Level(ClassicLevelXP, Total(ClassicLevelXP, 30)); l != 30 {
		t.Error("total to 30")
	}
	if Total(ClassicLevelXP, 3) != 1300 {
		t.Error("total to 3")
	}
}
