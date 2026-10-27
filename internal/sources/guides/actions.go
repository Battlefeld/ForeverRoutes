package guides

import (
	"regexp"
	"strconv"
	"strings"
)

var targetID = regexp.MustCompile(`::(\d+)`)
var objectiveInteraction = regexp.MustCompile(`(?i)\b(heal|fortify|resurrect|revive|waken|awaken|salute|feed|tame|dispel|purify)`)
var objectiveTalk = regexp.MustCompile(`(?i)\b(speak|spoken|talk|gauge|obtain instructions|receive instructions)`)

// objectiveAction retains only action facts and NPC identifiers. It never stores guide prose.
// A builder must still match the exact NPC before using these hints to reinterpret a creature credit.
func objectiveAction(f *Fact, lines []string) {
	gossip, use := false, false
	for _, l := range lines {
		if strings.HasPrefix(l, ".target ") {
			name := strings.TrimSpace(strings.TrimPrefix(l, ".target "))
			name = strings.TrimPrefix(name, "+")
			if m := targetID.FindStringSubmatch(name); m != nil {
				id, _ := strconv.Atoi(m[1])
				f.Targets = append(f.Targets, id)
				name = name[:strings.Index(name, "::")]
			}
			if name != "" {
				f.TargetNames = appendName(f.TargetNames, name)
			}
		}
		gossip = gossip || strings.HasPrefix(l, ".skipgossip") || strings.HasPrefix(l, ".gossip")
		use = use || strings.HasPrefix(l, ".use ")
		if m := rxpComplete.FindStringSubmatch(l); m != nil {
			q, _ := strconv.Atoi(m[1])
			n, _ := strconv.Atoi(m[2])
			if q != f.Quest || n != f.Obj {
				continue
			}
			switch {
			case objectiveInteraction.MatchString(l):
				f.Action = "interact"
			case objectiveTalk.MatchString(l):
				f.Action = "talk"
			case strings.Contains(strings.ToLower(l), "delivered"):
				f.Action = "use"
			}
		}
	}
	if f.Action == "" && gossip {
		f.Action = "talk"
	}
	if f.Action == "" && use {
		f.Action = "use"
	}
}
