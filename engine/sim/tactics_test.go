package sim

import (
	"encoding/json"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// tacticsWorld is a world of rows with a hostile wolf ('W') whose template edit changes.
func tacticsWorld(t *testing.T, edit func(wolf *content.Thing), rows ...string) (*World, *Thing) {
	t.Helper()
	w := fightWorld(t, func(p *content.Pack) {
		p.Levels["level.walk"].Things[0].Thing = "thing.wolf"
		if edit != nil {
			edit(p.Things["thing.wolf"])
		}
	}, rows...)
	return w, w.Current().Things[0]
}

// updateWolf runs the world until the wolf's next update has run.
func updateWolf(w *World, wolf *Thing) {
	next := wolf.next
	for range 1000 {
		w.Update(nil)
		if wolf.next != next {
			return
		}
	}
}

// TestWayAroundWall heads a creature the way around a wall rather than into it, and it reaches
// the player.
func TestWayAroundWall(t *testing.T) {
	w, wolf := tacticsWorld(t, nil,
		"#####",
		"#W..#",
		"###.#",
		"#@..#",
		"#####")
	u, v := wolf.U, wolf.V
	updateWolf(w, wolf)
	if wolf.U != u || wolf.V <= v {
		t.Fatalf("the wolf stepped from (%d, %d) to (%d, %d), want along +v around the wall", u, v, wolf.U, wolf.V)
	}
	p := w.Player()
	for range 5000 {
		w.Update(nil)
		if p.HitPoints < p.MaxHitPoints {
			break
		}
	}
	if p.HitPoints == p.MaxHitPoints {
		t.Errorf("the wolf never reached the player: at (%d, %d)", wolf.U, wolf.V)
	}
}

// TestWayKeepsHeading keeps the game's heading where nothing is in the way.
func TestWayKeepsHeading(t *testing.T) {
	w, wolf := tacticsWorld(t, nil,
		"#######",
		"#.....#",
		"#@...W#",
		"#.....#",
		"#######")
	p := w.Player()
	wolf.U += 10
	if d, ok := w.wayStep(w.Current(), wolf, p.Cell(), p.U, p.V, nil); !ok || d != towards(wolf.U, wolf.V, p.U, p.V) {
		t.Errorf("way step %d (%v), want the heading %d", d, ok, towards(wolf.U, wolf.V, p.U, p.V))
	}
}

func TestKeepDistance(t *testing.T) {
	w, wolf := tacticsWorld(t, func(wolf *content.Thing) { wolf.KeepDistance = 2 },
		"###########",
		"#@W.......#",
		"###########")
	p := w.Player()
	v := wolf.V
	updateWolf(w, wolf)
	if wolf.V <= v || wolf.Facing != 6 {
		t.Fatalf("the near wolf went from v %d to %d facing %d, want away facing the player (6)", v, wolf.V, wolf.Facing)
	}
	for range 200 {
		updateWolf(w, wolf)
	}
	if gap := wolf.V - p.V; gap < 2*CellUnits || gap > 4*CellUnits {
		t.Errorf("the wolf keeps %d units from the player, want 64..128", gap)
	}
	if wolf.Facing != 6 {
		t.Errorf("the wolf faces %d, want the player (6)", wolf.Facing)
	}

	wolf.V = p.V + 8*CellUnits
	v = wolf.V
	updateWolf(w, wolf)
	if wolf.V >= v {
		t.Errorf("the far wolf went from v %d to %d, want toward the player", v, wolf.V)
	}
}

// TestKeepDistanceCornered fights when it cannot back away.
func TestKeepDistanceCornered(t *testing.T) {
	w, _ := tacticsWorld(t, func(wolf *content.Thing) { wolf.KeepDistance = 2 },
		"#####",
		"#.@W#",
		"#####")
	p := w.Player()
	for range 2000 {
		w.Update(nil)
		if p.HitPoints < p.MaxHitPoints {
			break
		}
	}
	if p.HitPoints == p.MaxHitPoints {
		t.Error("the cornered wolf never struck")
	}
}

// TestKeepDistanceLinesUp sidesteps onto the player's line, then rests there and casts.
func TestKeepDistanceLinesUp(t *testing.T) {
	w, wolf := tacticsWorld(t, func(wolf *content.Thing) { wolf.KeepDistance = 2; wolf.Spell = "spell.bolt" },
		"#######",
		"#@....#",
		"#.....#",
		"#...W.#",
		"#######")
	w.pack.Spells["spell.bolt"] = &content.Spell{ID: "spell.bolt"}
	p := w.Player()
	u := wolf.U
	updateWolf(w, wolf)
	if wolf.U >= u || wolf.Facing != 6 {
		t.Fatalf("the wolf went from u %d to %d facing %d, want toward the player's line facing it", u, wolf.U, wolf.Facing)
	}
	for range 100 {
		if abs(wolf.U-p.U) < stepOf(wolf) {
			break
		}
		updateWolf(w, wolf)
	}
	if abs(wolf.U-p.U) >= stepOf(wolf) {
		t.Fatalf("the wolf never lined up: u gap %d", wolf.U-p.U)
	}
	cast := false
	for range 100 {
		updateWolf(w, wolf)
		if abs(wolf.U-p.U) >= stepOf(wolf) || wolf.Facing != 6 {
			t.Fatalf("the lined-up wolf left the line: u gap %d, facing %d", wolf.U-p.U, wolf.Facing)
		}
		if wolf.Pose == PoseAttack {
			cast = true
			break
		}
	}
	if !cast || wolf.walkDir != restDir || wolf.walkCount < 10 {
		t.Errorf("the lined-up wolf: cast %v, walk %d for %d", cast, wolf.walkDir, wolf.walkCount)
	}
}

// strikeOnce runs until the wolf's blow, and then its next update.
func strikeOnce(t *testing.T, w *World, wolf *Thing) {
	t.Helper()
	p := w.Player()
	for range 2000 {
		w.Update(nil)
		if p.HitPoints < p.MaxHitPoints {
			return
		}
	}
	t.Fatal("the wolf never struck")
}

func TestAfterBlowBackOff(t *testing.T) {
	w, wolf := tacticsWorld(t, func(wolf *content.Thing) { wolf.AfterBlow = content.AfterBlowBackOff },
		"#########",
		"#@W.....#",
		"#########")
	strikeOnce(t, w, wolf)
	if wolf.Recovery != content.AfterBlowBackOff || wolf.RecoverySteps < 2 || wolf.RecoverySteps > 3 {
		t.Fatalf("after its blow the wolf recovers %q for %d", wolf.Recovery, wolf.RecoverySteps)
	}
	s, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back State
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if w, err = Restore(w.pack, &back); err != nil {
		t.Fatal(err)
	}
	wolf = w.Current().Things[0]
	if wolf.Recovery != content.AfterBlowBackOff {
		t.Fatalf("the restored wolf recovers %q", wolf.Recovery)
	}
	v, steps := wolf.V, wolf.RecoverySteps
	for range steps {
		updateWolf(w, wolf)
	}
	if wolf.V <= v || wolf.Facing != 6 || wolf.Recovery != "" {
		t.Errorf("the wolf backed from v %d to %d facing %d, recovering %q", v, wolf.V, wolf.Facing, wolf.Recovery)
	}
	updateWolf(w, wolf)
	if wolf.V >= v+steps*8 {
		t.Errorf("the wolf did not come back: v %d", wolf.V)
	}
}

func TestAfterBlowFlank(t *testing.T) {
	w, wolf := tacticsWorld(t, func(wolf *content.Thing) { wolf.AfterBlow = content.AfterBlowFlank },
		"#######",
		"#.....#",
		"#.....#",
		"#..@W.#",
		"#.....#",
		"#.....#",
		"#######")
	p := w.Player()
	p.Facing = 2
	strikeOnce(t, w, wolf)
	if wolf.Recovery != content.AfterBlowFlank {
		t.Fatalf("after its blow the wolf recovers %q", wolf.Recovery)
	}
	behindCell := content.Cell{p.Cell()[0], p.Cell()[1] - 1}
	reached := false
	for range flankSteps {
		updateWolf(w, wolf)
		if wolf.Cell() == p.Cell() {
			t.Fatal("the wolf walked through the player")
		}
		if wolf.Cell() == behindCell {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("the wolf did not get behind the player: at %v, want %v", wolf.Cell(), behindCell)
	}
	for range 8 {
		if wolf.Recovery == "" {
			break
		}
		updateWolf(w, wolf)
	}
	if wolf.Recovery != "" {
		t.Fatalf("behind the player the wolf still flanks: at (%d, %d)", wolf.U, wolf.V)
	}
	hp := p.HitPoints
	for range 2000 {
		w.Update(nil)
		if p.HitPoints < hp {
			break
		}
	}
	if !behind(wolf.Facing, p.Facing) || hp-p.HitPoints != 6 {
		t.Errorf("the wolf struck facing %d for %d, want from behind the player facing %d for 6", wolf.Facing, hp-p.HitPoints, p.Facing)
	}
	if wolf.Recovery != "" || wolf.walkDir != restDir || wolf.walkCount < 10 {
		t.Errorf("after a blow from behind the wolf recovers %q, walk %d for %d; want a rest", wolf.Recovery, wolf.walkDir, wolf.walkCount)
	}
}

// TestAfterBlowFlankNoWay rests when there is no cell behind the player to make for.
func TestAfterBlowFlankNoWay(t *testing.T) {
	w, wolf := tacticsWorld(t, func(wolf *content.Thing) { wolf.AfterBlow = content.AfterBlowFlank },
		"######",
		"#@W..#",
		"######")
	w.Player().Facing = 2
	strikeOnce(t, w, wolf)
	updateWolf(w, wolf)
	if wolf.Recovery != "" || wolf.walkDir != restDir || wolf.walkCount < 9 {
		t.Errorf("with no way behind the wolf recovers %q, walk %d for %d; want a rest", wolf.Recovery, wolf.walkDir, wolf.walkCount)
	}
}
