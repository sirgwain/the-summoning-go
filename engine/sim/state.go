package sim

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The world as plain data, for saves (docs/design.md, Saves): everything that changes, with
// templates and things named by id, so that restoring it on the same pack gives the same world.

// StateFormat is the version of State this package writes and reads.
const StateFormat = 1

// State is the world's state as plain data.
type State struct {
	Format int `json:"format"`
	// Pack and Version name the pack the world ran, for a warning when they differ.
	Pack    string `json:"pack"`
	Version string `json:"version"`
	Step    int64  `json:"step"`
	Current string `json:"current"`
	// Status is the step of the next status tick; RNG the random source's state.
	Status   int64          `json:"status"`
	RNG      []byte         `json:"rng"`
	Counters map[string]int `json:"counters,omitempty"`
	// Shared are the shared charges of the items that have them, by item.
	Shared  map[string]int     `json:"shared,omitempty"`
	Records map[string]*Record `json:"records,omitempty"`
	// Log is the conversation log.
	Log    []*Logged     `json:"log,omitempty"`
	Player *PlayerState  `json:"player,omitempty"`
	Levels []*LevelState `json:"levels"`
}

// PlayerState is the player with its template's id and what it keeps to itself.
type PlayerState struct {
	Player
	Template    string       `json:"template"`
	Want        int          `json:"want"`
	Next        int64        `json:"next"`
	Vision      content.Cell `json:"vision"`
	StatusTicks int          `json:"status_ticks"`
	PoseLeft    int          `json:"pose_left"`
}

// LevelState is a level entered.
type LevelState struct {
	ID       string          `json:"id"`
	Grids    [2]content.Grid `json:"grids"`
	Features []FeatureState  `json:"features,omitempty"`
	Things   []ThingState    `json:"things,omitempty"`
	Flights  []FlightState   `json:"flights,omitempty"`
	Objects  []CellObjects   `json:"objects,omitempty"`
	Visible  []bool          `json:"visible,omitempty"`
	Seen     []bool          `json:"seen,omitempty"`
}

// FeatureState is a feature with the step its door moves next.
type FeatureState struct {
	Feature
	Next int64 `json:"next"`
}

// ThingState is a thing with its template's id and what it keeps to itself.
type ThingState struct {
	Thing
	Template  string `json:"template"`
	WalkDir   int    `json:"walk_dir"`
	WalkCount int    `json:"walk_count"`
	Next      int64  `json:"next"`
	Casting   int    `json:"casting,omitempty"`
	AnimCount int    `json:"anim_count,omitempty"`
	AnimStep  int    `json:"anim_step,omitempty"`
	Spin      int    `json:"spin,omitempty"`
	SpinNext  int64  `json:"spin_next,omitempty"`
}

// FlightState is a thing in flight with its thrower's id and what it keeps to itself.
type FlightState struct {
	Flight
	Thrower string `json:"thrower,omitempty"`
	Waited  bool   `json:"waited,omitempty"`
	Next    int64  `json:"next"`
	Ending  int    `json:"ending,omitempty"`
}

// CellObjects are the objects on a cell's floor, bottom first.
type CellObjects struct {
	Cell    content.Cell `json:"cell"`
	Objects []*Object    `json:"objects"`
}

// State returns the world's state, or an error while a conversation is open, unless it waits on
// a menu: the state then leaves the conversation out.
func (w *World) State() (*State, error) {
	if t := w.talk; t != nil && (t.Line == nil || t.Line.Kind != LineMenu) {
		return nil, fmt.Errorf("the world cannot be saved during a conversation")
	}
	rng, err := w.src.MarshalBinary()
	if err != nil {
		return nil, err
	}
	s := &State{Format: StateFormat, Pack: w.pack.Manifest.ID, Version: w.pack.Manifest.Version,
		Step: w.step, Current: w.current, Status: w.status, RNG: rng, Counters: w.counters, Shared: maps.Clone(w.shared), Records: w.records, Log: w.log}
	if p := w.player; p != nil {
		s.Player = &PlayerState{Player: *p, Template: p.Template.ID, Want: p.want, Next: p.next, Vision: p.vision,
			StatusTicks: p.statusTicks, PoseLeft: p.poseLeft}
	}
	for _, id := range sortedKeys(w.levels) {
		l := w.levels[id]
		ls := &LevelState{ID: id, Grids: [2]content.Grid{l.Floor, l.Wall}, Visible: l.visible, Seen: l.seen}
		for _, f := range l.Features {
			ls.Features = append(ls.Features, FeatureState{Feature: *f, Next: f.next})
		}
		for _, t := range l.Things {
			ls.Things = append(ls.Things, ThingState{Thing: *t, Template: t.Template.ID, WalkDir: t.walkDir, WalkCount: t.walkCount,
				Next: t.next, Casting: t.casting, AnimCount: t.animCount, AnimStep: t.animStep, Spin: t.spin, SpinNext: t.spinNext})
		}
		for _, f := range l.Flights {
			fs := FlightState{Flight: *f, Waited: f.waited, Next: f.next}
			if f.Thrower != nil {
				fs.Thrower = f.Thrower.ID
			}
			if f.Spell != nil {
				fs.Ending = f.Spell.ending
			}
			ls.Flights = append(ls.Flights, fs)
		}
		for c, objs := range l.Objects {
			ls.Objects = append(ls.Objects, CellObjects{Cell: c, Objects: objs})
		}
		slices.SortFunc(ls.Objects, func(a, b CellObjects) int {
			if a.Cell[0] != b.Cell[0] {
				return a.Cell[0] - b.Cell[0]
			}
			return a.Cell[1] - b.Cell[1]
		})
		s.Levels = append(s.Levels, ls)
	}
	return s, nil
}

