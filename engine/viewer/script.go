package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// Step is what a script does at one update (docs/design.md, Testing). Every field is optional;
// a step can do several things, in this order: center, select, then input, and after the update,
// shoot and dump.
type Step struct {
	// At is the update the step runs in, counting from 0.
	At int64 `json:"at"`
	// Center centers the view on a cell.
	Center *content.Cell `json:"center,omitempty"`
	// Select selects a cell; SelectThing a thing of the shown level.
	Select      *content.Cell `json:"select,omitempty"`
	SelectThing string        `json:"select_thing,omitempty"`
	// Keys are pressed in this update and held through it. Hold keys are held without being
	// pressed, for Shift with a click, for panning or for walking.
	Keys []string `json:"keys,omitempty"`
	Hold []string `json:"hold,omitempty"`
	// For holds the step's keys and hold keys through this many updates from At, pressing them
	// only in the first; 0 means 1.
	For int64 `json:"for,omitempty"`
	// Play switches to playing (true) or to the tools (false).
	Play *bool `json:"play,omitempty"`
	// PlacePlayer puts the player on a cell of the shown level, by the debug command.
	PlacePlayer *content.Cell `json:"place_player,omitempty"`
	// Click taps the left button on a cell, centering the view on it first if it is hidden.
	Click *content.Cell `json:"click,omitempty"`
	// ClickPlayer taps on the player's figure; Slot on the hotbar's box for a slot; Action on the
	// hotbar's button for [hand, action]. Right makes the step's tap the right button's.
	ClickPlayer bool  `json:"click_player,omitempty"`
	Slot        *int  `json:"slot,omitempty"`
	Action      []int `json:"action,omitempty"`
	Right       bool  `json:"right,omitempty"`
	// Wheel scrolls, positive to zoom in.
	Wheel float64 `json:"wheel,omitempty"`
	// Shot saves the screen after the update, and Dump the viewer's state, to these files.
	Shot string `json:"shot,omitempty"`
	Dump string `json:"dump,omitempty"`
}

// Script is a list of steps, run in update order.
type Script []Step

