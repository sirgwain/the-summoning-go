package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// gunWorld is a fight world whose pack has its own damage kinds (physical, plasma, emp) and a
// player holding a weapon of two fire actions: a pistol shot from its own charges (two) and a
// beam paid for with 3 magic points, both bolts of 6 plasma damage with no room needed. The
// player faces along v.
func gunWorld(t *testing.T, edit func(p *content.Pack), rows ...string) *World {
	t.Helper()
	w := fightWorld(t, func(p *content.Pack) {
		p.Things["thing.player"].Box = &content.Box{-10, 9, 16, 6}
		p.Things["thing.guard"].Box = &content.Box{-10, 9, 16, 12}
		p.Combat = &content.Combat{ID: "combat", Kinds: []content.DamageKind{{ID: "physical"}, {ID: "plasma"}, {ID: "emp"}}}
		p.Items["item.bolt"] = &content.Item{ID: "item.bolt", Weight: 1, Size: 1}
		shot := &content.Shot{Item: "item.bolt", Damage: content.Dice{Bonus: 6}, Kind: "plasma", Range: &content.Range{Base: 20}}
		p.Items["item.gun"] = &content.Item{ID: "item.gun", Weight: 3, Size: 2, Wear: content.WearHand, Charges: 2,
			Actions: []content.ItemAction{
				{Kind: content.ActionFire, Delay: 3, Charges: 1, Shot: shot},
				{Kind: content.ActionFire, Delay: 3, Power: 3, Shot: shot},
			}}
		if edit != nil {
			edit(p)
		}
	}, rows...)
	p := w.Player()
	p.Facing = 2
	p.Slots[SlotRightHand] = w.newObject("item.gun")
	return w
}

// TestChargeShotPointBlank fires a charge-fed shot at a guard a step ahead, which needs no
// room: each shot uses a charge and does 6 + (ACC − 10) / 2; the bolt vanishes; with no charges
// left the gun refuses (docs/pack.md, Items).
func TestChargeShotPointBlank(t *testing.T) {
	w := gunWorld(t, nil,
		"##########",
		"#@G......#",
		"##########")
	p, l, g := w.Player(), w.Current(), w.Current().Thing("guard")
	g.U, g.V = p.U, p.V+16
	gun := p.Slots[SlotRightHand]
	if ev := strike(w, 0, 0); has(ev, EventMessage, content.MsgNoRoomToShoot) || g.HitPoints != 20-7 || gun.Value != 1 {
		t.Fatalf("point-blank shot: %v; guard %d, charges %d", ev, g.HitPoints, gun.Value)
	}
	strike(w, 0, 0)
	if g.HitPoints != 20-14 || gun.Value != 0 {
		t.Fatalf("second shot: guard %d, charges %d", g.HitPoints, gun.Value)
	}
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgNoCharges) || g.HitPoints != 20-14 {
		t.Errorf("an empty gun fired: %v, guard %d", ev, g.HitPoints)
	}
	for c, objs := range l.Objects {
		for _, o := range objs {
			if o.Item == "item.bolt" {
				t.Errorf("a bolt lies at %v", c)
			}
		}
	}
}

// TestPowerShot pays a shot with magic points, and refuses one the player cannot pay for.
func TestPowerShot(t *testing.T) {
	w := gunWorld(t, nil,
		"##########",
		"#@....G..#",
		"##########")
	p, l := w.Player(), w.Current()
	p.MagicPoints = 5
	strike(w, 0, 1)
	if p.MagicPoints != 2 {
		t.Fatalf("magic points after a shot: %d, want 2", p.MagicPoints)
	}
	for range 60 {
		w.Update(nil)
	}
	if len(l.Flights) != 0 {
		t.Fatalf("%d flights left", len(l.Flights))
	}
	if ev := strike(w, 0, 1); !has(ev, EventMessage, content.MsgNoPower) || p.MagicPoints != 2 {
		t.Errorf("a shot without the power: %v, %d magic points", ev, p.MagicPoints)
	}
}

