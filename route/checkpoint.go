package route

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"
)

// Bump this when simulation semantics change. Data, walks, full combat profiles, constraints and feedback are hashed too.
const TimelineModelRevision = "foreverroutes.timeline-model.v2.3"
const timelineModelRevision = TimelineModelRevision

type OptimizationCheckpoint struct {
	Schema       string          `json:"schema"`
	Context      string          `json:"context"`
	Target       int             `json:"target"`
	Seconds      float64         `json:"seconds"`
	Preparations PreparationPlan `json:"preparations"`
	WandDetour   float64         `json:"wandDetour"`
	Events       []GuideEvent    `json:"events"`
}

func (w *World) optimizationKey(o OptimizeOptions, goals []int) (string, error) {
	required, excluded := slices.Clone(o.RequireDungeons), slices.Clone(o.ExcludeDungeons)
	sort.Strings(required)
	sort.Strings(excluded)
	overrides := w.classOverride
	if len(overrides) == 0 {
		overrides = nil
	}
	v := struct {
		Revision                 string
		Data                     any
		Character                Character
		Options                  Options
		Target                   int
		Goals                    []int
		Required, Excluded       []string
		Overrides                map[int]string
		Feedback                 *CombatFeedback
		FeedbackActive           bool
		ProfileType, ProfileName string
		WalkTable                string
	}{timelineModelRevision, w.Data, w.Char, w.Opt, o.Target, goals, required, excluded, overrides, w.feedback, w.Opt.Feedback.Active(), fmt.Sprintf("%T", w.Opt.Profile), w.Opt.Profile.Name(), w.walkTable.digest()}
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("checkpoint context: %w", err)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func loadCheckpoint(path, key string) (*OptimizationCheckpoint, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c OptimizationCheckpoint
	if err = json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	if c.Schema != "foreverroutes.optimization-checkpoint.v1" || c.Context != key {
		return nil, fmt.Errorf("checkpoint context differs (data, profile, model or selected options); use a new checkpoint path")
	}
	return &c, nil
}

var checkpointMu sync.Mutex

func (w *World) saveCheckpoint(path, key string, events []GuideEvent, p PreparationPlan, wand float64, target int) (retErr error) {
	checkpointMu.Lock()
	defer checkpointMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	// A named exclusive file is portable to Windows. Close before removing; never silently delete another writer's lock.
	lock := path + ".lock"
	until := time.Now().Add(time.Second)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, writeErr := fmt.Fprintf(f, "%d\n", os.Getpid())
			closeErr := f.Close()
			if writeErr != nil || closeErr != nil {
				os.Remove(lock)
				return fmt.Errorf("checkpoint lock write/close: %v %v", writeErr, closeErr)
			}
			break
		}
		if !os.IsExist(err) {
			return err
		}
		if time.Now().After(until) {
			return fmt.Errorf("checkpoint busy: %s", lock)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer func() {
		var err error
		for i := 0; i < 5; i++ {
			err = os.Remove(lock)
			if err == nil || os.IsNotExist(err) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		if retErr == nil {
			retErr = fmt.Errorf("checkpoint lock cleanup: %w", err)
		}
	}()
	before, oldWand := w.Preparations(), w.WandDetour()
	defer func() { w.preparation = before; w.SetWandDetour(oldWand) }()
	w.preparation = p
	w.SetWandDetour(wand)
	r, err := w.FollowGuide(events, target)
	if err != nil || r.Level < target {
		return fmt.Errorf("cannot save an invalid route: %v", err)
	}
	if previous, err := loadCheckpoint(path, key); err != nil {
		return err
	} else if previous != nil {
		if err := w.SetPreparations(previous.Preparations); err != nil {
			return err
		}
		w.SetWandDetour(previous.WandDetour)
		old, err := w.FollowGuide(previous.Events, target)
		if err != nil || old.Level < target {
			return fmt.Errorf("stored checkpoint failed re-evaluation")
		}
		if old.Seconds <= r.Seconds {
			if math.Abs(previous.Seconds-old.Seconds) < .05 {
				return nil
			}
			events, p, wand, r = previous.Events, previous.Preparations, previous.WandDetour, old
		}
	}
	c := OptimizationCheckpoint{"foreverroutes.optimization-checkpoint.v1", key, target, r.Seconds, p, wand, events}
	b, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".timeline-checkpoint-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
