package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// bowWorld is a fight world with the game's boxes and a player holding a bow, wearing a quiver of
// two arrows, facing along v.
func bowWorld(t *testing.T, rows ...string) *World {
	t.Helper()
	w := fightWorld(t, func(p *content.Pack) {
		p.Things["thing.player"].Box = &content.Box{-10, 9, 16, 6}
		p.Things["thing.guard"].Box = &content.Box{-10, 9, 16, 12}
		p.Items["item.arrow"] = &content.Item{ID: "item.arrow", Weight: 1, Size: 1, Tags: []string{"arrows"},
			Missile: &content.Missile{Damage: content.Dice{Bonus: 5}}}
		p.Items["item.quiver"] = &content.Item{ID: "item.quiver", Weight: 1, Size: 2, Wear: content.WearQuiver,
			Container: &content.Container{Capacity: 4, Largest: 1}}
		bow := 3
		p.Items["item.bow"] = &content.Item{ID: "item.bow", Weight: 3, Size: 3, Wear: content.WearHand, WeaponClass: &bow,
			Actions: []content.ItemAction{{Kind: content.ActionFire, Delay: 3}}}
	}, rows...)
	p := w.Player()
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.bow")
	q := w.newObject("item.quiver")
	q.Contents = []*Object{w.newObject("item.arrow"), w.newObject("item.arrow")}
	p.Slots[SlotQuiver] = q
	return w
}

// TestBowNeedsRoom refuses to draw the bow while a creature stands within a step ahead, but not
// for a corpse, and fires once the player stands back (docs/spec/items.md, Throwing).
func TestBowNeedsRoom(t *testing.T) {
	w := bowWorld(t,
		"##########",
		"#@G......#",
		"##########")
	p, g := w.Player(), w.Current().Thing("guard")
	g.U, g.V = p.U, p.V+16
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgNoRoomToShoot) || len(p.Slots[SlotQuiver].Contents) != 2 {
		t.Fatalf("firing at a guard a step ahead: %v", ev)
	}
	g.HitPoints = 0
	if ev := strike(w, 0, 0); has(ev, EventMessage, content.MsgNoRoomToShoot) || len(p.Slots[SlotQuiver].Contents) != 1 {
		t.Errorf("a corpse a step ahead stopped the bow: %v", ev)
	}
	g.HitPoints, g.V = 20, p.V+40
	if ev := strike(w, 0, 0); has(ev, EventMessage, content.MsgNoRoomToShoot) || len(p.Slots[SlotQuiver].Contents) != 0 {
		t.Errorf("firing at a guard more than a step away: %v", ev)
	}
}

// TestFlightStartsAhead starts an arrow its launch distance ahead of the player: 12 units along
// direction 2 (docs/spec/entities.md, Things in flight).
func TestFlightStartsAhead(t *testing.T) {
	w := bowWorld(t,
		"##########",
		"#@.......#",
		"##########")
	p, l := w.Player(), w.Current()
	w.Update([]input.Command{input.Strike{Hand: 0}})
	if len(l.Flights) != 1 {
		t.Fatalf("%d flights", len(l.Flights))
	}
	if f := l.Flights[0]; f.U != p.U || f.V != p.V+12 {
		t.Errorf("the arrow starts at (%d, %d), want (%d, %d)", f.U, f.V, p.U, p.V+12)
	}
}

// TestThrowTooCloseToWall refuses a throw whose flight would start in the wall: the rock stays in
// hand and the hand does not wait (docs/spec/entities.md, Too close to a wall).
func TestThrowTooCloseToWall(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@..#",
		"#####")
	p, l := w.Player(), w.Current()
	p.Facing, p.V = 6, 40
	rock := w.newObject("item.rock")
	p.Slots[SlotRightHand] = rock
	ev := w.Update([]input.Command{input.Throw{Hand: 0}})
	if !has(ev, EventMessage, content.MsgTooCloseToWall) || p.Slots[SlotRightHand] != rock || p.Delays[0] != 0 || len(l.Flights) != 0 {
		t.Errorf("throwing into the wall: %v, hand %v, delay %d, %d flights", ev, p.Slots[SlotRightHand], p.Delays[0], len(l.Flights))
	}
	// A step back, the flight starts on the floor.
	p.V = 60
	if w.Update([]input.Command{input.Throw{Hand: 0}}); p.Slots[SlotRightHand] != nil || len(l.Flights) != 1 {
		t.Errorf("throwing from a step back: hand %v, %d flights", p.Slots[SlotRightHand], len(l.Flights))
	}
}

// TestSpellIntoWallHitsCaster sets off the player's fire missile at the player when it would start
// in a wall, and an opening one at the wall (docs/spec/spells.md, Missiles).
func TestSpellIntoWallHitsCaster(t *testing.T) {
	w := fightWorld(t, nil,
		"#####",
		"#@..#",
		"#####")
	p, l := w.Player(), w.Current()
	p.Facing, p.V = 6, 40
	w.castMissile(l, w.playerCaster(), &content.MissileSpell{Hurts: content.HurtsFire}, 5)
	if p.HitPoints != 30-5 {
		t.Errorf("a fire missile cast into the wall: %d hit points, want 25", p.HitPoints)
	}
	w.events = nil
	w.castMissile(l, w.playerCaster(), &content.MissileSpell{Opens: true}, 5)
	if p.HitPoints != 25 || !has(w.events, EventMessage, content.MsgMissileWall) {
		t.Errorf("an opening missile cast into the wall: %d hit points, %v", p.HitPoints, w.events)
	}
}
