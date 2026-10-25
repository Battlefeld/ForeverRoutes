package build

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
)

func TestPrerequisitesFromBreadcrumbsAndGuides(t *testing.T) {
	src := func() map[string]string { return map[string]string{} }
	b := &builder{stats: map[string]int{}, out: []dataset.Quest{
		{ID: 787, Breadcrumb: 788, Sources: src()},
		{ID: 788, PreAll: []int{787}, Sources: src()},                // a breadcrumb is no prerequisite
		{ID: 871, Sources: src()},                                    // taken together with 5041 by the guide
		{ID: 5041, PreAny: []int{871}, Sources: src()},               // the guide takes it while 871 is open
		{ID: 785, Sources: src()},                                    // no guide does it
		{ID: 786, PreAll: []int{785}, Sources: src()},                // so it is not needed for 786
		{ID: 1688, PreAll: []int{785}, Classes: 256, Sources: src()}, // a class quest keeps its prerequisites
	}, rxpOrder: map[string][]guides.OrderStep{"a.lua": {
		{Op: "accept", Quest: 788}, {Op: "turnin", Quest: 788},
		{Op: "accept", Quest: 871}, {Op: "accept", Quest: 5041}, {Op: "turnin", Quest: 871}, {Op: "turnin", Quest: 5041},
		{Op: "accept", Quest: 786}, {Op: "accept", Quest: 1688},
	}}}
	b.breadcrumbPrereqs()
	b.rxpPrereqs()
	pre := map[int]int{}
	for _, q := range b.out {
		pre[q.ID] = len(q.PreAll) + len(q.PreAny)
	}
	if pre[788] != 0 || pre[5041] != 0 || pre[786] != 0 || pre[1688] != 1 {
		t.Errorf("prerequisites left: %v", pre)
	}
}
