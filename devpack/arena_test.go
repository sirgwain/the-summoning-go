package devpack

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// TestArenaFight plays the arena by its script (scripts/arena.json) through the viewer, as a
// person would: the player takes up the sword and puts it in the right hand, turns toward the
// goblin, and strikes whenever the hand is ready. The goblin comes and fights. The test checks
// what a person would see: the sword drawn in the player's hand, damage shown over both, and the
// goblin lying as a corpse at the end. Watch it with
//
//	summoning view packs/dev --level level.arena --script devpack/scripts/arena.json --out /tmp/arena
func TestArenaFight(t *testing.T) {
	script, err := viewer.LoadScript("scripts/arena.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := viewer.New(Build(), 1, Arena, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	r := viewer.NewRunner(script, t.TempDir())
	var swordDrawn, goblinShown, playerShown bool
	for !r.Done() {
		in, err := r.Before(v)
		if err != nil {
			t.Fatal(err)
		}
		v.Update(in)
		if _, err := r.After(v); err != nil {
			t.Fatal(err)
		}
		for _, c := range v.Cmds {
			if c.Thing == sim.PlayerID && (c.Sprite == spriteSwordFront || c.Sprite == spriteSwordBehind) {
				swordDrawn = true
			}
		}
		// One label shows over each of the goblin and the player while its damage shows.
		step, goblin, player := v.World.Step(), v.World.Current().Thing("goblin"), v.World.Player()
		want := 0
		if goblin != nil && goblin.ShownUntil > step {
			want++
			goblinShown = true
		}
		if player.ShownUntil > step {
			want++
			playerShown = true
		}
		if len(v.Labels) != want {
			t.Fatalf("update %d: %d labels, want %d", v.Ticks, len(v.Labels), want)
		}
	}
	t.Cleanup(func() {
		if t.Failed() {
			for _, e := range v.Events {
				t.Log(e)
			}
		}
	})
	p, g := v.World.Player(), v.World.Current().Thing("goblin")
	switch {
	case v.World.Current().ID != Arena:
		t.Fatalf("the fight is on %s", v.World.Current().ID)
	case p.Dead:
		t.Fatalf("the goblin killed the player")
	case g == nil || g.HitPoints != 0:
		t.Fatalf("the goblin is not a corpse: %+v", g)
	}
	if !swordDrawn {
		t.Error("the sword was never drawn in the player's hand")
	}
	if !goblinShown {
		t.Error("no damage was shown over the goblin")
	}
	if p.HitPoints < p.MaxHitPoints && !playerShown {
		t.Error("the player was hurt but no damage was shown over it")
	}
	corpse := false
	for _, c := range v.Cmds {
		if c.Thing == "goblin" && c.Frame == figureCorpse {
			corpse = true
		}
	}
	if !corpse {
		t.Error("the goblin is not drawn lying dead")
	}
	if p.Experience == 0 {
		t.Error("killing the goblin gave no experience")
	}
}