// LoadScript reads a script from a JSON file.
func LoadScript(path string) (Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := ParseScript(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// ParseScript reads a script, checking its keys and putting its steps in update order.
func ParseScript(data []byte) (Script, error) {
	var s Script
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	for i, st := range s {
		if st.At < 0 {
			return nil, fmt.Errorf("step %d: negative update %d", i, st.At)
		}
		for _, keys := range [][]string{st.Keys, st.Hold} {
			for j, k := range keys {
				key, err := ParseKey(k)
				if err != nil {
					return nil, fmt.Errorf("step %d: %w", i, err)
				}
				keys[j] = string(key)
			}
		}
	}
	slices.SortStableFunc(s, func(a, b Step) int { return int(a.At - b.At) })
	return s, nil
}

// Runner runs a script on a viewer, one update at a time.
type Runner struct {
	script Script
	// dir is where shot and dump paths are relative to.
	dir string
	// next is the first step not yet run; at the update being run.
	next int
	at   int64
	// first and last are the steps of the update being run.
	first, last int
	// held are earlier steps whose keys are still held.
	held []Step
}

// NewRunner returns a runner for script s that writes its files relative to dir.
func NewRunner(s Script, dir string) *Runner { return &Runner{script: s, dir: dir} }

// Done reports whether every step has run and no keys are still held.
func (r *Runner) Done() bool {
	if r.next < len(r.script) {
		return false
	}
	for _, st := range r.held {
		if r.at+1 < st.At+st.For {
			return false
		}
	}
	return true
}

// Before runs the update's steps on v and returns its input. Call it before each update, and
// After after it.
func (r *Runner) Before(v *Viewer) (Input, error) {
	r.at, r.first = v.Ticks, r.next
	for r.next < len(r.script) && r.script[r.next].At <= r.at {
		r.next++
	}
	r.last = r.next
	in := Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(-1, -1)}
	held := r.held[:0]
	for _, st := range r.held {
		if r.at < st.At+st.For {
			held = append(held, st)
			for _, k := range append(slices.Clone(st.Keys), st.Hold...) {
				in.Held[Key(k)] = true
			}
		}
	}
	r.held = held
	for _, st := range r.script[r.first:r.last] {
		if st.For > 1 {
			r.held = append(r.held, st)
		}
		if st.Play != nil {
			v.SetPlay(*st.Play)
		}
		if st.PlacePlayer != nil {
			v.World.Update([]input.Command{input.PlacePlayer{Cell: *st.PlacePlayer}})
			v.rebuild()
			if v.Play {
				v.Recenter()
			}
		}
		if st.Center != nil {
			v.CenterOn(*st.Center)
		}
		if st.Select != nil {
			v.Select(*st.Select)
		}
		if st.SelectThing != "" {
			if err := v.SelectThing(st.SelectThing); err != nil {
				return in, fmt.Errorf("update %d: %w", r.at, err)
			}
		}
		for _, k := range st.Hold {
			in.Held[Key(k)] = true
		}
		for _, k := range st.Keys {
			in.Held[Key(k)], in.Pressed[Key(k)] = true, true
		}
		tap := func(pt image.Point) {
			in.Cursor = pt
			if st.Right {
				in.Right = Button{Pressed: true, Released: true}
			} else {
				in.Left = Button{Pressed: true, Released: true}
			}
		}
		if st.ClickPlayer {
			p := v.World.Player()
			if p == nil {
				return in, fmt.Errorf("update %d: no player to click", r.at)
			}
			c := p.Cell()
			st.Click = &c
		}
		if st.Click != nil {
			pt := v.CellPoint(*st.Click)
			if !pt.In(image.Rectangle{Max: v.Screen}) || v.covered(pt) {
				v.CenterOn(*st.Click)
				pt = v.CellPoint(*st.Click)
			}
			tap(pt)
		}
		if st.Slot != nil || st.Action != nil {
			pt, err := hotbarPoint(v, st)
			if err != nil {
				return in, fmt.Errorf("update %d: %w", r.at, err)
			}
			tap(pt)
		}
		in.Wheel += st.Wheel
	}
	return in, nil
}

// covered reports whether screen point pt is under the panel, the log or the hotbar.
func (v *Viewer) covered(pt image.Point) bool {
	if pt.In(v.PanelRect()) || pt.In(v.LogRect()) {
		return true
	}
	h := v.Hotbar()
	return h != nil && pt.In(h.Rect)
}

// hotbarPoint returns the middle of the hotbar box or button a step taps.
func hotbarPoint(v *Viewer, st Step) (image.Point, error) {
	h := v.Hotbar()
	if h == nil {
		return image.Point{}, fmt.Errorf("no hotbar while not playing")
	}
	mid := func(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }
	if st.Slot != nil {
		r, ok := h.SlotRect(*st.Slot)
		if !ok {
			return image.Point{}, fmt.Errorf("the hotbar has no slot %d", *st.Slot)
		}
		return mid(r), nil
	}
	if len(st.Action) != 2 {
		return image.Point{}, fmt.Errorf("action wants [hand, action]")
	}
	for _, a := range h.Actions {
		if a.Hand == st.Action[0] && a.Action == st.Action[1] {
			return mid(h.Screen(a.Rect)), nil
		}
	}
	return image.Point{}, fmt.Errorf("the hotbar has no action %v", st.Action)
}

// After writes the update's dumps and returns the paths of the screenshots to take.
func (r *Runner) After(v *Viewer) ([]string, error) {
	var shots []string
	for _, st := range r.script[r.first:r.last] {
		if st.Dump != "" {
			if err := WriteDump(v, r.path(st.Dump)); err != nil {
				return nil, err
			}
		}
		if st.Shot != "" {
			shots = append(shots, r.path(st.Shot))
		}
	}
	r.first = r.last
	return shots, nil
}

func (r *Runner) path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(r.dir, p)
}

// Run runs the whole script on v without a screen, skipping screenshots.
func (r *Runner) Run(v *Viewer) error {
	for !r.Done() {
		in, err := r.Before(v)
		if err != nil {
			return err
		}
		v.Update(in)
		if _, err := r.After(v); err != nil {
			return err
		}
	}
	return nil
}

// WriteDump writes the viewer's state to a JSON file, making its directory.
func WriteDump(v *Viewer, path string) error {
	data, err := json.MarshalIndent(v.Dump(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
