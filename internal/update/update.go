// Package update keeps the route dataset current. It checks every source for a new state (QuestieDB, EverythingQuests
// and cmangos by their newest commit, ForeverDB by its API, WarcraftDB for quests that still lack facts), rebuilds the
// merged quest database and the dataset, and checks the result against the previous one. Revisions are pinned in
// data/sources/lock.json, so every dataset can be rebuilt exactly from the repository (Rebuild).
package update

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/build"
	"github.com/Battlefeld/ForeverRoutes/internal/merge"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/everythingquests"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverchanges"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverdb"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/questie"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/wclbox"
	"github.com/Battlefeld/ForeverRoutes/internal/warcraftdb"
)

// Files in the repository.
const (
	LockFile     = "data/sources/lock.json"
	SnapshotFile = "data/sources/foreverdb-snapshot.json.gz"
	FactsFile    = "data/sources/manual-facts.json"
	OverlayFile  = "data/sources/warcraftdb-forever.json"
	WCLBoxFile   = "data/sources/wclbox-items.json"
	FChangesFile = "data/sources/foreverchanges-quests.json"
	DatasetFile  = "data/routes-data.json.gz"
)

// Upstream repositories and the branch followed.
var repos = map[string][2]string{
	"questie":          {"https://github.com/Questie/QuestieDB", "master"},
	"everythingquests": {"https://github.com/wheelbarrel00/EverythingQuests", "main"},
	"cmangos":          {"https://github.com/cmangos/classic-db", "master"},
	"polestar":         {"https://github.com/fitzforge/polestar", "next"}, // the default branch carries forever-db
	"rxp":              {"https://github.com/RestedXP/RXPGuides", "main"},
}

// Lock pins the state of every source a dataset was built from.
type Lock struct {
	Schema           string `json:"schema"`
	Questie          string `json:"questie"`
	EverythingQuests string `json:"everythingquests"`
	CMangos          string `json:"cmangos"`
	Polestar         string `json:"polestar,omitempty"` // forever-db (Polestar), empty: not used
	RXP              string `json:"rxp,omitempty"`      // RestedXP's guides, empty: not used
	ForeverDBBuild   string `json:"foreverdbBuild"`
	ForeverDBHash    string `json:"foreverdbHash"`    // of the stored snapshot
	ForeverDBFetched string `json:"foreverdbFetched"` // when the API was last asked (RFC 3339)
	Updated          string `json:"updated"`          // when the dataset last changed (RFC 3339)
}

// Options of a run. Root is the repository; Cache keeps downloads between runs.
type Options struct {
	Root             string
	Cache            string
	ForeverDBKey     string        // empty: keep the stored ForeverDB snapshot
	RequireForeverDB bool          // fail without a key (the scheduled run)
	ForeverDBMinAge  time.Duration // do not ask the API more often than this (1 h)
	WarcraftDBBudget int           // WarcraftDB requests per run (150)
	WCLBoxBudget     int           // WCLBox page requests per run (400)
	FChangesBudget   int           // ForeverChanges page requests per run (400)
	Force            bool          // rebuild even when no source changed
	Now              func() time.Time
	HTTP             *http.Client
	Log              func(format string, a ...any)
}

