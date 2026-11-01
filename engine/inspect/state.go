package inspect

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// WorldState is the world's state as plain data in the pack's terms, for dumps that tests and
// tools read (docs/design.md, Testing).
type WorldState struct {
	Step    int64  `json:"step"`
	Current string `json:"current"`
	// Player is the player, if the pack has one.
	Player *PlayerState `json:"player,omitempty"`
	// Talk is the open conversation, if one is; Counters the story's counters; Records what the
	// world keeps about thing templates.
	Talk     *TalkState             `json:"talk,omitempty"`
	Counters map[string]int         `json:"counters,omitempty"`
	Records  map[string]*sim.Record `json:"records,omitempty"`
	// Levels are the levels entered so far, by id.
	Levels map[string]*LevelState `json:"levels"`
}

// PlayerState is the player's state.
type PlayerState struct {
	Cell        content.Cell `json:"cell"`
	At          [2]int       `json:"at"`
	Facing      int          `json:"facing"`
	HitPoints   int          `json:"hit_points"`
	MagicPoints int          `json:"magic_points"`
	Level       int          `json:"level"`
	Experience  int          `json:"experience"`
	ArmorClass  int          `json:"armor_class,omitempty"`
	Skills      []sim.Skill  `json:"skills"`
	Poisoned    bool         `json:"poisoned,omitempty"`
	Status      sim.Status   `json:"status"`
	Timed       []int        `json:"timed"`
	Dead        bool         `json:"dead,omitempty"`
	// Hand is the object held; Slots the inventory's filled slots by name.
	Hand  *content.Object           `json:"hand,omitempty"`
	Slots map[string]content.Object `json:"slots,omitempty"`
	// Magic is the player's spells; Look a look taken by a Shape Change.
	Magic sim.Magic `json:"magic"`
	Look  string    `json:"look,omitempty"`
}

// TalkState is an open conversation.
type TalkState struct {
	Conversation string    `json:"conversation"`
	Speaker      string    `json:"speaker,omitempty"`
	Portrait     string    `json:"portrait,omitempty"`
	Offered      []string  `json:"offered,omitempty"`
	Line         *sim.Line `json:"line,omitempty"`
	Transcript   []string  `json:"transcript,omitempty"`
}

// LevelState is a level's state.
type LevelState struct {
	Features []FeatureState `json:"features,omitempty"`
	Things   []ThingState   `json:"things,omitempty"`
	Objects  []CellObjects  `json:"objects,omitempty"`
	// Changed lists the cells whose floor or wall differs from the level's definition.
	Changed []CellChange `json:"changed,omitempty"`
	// Visible counts the cells the player sees, and Seen those it has seen.
	Visible int `json:"visible,omitempty"`
	Seen    int `json:"seen,omitempty"`
}

// FeatureState is a feature's fields as they are now, as the pack writes them, with what the
// simulation keeps besides.
type FeatureState struct {
	content.Feature
	// Shows is the look state the feature is drawn in.
	Shows   string `json:"shows"`
	Stage   int    `json:"stage,omitempty"`
	Counter int    `json:"counter,omitempty"`
}

// ThingState is a thing's state.
type ThingState struct {
	ID        string           `json:"id"`
	Thing     string           `json:"thing"`
	Cell      content.Cell     `json:"cell"`
	At        [2]int           `json:"at"`
	Facing    int              `json:"facing"`
	HitPoints int              `json:"hit_points"`
	Hostile   bool             `json:"hostile,omitempty"`
	Items     []content.Object `json:"items,omitempty"`
	Rolling   *content.Rolling `json:"rolling,omitempty"`
}

// CellObjects are the objects on a cell's floor, bottom first.
type CellObjects struct {
	Cell    content.Cell     `json:"cell"`
	Objects []content.Object `json:"objects"`
}

// CellChange is a cell's floor type and wall shape now, content.NoFrame for none.
type CellChange struct {
	Cell  content.Cell `json:"cell"`
	Floor int          `json:"floor"`
	Wall  int          `json:"wall"`
}

