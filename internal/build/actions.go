package build

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
)

var scriptedInteraction = regexp.MustCompile(`(?i)\b(heal|fortify|resurrect|revive|awoken|awaken|waken|salute)\b`)

// legacyAction uses the actual objective text/spell requirement, never a friendly NPC's level as a substitute.
func (b *builder) legacyAction(q *dataset.Quest, o *dataset.Objective) string {
	cm := b.w.quest[q.ID]
	if cm == nil {
		return ""
	}
	t := b.w.qt
	for i := 1; i <= 4; i++ {
		if t.Int(cm, fmt.Sprintf("ReqCreatureOrGOId%d", i)) != o.ID {
			continue
		}
		if t.Int(cm, fmt.Sprintf("ReqSpellCast%d", i)) > 0 || scriptedInteraction.MatchString(t.String(cm, fmt.Sprintf("ObjectiveText%d", i))) {
			return "interact"
		}
	}
	return ""
}

func creditMarker(name string) bool {
	n := strings.ToLower(name)
	return n == "credit" || strings.Contains(n, "kill credit") || strings.Contains(n, "quest credit") || strings.HasPrefix(n, "[dnt]")
}

func actionTarget(f guides.Fact, n dataset.NPC) bool {
	if slices.Contains(f.Targets, n.ID) {
		return true
	}
	for _, name := range f.TargetNames {
		if strings.EqualFold(strings.TrimSpace(name), n.Name) {
			return true
		}
	}
	return false
}

// objectiveActions corrects the common client creature-credit encoding. A credit counter is not evidence of a kill.
// Exact guide actions win; hidden credit markers use their recorded places and never generate combat or kill XP.
func (b *builder) objectiveActions() {
	ns := map[int]dataset.NPC{}
	for _, n := range b.npcOut {
		ns[n.ID] = n
	}
	for i := range b.out {
		q := &b.out[i]
		for oi := range q.Objectives {
			o := &q.Objectives[oi]
			if o.Kind != "kill" {
				continue
			}
			if kind := b.legacyAction(q, o); kind != "" {
				o.Kind = kind
				o.Source += "+cmangos_action"
				b.stats["objective_actions_from_classic_flags"]++
				continue
			}
			n, ok := ns[o.ID]
			if !ok {
				continue
			}
			var facts []guides.Fact
			if b.gdx != nil {
				facts = b.gdx.byQuest[q.ID]
			}
			for _, f := range facts {
				if !f.Complete || (f.Obj != oi+1 && !(f.Obj == 0 && len(q.Objectives) == 1)) || f.Action == "" {
					continue
				}
				switch {
				case f.Action == "talk" && actionTarget(f, n):
					o.Kind = "talk"
				case (f.Action == "interact" || f.Action == "use") && actionTarget(f, n):
					o.Kind = "interact"
				case f.Action == "use" && creditMarker(n.Name):
					o.Kind = "event"
					o.Places = b.guidePoints(f.Places)
				default:
					continue
				}
				if o.Kind == "event" && len(o.Places) == 0 {
					o.Kind = "kill"
					continue
				}
				o.Source += "+" + f.Source + "_action"
				q.Sources["objectiveAction"] = f.Source + ":" + f.File
				b.stats["objective_actions_from_guides"]++
				break
			}
			if o.Kind == "kill" && creditMarker(n.Name) {
				var pts []dataset.Point
				for _, f := range facts {
					if f.Complete && (f.Obj == oi+1 || (f.Obj == 0 && len(q.Objectives) == 1)) {
						pts = append(pts, b.guidePoints(f.Places)...)
					}
				}
				if len(pts) == 0 {
					pts = n.Spawns
				}
				if len(pts) > 0 {
					o.Kind = "event"
					o.Places = dedupePoints(pts)
					o.Source += "+credit_marker"
					b.stats["objective_credit_markers"]++
				} else {
					b.unsupported(q, "scripted objective credit without a recorded place")
				}
			}
		}
	}
	b.scriptedFights()
}

// Some credits name the spirit spoken to after a summoned enemy has been defeated. Price both stages.
func (b *builder) scriptedFights() {
	for _, f := range b.questFixes {
		if f.AddFight <= 0 {
			continue
		}
		for i := range b.out {
			q := &b.out[i]
			if q.ID != f.Quest {
				continue
			}
			var pts []dataset.Point
			if b.gdx != nil {
				for _, g := range b.gdx.byQuest[q.ID] {
					if g.Complete {
						pts = append(pts, b.guidePoints(g.Places)...)
					}
				}
			}
			if len(pts) == 0 {
				b.unsupported(q, "summoned fight without a recorded place")
				continue
			}
			// The spirit appears where this quest's fight took place, including the Tauren version in Mulgore.
			// Its generic NPC table can contain only the Night Elf spawn in Darkshore.
			for j := range q.Objectives {
				if q.Objectives[j].Kind == "talk" {
					q.Objectives[j].Places = dedupePoints(pts)
				}
			}
			q.Objectives = append([]dataset.Objective{{Kind: "kill", ID: f.AddFight, Count: 1, Source: f.Source, Places: dedupePoints(pts)}}, q.Objectives...)
			b.stats["scripted_fights_restored"]++
		}
	}
}