// TestAmmoByTag fires the first object of the ammunition's tag in what the named slot holds,
// passing over others.
func TestAmmoByTag(t *testing.T) {
	w := gunWorld(t, func(p *content.Pack) {
		p.Items["item.shell"] = &content.Item{ID: "item.shell", Weight: 1, Size: 1, Tags: []string{"shells"},
			Missile: &content.Missile{Damage: content.Dice{Bonus: 2}, Kind: "emp"}}
		p.Items["item.belt"] = &content.Item{ID: "item.belt", Weight: 1, Size: 2, Wear: content.WearQuiver,
			Container: &content.Container{Capacity: 4, Largest: 1}}
		p.Items["item.shotgun"] = &content.Item{ID: "item.shotgun", Weight: 3, Size: 3, Wear: content.WearHand,
			Actions: []content.ItemAction{{Kind: content.ActionFire, Delay: 3, Ammo: &content.Ammo{Tag: "shells", Slot: SlotQuiver}}}}
	},
		"##########",
		"#@....G..#",
		"##########")
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject("item.shotgun")
	belt := w.newObject("item.belt")
	belt.Contents = []*Object{w.newObject("item.bolt"), w.newObject("item.shell")}
	p.Slots[SlotQuiver] = belt
	strike(w, 0, 0)
	if len(belt.Contents) != 1 || belt.Contents[0].Item != "item.bolt" {
		t.Fatalf("the belt holds %v after a shot", belt.Contents)
	}
	if ev := strike(w, 0, 0); !has(ev, EventMessage, content.MsgNoArrows) {
		t.Errorf("fired without shells: %v", ev)
	}
}

// TestKindsByName works damage of the pack's kinds through a creature's lists: a quarter of what
// it resists, none of what it ignores, a quarter more of what it is weak to, and all of anything
// else; and the book learns each.
func TestKindsByName(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(tm *content.Thing)
		want int
		def  string
	}{
		{"resists", func(tm *content.Thing) { tm.Resists = []string{"plasma"} }, 2, DefenseResists + "plasma"},
		{"ignores", func(tm *content.Thing) { tm.Ignores = []string{"plasma"} }, 0, DefenseIgnores + "plasma"},
		{"weak", func(tm *content.Thing) { tm.WeakTo = []string{"plasma"} }, 10, DefenseWeak + "plasma"},
		{"other", func(tm *content.Thing) { tm.Resists = []string{"emp"} }, 8, ""},
	} {
		w := gunWorld(t, func(p *content.Pack) { c.set(p.Things["thing.guard"]) },
			"#####",
			"#@G.#",
			"#####")
		l, g := w.Current(), w.Current().Thing("guard")
		w.hurtThing(l, g, 8, w.kind("plasma"))
		if g.HitPoints != 20-c.want {
			t.Errorf("%s: guard at %d, want %d", c.name, g.HitPoints, 20-c.want)
		}
		w.meetKind(l, g, w.kind("plasma"))
		if e := w.book.Creatures[g.Template.ID]; c.def != "" && (e == nil || !e.Knows(c.def)) {
			t.Errorf("%s: the book did not learn %s: %+v", c.name, c.def, e)
		}
	}
}

// TestCreatureShoots has a hostile creature with a ranged attack shoot the player in line within
// reach, once each interval, its shot vanishing; out of line it does not shoot.
func TestCreatureShoots(t *testing.T) {
	w := gunWorld(t, func(p *content.Pack) {
		g := p.Things["thing.guard"]
		g.Behaviors = append(g.Behaviors, content.BehaviorHostile)
		g.IntervalMS = 100
		g.Shoots = &content.Shoots{Item: "item.bolt", Damage: content.Dice{Bonus: 3}, Kind: "plasma", Range: 20, IntervalMS: 1000}
	},
		"##########",
		"#@..G....#",
		"#........#",
		"##########")
	p, l, g := w.Player(), w.Current(), w.Current().Thing("guard")
	p.HitPoints, p.MaxHitPoints = 50, 50
	shots := 0
	for range 2 * StepsPerSecond {
		for _, e := range w.Update(nil) {
			if e.Kind == EventThing && e.Detail == "thrown" && e.Item == "item.bolt" {
				shots++
			}
		}
	}
	if shots < 1 || shots > 2 || p.HitPoints >= 50 {
		t.Fatalf("in line: %d shots, player at %d", shots, p.HitPoints)
	}
	if g.Facing != 6 {
		t.Errorf("the guard faces %d, want 6", g.Facing)
	}
	p.U, p.V = g.U+CellUnits, g.V-3*CellUnits
	for range 40 {
		w.Update(nil)
	}
	shots = 0
	before := p.HitPoints
	for range 2 * StepsPerSecond {
		for _, e := range w.Update(nil) {
			if e.Kind == EventThing && e.Detail == "thrown" && e.Item == "item.bolt" {
				shots++
			}
		}
	}
	if shots != 0 || p.HitPoints != before {
		t.Errorf("out of line: %d shots, player %d → %d", shots, before, p.HitPoints)
	}
	for c, objs := range l.Objects {
		for _, o := range objs {
			if o.Item == "item.bolt" {
				t.Errorf("a bolt lies at %v", c)
			}
		}
	}
}