// State returns the world's state.
func State(w *sim.World) *WorldState {
	s := &WorldState{Step: w.Step(), Current: w.Current().ID, Levels: map[string]*LevelState{}}
	if pl := w.Player(); pl != nil {
		ps := &PlayerState{Cell: pl.Cell(), At: [2]int{pl.U, pl.V}, Facing: pl.Facing,
			HitPoints: pl.HitPoints, MagicPoints: pl.MagicPoints, Level: pl.Level, Experience: pl.Experience,
			ArmorClass: pl.ArmorClass, Skills: pl.Skills[:], Poisoned: pl.Poisoned, Dead: pl.Dead,
			Status: pl.Status, Timed: pl.Timed[:], Magic: pl.Magic, Look: pl.Look}
		if pl.Hand != nil {
			o := objects([]*sim.Object{pl.Hand})[0]
			ps.Hand = &o
		}
		n := sim.SlotCount
		if bag := pl.Pack(w.Pack()); bag != nil {
			n = sim.SlotPack + len(bag.Contents)
		}
		for i := range n {
			if o := pl.Slot(w.Pack(), i); o != nil {
				if ps.Slots == nil {
					ps.Slots = map[string]content.Object{}
				}
				ps.Slots[SlotName(i)] = objects([]*sim.Object{o})[0]
			}
		}
		s.Player = ps
	}
	if t := w.Talk(); t != nil {
		ts := &TalkState{Conversation: t.Def.ID, Portrait: t.Portrait, Line: t.Line, Transcript: t.Transcript}
		if t.Speaker != nil {
			ts.Speaker = t.Speaker.ID
		}
		for _, k := range t.Offered {
			if k != "" {
				ts.Offered = append(ts.Offered, k)
			}
		}
		s.Talk = ts
	}
	if c := w.Counters(); len(c) > 0 {
		s.Counters = c
	}
	if r := w.Records(); len(r) > 0 {
		s.Records = r
	}
	for id := range w.Pack().Levels {
		if l := w.Level(id); l != nil {
			s.Levels[id] = levelState(l)
		}
	}
	return s
}

func levelState(l *sim.Level) *LevelState {
	s := &LevelState{}
	for _, f := range l.Features {
		fs := FeatureState{Feature: f.Feature, Shows: f.LookState(), Stage: f.Stage, Counter: f.Counter}
		s.Features = append(s.Features, fs)
	}
	for _, t := range l.Things {
		s.Things = append(s.Things, ThingState{
			ID: t.ID, Thing: t.Template.ID, Cell: t.Cell(), At: [2]int{t.U, t.V},
			Facing: t.Facing, HitPoints: t.HitPoints, Hostile: t.Hostile, Items: objects(t.Items), Rolling: t.Rolling,
		})
	}
	for c, objs := range l.Objects {
		if len(objs) > 0 {
			s.Objects = append(s.Objects, CellObjects{Cell: c, Objects: objects(objs)})
		}
	}
	slices.SortFunc(s.Objects, func(a, b CellObjects) int { return compareCells(a.Cell, b.Cell) })
	for x := range l.Floor {
		for y := range l.Floor[x] {
			floor, wall := l.Floor[x][y], l.Wall[x][y]
			if floor != l.Def.Floor[x][y] || wall != l.Def.Wall[x][y] {
				s.Changed = append(s.Changed, CellChange{Cell: content.Cell{x, y}, Floor: floor, Wall: wall})
			}
			if c := (content.Cell{x, y}); l.Sighted() && l.Visible(c) {
				s.Visible++
			}
			if l.Seen(content.Cell{x, y}) {
				s.Seen++
			}
		}
	}
	return s
}

func objects(objs []*sim.Object) []content.Object {
	var out []content.Object
	for _, o := range objs {
		out = append(out, content.Object{Item: o.Item, Contents: objects(o.Contents)})
	}
	return out
}

func compareCells(a, b content.Cell) int {
	if a[0] != b[0] {
		return a[0] - b[0]
	}
	return a[1] - b[1]
}