func (o *Options) defaults() {
	if o.Root == "" {
		o.Root = "."
	}
	if o.Cache == "" {
		o.Cache = filepath.Join(o.Root, ".cache")
	}
	if o.ForeverDBMinAge == 0 {
		o.ForeverDBMinAge = time.Hour
	}
	if o.WarcraftDBBudget == 0 {
		o.WarcraftDBBudget = 150
	}
	if o.WCLBoxBudget == 0 {
		o.WCLBoxBudget = 400
	}
	if o.FChangesBudget == 0 {
		o.FChangesBudget = 400
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.HTTP == nil {
		o.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	if o.Log == nil {
		o.Log = func(string, ...any) {}
	}
}

func (o *Options) path(rel string) string { return filepath.Join(o.Root, rel) }

// Report says what a run found and did.
type Report struct {
	Changed  bool     `json:"changed"`  // the dataset changed and was written
	Sources  []string `json:"sources"`  // sources with a new state
	Summary  []string `json:"summary"`  // counts before -> after
	Added    []int    `json:"added"`    // quests new in the dataset
	Removed  []int    `json:"removed"`  // quests gone from the dataset
	Problems []string `json:"problems"` // checks that failed; nothing was written then
}

// Markdown renders the report for a commit message or an issue.
func (r *Report) Markdown() string {
	var b strings.Builder
	if len(r.Problems) > 0 {
		b.WriteString("Checks failed, the dataset was not changed:\n\n")
		for _, p := range r.Problems {
			b.WriteString("- " + p + "\n")
		}
		b.WriteString("\n")
	}
	if len(r.Sources) > 0 {
		b.WriteString("New source states:\n\n")
		for _, s := range r.Sources {
			b.WriteString("- " + s + "\n")
		}
		b.WriteString("\n")
	}
	for _, s := range r.Summary {
		b.WriteString("- " + s + "\n")
	}
	if len(r.Added) > 0 {
		fmt.Fprintf(&b, "- quests added: %s\n", ids(r.Added))
	}
	if len(r.Removed) > 0 {
		fmt.Fprintf(&b, "- quests removed: %s\n", ids(r.Removed))
	}
	return b.String()
}

func ids(l []int) string {
	var parts []string
	for i, id := range l {
		if i == 40 {
			parts = append(parts, fmt.Sprintf("… (%d in all)", len(l)))
			break
		}
		parts = append(parts, fmt.Sprint(id))
	}
	return strings.Join(parts, ", ")
}

// ReadLock reads the lock file.
func ReadLock(root string) (*Lock, error) {
	b, err := os.ReadFile(filepath.Join(root, LockFile))
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	if l.Schema != "foreverroutes.lock.v1" {
		return nil, fmt.Errorf("lock: unknown schema %q", l.Schema)
	}
	return &l, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// head is the newest commit of a branch.
func head(ctx context.Context, repo, branch string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "ls-remote", repo, "refs/heads/"+branch).Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s: %w", repo, err)
	}
	f := strings.Fields(string(out))
	if len(f) < 1 || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(f[0]) {
		return "", fmt.Errorf("git ls-remote %s: no commit for %s", repo, branch)
	}
	return f[0], nil
}

// sparseCheckout checks out only some folders of one commit of a repository (shallow, without other files' contents)
// into cache/<name>-<commit> and returns that folder.
func sparseCheckout(ctx context.Context, cache, name, repo, rev string, paths ...string) (string, error) {
	dir := filepath.Join(cache, name+"-"+rev)
	if _, err := os.Stat(filepath.Join(dir, ".done")); err == nil {
		return dir, nil
	}
	os.RemoveAll(dir)
	steps := [][]string{
		{"init", "-q", dir},
		{"-C", dir, "remote", "add", "origin", repo},
		{"-C", dir, "sparse-checkout", "set", "--no-cone"},
		{"-C", dir, "fetch", "-q", "--depth", "1", "--filter=blob:none", "origin", rev},
		{"-C", dir, "checkout", "-q", "FETCH_HEAD"},
	}
	for _, p := range paths {
		steps[2] = append(steps[2], "/"+p+"/")
	}
	for _, args := range steps {
		if out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	return dir, os.WriteFile(filepath.Join(dir, ".done"), nil, 0o644)
}

// cmangosDump checks out one commit of cmangos classic-db (shallow) and returns its Full_DB dump.
func cmangosDump(ctx context.Context, cache, rev string) (string, error) {
	dir := filepath.Join(cache, "classic-db-"+rev)
	if _, err := os.Stat(filepath.Join(dir, ".done")); err != nil {
		os.RemoveAll(dir)
		for _, args := range [][]string{
			{"init", "-q", dir},
			{"-C", dir, "fetch", "-q", "--depth", "1", repos["cmangos"][0], rev},
			{"-C", dir, "checkout", "-q", "FETCH_HEAD"},
		} {
			if out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
				return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, out)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, ".done"), nil, 0o644); err != nil {
			return "", err
		}
	}
	dumps, _ := filepath.Glob(filepath.Join(dir, "Full_DB", "*.sql.gz"))
	if len(dumps) != 1 {
		return "", fmt.Errorf("cmangos %s: expected one Full_DB dump, found %d", rev, len(dumps))
	}
	return dumps[0], nil
}

