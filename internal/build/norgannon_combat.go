package build

// Fill missing combat metadata and retain sourced elite restrictions. Explicit
// client ranks take precedence; curated quest fixes are applied afterwards.
func (b *builder) norgannonCombat() {
	if b.ngn == nil {
		return
	}
	for i := range b.npcOut {
		n := &b.npcOut[i]
		m, ok := b.ngn.NPCs[n.ID]
		if !ok {
			continue
		}
		changed := false
		if n.MinLevel == 0 && m.MinLevel > 0 {
			n.MinLevel = m.MinLevel
			changed = true
		}
		if n.MaxLevel == 0 && m.MaxLevel > 0 {
			n.MaxLevel = m.MaxLevel
			changed = true
		}
		clientRankKnown := b.pdb != nil && b.pdb.NPCs[n.ID] != nil && b.pdb.NPCs[n.ID].Rank != nil
		if !clientRankKnown && m.Rank != nil && *m.Rank >= 1 && *m.Rank <= 3 && (n.Rank == 0 || n.Rank == 4 || n.Rank < *m.Rank) {
			n.Rank = *m.Rank
			changed = true
		}
		if changed {
			n.Source += "+norgannon_combat"
			b.stats["npc_combat_metadata_from_norgannon"]++
		}
	}
}
