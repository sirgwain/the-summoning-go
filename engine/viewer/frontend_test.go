package viewer

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func pressKey(v *Viewer, k Key) {
	v.Update(Input{Held: map[Key]bool{k: true}, Pressed: map[Key]bool{k: true}, Cursor: image.Pt(-1, -1)})
}

func typeText(v *Viewer, s string) {
	v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}, Cursor: image.Pt(-1, -1), Chars: []rune(s)})
}

// clickButton clicks the frontend's button name.
func clickButton(t *testing.T, v *Viewer, name string) {
	t.Helper()
	for _, b := range v.Front.Buttons {
		if b.Name == name {
			tapAt(v, b.Rect.Min.Add(image.Pt(2, 2)))
			return
		}
	}
	t.Fatalf("screen %s has no button %q: %+v", v.Front.Screen, name, v.Front.Buttons)
}

// TestCreateCharacter makes a character through the frontend's screens, with keys and clicks,
// spending points by hand, and starts play with it.
func TestCreateCharacter(t *testing.T) {
	v := newViewer(t)
	v.OpenFrontend()
	if v.Front.Screen != ScreenMenu || v.Front.Buttons[0].Name != "create" {
		t.Fatalf("menu %+v", v.Front.Buttons)
	}
	pressKey(v, KeyEnter)
	if v.Front.Screen != ScreenPortrait || v.Front.Portrait == nil {
		t.Fatalf("screen %s, portrait %v", v.Front.Screen, v.Front.Portrait)
	}
	clickButton(t, v, "ok")
	typeText(v, "Ash")
	pressKey(v, KeyBackspace)
	typeText(v, "h!")
	pressKey(v, KeyEnter)
	if v.Front.Screen != ScreenMethod {
		t.Fatalf("after the name: %s", v.Front.Screen)
	}
	// Spent: every attribute starts at 0 with points to spend; Keep shows only when all are spent.
	clickButton(t, v, "method.1")
	f := v.Front
	if f.Screen != ScreenAttributes || f.char.Attributes[sim.AttrStrength] != 0 || f.points < 28 {
		t.Fatalf("spent: %+v, %d points", f.char.Attributes, f.points)
	}
	for _, b := range f.Buttons {
		if b.Name == "ok" {
			t.Fatal("Keep before the points are spent")
		}
	}
	clickButton(t, v, "dec.0")
	if f.char.Attributes[sim.AttrStrength] != 0 {
		t.Error("an attribute went below its roll")
	}
	for f.points > 0 {
		before := f.points
		for i := range Attributes {
			if f.points > 0 {
				clickButton(t, v, "inc."+string(rune('0'+i)))
			}
		}
		if f.points == before {
			t.Fatalf("points stuck at %d: %v", f.points, f.char.Attributes)
		}
	}
	attrs := map[string]int{}
	for k, n := range f.char.Attributes {
		if n > 20 {
			t.Errorf("%s at %d, over 20", k, n)
		}
		attrs[k] = n
	}
	clickButton(t, v, "ok")
	clickButton(t, v, "train.0")
	clickButton(t, v, "train.0")
	if v.Front != nil {
		t.Fatalf("the frontend stayed on %s", v.Front.Screen)
	}
	p := v.World.Player()
	if !v.Play || p.Name != "Ash!" || p.Attributes[sim.AttrPower] != attrs[sim.AttrPower] || p.Slots[sim.SlotRightHand] == nil {
		t.Errorf("play %v, name %q, attributes %v", v.Play, p.Name, p.Attributes)
	}
	if v.playerName() != "Ash!" {
		t.Errorf("texts name the player %q", v.playerName())
	}
}

// TestPremadeAndBack picks the premade character, after backing out of creation with Escape.
func TestPremadeAndBack(t *testing.T) {
	v := newViewer(t)
	v.OpenFrontend()
	pressKey(v, KeyEnter)
	pressKey(v, KeyEscape)
	if v.Front.Screen != ScreenMenu {
		t.Fatalf("Escape from the portrait: %s", v.Front.Screen)
	}
	clickButton(t, v, "premade")
	clickButton(t, v, "premade.0")
	if p := v.World.Player(); v.Front != nil || p.Name != "Arden" || p.Attributes[sim.AttrAgility] != 20 {
		t.Errorf("the premade character: front %v, %q %v", v.Front, p.Name, p.Attributes)
	}
}

// TestDeathAndEndingGoToMenu: a dead player shows for a while, then the main menu; an ending
// plays its scene, then the menu.
func TestDeathAndEndingGoToMenu(t *testing.T) {
	v := playIn(t, devpack.Hall, devpack.HallStart)
	v.World.Player().Dead = true
	for range deathUpdates {
		if v.Front != nil {
			t.Fatal("the menu came at once")
		}
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	}
	if v.Front == nil || v.Front.Screen != ScreenMenu {
		t.Fatalf("no menu after death: %+v", v.Front)
	}

	p := devpack.Build()
	p.Scenes["scene.end"] = &content.Scene{ID: "scene.end", Steps: []content.SceneStep{
		{Picture: &content.Image{Sprite: "sprite.portrait"}, Clear: true, MS: 100},
		{Fade: "#ffffff", MS: 100}}}
	p.Creation.Ending = &content.Ending{Choices: []content.EndingChoice{{Name: "S", Holding: devpack.ItemSword, Scene: "scene.end"}}}
	p.Items[devpack.ItemRock].Power = 0x5E
	p.Items[devpack.ItemRock].Actions = []content.ItemAction{{Kind: content.ActionUse, Delay: 3}}
	v, err := New(p, 1, "", 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	v.SetPlay(true)
	pl := v.World.Player()
	pl.Slots[sim.SlotRightHand] = &sim.Object{Item: devpack.ItemRock}
	pl.Slots[sim.SlotLeftHand] = &sim.Object{Item: devpack.ItemSword}
	pressKey(v, KeySpace)
	if v.Front == nil || v.Front.Screen != ScreenScene || len(v.Front.Scene.Pictures) != 1 {
		t.Fatalf("no ending scene: %+v", v.Front)
	}
	for range 20 {
		v.Update(Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}})
	}
	if v.Front.Screen != ScreenMenu {
		t.Errorf("after the scene: %s, faded %v", v.Front.Screen, v.Front.Scene)
	}
}

// TestSceneRun plays a part's steps and a fade by the clock.
func TestSceneRun(t *testing.T) {
	r := NewSceneRun(&content.Scene{ID: "s", Steps: []content.SceneStep{
		{Part: &content.Part{From: 0, To: 2, StepMS: 50}, MS: 0},
		{Fade: "#000000", MS: 100},
	}})
	var spreads []int
	for !r.Update() {
		if s, ok := r.Spread(); ok {
			spreads = append(spreads, s)
		}
		if len(spreads) > 100 {
			t.Fatal("the scene does not end")
		}
	}
	if len(spreads) < 3 || spreads[0] != 0 || spreads[len(spreads)-1] != 2 {
		t.Errorf("spreads %v", spreads)
	}
	if r.Faded != 1 {
		t.Errorf("faded %v", r.Faded)
	}
}