// ReadSnapshot reads the stored ForeverDB snapshot.
func ReadSnapshot(path string) (*foreverdb.Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var s foreverdb.Snapshot
	if err := json.NewDecoder(zr).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// encodeSnapshot is the stored form: compact JSON, gzip without a time stamp, so equal data gives equal bytes.
func encodeSnapshot(s *foreverdb.Snapshot) ([]byte, string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(raw); err != nil {
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), hex.EncodeToString(sum[:]), nil
}

// Inputs are the source files a rebuild reads, besides what the lock pins.
type Inputs struct {
	Snapshot *foreverdb.Snapshot
	Overlay  string // WarcraftDB overlay file, empty for none
	WCLBox   string // WCLBox item sources file, empty for none
	FChanges string // ForeverChanges quest summaries file, empty for none
}

// Rebuild makes the merged quest database and the dataset from a lock. It fetches QuestieDB and EverythingQuests at
// the pinned commits and checks out cmangos; nothing else touches the network.
func Rebuild(ctx context.Context, o Options, l *Lock, in Inputs) (*dataset.Dataset, *merge.Doc, error) {
	o.defaults()
	raw, err := questie.Fetch(ctx, o.HTTP, l.Questie, filepath.Join(o.Cache, "questie"))
	if err != nil {
		return nil, nil, err
	}
	cat, err := questie.Build(raw, l.Questie, questie.Options{})
	if err != nil {
		return nil, nil, err
	}
	eraw, err := everythingquests.Fetch(ctx, o.HTTP, l.EverythingQuests)
	if err != nil {
		return nil, nil, err
	}
	eq, err := everythingquests.Build(eraw, l.EverythingQuests)
	if err != nil {
		return nil, nil, err
	}
	var facts merge.Facts
	b, err := os.ReadFile(o.path(FactsFile))
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(b, &facts); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", FactsFile, err)
	}
	doc, err := merge.Merge(merge.Inputs{Questie: cat, ForeverDB: in.Snapshot, Addon: eq, Facts: &facts})
	if err != nil {
		return nil, nil, err
	}
	dump, err := cmangosDump(ctx, o.Cache, l.CMangos)
	if err != nil {
		return nil, nil, err
	}
	mergedPath := filepath.Join(o.Cache, "merged-quests.json")
	if err := os.MkdirAll(o.Cache, 0o755); err != nil {
		return nil, nil, err
	}
	mb, err := json.Marshal(doc)
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(mergedPath, mb, 0o644); err != nil {
		return nil, nil, err
	}
	bin := build.Inputs{Merged: mergedPath, CMangos: dump, CMangosRevision: l.CMangos, WarcraftDB: in.Overlay, WCLBox: in.WCLBox, ForeverChanges: in.FChanges}
	if l.Polestar != "" {
		if bin.Polestar, err = polestar.Fetch(ctx, o.HTTP, o.Cache, l.Polestar); err != nil {
			return nil, nil, err
		}
		bin.PolestarRevision = l.Polestar
		if bin.GuidesPolestar, err = sparseCheckout(ctx, o.Cache, "polestar-guides", repos["polestar"][0], l.Polestar, "guides/forever"); err != nil {
			return nil, nil, err
		}
		bin.GuidesPolestar = filepath.Join(bin.GuidesPolestar, "guides", "forever")
	}
	if l.RXP != "" {
		dir, err := sparseCheckout(ctx, o.Cache, "rxp", repos["rxp"][0], l.RXP, "Guides/forever", "Guides/Forever")
		if err != nil {
			return nil, nil, err
		}
		bin.GuidesRXP, bin.GuidesRXPRevision = filepath.Join(dir, "Guides"), l.RXP
	}
	d, err := build.Build(bin)
	if err != nil {
		return nil, nil, err
	}
	return d, doc, nil
}

