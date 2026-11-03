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
			w.startBoulder(t, a.Roll.Direction, a.Roll.Shuttle)
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "rolling"})
		}
	case a.Stop != nil:
		if t := l.Thing(*a.Stop); t != nil {
			t.Rolling = nil
		}
	case a.ShootItem != nil:
		w.shootItem(l, a.ShootItem.Cell, a.ShootItem.Item, a.ShootItem.Direction)
	case a.ShootSpell != nil:
		w.shootSpell(l, a.ShootSpell.Cell, a.ShootSpell.Spell, a.ShootSpell.Direction, a.ShootSpell.Power)
	case a.Create != nil:
		w.drop(l, a.Create.Cell, w.newObject(a.Create.Item), byWorld, depth)
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
			w.drop(l, c.Cell, w.newObject(c.Item), byPlayer, 0)
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
			t := w.newThing(w.newThingID(l, tmpl.Kind), tmpl)
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
	case input.Weigh:
		if f := l.byCell[c.Cell]; f == nil || f.Kind != content.FeaturePlate {
			refuse(c.Cell, "no plate")
			return
		}
		w.weigh(l, c.Cell, c.Weight, 0)
	case input.Reset:
		if !w.resetFeature(l, c.Feature, c.Unweigh) {
			refuse(content.Cell{}, "no feature "+c.Feature)
		}
	case input.Run:
		w.run(l, c.Procedure, 0)
	case input.FollowExit:
		f := l.byID[c.Feature]
		if f == nil || f.To == nil {
			refuse(content.Cell{}, "no exit "+c.Feature)
			return
		}
		if w.player != nil {
			w.travel(f.To, "")
			return
		}
		if err := w.enter(f.To.Level); err != nil {
			refuse(f.Cell, err.Error())
		}
	case input.SetCounter:
		w.setCounter(c.Counter, c.Value)
	case input.GoTo:
		if w.player != nil {
			if dest, err := w.load(c.Level); err != nil {
				refuse(content.Cell{}, err.Error())
			} else {
				w.travel(&content.Destination{Level: c.Level, Cell: w.arrival(dest)}, "")
			}
			return
		}
		if err := w.enter(c.Level); err != nil {
			refuse(content.Cell{}, err.Error())
		}
	case input.Walk, input.PickUp, input.PutDown, input.Swap, input.Open, input.Activate, input.PlacePlayer, input.Strike, input.PutOn, input.UseSlot, input.MoveSlot, input.Throw,
		input.Learn, input.Memorize, input.Cast, input.NextSpell, input.Read, input.Talk, input.ShowMap, input.Menu:
		w.playerCommand(c)
	}
}

// playerCommand carries out a command of the player's.
func (w *World) playerCommand(cmd input.Command) {
	p, l := w.player, w.Current()
	if p == nil {
		w.emit(Event{Kind: EventRefused, Level: l.ID, Detail: "no player"})
		return
	}
	switch c := cmd.(type) {
	case input.Walk:
		if c.Direction >= 0 && c.Direction < len(Dirs) {
			p.want = c.Direction
		}
	case input.PickUp:
		w.pickUp(c.Hand)
	case input.PutDown:
		w.putDown()
	case input.Swap:
		w.swap(c.Slot)
	case input.Open:
		w.open(c.Slot)
	case input.Activate:
		w.activate()
	case input.Strike:
		w.strike(c.Hand, c.Action)
	case input.PutOn:
		w.putOnHand()
	case input.UseSlot:
		w.useSlot(c.Slot)
	case input.MoveSlot:
		w.moveSlot(c.Slot, c.To)
	case input.Throw:
		w.throwFrom(c.Hand)
	case input.Learn:
		w.learn(c.Gesture)
	case input.Memorize:
		w.memorizeLearned(c.Spell)
	case input.Cast:
		w.cast(c.School)
	case input.NextSpell:
		w.nextSpell(c.School)
	case input.Read:
		w.read(c.Slot)
	case input.Talk:
		w.talkTo(c.Thing)
	case input.ShowMap:
		w.showMap()
	case input.Menu:
		if t := w.pack.Things[w.pack.Manifest.Menu]; t != nil {
			w.startTalk(t, nil)
		}
	case input.PlacePlayer:
		if !l.InBounds(c.Cell) || l.Blocked(c.Cell) {
			w.emit(Event{Kind: EventRefused, Level: l.ID, Cell: c.Cell, Detail: "the player cannot stand there"})
			return
		}
		w.weigh(l, p.Cell(), -p.weight(), 0)
		w.place(c.Cell, false)
		w.playerEntered(l, c.Cell, 0)
	}
}

// arrival is where the player arrives on level l when no exit says: the pack's start cell on the
// start level, otherwise the first open cell in row order with no feature and nothing standing.
func (w *World) arrival(l *Level) content.Cell {
	if s := w.pack.Manifest.Start; s.Level == l.ID {
		return s.Cell
	}
	for x := range l.Def.Size[0] {
		for y := range l.Def.Size[1] {
			if c := (content.Cell{x, y}); !l.Blocked(c) && l.byCell[c] == nil && len(l.ThingsAt(c)) == 0 {
				return c
			}
		}
	}
	return content.Cell{}
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
