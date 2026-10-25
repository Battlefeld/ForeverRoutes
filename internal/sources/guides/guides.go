// Package guides reads what leveling guides tell the player about a quest objective: which creatures to kill for it,
// where to go, whether its items lie on the ground, which object to use. For Forever's own quests this is often the
// only public record of where an item drops. Two guide sets are read, both under licences that allow it:
//
//   - RestedXP's Forever guides (github.com/RestedXP/RXPGuides, Guides/forever and Guides/Forever; CC BY-NC-SA 4.0):
//     a step's ".complete Q,N" and ".collect ITEM,COUNT,Q,N" name the objective, ".mob" the creatures, ".goto" the
//     places (world yards "uiMap/instance,y,x" or map percent "uiMap,x,y"), ".use ITEM" an item that starts a quest.
//   - Polestar's Forever guides (github.com/fitzforge/polestar, guides/forever; GPL-3.0): "kill NPC quest Q.N",
//     "collect ITEM quest Q.N", "interact OBJECT quest Q.N", with "@x,y" or a step's "goto x,y" on the zone's map.
//
// Only facts are taken (ids, names, places); no guide text.
package guides

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Place is where a guide sends the player: world yards (World: Map is the instance, X north, Y west) or percent on an
// in-game map (UIMap, or Zone by name when the guide names the zone instead).
type Place struct {
	World bool    `json:"world,omitempty"`
	Map   int     `json:"map,omitempty"`
	UIMap int     `json:"ui,omitempty"`
	Zone  string  `json:"zone,omitempty"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// Fact is what one guide step says about one objective (Obj, numbered as the game numbers them, from 1) of a quest, or
// about the item that starts a quest (Start).
type Fact struct {
	Source  string   `json:"source"` // rxp | polestar
	File    string   `json:"file"`
	Quest   int      `json:"quest"`
	Obj     int      `json:"obj,omitempty"`
	Item    int      `json:"item,omitempty"`
	Start   bool     `json:"start,omitempty"`
	Mobs    []string `json:"mobs,omitempty"`
	NPCs    []int    `json:"npcs,omitempty"`
	Objects []int    `json:"objects,omitempty"`
	Places  []Place  `json:"places,omitempty"`
	Ground  bool     `json:"ground,omitempty"` // the items lie on the ground (or in the open): picked up, not dropped
	// Polestar steps without coordinates: the step's zone, and the creatures its other objectives kill there (the items
	// are found while doing them).
	Zone string `json:"zone,omitempty"`
	Near []int  `json:"near,omitempty"`
	// Complete: a "complete" line (an objective done, or the whole quest when Obj is 0); Accept, Turnin: the same step
	// also takes or hands in the quest (an event done at the giver or the turn-in).
	Complete bool `json:"complete,omitempty"`
	Accept   bool `json:"accept,omitempty"`
	Turnin   bool `json:"turnin,omitempty"`
}

var (
	rxpGuide    = regexp.MustCompile(`(?s)RegisterGuide\(\[\[(.*?)\]\]`)
	rxpGoto     = regexp.MustCompile(`^\.goto\s+(\d+)(?:/(\d+))?,\s*(-?[\d.]+),\s*(-?[\d.]+)`)
	rxpComplete = regexp.MustCompile(`^\.complete\s+(\d+)(?:,(\d+))?`)
	rxpTurnin   = regexp.MustCompile(`^\.turnin\s+(\d+)`)
	rxpCollect  = regexp.MustCompile(`^\.collect\s+(\d+),\d+(?:,(\d+),(\d+))?`)
	rxpAccept   = regexp.MustCompile(`^\.accept\s+(\d+)`)
	rxpUse      = regexp.MustCompile(`^\.use\s+(\d+)`)
	rxpEnemy    = regexp.MustCompile(`cRXP_ENEMY_([^|]+)\|r`)
)

// ParseRXP reads RestedXP guide files (name -> content).
func ParseRXP(files map[string]string) []Fact {
	var out []Fact
	for _, name := range sortedKeys(files) {
		for _, body := range rxpGuide.FindAllStringSubmatch(files[name], -1) {
			var step []string
			flush := func() {
				out = append(out, rxpStep(name, step)...)
				step = nil
			}
			for _, raw := range strings.Split(body[1], "\n") {
				line := strings.TrimSpace(raw)
				if strings.HasPrefix(line, "--") || line == "" {
					continue
				}
				if strings.HasPrefix(line, "step") {
					flush()
					continue
				}
				// a line's own class or race filter does not change where things are
				if i := strings.Index(line, " << "); i > 0 {
					line = strings.TrimSpace(line[:i])
				}
				if i := strings.Index(line, " --"); i > 0 {
					line = strings.TrimSpace(line[:i])
				}
				step = append(step, line)
			}
			flush()
		}
	}
	return out
}

func rxpStep(file string, lines []string) []Fact {
	var places []Place
	var facts []*Fact
	var cur *Fact
	var stepMobs, textMobs []string
	ground, use, accepted := false, 0, 0
	accepts, turnins := map[int]bool{}, map[int]bool{}
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, ".goto "):
			if p, ok := parseGoto(l); ok {
				places = append(places, p)
			}
		case rxpComplete.MatchString(l):
			m := rxpComplete.FindStringSubmatch(l)
			q, _ := strconv.Atoi(m[1])
			n, _ := strconv.Atoi(m[2])
			cur = &Fact{Source: "rxp", File: file, Quest: q, Obj: n, Complete: true}
			facts = append(facts, cur)
		case rxpCollect.MatchString(l):
			m := rxpCollect.FindStringSubmatch(l)
			item, _ := strconv.Atoi(m[1])
			cur = &Fact{Source: "rxp", File: file, Item: item}
			if m[2] != "" {
				cur.Quest, _ = strconv.Atoi(m[2])
				cur.Obj, _ = strconv.Atoi(m[3])
			}
			facts = append(facts, cur)
		case strings.HasPrefix(l, ".mob "):
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(l, ".mob "), "+"))
			if cur != nil {
				cur.Mobs = appendName(cur.Mobs, name)
			} else {
				stepMobs = appendName(stepMobs, name)
			}
		case rxpUse.MatchString(l):
			use, _ = strconv.Atoi(rxpUse.FindStringSubmatch(l)[1])
		case rxpAccept.MatchString(l):
			accepted, _ = strconv.Atoi(rxpAccept.FindStringSubmatch(l)[1])
			accepts[accepted] = true
		case rxpTurnin.MatchString(l):
			q, _ := strconv.Atoi(rxpTurnin.FindStringSubmatch(l)[1])
			turnins[q] = true
		case strings.HasPrefix(l, ">>"):
			if strings.Contains(l, "on the ground") || strings.Contains(l, "cRXP_PICK_") {
				ground = true
			}
			for _, m := range rxpEnemy.FindAllStringSubmatch(l, -1) {
				textMobs = appendName(textMobs, singular(strings.TrimSpace(m[1])))
			}
		}
	}
	var out []Fact
	for _, f := range facts {
		if len(f.Mobs) == 0 {
			f.Mobs = stepMobs
		}
		if len(f.Mobs) == 0 {
			f.Mobs = textMobs
		}
		f.Places, f.Ground = places, ground && len(f.Mobs) == 0
		// ".collect ITEM,1,Q" with ".use ITEM" and ".accept Q": the item that starts the quest
		if f.Item != 0 && f.Item == use && accepted != 0 {
			f.Start, f.Quest, f.Obj = true, accepted, 0
		}
		f.Accept, f.Turnin = accepts[f.Quest], turnins[f.Quest]
		if f.Quest != 0 && (len(f.Mobs) > 0 || len(f.Places) > 0 || f.Complete) {
			out = append(out, *f)
		}
	}
	return out
}

var (
	psStep    = regexp.MustCompile(`^step\b`)
	psObj     = regexp.MustCompile(`^(kill|collect|interact)\s+(\d+)(?:\s+x\d+)?\s+quest\s+(\d+)(?:\.(\d+))?(.*)$`)
	psAt      = regexp.MustCompile(`@(-?[\d.]+),(-?[\d.]+)`)
	psGoto    = regexp.MustCompile(`^goto\s+(-?[\d.]+),(-?[\d.]+)`)
	psZone    = regexp.MustCompile(`^zone\s+(.+)$`)
	psDone    = regexp.MustCompile(`^complete\s+(\d+)(?:\.(\d+))?`)
	psTake    = regexp.MustCompile(`^(accept|turnin)\s+(\d+)`)
	psComment = regexp.MustCompile(`\s+--.*$`)
	psTip     = regexp.MustCompile(`^tip\s+(.*)$`)
	psName    = regexp.MustCompile(`[A-Z][a-z'-]+(?:\s+[A-Z][a-z'-]+)*`)
)

// ParsePolestar reads Polestar guide files (name -> content).
func ParsePolestar(files map[string]string) []Fact {
	var out []Fact
	for _, name := range sortedKeys(files) {
		guideZone, zone := "", ""
		var step []string
		flush := func() {
			out = append(out, polestarStep(name, zone, step)...)
			step, zone = nil, guideZone
		}
		for _, raw := range strings.Split(files[name], "\n") {
			if strings.HasPrefix(strings.TrimSpace(raw), "#") {
				continue
			}
			line := strings.TrimSpace(psComment.ReplaceAllString(raw, ""))
			if line == "" {
				continue
			}
			indented := strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")
			switch {
			case !indented && psStep.MatchString(line):
				flush()
			case !indented && psZone.MatchString(line):
				guideZone = psZone.FindStringSubmatch(line)[1]
				zone = guideZone
			case indented && psZone.MatchString(line):
				zone = psZone.FindStringSubmatch(line)[1]
			case indented:
				step = append(step, line)
			}
		}
		flush()
	}
	return out
}

func polestarStep(file, zone string, lines []string) []Fact {
	var stepPlaces []Place
	var kills []int
	var tipNames []string
	for _, l := range lines {
		if m := psGoto.FindStringSubmatch(l); m != nil {
			x, _ := strconv.ParseFloat(m[1], 64)
			y, _ := strconv.ParseFloat(m[2], 64)
			stepPlaces = append(stepPlaces, Place{Zone: zone, X: x, Y: y})
		}
		if m := psObj.FindStringSubmatch(l); m != nil && m[1] == "kill" {
			id, _ := strconv.Atoi(m[2])
			kills = append(kills, id)
		}
		// the creatures a tip names ("Gray Forest Wolves roam the woods"); the build keeps those that exist in the zone
		if m := psTip.FindStringSubmatch(l); m != nil {
			for _, n := range psName.FindAllString(m[1], -1) {
				tipNames = appendName(tipNames, singular(n))
			}
		}
	}
	var out []Fact
	accepts, turnins := map[int]bool{}, map[int]bool{}
	for _, l := range lines {
		if m := psTake.FindStringSubmatch(l); m != nil {
			q, _ := strconv.Atoi(m[2])
			if m[1] == "accept" {
				accepts[q] = true
			} else {
				turnins[q] = true
			}
		}
	}
	for _, l := range lines {
		if m := psDone.FindStringSubmatch(l); m != nil {
			q, _ := strconv.Atoi(m[1])
			n, _ := strconv.Atoi(m[2])
			out = append(out, Fact{Source: "polestar", File: file, Quest: q, Obj: n, Places: stepPlaces, Zone: zone, Complete: true, Accept: accepts[q], Turnin: turnins[q]})
			continue
		}
		m := psObj.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[2])
		q, _ := strconv.Atoi(m[3])
		n, _ := strconv.Atoi(m[4])
		f := Fact{Source: "polestar", File: file, Quest: q, Obj: n, Places: stepPlaces, Zone: zone}
		if at := psAt.FindStringSubmatch(m[5]); at != nil {
			x, _ := strconv.ParseFloat(at[1], 64)
			y, _ := strconv.ParseFloat(at[2], 64)
			f.Places = []Place{{Zone: zone, X: x, Y: y}}
		}
		switch m[1] {
		case "kill":
			f.NPCs = []int{id}
		case "collect":
			f.Item = id
		case "interact":
			f.Objects = []int{id}
		}
		// an item without a place: found while killing the step's creatures, or dropped by the creatures its tip
		// names when the step kills nothing
		if f.Item != 0 && len(f.Places) == 0 {
			if len(kills) > 0 {
				f.Near = kills
			} else {
				f.Mobs = tipNames
			}
		}
		out = append(out, f)
	}
	return out
}

// singular turns a guide's plural creature name ("Kobold Vermins") into the creature's.
func singular(n string) string {
	switch {
	case strings.HasSuffix(n, "ies"):
		return n[:len(n)-3] + "y"
	case strings.HasSuffix(n, "ves"):
		return n[:len(n)-3] + "f"
	case strings.HasSuffix(n, "s") && !strings.HasSuffix(n, "ss"):
		return n[:len(n)-1]
	}
	return n
}

func appendName(l []string, n string) []string {
	if n == "" {
		return l
	}
	for _, x := range l {
		if x == n {
			return l
		}
	}
	return append(l, n)
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ReadDir reads every file below dir whose name ends in suffix (name relative to dir -> content).
func ReadDir(dir, suffix string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, suffix) {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return files, err
}

// Deathskip is a deathskip a guide uses: where to die, where the guide goes next (by the graveyard's Spirit Healer),
// and the step's race, class and faction filter as written ("Horde Shaman", "!Warlock", "Human Warrior/Dwarf Paladin").
type Deathskip struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Die    Place  `json:"die"`
	Rise   Place  `json:"rise"`
	Filter string `json:"filter,omitempty"`
}

var rxpStepFilter = regexp.MustCompile(`^step\b(?:\s*<<\s*(.*))?`)

// RXPDeathskips reads the deathskips of RestedXP guide files: the last place before a ".deathskip" line and the first
// after it. Steps marked skip (switched off in the guide) are left out.
func RXPDeathskips(files map[string]string) []Deathskip {
	var out []Deathskip
	for _, name := range sortedKeys(files) {
		lines := strings.Split(files[name], "\n")
		for i, raw := range lines {
			if !strings.HasPrefix(strings.TrimSpace(raw), ".deathskip") {
				continue
			}
			filter, skip := "", false
			for k := i; k >= 0 && k > i-40; k-- {
				if m := rxpStepFilter.FindStringSubmatch(strings.TrimSpace(lines[k])); m != nil {
					filter = strings.TrimSpace(m[1])
					if c := strings.Index(filter, "--"); c >= 0 {
						filter = strings.TrimSpace(filter[:c])
					}
					skip = strings.Contains(" "+filter+" ", " skip ")
					break
				}
			}
			die, okD := nearGoto(lines, i, -1)
			rise, okR := nearGoto(lines, i, 1)
			if skip || !okD || !okR {
				continue
			}
			out = append(out, Deathskip{File: name, Line: i + 1, Die: die, Rise: rise, Filter: filter})
		}
	}
	return out
}

// nearGoto is the first ".goto" from line i going in direction dir, within 30 lines.
func nearGoto(lines []string, i, dir int) (Place, bool) {
	for k := i + dir; k >= 0 && k < len(lines) && k != i+30*dir; k += dir {
		l := strings.TrimSpace(lines[k])
		if j := strings.Index(l, " << "); j > 0 {
			l = strings.TrimSpace(l[:j])
		}
		if p, ok := parseGoto(l); ok {
			return p, true
		}
	}
	return Place{}, false
}

var rxpGotoZone = regexp.MustCompile(`^\.goto\s+([A-Za-z][A-Za-z' ]*?),\s*(-?[\d.]+),\s*(-?[\d.]+)`)

// parseGoto reads a ".goto" line: "UIMAP/INSTANCE,y,x" in world yards, "UIMAP,x,y" or "Zone Name,x,y" in map percent.
func parseGoto(l string) (Place, bool) {
	if m := rxpGoto.FindStringSubmatch(l); m != nil {
		ui, _ := strconv.Atoi(m[1])
		a, _ := strconv.ParseFloat(m[3], 64)
		b, _ := strconv.ParseFloat(m[4], 64)
		if m[2] != "" {
			inst, _ := strconv.Atoi(m[2])
			return Place{World: true, Map: inst, UIMap: ui, X: b, Y: a}, true
		}
		return Place{UIMap: ui, X: a, Y: b}, true
	}
	if m := rxpGotoZone.FindStringSubmatch(l); m != nil {
		a, _ := strconv.ParseFloat(m[2], 64)
		b, _ := strconv.ParseFloat(m[3], 64)
		return Place{Zone: strings.TrimSpace(m[1]), X: a, Y: b}, true
	}
	return Place{}, false
}

// Sighting is where a guide has the player talk to an NPC: a step's ".target Name::ID" with the step's place.
type Sighting struct {
	NPC   int   `json:"npc"`
	Place Place `json:"place"`
	File  string
	Line  int
}

var rxpTarget = regexp.MustCompile(`^\.target\s+[^:]*::(\d+)`)

// RXPTargets reads where RestedXP guides send the player to an NPC named with its id: the last ".goto" of the step
// before the ".target" line (the place the step talks at). Steps marked skip are left out.
func RXPTargets(files map[string]string) []Sighting {
	var out []Sighting
	for _, name := range sortedKeys(files) {
		var place *Place
		skip := false
		for i, raw := range strings.Split(files[name], "\n") {
			l := strings.TrimSpace(raw)
			if m := rxpStepFilter.FindStringSubmatch(l); m != nil || l == "step" {
				place, skip = nil, m != nil && strings.Contains(" "+m[1]+" ", " skip ")
				continue
			}
			if j := strings.Index(l, " << "); j > 0 {
				l = strings.TrimSpace(l[:j])
			}
			if p, ok := parseGoto(l); ok {
				place = &p
				continue
			}
			if m := rxpTarget.FindStringSubmatch(l); m != nil && place != nil && !skip {
				id, _ := strconv.Atoi(m[1])
				out = append(out, Sighting{NPC: id, Place: *place, File: name, Line: i + 1})
			}
		}
	}
	return out
}

// QuestSteps is what RestedXP's guides do with a quest: whether a step takes it, hands it in or works an objective
// (".complete" or ".collect" for it), and where a step takes it by using an item (".use" in the accepting step, without
// a creature to kill: the item is at hand there).
type QuestSteps struct {
	Accept, Turnin, Work bool
	UseAt                []Place
}

// RXPQuests reads the quest steps of RestedXP guide files (steps marked skip are left out).
func RXPQuests(files map[string]string) map[int]*QuestSteps {
	out := map[int]*QuestSteps{}
	get := func(q int) *QuestSteps {
		if out[q] == nil {
			out[q] = &QuestSteps{}
		}
		return out[q]
	}
	for _, name := range sortedKeys(files) {
		lines := strings.Split(files[name], "\n")
		for i := 0; i < len(lines); i++ {
			m := rxpStepFilter.FindStringSubmatch(strings.TrimSpace(lines[i]))
			if m == nil || strings.Contains(" "+m[1]+" ", " skip ") {
				continue
			}
			var places []Place
			accepted, use, mob := 0, false, false
			for j := i + 1; j < len(lines); j++ {
				l := strings.TrimSpace(lines[j])
				if rxpStepFilter.MatchString(l) || strings.HasPrefix(l, "]]") {
					break
				}
				if k := strings.Index(l, " << "); k > 0 {
					l = strings.TrimSpace(l[:k])
				}
				switch {
				case strings.HasPrefix(l, ".goto "):
					if p, ok := parseGoto(l); ok {
						places = append(places, p)
					}
				case rxpAccept.MatchString(l):
					accepted, _ = strconv.Atoi(rxpAccept.FindStringSubmatch(l)[1])
					get(accepted).Accept = true
				case rxpTurnin.MatchString(l):
					q, _ := strconv.Atoi(rxpTurnin.FindStringSubmatch(l)[1])
					get(q).Turnin = true
				case rxpComplete.MatchString(l):
					q, _ := strconv.Atoi(rxpComplete.FindStringSubmatch(l)[1])
					get(q).Work = true
				case rxpCollect.MatchString(l):
					if c := rxpCollect.FindStringSubmatch(l); c[2] != "" {
						q, _ := strconv.Atoi(c[2])
						get(q).Work = true
					}
				case rxpUse.MatchString(l):
					use = true
				case strings.HasPrefix(l, ".mob ") || strings.HasPrefix(l, ".unitscan "):
					mob = true
				}
			}
			if accepted != 0 && use && !mob && len(places) > 0 {
				get(accepted).UseAt = append(get(accepted).UseAt, places[len(places)-1])
			}
		}
	}
	return out
}

// OrderStep is a quest step of a guide file: accept or turnin.
type OrderStep struct {
	Op    string
	Quest int
}

// RXPOrder lists the accept and turnin steps of each RestedXP guide file in the guide's order (steps marked skip are
// left out).
func RXPOrder(files map[string]string) map[string][]OrderStep {
	out := map[string][]OrderStep{}
	for _, name := range sortedKeys(files) {
		skip := false
		for _, raw := range strings.Split(files[name], "\n") {
			l := strings.TrimSpace(raw)
			if m := rxpStepFilter.FindStringSubmatch(l); m != nil {
				skip = strings.Contains(" "+m[1]+" ", " skip ")
				continue
			}
			if skip {
				continue
			}
			if k := strings.Index(l, " << "); k > 0 {
				l = strings.TrimSpace(l[:k])
			}
			if m := rxpAccept.FindStringSubmatch(l); m != nil {
				q, _ := strconv.Atoi(m[1])
				out[name] = append(out[name], OrderStep{"accept", q})
			} else if m := rxpTurnin.FindStringSubmatch(l); m != nil {
				q, _ := strconv.Atoi(m[1])
				out[name] = append(out[name], OrderStep{"turnin", q})
			}
		}
	}
	return out
}