// Run checks the sources and rebuilds when one changed. The files are only written when every check passes.
func Run(ctx context.Context, o Options) (*Report, error) {
	o.defaults()
	rep := &Report{}
	old, err := ReadLock(o.Root)
	if err != nil {
		return nil, err
	}
	next := *old
	for name, r := range repos {
		h, err := head(ctx, r[0], r[1])
		if err != nil {
			return nil, err
		}
		cur := map[string]*string{"questie": &next.Questie, "everythingquests": &next.EverythingQuests, "cmangos": &next.CMangos, "polestar": &next.Polestar, "rxp": &next.RXP}[name]
		if h != *cur {
			rep.Sources = append(rep.Sources, fmt.Sprintf("%s %s → %s", name, short(*cur), short(h)))
			*cur = h
		}
	}
	snap, err := ReadSnapshot(o.path(SnapshotFile))
	if err != nil {
		return nil, err
	}
	var newSnap []byte
	now := o.Now().UTC()
	switch {
	case o.ForeverDBKey == "" && o.RequireForeverDB:
		return nil, fmt.Errorf("FOREVERDB_API_KEY is not set")
	case o.ForeverDBKey == "":
		o.Log("ForeverDB: no key, keeping the stored snapshot")
	default:
		last, _ := time.Parse(time.RFC3339, old.ForeverDBFetched)
		if !o.Force && now.Sub(last) < o.ForeverDBMinAge {
			o.Log("ForeverDB: asked %s ago, not again yet", now.Sub(last).Round(time.Minute))
			break
		}
		c, err := foreverdb.NewClient(o.ForeverDBKey)
		if err != nil {
			return nil, err
		}
		fresh, err := c.Fetch(ctx)
		if err != nil {
			return nil, err
		}
		next.ForeverDBFetched = now.Format(time.RFC3339)
		b, sum, err := encodeSnapshot(fresh)
		if err != nil {
			return nil, err
		}
		o.Log("ForeverDB: %d quests, %d calls", len(fresh.Quests), c.Calls)
		if sum != old.ForeverDBHash {
			rep.Sources = append(rep.Sources, fmt.Sprintf("ForeverDB build %s, %d quests (data changed)", fresh.Build(), len(fresh.Quests)))
			snap, newSnap, next.ForeverDBHash, next.ForeverDBBuild = fresh, b, sum, fresh.Build()
		}
	}
	if len(rep.Sources) == 0 && !o.Force {
		// only the time of the ForeverDB check changed
		if next.ForeverDBFetched != old.ForeverDBFetched {
			if err := writeJSON(o.path(LockFile), next); err != nil {
				return nil, err
			}
		}
		return rep, nil
	}
	// first pass, then WarcraftDB and WCLBox for what is still missing, then the final pass
	in := Inputs{Snapshot: snap, Overlay: o.path(OverlayFile), WCLBox: o.path(WCLBoxFile), FChanges: o.path(FChangesFile)}
	d, doc, err := Rebuild(ctx, o, &next, in)
	if err != nil {
		return nil, err
	}
	ov, fetched, err := fillFromWarcraftDB(o, d)
	if err != nil {
		return nil, err
	}
	if fetched > 0 {
		in.Overlay = filepath.Join(o.Cache, "warcraftdb-forever.json")
		if err := writeJSON(in.Overlay, ov); err != nil {
			return nil, err
		}
		rep.Sources = append(rep.Sources, fmt.Sprintf("WarcraftDB: %d quests fetched", fetched))
		if d, doc, err = Rebuild(ctx, o, &next, in); err != nil {
			return nil, err
		}
	}
	wcl, wfetched, err := fillFromWCLBox(o, d)
	if err != nil {
		return nil, err
	}
	fch, cfetched, err := fillFromForeverChanges(o, d)
	if err != nil {
		return nil, err
	}
	if wfetched+cfetched > 0 {
		if wfetched > 0 {
			in.WCLBox = filepath.Join(o.Cache, "wclbox-items.json")
			if err := writeJSON(in.WCLBox, wcl); err != nil {
				return nil, err
			}
			rep.Sources = append(rep.Sources, fmt.Sprintf("WCLBox: %d item pages read", wfetched))
		}
		if cfetched > 0 {
			in.FChanges = filepath.Join(o.Cache, "foreverchanges-quests.json")
			if err := writeJSON(in.FChanges, fch); err != nil {
				return nil, err
			}
			rep.Sources = append(rep.Sources, fmt.Sprintf("ForeverChanges: %d quest pages read", cfetched))
		}
		if d, doc, err = Rebuild(ctx, o, &next, in); err != nil {
			return nil, err
		}
	}
	// what was read is kept even when the dataset stays the same, so it is not asked for again
	persist := func() error {
		if fetched > 0 {
			if err := writeJSON(o.path(OverlayFile), ov); err != nil {
				return err
			}
		}
		if cfetched > 0 {
			if err := writeJSON(o.path(FChangesFile), fch); err != nil {
				return err
			}
		}
		if wfetched > 0 {
			return writeJSON(o.path(WCLBoxFile), wcl)
		}
		return nil
	}
	prev, err := dataset.Load(o.path(DatasetFile))
	if err != nil {
		return nil, err
	}
	compare(rep, prev, d, doc)
	if len(rep.Problems) > 0 {
		return rep, nil
	}
	// only content counts: new commits upstream that change nothing here leave the repository as it is
	if sameContent(prev, d) {
		rep.Sources = append(rep.Sources, "(no change to the dataset)")
		if err := persist(); err != nil {
			return nil, err
		}
		if next.ForeverDBFetched != old.ForeverDBFetched {
			kept := *old
			kept.ForeverDBFetched = next.ForeverDBFetched
			return rep, writeJSON(o.path(LockFile), kept)
		}
		return rep, nil
	}
	var newDS bytes.Buffer
	if err := dataset.Encode(&newDS, d); err != nil {
		return nil, err
	}
	rep.Changed = true
	next.Updated = now.Format(time.RFC3339)
	if err := os.WriteFile(o.path(DatasetFile), newDS.Bytes(), 0o644); err != nil {
		return nil, err
	}
	if newSnap != nil {
		if err := os.WriteFile(o.path(SnapshotFile), newSnap, 0o644); err != nil {
			return nil, err
		}
	}
	if err := persist(); err != nil {
		return nil, err
	}
	return rep, writeJSON(o.path(LockFile), next)
}

