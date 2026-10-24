// Package dataset is the compact, versioned quest world the route engine plans with. It is built by
// `foreverroutes build-data` from the merged quest database (ForeverDB, QuestieDB, EverythingQuests, WarcraftDB) and the
// cmangos classic world database, and shipped with the module (data/routes-data.json.gz).
//
// Every value keeps the source it came from; nothing is invented. A quest the engine cannot model yet is kept but
// marked unsupported with the reason, so the planner never relies on a guess.
package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Schema is the format version of the file; the engine refuses other versions.
const Schema = "foreverroutes.dataset.v1"

// Point is a position in world coordinates (yards) on one continent map (0 Eastern Kingdoms, 1 Kalimdor).
type Point struct {
	Map int     `json:"m"`
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
}

// Ref names who starts or ends a quest: an NPC, an object or an item.
type Ref struct {
	Kind string `json:"k"` // npc | object | item
	ID   int    `json:"id"`
}

// Drop is where a quest item comes from and how likely (percent per kill or per use).
type Drop struct {
	NPC     int     `json:"npc,omitempty"`
	Object  int     `json:"obj,omitempty"`
	Chance  float64 `json:"p"`
	Unknown bool    `json:"unknown,omitempty"` // a source names the creature but no chance: Chance is 0, not a value
}

// Objective is one requirement of a quest.
type Objective struct {
	Kind    string `json:"k"` // kill | item | object
	ID      int    `json:"id"`
	Count   int    `json:"n"`
	Source  string `json:"src"`
	Drops   []Drop `json:"drops,omitempty"`
	Vendors []int  `json:"vendors,omitempty"` // NPCs that sell the item
}

// Quest is one quest with everything the route engine needs.
type Quest struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Level       int               `json:"lvl"`               // quest level (XP decay)
	MinLevel    int               `json:"min"`               // level needed to accept
	Races       int               `json:"races,omitempty"`   // bit mask, 0 = all
	Classes     int               `json:"classes,omitempty"` // bit mask, 0 = all
	XP          int               `json:"xp"`
	Start       []Ref             `json:"start"`
	End         []Ref             `json:"end"`
	StartAt     *Point            `json:"startAt,omitempty"`
	StartDrops  []Drop            `json:"startDrops,omitempty"` // started by an item: where the item drops
	EndAt       *Point            `json:"endAt,omitempty"`
	PreAll      []int             `json:"preAll,omitempty"`
	PreAny      []int             `json:"preAny,omitempty"`
	Exclusive   int               `json:"excl,omitempty"` // exclusive group: only one quest of a positive group
	Breadcrumb  int               `json:"breadcrumb,omitempty"`
	Repeatable  bool              `json:"repeat,omitempty"`
	Type        string            `json:"type,omitempty"` // dungeon, raid or pvp (Forever client quest type); empty = open world
	Objectives  []Objective       `json:"obj,omitempty"`
	Unsupported string            `json:"unsupported,omitempty"`
	NeedItems   []int             `json:"needItems,omitempty"` // items the quest needs whose source no data has yet
	Sources     map[string]string `json:"src"`                 // field -> source label
}

// NPC is a creature: quest giver, target or grind mob.
type NPC struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	MinLevel int     `json:"min"`
	MaxLevel int     `json:"max"`
	HP       int     `json:"hp"`
	Dmg      float64 `json:"dmg,omitempty"`  // average melee hit (cmangos); 0 = unknown
	Rank     int     `json:"rank,omitempty"` // 0 normal, 1 elite, 2 rare elite, 3 boss, 4 rare
	Respawn  int     `json:"respawn,omitempty"`
	Hostile  string  `json:"hostile,omitempty"` // A, H, AH: which factions it attacks; empty = unknown
	Spawns   []Point `json:"spawns,omitempty"`
	Source   string  `json:"src"`
}

// Object is a game object: chests, herbs, quest objects.
type Object struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Spawns []Point `json:"spawns,omitempty"`
}

// Source is one input of the build with its pinned revision.
type Source struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Revision string `json:"revision,omitempty"`
	License  string `json:"license,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Node is a travel point: a flight master or a transport dock.
type Node struct {
	NPC     int    `json:"npc"`
	Name    string `json:"name"`
	Faction string `json:"faction"` // A or H: who may use it; empty: both
	At      Point  `json:"at"`
}

// Link is a fixed connection between two docks (zeppelin, boat) with its usual time including the wait.
type Link struct {
	From    int     `json:"from"` // NPC of the dock
	To      int     `json:"to"`
	Seconds float64 `json:"seconds"`
	Name    string  `json:"name"`
}

// Travel holds what makes long distances shorter.
type Travel struct {
	FlightMasters []Node `json:"flightMasters"`
	Inns          []Node `json:"inns"` // innkeepers: where the hearthstone can be bound
	Docks         []Node `json:"docks"`
	Links         []Link `json:"links"`
}

// Dataset is the whole file.
type Dataset struct {
	Schema  string   `json:"schema"`
	Build   string   `json:"build"`
	Sources []Source `json:"sources"`
	// LevelXP[i] is the XP needed from level i+1 to i+2.
	LevelXP []int          `json:"levelXp"`
	Quests  []Quest        `json:"quests"`
	NPCs    []NPC          `json:"npcs"`
	Objects []Object       `json:"objects"`
	Travel  Travel         `json:"travel"`
	Stats   map[string]int `json:"stats"`
}

// Load reads a dataset file (plain or gzip JSON) and checks its schema.
func Load(path string) (*Dataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Read decodes a dataset from r; gzip is detected by its magic bytes.
func Read(r io.Reader) (*Dataset, error) {
	br := &peekReader{r: r}
	head, err := br.peek(2)
	if err != nil {
		return nil, err
	}
	var src io.Reader = br
	if head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		src = gz
	}
	var d Dataset
	if err := json.NewDecoder(src).Decode(&d); err != nil {
		return nil, fmt.Errorf("dataset: %w", err)
	}
	if d.Schema != Schema {
		return nil, fmt.Errorf("dataset: schema %q, want %q", d.Schema, Schema)
	}
	return &d, nil
}

// Write stores the dataset as gzip JSON with a stable field order.
func Write(path string, d *Dataset) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := Encode(f, d); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Encode writes the dataset as gzip JSON; equal datasets give equal bytes.
func Encode(w io.Writer, d *Dataset) error {
	gz, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err := json.NewEncoder(gz).Encode(d); err != nil {
		return err
	}
	return gz.Close()
}

type peekReader struct {
	r   io.Reader
	buf []byte
}

func (p *peekReader) peek(n int) ([]byte, error) {
	for len(p.buf) < n {
		b := make([]byte, n-len(p.buf))
		k, err := p.r.Read(b)
		p.buf = append(p.buf, b[:k]...)
		if err != nil {
			if len(p.buf) >= n {
				break
			}
			return nil, err
		}
	}
	return p.buf[:n], nil
}

func (p *peekReader) Read(b []byte) (int, error) {
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}
	return p.r.Read(b)
}