// Restore returns the world s describes, running pack p. The pack should be the one the state
// was saved with; templates it no longer has are an error.
func Restore(p *content.Pack, s *State) (*World, error) {
	if s.Format != StateFormat {
		return nil, fmt.Errorf("save format %d, want %d", s.Format, StateFormat)
	}
	src := &rand.PCG{}
	if err := src.UnmarshalBinary(s.RNG); err != nil {
		return nil, fmt.Errorf("random source: %w", err)
	}
	w := &World{pack: p, src: src, rng: rand.New(src), levels: map[string]*Level{}, rules: rulesFor(p),
		step: s.Step, current: s.Current, status: s.Status, records: s.Records, counters: s.Counters, shared: s.Shared}
	if w.shared == nil {
		w.shared = map[string]int{}
	}
	if w.records == nil {
		w.records = map[string]*Record{}
	}
	if w.counters == nil {
		w.counters = map[string]int{}
	}
	w.log = w.filterLog(s.Log)
	for _, id := range sortedKeys(p.Items) {
		if p.Items[id].Random {
			w.randomItems = append(w.randomItems, id)
		}
	}
	thing := func(id string) (*content.Thing, error) {
		t := p.Things[id]
		if t == nil {
			return nil, fmt.Errorf("the pack has no thing %q", id)
		}
		return t, nil
	}
	for _, ls := range s.Levels {
		def := p.Levels[ls.ID]
		if def == nil {
			return nil, fmt.Errorf("the pack has no level %q", ls.ID)
		}
		l := &Level{ID: ls.ID, Def: def, Floor: ls.Grids[0], Wall: ls.Grids[1], Objects: map[content.Cell][]*Object{},
			byID: map[string]*Feature{}, byCell: map[content.Cell]*Feature{}, visible: ls.Visible, seen: ls.Seen}
		if len(l.Floor) != def.Size[0] || len(l.Wall) != def.Size[0] {
			return nil, fmt.Errorf("level %s: the saved grids do not fit its size", ls.ID)
		}
		if l.visible != nil && len(l.visible) != def.Size[0]*def.Size[1] {
			return nil, fmt.Errorf("level %s: the saved visibility does not fit its size", ls.ID)
		}
		l.takeShapes(p.Walls[def.Walls])
		for _, fs := range ls.Features {
			f := fs.Feature
			f.next = fs.Next
			l.Features = append(l.Features, &f)
			l.byID[f.ID], l.byCell[f.Cell] = &f, &f
		}
		things := map[string]*Thing{}
		for _, ts := range ls.Things {
			tm, err := thing(ts.Template)
			if err != nil {
				return nil, fmt.Errorf("level %s: %w", ls.ID, err)
			}
			t := ts.Thing
			t.Template, t.walkDir, t.walkCount, t.next = tm, ts.WalkDir, ts.WalkCount, ts.Next
			t.casting, t.animCount, t.animStep, t.spin, t.spinNext = ts.Casting, ts.AnimCount, ts.AnimStep, ts.Spin, ts.SpinNext
			l.Things = append(l.Things, &t)
			things[t.ID] = &t
		}
		for _, fs := range ls.Flights {
			f := fs.Flight
			f.Thrower, f.waited, f.next = things[fs.Thrower], fs.Waited, fs.Next
			if f.Spell != nil {
				sp := *f.Spell
				sp.ending = fs.Ending
				f.Spell = &sp
			}
			l.Flights = append(l.Flights, &f)
		}
		for _, co := range ls.Objects {
			l.Objects[co.Cell] = co.Objects
		}
		w.levels[ls.ID] = l
	}
	if w.levels[w.current] == nil {
		return nil, fmt.Errorf("the current level %q was not saved", w.current)
	}
	if ps := s.Player; ps != nil {
		tm, err := thing(ps.Template)
		if err != nil {
			return nil, err
		}
		pl := ps.Player
		pl.Template, pl.want, pl.next, pl.vision, pl.statusTicks, pl.poseLeft = tm, ps.Want, ps.Next, ps.Vision, ps.StatusTicks, ps.PoseLeft
		if pl.Magic.Memorized == nil {
			pl.Magic.Memorized = map[string]int{}
		}
		w.player = &pl
	}
	return w, nil
}