// sameContent compares two datasets without their source revisions.
func sameContent(a, b *dataset.Dataset) bool {
	ca, cb := *a, *b
	ca.Sources, cb.Sources = nil, nil
	ja, _ := json.Marshal(ca)
	jb, _ := json.Marshal(cb)
	return bytes.Equal(ja, jb)
}

func short(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	if s == "" {
		return "–"
	}
	return s
}

// compare checks the new dataset against the previous one and fills the summary. A large loss means a source broke.
func compare(rep *Report, prev, next *dataset.Dataset, doc *merge.Doc) {
	s := doc.Stats
	check := func(ok bool, format string, a ...any) {
		if !ok {
			rep.Problems = append(rep.Problems, fmt.Sprintf(format, a...))
		}
	}
	check(s["identified_quests"] >= 2000, "merged database: only %d quests (at least 2000 expected)", s["identified_quests"])
	check(s["foreverdb_quests"] >= 1000, "merged database: only %d ForeverDB quests (at least 1000 expected)", s["foreverdb_quests"])
	check(s["npc_hp_classic_references"] >= 4500, "merged database: only %d NPCs with health (at least 4500 expected)", s["npc_hp_classic_references"])
	pq, nq := len(prev.Quests), len(next.Quests)
	check(nq*100 >= pq*97, "dataset: quests fell from %d to %d (more than 3 %%)", pq, nq)
	check(len(next.NPCs)*100 >= len(prev.NPCs)*95, "dataset: NPCs fell from %d to %d (more than 5 %%)", len(prev.NPCs), len(next.NPCs))
	check(next.Stats["map_transforms_uimap"] >= 25, "dataset: only %d map transforms (positions would be lost)", next.Stats["map_transforms_uimap"])
	check(len(next.Travel.FlightMasters) >= 50, "dataset: only %d flight masters", len(next.Travel.FlightMasters))
	check(next.Stats["quests_unsupported"] <= prev.Stats["quests_unsupported"]+50, "dataset: unsupported quests rose from %d to %d",
		prev.Stats["quests_unsupported"], next.Stats["quests_unsupported"])
	line := func(name string, a, b int) {
		if a != b {
			rep.Summary = append(rep.Summary, fmt.Sprintf("%s: %d → %d", name, a, b))
		}
	}
	line("quests", pq, nq)
	line("NPCs", len(prev.NPCs), len(next.NPCs))
	line("objects", len(prev.Objects), len(next.Objects))
	for _, k := range []string{"quests_unsupported", "xp_unknown", "objectives_count_unknown", "npcs_without_positions"} {
		line(k, prev.Stats[k], next.Stats[k])
	}
	before, after := map[int]bool{}, map[int]bool{}
	for _, q := range prev.Quests {
		before[q.ID] = true
	}
	for _, q := range next.Quests {
		after[q.ID] = true
		if !before[q.ID] {
			rep.Added = append(rep.Added, q.ID)
		}
	}
	for id := range before {
		if !after[id] {
			rep.Removed = append(rep.Removed, id)
		}
	}
	sort.Ints(rep.Added)
	sort.Ints(rep.Removed)
}

