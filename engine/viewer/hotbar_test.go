package viewer

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestHotbarLayout(t *testing.T) {
	v := newViewer(t)
	if v.Hotbar() != nil {
		t.Fatal("a hotbar while not playing")
	}
	v.SetPlay(true)
	h := v.Hotbar()
	if h == nil {
		t.Fatal("no hotbar while playing")
	}
	// 1280 pixels across: scale 2, centered along the bottom.
	if h.Scale != 2 || h.Rect.Max.Y != 800-Margin || h.Rect.Min.X+h.Rect.Max.X != 1280 {
		t.Errorf("hotbar at %v, scale %d", h.Rect, h.Scale)
	}
	if !h.Has || h.Portrait.Sprite == "" {
		t.Error("no portrait")
	}
	if len(h.Slots) != 2+sim.Shelves {
		t.Fatalf("%d slot boxes", len(h.Slots))
	}
	// Shelves 0–9 along the top row, 10–19 under them.
	s0, _ := h.SlotRect(0)
	s9, _ := h.SlotRect(9)
	s10, _ := h.SlotRect(10)
	if s9.Min.Y != s0.Min.Y || s9.Min.X-s0.Min.X != 9*SlotSize*h.Scale || s10.Min.X != s0.Min.X || s10.Min.Y != s0.Max.Y {
		t.Errorf("shelves 0 %v, 9 %v, 10 %v", s0, s9, s10)
	}
	// The schools two by two, row by row: Sorcery right of Wizardry, Enchantment under it.
	if sp := h.Spells; sp[1].Rect.Min.Y != sp[0].Rect.Min.Y || sp[1].Rect.Min.X <= sp[0].Rect.Min.X ||
		sp[2].Rect.Min.X != sp[0].Rect.Min.X || sp[2].Rect.Min.Y <= sp[0].Rect.Min.Y || sp[3].Rect.Min != image.Pt(sp[1].Rect.Min.X, sp[2].Rect.Min.Y) {
		t.Errorf("school boxes %v %v %v %v", sp[0].Rect, sp[1].Rect, sp[2].Rect, sp[3].Rect)
	}
	// Bare hands strike.
	if len(h.Actions) != 2 || h.Actions[0].Label != "strike" || !h.Actions[0].Ready {
		t.Errorf("actions %+v", h.Actions)
	}
	if g := h.Gauges[GaugeHitPoints]; g.Value != 56 || g.Max != 56 {
		t.Errorf("hit point gauge %+v", g)
	}
	// The log sits above the hotbar.
	v.Events = []string{"x"}
	if r := v.LogRect(); r.Max.Y > h.Rect.Min.Y {
		t.Errorf("log %v over the hotbar %v", r, h.Rect)
	}
}

// TestKeyIntoSlotWithMouse takes the key with a click on the player, puts it on a shelf and takes
// it back with clicks on the hotbar, and puts it in slot C with a click on the player standing
// there, which opens door C.
func TestKeyIntoSlotWithMouse(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	at := devpack.SlotC
	at[1]--
	v.World.Update([]input.Command{input.PlacePlayer{Cell: at}, input.Drop{Cell: at, Item: devpack.ItemKey}})
	if err := NewRunner(mustScript(t, `[
		{"at": 0, "click_player": true},
		{"at": 1, "slot": 0},
		{"at": 2, "slot": 0},
		{"at": 3, "slot": 0},
		{"at": 4, "slot": 0}
	]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	p := v.World.Player()
	if p.Hand == nil || p.Hand.Item != devpack.ItemKey || p.Slots[0] != nil {
		t.Fatalf("hand %v, shelf 0 %v", p.Hand, p.Slots[0])
	}
	v.World.Update([]input.Command{input.PlacePlayer{Cell: devpack.SlotC}})
	if err := NewRunner(mustScript(t, `[{"at": 0, "click_player": true}, {"at": 120}]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	if p.Hand != nil {
		t.Fatalf("the key is still in hand: %v", v.Events)
	}
	if v.World.Current().Blocked(devpack.DoorC) {
		t.Error("the key did not open door C")
	}
}

func TestRightClickStows(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	at := devpack.RocksB[0]
	v.World.Update([]input.Command{input.PlacePlayer{Cell: at}})
	p := v.World.Player()
	p.Slots[0] = &sim.Object{Item: devpack.ItemKey}
	if err := NewRunner(mustScript(t, `[{"at": 0, "click_player": true, "right": true}]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	if p.Hand != nil || p.Slots[1] == nil || p.Slots[1].Item != devpack.ItemRock {
		t.Fatalf("right click on the player: hand %v, shelf 1 %v", p.Hand, p.Slots[1])
	}
	v.Update(Input{Cursor: image.Pt(-1, -1)})
	p.Hand = &sim.Object{Item: devpack.ItemKey}
	if err := NewRunner(mustScript(t, `[{"at": 0, "slot": 5, "right": true}]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	if p.Hand != nil || p.Slots[2] == nil {
		t.Errorf("right click with a key in hand: hand %v, shelf 2 %v", p.Hand, p.Slots[2])
	}
}

func TestFigureColorsFromPortrait(t *testing.T) {
	v := newViewer(t)
	pal := v.Pack.Palettes[v.Pack.Manifest.Palette]
	pt := v.Pack.Portraits[v.World.Player().Portrait]
	c := v.FixedColors(pal)
	for i, k := range pt.FigureColors {
		if c[224+i] != pal.Colors[k] {
			t.Errorf("figure color %d is %v, want palette %d", 224+i, c[224+i], k)
		}
	}
	if c[230] != pal.Colors[230] {
		t.Error("a color past the figure's changed")
	}
}

func mustScript(t *testing.T, s string) Script {
	t.Helper()
	sc, err := ParseScript([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// TestRightClickDrinks right-clicks a potion on a shelf with nothing on the pointer: the player
// drinks it where it lies.
func TestRightClickDrinks(t *testing.T) {
	v := newViewer(t)
	v.SetPlay(true)
	p := v.World.Player()
	p.HitPoints = 1
	p.Slots[4] = &sim.Object{Item: devpack.ItemPotion}
	if err := NewRunner(mustScript(t, `[{"at": 0, "slot": 4, "right": true}]`), t.TempDir()).Run(v); err != nil {
		t.Fatal(err)
	}
	if p.Slots[4] != nil && p.Slots[4].Item == devpack.ItemPotion || p.HitPoints <= 1 || p.Hand != nil {
		t.Errorf("right-clicking the potion: shelf %+v, %d hit points, hand %+v", p.Slots[4], p.HitPoints, p.Hand)
	}
}
