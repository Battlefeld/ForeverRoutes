package route

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestClassProfileIncomingDamage(t *testing.T) {
	for _, tc := range []struct {
		class          string
		contactSeconds float64
	}{
		{"Warrior", 7.5}, {"Rogue", 7.5}, {"Paladin", 7.5},
		{"Mage", 4.5}, {"Priest", 4.5}, {"Warlock", 4.5}, {"Hunter", 4.5},
		// The shipped hybrid profiles cast spells; their class alone does not imply melee.
		{"Shaman", 4.5}, {"Druid", 4.5},
	} {
		t.Run(tc.class, func(t *testing.T) {
			p, err := LoadTableProfile(strings.NewReader(strings.Replace(tableJSON, `"class":"mage"`, `"class":"`+strings.ToLower(tc.class)+`"`, 1)))
			if err != nil {
				t.Fatal(err)
			}
			// Isolate incoming damage from drinking: all profiles have the same damage curve.
			p.byLevel[10].Mana = 0
			fight, rest := p.KillSeconds(Character{Class: tc.class}, 10, &dataset.NPC{HP: 150, MaxLevel: 10, Dmg: 20})
			food, _ := restRates(10)
			want := 10*tc.contactSeconds/food + 4
			if fight != 7.5 || math.Abs(rest-want) > 1e-9 {
				t.Fatalf("fight/rest = %.6f/%.6f, want 7.5/%.6f", fight, rest, want)
			}
		})
	}
}

func TestClassProfilePetAndMeasuredContact(t *testing.T) {
	p, err := LoadTableProfile(strings.NewReader(strings.Replace(tableJSON, `"class":"mage"`, `"class":"warlock"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	p.byLevel[10].Mana = 0
	p.ClassOptions = map[string]any{"summon": float64(2)}
	_, rest := p.KillSeconds(Character{Class: "Warlock"}, 10, &dataset.NPC{HP: 150, MaxLevel: 10, Dmg: 20})
	food, _ := restRates(10)
	if want := 10*7.5*.25/food + 4; math.Abs(rest-want) > 1e-9 {
		t.Fatalf("voidwalker rest = %.6f, want %.6f", rest, want)
	}
	// Explicit measured contact takes precedence over both pet and class defaults.
	p.MeleeShare = .6
	_, rest = p.KillSeconds(Character{Class: "Warlock"}, 10, &dataset.NPC{HP: 150, MaxLevel: 10, Dmg: 20})
	if want := 10*7.5*.6/food + 4; math.Abs(rest-want) > 1e-9 {
		t.Fatalf("measured rest = %.6f, want %.6f", rest, want)
	}
}

func TestClassChainsRejectPrerequisiteCycles(t *testing.T) {
	for _, tc := range []struct {
		name             string
		goalAll, goalAny []int
		prerequisites    []dataset.Quest
		want             []int
	}{
		{"and cycle", []int{1001}, nil, []dataset.Quest{{ID: 1001, PreAll: []int{1470}}}, nil},
		{"or self cycle", nil, []int{1470}, nil, nil},
		{"or cycle", nil, []int{1001}, []dataset.Quest{{ID: 1001, PreAny: []int{1470}}}, nil},
		{"mixed cycle", []int{1001}, nil, []dataset.Quest{{ID: 1001, PreAny: []int{1470}}}, nil},
		{"cyclic alternative with valid branch", nil, []int{1001, 1002}, []dataset.Quest{{ID: 1001, PreAny: []int{1470}}, {ID: 1002}}, []int{1002, 1470}},
		// An earlier failed OR branch must not mark shared prerequisites as processed.
		{"failed alternative does not leak ancestors", nil, []int{1001, 1002}, []dataset.Quest{{ID: 1001, PreAll: []int{1003, 1470}}, {ID: 1002, PreAll: []int{1003}}, {ID: 1003}}, []int{1003, 1002, 1470}},
		// A diamond is not a cycle, and a shared prerequisite must occur only once.
		{"shared prerequisite", []int{1001, 1002}, nil, []dataset.Quest{{ID: 1001, PreAll: []int{1003}}, {ID: 1002, PreAll: []int{1003}}, {ID: 1003}}, []int{1003, 1001, 1002, 1470}},
		{"and or shared prerequisite", []int{1001}, []int{1001}, []dataset.Quest{{ID: 1001}}, []int{1001, 1470}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			goal := &Quest{Quest: &dataset.Quest{ID: 1470, Name: "Imp training", MinLevel: 4, Classes: ClassBits["Warlock"], PreAll: tc.goalAll, PreAny: tc.goalAny}, StartPos: dataset.Point{Map: 1}}
			w := &World{Char: Character{Race: "Orc", Class: "Warlock"}, Start: dataset.Point{Map: 1}, Quests: []*Quest{goal}, ByID: map[int]*Quest{goal.ID: goal}}
			for i := range tc.prerequisites {
				q := &Quest{Quest: &tc.prerequisites[i], StartPos: dataset.Point{Map: 1}}
				w.ByID[q.ID] = q
			}
			chains := w.ClassChains(20)
			var got []int
			if len(chains) > 0 {
				if len(chains) != 1 || chains[0].Tier != "required" {
					t.Fatalf("unexpected class chains: %+v", chains)
				}
				for _, q := range chains[0].Quests {
					got = append(got, q.ID)
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("prerequisite order = %v, want %v", got, tc.want)
			}
		})
	}
}