// NeedsFacts lists the quests of a dataset whose objectives, XP or positions are not known.
func NeedsFacts(d *dataset.Dataset) []int {
	spawned := map[int]bool{}
	for _, n := range d.NPCs {
		if len(n.Spawns) > 0 {
			spawned[n.ID] = true
		}
	}
	located := func(refs []dataset.Ref, at *dataset.Point) bool {
		if at != nil {
			return true
		}
		for _, r := range refs {
			if r.Kind == "npc" && spawned[r.ID] {
				return true
			}
		}
		return false
	}
	var out []int
	for _, q := range d.Quests {
		if strings.Contains(q.Unsupported, "unknown") || q.Sources["xp"] == "unknown" || !located(q.Start, q.StartAt) || !located(q.End, q.EndAt) {
			out = append(out, q.ID)
		}
	}
	sort.Ints(out)
	return out
}

// fillFromForeverChanges reads the summary of every quest the dataset still cannot use, unless it is read already.
func fillFromForeverChanges(o Options, d *dataset.Dataset) (*foreverchanges.Overlay, int, error) {
	ov := &foreverchanges.Overlay{Schema: "foreverroutes.foreverchanges.v1", Source: foreverchanges.Credit, Quests: map[int]foreverchanges.Quest{}}
	if b, err := os.ReadFile(o.path(FChangesFile)); err == nil {
		if err := json.Unmarshal(b, ov); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", FChangesFile, err)
		}
		if ov.Quests == nil {
			ov.Quests = map[int]foreverchanges.Quest{}
		}
	}
	var ids []int
	for _, q := range d.Quests {
		if q.Unsupported == "" {
			continue
		}
		if _, have := ov.Quests[q.ID]; !have {
			ids = append(ids, q.ID)
		}
	}
	sort.Ints(ids)
	c := &foreverchanges.Client{HTTP: &http.Client{Timeout: 30 * time.Second}, Cache: filepath.Join(o.Cache, "foreverchanges"), Delay: time.Second}
	fetched := 0
	for _, id := range ids {
		if c.Calls >= o.FChangesBudget {
			break
		}
		q, err := c.Quest(id)
		if foreverchanges.IsNotFound(err) {
			ov.Quests[id] = foreverchanges.Quest{ID: id, URL: foreverchanges.Base + "/quest/" + fmt.Sprint(id)}
			fetched++
			continue
		}
		if err != nil {
			o.Log("ForeverChanges: skipping quest %d: %v", id, err)
			if strings.Contains(err.Error(), "HTTP 429") || strings.Contains(err.Error(), "HTTP 403") {
				break
			}
			continue
		}
		ov.Quests[id] = q
		fetched++
	}
	if fetched > 0 {
		ov.Fetched = o.Now().UTC().Format("2006-01-02")
	}
	return ov, fetched, nil
}

