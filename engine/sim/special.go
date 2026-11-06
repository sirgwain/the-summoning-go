package sim

import (
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Special weapons, charges, breaking objects, animated creatures and teleported creatures
// spinning (docs/spec/combat.md, Special weapons; docs/spec/items.md, Breaking;
// docs/spec/creatures.md).

const (
	// powerCharges is the power of an object whose charges absorb damage (items.md, Powers).
	powerCharges = 0x23
	// powerRandomize turns the object used into a random object (items.md, Powers).
	powerRandomize = 0x1B
	// spinTicks is how long a teleported creature spins.
	spinTicks = 30
	// A flickering creature counts flickerBase to flickerBase+flickerRange updates, flickering
	// while the count is above flickerBase; a toggling one toggles every toggleBase to
	// toggleBase+toggleRange updates.
	flickerBase  = 15
	flickerRange = 15
	toggleBase   = 2
	toggleRange  = 8
)

// countWeapon puts the damage a counted weapon in hand dealt into the player's weapon count: a
// growing weapon talks at its first hit and grows at its count; a grown one rises a level each
// time the count passes its step (docs/spec/combat.md, Special weapons).
func (w *World) countWeapon(o *Object, it *content.Item, damage int) {
	p := w.player
	talk := func(id string) {
		if t := w.pack.Things[id]; t != nil {
			w.startTalk(t, nil)
		}
	}
	switch {
	case it.Grows != nil:
		g := it.Grows
		switch {
		case p.WeaponCount == 0:
			talk(g.First)
		case p.WeaponCount >= g.At && w.pack.Items[g.Into] != nil:
			for _, slot := range []int{SlotRightHand, SlotLeftHand} {
				if p.Slots[slot] == o {
					w.takeOff(o)
					grown := w.newObject(g.Into)
					p.Slots[slot] = grown
					w.putOn(grown)
					w.emit(Event{Kind: EventObject, Level: w.current, Cell: p.Cell(), Item: g.Into, Detail: "grown"})
				}
			}
			talk(g.Grown)
			if g.Counter != "" {
				w.setCounter(g.Counter, 1)
			}
		}
		p.WeaponCount += damage
	case it.Levels != nil:
		lv := it.Levels
		p.WeaponCount += damage
		if p.WeaponCount > lv.Every && w.counters[lv.Counter] < lv.Max {
			w.setCounter(lv.Counter, w.counters[lv.Counter]+1)
			p.WeaponCount = 0
			talk(lv.Talk)
		}
	}
}

// absorb takes what the charges of the object in a hand with the charges power absorb of damage
// d, the right hand's first, and returns the rest (docs/spec/combat.md, Hurting the player).
func (w *World) absorb(d int) int {
	p := w.player
	for _, slot := range []int{SlotRightHand, SlotLeftHand} {
		o := p.Slots[slot]
		if o == nil {
			continue
		}
		it := w.pack.Items[o.Item]
		if it == nil || it.Power != powerCharges {
			continue
		}
		used := min(d, w.Quantity(o))
		for range used {
			w.useOne(o)
		}
		if used > 0 {
			w.emit(Event{Kind: EventMessage, Level: w.current, Cell: p.Cell(), Text: w.messageText(content.MsgAbsorbed),
				Detail: content.MsgAbsorbed, Damage: used})
		}
		return d - used
	}
	return d
}

// quiet reports whether combat message key is hidden: the pack's quiet-combat counter is set.
func (w *World) quiet(key string) bool {
	switch key {
	case content.MsgMiss, content.MsgHit, content.MsgCreatureHits, content.MsgCreatureMiss, content.MsgNotACreature:
	default:
		return false
	}
	c := w.pack.Manifest.Counters
	return c != nil && c.QuietCombat != "" && w.counters[c.QuietCombat] != 0
}

// broke runs what item it does when it breaks on level l (docs/spec/items.md, Breaking).
func (w *World) broke(l *Level, it *content.Item) {
	for _, a := range it.OnBreak {
		switch {
		case a.SetCounter != nil:
			w.setCounter(a.SetCounter.Counter, a.SetCounter.Value)
		case a.Converse != nil:
			if t := w.pack.Things[*a.Converse]; t != nil {
				w.startTalk(t, nil)
			}
		case a.RemoveAll != nil:
			for _, t := range append([]*Thing(nil), l.Things...) {
				if t.Template.ID == *a.RemoveAll {
					w.removeThing(l, t, "removed")
				}
			}
		case a.Experience != nil:
			w.gainExperience(*a.Experience)
		}
	}
}

// animate changes animated creature t's frame for one of its updates (docs/spec/creatures.md,
// Animated objects).
func (w *World) animate(l *Level, t *Thing) {
	a := t.Template.Animate
	frame := 0
	if t.Frame != nil {
		frame = *t.Frame
	}
	old := frame
	switch a.Kind {
	case content.AnimateFlicker:
		if t.animCount == 0 {
			t.animCount = flickerBase + w.rng.IntN(flickerRange)
		} else {
			t.animCount--
		}
		frame = a.Frames[0]
		if rest := a.Frames[1:]; t.animCount > flickerBase && len(rest) > 0 {
			if a.Random {
				frame = rest[w.rng.IntN(len(rest))]
			} else {
				frame = rest[t.animCount%len(rest)]
			}
		}
	case content.AnimateToggle:
		if t.animCount == 0 {
			t.animCount = toggleBase + w.rng.IntN(toggleRange)
			t.animStep = (t.animStep + 1) % len(a.Frames)
		} else {
			t.animCount--
		}
		frame = a.Frames[t.animStep]
	case content.AnimateRandom:
		frame = a.Frames[w.rng.IntN(len(a.Frames))]
	case content.AnimateCycle:
		t.animStep = (t.animStep + 1) % len(a.Frames)
		frame = a.Frames[t.animStep]
	}
	t.Frame = &frame
	if frame != old {
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "animates"})
	}
}

// spin turns a creature on a teleporter to its next direction each tick until its spin runs
// out, when the teleporter under it, if still armed, sends it on; it reports whether it was
// spinning.
func (w *World) spin(l *Level, t *Thing) bool {
	if t.spin == 0 {
		return false
	}
	if w.step >= t.spinNext {
		t.spin--
		t.spinNext = w.step + Steps(1)
		t.Facing = (t.Facing + 2) % 8
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "spins"})
		if t.spin == 0 {
			t.next, t.frozen = w.step, 0
			if f := l.byCell[t.Cell()]; f != nil && f.Kind == content.FeatureTeleporter && f.Armed {
				w.teleportThing(l, t, w.jump(l, f), 1)
			}
		}
	}
	return true
}

// killedBy reports what a player's blow that left creature t at 0 hit points does: its name
// shown when it has no conversation, and the practice it gives.
func (w *World) killedBy(l *Level, t *Thing) int {
	tm := t.Template
	if tm.Conversation == "" && tm.Name != "" {
		w.emit(Event{Kind: EventMessage, Level: l.ID, Cell: t.Cell(), Text: w.messageText(content.MsgKilled),
			Detail: content.MsgKilled, Item: tm.Name})
	}
	return max(1, tm.KillPractice)
}

// mirrored hurts the player for the damage creature t would take, when it mirrors, and reports
// whether it did.
func (w *World) mirrored(t *Thing, d int, k combat.Kind) bool {
	if !t.Template.Has(content.BehaviorMirrors) || !alive(t) {
		return false
	}
	w.hurtPlayer(d, k, t.ID)
	return true
}
