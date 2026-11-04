package sim

import (
	"fmt"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// corpseWorld is a corridor with the player at one end, a guard in the middle made a corpse,
// and a guard at the other end: the nearer and the farther thing.
func corpseWorld(t *testing.T, edit func(p *content.Pack)) (*World, *Thing, *Thing) {
	t.Helper()
	w := fightWorld(t, func(p *content.Pack) {
		for i, pt := range p.Levels["level.walk"].Things {
			pt.ID = fmt.Sprintf("guard.%d", i)
		}
		if edit != nil {
			edit(p)
		}
	},
		"#######",
		"#@.G.G#",
		"#######")
	l := w.Current()
	near, far := l.Things[0], l.Things[1]
	p := w.Player()
	if dist(near, p) > dist(far, p) {
		near, far = far, near
	}
	return w, near, far
}

func dist(t *Thing, p *Player) int { return abs(t.U-p.U) + abs(t.V-p.V) }

// TestCreatureWalksOverCorpse has a hostile creature chase the player straight through a
// corpse in its way, which does not block it (docs/spec/entities.md, Moving).
func TestCreatureWalksOverCorpse(t *testing.T) {
	w, body, c := corpseWorld(t, func(p *content.Pack) {
		g := p.Things["thing.guard"]
		g.Behaviors, g.Speed = []string{content.BehaviorLeavesCorpse, content.BehaviorHostile}, 8
	})
	body.HitPoints = 0
	c.U, c.V = body.U, body.V+8
	u, v := c.U, c.V
	for range 100 {
		w.Update(nil)
		if c.V != v {
			break
		}
	}
	if c.U != u || c.V != v-8 {
		t.Errorf("the creature behind the corpse moved from %d,%d to %d,%d, want straight on 8", u, v, c.U, c.V)
	}
}

// TestLevitatingOverCorpse walks a levitating player over a corpse, which does not move
// (docs/spec/entities.md, Pushing).
func TestLevitatingOverCorpse(t *testing.T) {
	w, body, far := corpseWorld(t, nil)
	body.HitPoints, far.HitPoints = 0, 0
	p := w.Player()
	p.Status.Levitating = true
	at := [2]int{body.U, body.V}
	start := dist(body, p)
	for range 200 {
		w.Update([]input.Command{input.Walk{Direction: 2}})
	}
	if [2]int{body.U, body.V} != at {
		t.Errorf("the levitating player pushed the corpse from %v to %v", at, [2]int{body.U, body.V})
	}
	if d := abs(p.U-far.U) + abs(p.V-far.V); d >= abs(at[0]-far.U)+abs(at[1]-far.V) {
		t.Errorf("the levitating player did not pass the corpse (%d from it at the start)", start)
	}
}

// TestCreatureArrowHitsCreature fires a creature's arrow at the player past another creature,
// which it hurts (docs/spec/entities.md, Things in flight).
func TestCreatureArrowHitsCreature(t *testing.T) {
	w, near, far := corpseWorld(t, func(p *content.Pack) {
		p.Items["item.arrow"] = &content.Item{ID: "item.arrow", Weight: 1, Size: 1, Tags: []string{"arrows"},
			Missile: &content.Missile{Damage: content.Dice{Bonus: 5}}}
	})
	l, p := w.Current(), w.Player()
	hp, php := near.HitPoints, p.HitPoints
	w.launch(l, w.newObject("item.arrow"), far.U, far.V, 6, firedRange, 5, false, true, far)
	for range 100 {
		w.Update(nil)
	}
	if near.HitPoints >= hp || p.HitPoints != php {
		t.Errorf("the arrow: the creature between %d of %d hit points, the player %d of %d", near.HitPoints, hp, p.HitPoints, php)
	}
}
