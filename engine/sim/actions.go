package sim

import (
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// run runs procedure name of level l, if there is one. depth counts procedures running
// procedures.
func (w *World) run(l *Level, name string, depth int) {
	if name == "" {
		return
	}
	actions, ok := l.Def.Procedures[name]
	if !ok {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: "no procedure " + name})
		return
	}
	if depth >= maxDepth {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: "procedure " + name + " runs too deep"})
		return
	}
	w.emit(Event{Kind: EventProcedure, Level: l.ID, Detail: name})
	for _, a := range actions {
		w.act(l, a, depth)
	}
}

// act carries out one action (docs/pack.md, Procedures).
func (w *World) act(l *Level, a content.Action, depth int) {
	feature := func(id string) *Feature {
		f := l.byID[id]
		if f == nil {
			w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: "no feature " + id})
		}
		return f
	}
	switch {
	case a.On != nil:
		if f := feature(*a.On); f != nil {
			w.switchFeature(l, f, input.SwitchOn, depth)
		}
	case a.Off != nil:
		if f := feature(*a.Off); f != nil {
			w.switchFeature(l, f, input.SwitchOff, depth)
		}
	case a.Toggle != nil:
		if f := feature(*a.Toggle); f != nil {
			w.switchFeature(l, f, input.SwitchToggle, depth)
		}
	case a.SetHook != nil:
		if f := feature(a.SetHook.Feature); f != nil {
			if hook := f.Hook(a.SetHook.Hook); hook != nil {
				*hook = ""
				if a.SetHook.Procedure != nil {
					*hook = *a.SetHook.Procedure
				}
			}
		}
	case a.SetThreshold != nil:
		if f := feature(a.SetThreshold.Feature); f != nil {
			f.Threshold = a.SetThreshold.Value
		}
	case a.SetAccepts != nil:
		if f := feature(a.SetAccepts.Feature); f != nil {
			f.Accepts = a.SetAccepts.Item
		}
	case a.SetMessage != nil:
		if f := feature(a.SetMessage.Feature); f != nil {
			f.Message = a.SetMessage.Message
		}
	case a.Roll != nil:
		if t := l.Thing(a.Roll.Thing); t != nil {
			r := content.Rolling{Direction: a.Roll.Direction, Shuttle: a.Roll.Shuttle}
			t.Rolling = &r
			w.emit(Event{Kind: EventNotYet, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "rolling boulders"})
		}
	case a.Stop != nil:
		if t := l.Thing(*a.Stop); t != nil {
			t.Rolling = nil
		}
	case a.ShootItem != nil:
		w.emit(Event{Kind: EventNotYet, Level: l.ID, Cell: a.ShootItem.Cell, Item: a.ShootItem.Item, Detail: "shooting objects"})
	case a.ShootSpell != nil:
		w.emit(Event{Kind: EventNotYet, Level: l.ID, Cell: a.ShootSpell.Cell, Detail: "shooting spells"})
	case a.Create != nil:
		w.drop(l, a.Create.Cell, &Object{Item: a.Create.Item}, byWorld, depth)
	case a.SetFloor != nil:
		c := a.SetFloor.Cell
		l.Floor[c[0]][c[1]] = a.SetFloor.Type
		l.Wall[c[0]][c[1]] = content.NoFrame
		w.emit(Event{Kind: EventMap, Level: l.ID, Cell: c})
	case a.SetWall != nil:
		c := a.SetWall.Cell
		l.Wall[c[0]][c[1]] = a.SetWall.Shape
		w.emit(Event{Kind: EventMap, Level: l.ID, Cell: c})
	}
}

// apply carries out a command on the current level.
func (w *World) apply(cmd input.Command) {
	l := w.Current()
	refuse := func(c content.Cell, why string) {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c, Detail: why})
	}
	switch c := cmd.(type) {
	case input.Use:
		w.use(l, c.Cell)
	case input.Drop:
		switch {
		case !l.InBounds(c.Cell):
			refuse(c.Cell, "outside the level")
		case w.pack.Items[c.Item] == nil:
			refuse(c.Cell, "no item "+c.Item)
		default:
			w.drop(l, c.Cell, &Object{Item: c.Item}, byPlayer, 0)
		}
	case input.Remove:
		objs := l.Objects[c.Cell]
		if len(objs) == 0 {
			refuse(c.Cell, "nothing to take")
			return
		}
		o := objs[len(objs)-1]
		l.Objects[c.Cell] = objs[:len(objs)-1]
		if len(l.Objects[c.Cell]) == 0 {
			delete(l.Objects, c.Cell)
		}
		w.emit(Event{Kind: EventObject, Level: l.ID, Cell: c.Cell, Item: o.Item, Detail: "taken"})
		w.weigh(l, c.Cell, -o.Weight(w.pack), 0)
	case input.Place:
		tmpl := w.pack.Things[c.Thing]
		switch {
		case !l.InBounds(c.Cell):
			refuse(c.Cell, "outside the level")
		case tmpl == nil:
			refuse(c.Cell, "no thing "+c.Thing)
		default:
			t := &Thing{ID: w.newThingID(l, tmpl.Kind), Template: tmpl, Facing: tmpl.Facing, HitPoints: w.roll(tmpl.HitPoints)}
			t.U, t.V = cellCenter(c.Cell)
			l.Things = append(l.Things, t)
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: c.Cell, Thing: t.ID, Detail: "placed"})
			w.weigh(l, c.Cell, t.weight(), 0)
			w.entered(l, t, 0)
		}
	case input.Move:
		t := l.Thing(c.Thing)
		switch {
		case t == nil:
			refuse(c.Cell, "no thing "+c.Thing)
		case !l.InBounds(c.Cell):
			refuse(c.Cell, "outside the level")
		default:
			w.moveThing(l, t, c.Cell, 0)
		}
	case input.Switch:
		f := l.byID[c.Feature]
		if f == nil {
			refuse(content.Cell{}, "no feature "+c.Feature)
			return
		}
		w.switchFeature(l, f, c.Mode, 0)
	case input.Run:
		w.run(l, c.Procedure, 0)
	case input.FollowExit:
		f := l.byID[c.Feature]
		if f == nil || f.To == nil {
			refuse(content.Cell{}, "no exit "+c.Feature)
			return
		}
		if err := w.enter(f.To.Level); err != nil {
			refuse(f.Cell, err.Error())
		}
	case input.GoTo:
		if err := w.enter(c.Level); err != nil {
			refuse(content.Cell{}, err.Error())
		}
	}
}

// newThingID returns an id for a new thing of kind not yet used in l.
func (w *World) newThingID(l *Level, kind string) string {
	for n := 1; ; n++ {
		id := kind + ".new." + strconv.Itoa(n)
		if l.Thing(id) == nil {
			return id
		}
	}
}