// fillFromWCLBox reads the item pages of the items the dataset's quests need and no source explains yet, within the
// request budget. An item stays in the file even when the page names no source, so it is not asked for twice.
func fillFromWCLBox(o Options, d *dataset.Dataset) (*wclbox.Overlay, int, error) {
	ov := &wclbox.Overlay{Schema: "foreverroutes.wclbox.v1", Source: wclbox.Credit, Items: map[int]wclbox.Item{}}
	if b, err := os.ReadFile(o.path(WCLBoxFile)); err == nil {
		if err := json.Unmarshal(b, ov); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", WCLBoxFile, err)
		}
		if ov.Items == nil {
			ov.Items = map[int]wclbox.Item{}
		}
	}
	if ov.Quests == nil {
		ov.Quests = map[int]wclbox.Quest{}
	}
	need := map[int]bool{}
	for _, q := range d.Quests {
		for _, id := range q.NeedItems {
			need[id] = true
		}
	}
	ids := make([]int, 0, len(need))
	for id := range need {
		if _, have := ov.Items[id]; !have {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	c := &wclbox.Client{HTTP: &http.Client{Timeout: 30 * time.Second}, Cache: filepath.Join(o.Cache, "wclbox"), Delay: time.Second}
	fetched := 0
	for _, id := range ids {
		if c.Calls >= o.WCLBoxBudget {
			break
		}
		it, err := c.Item(id)
		if wclbox.IsNotFound(err) {
			ov.Items[id] = wclbox.Item{ID: id, URL: wclbox.Base + "/en/wupin/" + fmt.Sprint(id)}
			fetched++
			continue
		}
		if err != nil {
			o.Log("WCLBox: stopping: %v", err)
			break
		}
		ov.Items[id] = it
		fetched++
	}
	// quest pages: for quests still without a source for an item, or whose item the build only assumes is handed over
	var quests []int
	for _, q := range d.Quests {
		if _, have := ov.Quests[q.ID]; have {
			continue
		}
		if strings.Contains(q.Unsupported, "item") || strings.Contains(q.Unsupported, "objectives unknown") ||
			strings.Contains(q.Sources["objectives"], "assumed") {
			quests = append(quests, q.ID)
		}
	}
	sort.Ints(quests)
	for _, id := range quests {
		if c.Calls >= o.WCLBoxBudget {
			break
		}
		q, err := c.Quest(id)
		if wclbox.IsNotFound(err) {
			ov.Quests[id] = wclbox.Quest{ID: id, URL: wclbox.Base + "/en/renwu/" + fmt.Sprint(id)}
			fetched++
			continue
		}
		if err != nil {
			o.Log("WCLBox: stopping: %v", err)
			break
		}
		ov.Quests[id] = q
		fetched++
	}
	if fetched > 0 {
		ov.Fetched = o.Now().UTC().Format("2006-01-02")
		if c.Build != "" {
			ov.Build = c.Build
		}
	}
	return ov, fetched, nil
}

// fillFromWarcraftDB fetches quests that still lack facts and are not in the overlay yet (or only as missing), within
// the request budget.
func fillFromWarcraftDB(o Options, d *dataset.Dataset) (*warcraftdb.Overlay, int, error) {
	ov, err := warcraftdb.Load(o.path(OverlayFile))
	if err != nil {
		return nil, 0, err
	}
	if ov.Quests == nil {
		ov.Quests = map[int]warcraftdb.Quest{}
	}
	missing := map[int]bool{}
	for _, id := range ov.Missing {
		missing[id] = true
	}
	c := &warcraftdb.Client{HTTP: &http.Client{Timeout: 30 * time.Second}, Cache: filepath.Join(o.Cache, "warcraftdb"), Delay: 1500 * time.Millisecond}
	fetched := 0
	for _, id := range NeedsFacts(d) {
		if _, have := ov.Quests[id]; have || missing[id] {
			continue
		}
		if c.Calls >= o.WarcraftDBBudget {
			break
		}
		q, err := c.Quest(id)
		if warcraftdb.IsNotFound(err) {
			missing[id] = true
			fetched++
			continue
		}
		if err != nil {
			o.Log("WarcraftDB: stopping: %v", err)
			break
		}
		ov.Quests[id] = q
		fetched++
	}
	if fetched > 0 {
		ov.Missing = ov.Missing[:0]
		for id := range missing {
			ov.Missing = append(ov.Missing, id)
		}
		sort.Ints(ov.Missing)
		ov.Fetched = o.Now().UTC().Format("2006-01-02")
	}
	return ov, fetched, nil
}
