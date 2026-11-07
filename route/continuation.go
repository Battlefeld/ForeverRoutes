package route

import "fmt"

// ContinuationTrial records a complete continuation, not a short-window XP rate.
type ContinuationTrial struct {
	Name      string  `json:"name"`
	Lookahead int     `json:"lookahead"`
	Depth     int     `json:"depth"`
	Seconds   float64 `json:"seconds,omitempty"`
	Level     int     `json:"level,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// PlanContinuation is an offline portfolio experiment. Every policy starts from
// the same replayed state and must reach the target. The ordinary 6/3 policy is
// always evaluated first, so adding policies cannot replace it with a slower run.
// It deliberately does not change the latency-sensitive live planner.
func (w *World) PlanContinuation(s *State, target int) (*Result, []ContinuationTrial, error) {
	policies := []ContinuationTrial{
		{Name: "baseline", Lookahead: 6, Depth: 3},
		{Name: "short", Lookahead: 6, Depth: 1},
		{Name: "wide", Lookahead: 12, Depth: 3},
		{Name: "deep", Lookahead: 6, Depth: 6},
	}
	var best *Result
	for i := range policies {
		p := &policies[i]
		r, err := w.PlanAhead(s, PlanOptions{TargetLevel: target, Lookahead: p.Lookahead, Depth: p.Depth}, 1e9)
		if err != nil {
			p.Error = err.Error()
			continue
		}
		p.Level = r.Level
		if r.Level < target {
			p.Error = "target not reached"
			continue
		}
		for _, chain := range w.ClassChains(target) {
			if chain.Tier != "required" || s.Status[chain.Goal.Index] == done {
				continue
			}
			finished := false
			for _, step := range r.Steps {
				finished = finished || step.Op == "turnin" && step.Quest == chain.Goal.ID
			}
			if !finished {
				p.Error = fmt.Sprintf("required class quest %d not finished", chain.Goal.ID)
				break
			}
		}
		if p.Error != "" {
			continue
		}
		p.Seconds = r.Seconds
		if best == nil || r.Seconds < best.Seconds {
			best = r
		}
	}
	if best == nil {
		return nil, policies, fmt.Errorf("no continuation reaches level %d", target)
	}
	return best, policies, nil
}
