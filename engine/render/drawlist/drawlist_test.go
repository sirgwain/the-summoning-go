package drawlist

import (
	"image"
	"reflect"
	"slices"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

func TestTileBox(t *testing.T) {
	tests := map[content.Cell]image.Point{
		{0, 0}: {0, 0},
		{0, 1}: {32, 16},
		{1, 0}: {-32, 16},
		{1, 1}: {0, 32},
		{2, 5}: {96, 112},
	}
	for c, want := range tests {
		if got := TileBox(c); got != want {
			t.Errorf("TileBox(%v) = %v, want %v", c, got, want)
		}
	}
}

func testPack() (*content.Pack, *content.Level) {
	p := content.NewPack(content.Manifest{})
	p.Floors["f"] = &content.FloorSet{ID: "f", Sprite: "floor", Variants: 2}
	p.Walls["w"] = &content.WallSet{ID: "w", Sprite: "wall", Offset: content.Offset{0, -8},
		Shapes: []*content.WallShape{{Frame: 0}, {Frame: 1}, {Frame: 2}, {Frame: 3}}}
	p.Looks["exit"] = &content.Look{ID: "exit", Layer: content.LayerWall, States: map[string][]content.LookPart{
		"exit": {{Sprite: "exit", Frame: 1, Offset: content.Offset{0, -16}}},
	}}
	p.Looks["plate"] = &content.Look{ID: "plate", Layer: content.LayerFloor, States: map[string][]content.LookPart{
		"up": {{Sprite: "plate", Frame: 0, Offset: content.Offset{-16, 8}}}, "down": {{Sprite: "plate", Frame: 1, Offset: content.Offset{-16, 8}}},
	}}
	p.Looks["lever"] = &content.Look{ID: "lever", Layer: content.LayerStand, States: map[string][]content.LookPart{
		"off": {{Sprite: "lever", Frame: 0}}, "on": {{Sprite: "lever", Frame: 1, Offset: content.Offset{4, 2}}},
	}}
	p.Sprites["tiny"] = &content.SpriteBank{ID: "tiny", FrameSize: content.Offset{8, 6}, Frames: 1}
	p.Sprites["icon"] = &content.SpriteBank{ID: "icon", FrameSize: content.Offset{16, 16}, Frames: 1}
	p.Items["key"] = &content.Item{ID: "key", Icon: content.Image{Sprite: "icon"}, Floor: content.Image{Sprite: "tiny"}}
	p.Sprites["npc"] = &content.SpriteBank{ID: "npc", FrameSize: content.Offset{32, 32}, Frames: 3, Anchor: &content.Offset{18, 19}}
	p.Things["guard"] = &content.Thing{ID: "guard", Kind: content.ThingCreature, Sprite: "npc", Facing: 4,
		Animations: map[string]content.Animation{"idle.4": {Frames: []int{2}}}}
	n := content.NoFrame
	l := &content.Level{
		Size: [2]int{2, 2}, Floors: "f", Walls: "w",
		// (0, 0) floor with a lever; (0, 1) wall 1 with a decoration; (1, 0) exit in wall 3;
		// (1, 1) floor with a plate.
		FloorTypes:  []content.FloorType{{Variant: 1}, {Variant: 0}},
		Floor:       content.Grid{{0, 1}, {1, 1}},
		Wall:        content.Grid{{n, 1}, {3, n}},
		Overlays:    []content.Placed{{Cell: content.Cell{1, 1}, Sprite: "hazard", Frame: 2}},
		Decorations: []content.Placed{{Cell: content.Cell{0, 1}, Sprite: "deco", Frame: 4, Offset: content.Offset{8, 8}}, {Cell: content.Cell{1, 0}, Sprite: "deco", Frame: 5}},
		Props:       []content.Placed{{Cell: content.Cell{0, 0}, Sprite: "door", Frame: 0, Offset: content.Offset{-17, -3}}},
		Features: []*content.Feature{
			{ID: "exit", Kind: content.FeatureExit, Cell: content.Cell{1, 0}, Look: "exit"},
			{ID: "plate", Kind: content.FeaturePlate, Cell: content.Cell{1, 1}, Look: "plate", Down: true},
			{ID: "lever", Kind: content.FeatureLever, Cell: content.Cell{0, 0}, Look: "lever", On: true},
		},
		Objects: []*content.FloorObject{{Cell: content.Cell{0, 0}, Object: content.Object{Item: "key"}}},
		// A creature in (1, 1) off center: its ground point is (42, 60).
		Things: []*content.PlacedThing{{ID: "guard", Thing: "guard", Cell: content.Cell{1, 1}, Offset: &content.Offset{-6, 12}}},
	}
	return p, l
}

func TestBuild(t *testing.T) {
	p, l := testPack()
	got := Build(p, LevelScene(p, l), Options{})
	want := []Cmd{
		// Floors back to front, row by row and left to right, with the cells past the far edges
		// blanked.
		{"floor", 1, 0, 0, Whole, "", false},
		{"floor", 0, -32, 16, Whole, "", false},
		{"floor", 0, 32, 16, Whole, "", false},
		{"", 0, -64, 32, Whole, "", false},
		{"floor", 0, 0, 32, Whole, "", false},
		{"", 0, 64, 32, Whole, "", false},
		{"", 0, -32, 48, Whole, "", false},
		{"", 0, 32, 48, Whole, "", false},
		{"", 0, 0, 64, Whole, "", false},
		// Overlays, then floor looks in their state.
		{"hazard", 2, 0, 32, Whole, "", false},
		{"plate", 1, -16, 40, Whole, "", false},
		// The key centered in (0, 0).
		{"tiny", 0, 12, 13, Whole, "", false},
		// Row 0: the lever and the door; then row 1: the exit look in place of its wall and
		// decoration, and the wall with its decoration.
		{"lever", 1, 4, 2, Whole, "", false},
		{"door", 0, -17, -3, Whole, "", false},
		{"exit", 1, -32, 0, Whole, "", false},
		{"wall", 1, 32, 8, Whole, "", false},
		{"deco", 4, 40, 24, Whole, "", false},
		// Row 2: the guard, facing 4, its anchor at the reference point (34, 51).
		{"npc", 2, 16, 32, Whole, "guard", false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}

func TestBuildFloorHalves(t *testing.T) {
	p, l := testPack()
	l.FloorHalves = content.Grid{{content.HalvesBoth, content.HalfLower}, {0, content.HalvesBoth}}
	got := Build(p, LevelScene(p, l), Options{})[:5]
	want := []Cmd{
		{"floor", 1, 0, 0, Whole, "", false},
		// (1, 0): both halves blanked.
		{"", 0, -32, 16, UpperHalf, "", false},
		{"", 0, -32, 32, LowerHalf, "", false},
		// (0, 1): only the lower half drawn.
		{"", 0, 32, 16, UpperHalf, "", false},
		{"floor", 0, 32, 32, LowerHalf, "", false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}

func TestBuildDiamondFloors(t *testing.T) {
	p, l := testPack()
	p.Floors["f"].Shape = content.FloorDiamond
	l.FloorHalves = content.Grid{{content.HalvesBoth, content.HalfLower}, {0, content.HalvesBoth}}
	var floors []Cmd
	for _, c := range Build(p, LevelScene(p, l), Options{}) {
		if c.Sprite == "floor" || c.Sprite == "" {
			floors = append(floors, c)
		}
	}
	want := []Cmd{
		{"floor", 1, -16, 0, Whole, "", false},
		// (1, 0) draws nothing; (0, 1) only its lower half; nothing is blanked.
		{"floor", 0, 16, 32, LowerHalf, "", false},
		{"floor", 0, -16, 32, Whole, "", false},
	}
	if !reflect.DeepEqual(floors, want) {
		t.Errorf("got\n%v\nwant\n%v", floors, want)
	}
}

func TestBuildMagnified(t *testing.T) {
	p, l := testPack()
	for _, c := range Build(p, LevelScene(p, l), Options{Magnify: true}) {
		if c.Sprite == "icon" && (c.X != 8 || c.Y != 8) {
			t.Errorf("magnified key at (%d, %d), want (8, 8)", c.X, c.Y)
		}
		if c.Sprite == "tiny" {
			t.Error("magnified key drawn with its floor image")
		}
	}
}

func TestRefPoint(t *testing.T) {
	// A cell's corner lands 16 pixels right of its tile box; odd units round down.
	if got := RefPoint(32, 64); got != TileBox(content.Cell{1, 2}).Add(image.Pt(16, 0)) {
		t.Errorf("corner: %v", got)
	}
	if got := RefPoint(-1, 3); got != image.Pt(20, 0) {
		t.Errorf("RefPoint(-1, 3) = %v, want (20, 0)", got)
	}
}

func TestBounds(t *testing.T) {
	_, l := testPack()
	if got, want := Bounds(l), image.Rect(-32, -32, 64, 64); got != want {
		t.Errorf("Bounds = %v, want %v", got, want)
	}
}

func TestVisibleCellsOnly(t *testing.T) {
	p, l := testPack()
	s := LevelScene(p, l)
	s.Visible = func(c content.Cell) bool { return c == content.Cell{0, 0} }
	for _, c := range Build(p, s, Options{}) {
		switch c.Sprite {
		case "wall", "exit", "plate", "hazard", "deco", "npc":
			t.Errorf("drew %s %d of a cell not visible", c.Sprite, c.Frame)
		}
	}
	var floors int
	for _, c := range Build(p, s, Options{}) {
		if c.Sprite == "floor" {
			floors++
		}
	}
	if floors != 1 {
		t.Errorf("drew %d floors, want the visible one", floors)
	}
}

func TestThingSteppingAnimation(t *testing.T) {
	p, l := testPack()
	p.Things["guard"].Animations["walk.4"] = content.Animation{Frames: []int{0, 1}}
	s := LevelScene(p, l)
	frame := func(anim string, phase int) int {
		s.Things[0].Anim, s.Things[0].Phase = anim, phase
		for _, c := range Build(p, s, Options{}) {
			if c.Thing == s.Things[0].ID {
				return c.Frame
			}
		}
		t.Fatal("the guard is not drawn")
		return -1
	}
	if f := frame("walk.4", 3); f != 1 {
		t.Errorf("walk step 3 drew frame %d, want 1", f)
	}
	if f := frame("roll.4", 3); f != 2 {
		t.Errorf("a missing animation drew frame %d, want the idle frame 2", f)
	}
}

// TestPlayerFixedColors draws the player's figure and what it holds with the palette's own
// colors, and other things with its animated ones.
func TestPlayerFixedColors(t *testing.T) {
	p, l := testPack()
	p.Things["hero"] = &content.Thing{ID: "hero", Kind: content.ThingPlayer, Sprite: "npc"}
	p.Items["sword"] = &content.Item{ID: "sword", Held: &content.Held{Right: &content.HeldLayers{Behind: "npc", Front: "npc"}}}
	s := LevelScene(p, l)
	s.Things = append(s.Things, Thing{ID: "hero", Template: "hero", U: 32, V: 32, Held: [2]string{"sword"}})
	drawn, fixed := map[string]int{}, map[string]int{}
	for _, c := range Build(p, s, Options{}) {
		drawn[c.Thing]++
		if c.Fixed {
			fixed[c.Thing]++
		}
	}
	if drawn["hero"] != 3 || fixed["hero"] != 3 {
		t.Errorf("%d of the player's %d commands are fixed, want its figure and both held layers", fixed["hero"], drawn["hero"])
	}
	if drawn["guard"] == 0 || fixed["guard"] != 0 {
		t.Errorf("%d of the guard's %d commands are fixed, want none", fixed["guard"], drawn["guard"])
	}
}

// TestFigureLayers draws a thing with looks as its layers, bottom up, in place of its own
// sprite, between the held layers behind and in front.
func TestFigureLayers(t *testing.T) {
	p, l := testPack()
	p.Things["hero"] = &content.Thing{ID: "hero", Kind: content.ThingPlayer, Sprite: "npc"}
	p.Items["sword"] = &content.Item{ID: "sword", Held: &content.Held{Right: &content.HeldLayers{Behind: "npc", Front: "npc"}}}
	legs := *p.Sprites["npc"]
	legs.ID = "legs"
	p.Sprites["legs"] = &legs
	s := LevelScene(p, l)
	s.Things = append(s.Things, Thing{ID: "hero", Template: "hero", U: 32, V: 32, Held: [2]string{"sword"},
		Looks: []string{"legs", "missing", "legs"}})
	var sprites []string
	for _, c := range Build(p, s, Options{}) {
		if c.Thing == "hero" {
			if !c.Fixed {
				t.Errorf("a figure layer %s is not fixed", c.Sprite)
			}
			sprites = append(sprites, c.Sprite)
		}
	}
	if want := []string{"npc", "legs", "legs", "npc"}; !slices.Equal(sprites, want) {
		t.Errorf("the hero draws %v, want %v", sprites, want)
	}
}

// TestFloorSight hides the halves of a closed door's floor whose sight neighbors are not visible,
// and shows the floor whole once the door opens.
func TestFloorSight(t *testing.T) {
	p, l := testPack()
	l.Wall = content.Grid{{content.NoFrame, content.NoFrame}, {content.NoFrame, content.NoFrame}}
	l.Features = []*content.Feature{{ID: "door", Kind: content.FeatureDoor, Cell: content.Cell{0, 0}, State: content.DoorClosed}}
	l.Overlays, l.Decorations, l.Props, l.Objects, l.Things = nil, nil, nil, nil, nil
	n := content.NoFrame
	// The door's upper half shows while (0, 1) is visible (direction 2), its lower while (1, 0) is
	// (direction 4).
	l.UpperSight = content.Grid{{2, n}, {n, n}}
	l.LowerSight = content.Grid{{4, n}, {n, n}}
	parts := func(state string, visible ...content.Cell) []Part {
		s := LevelScene(p, l)
		s.Features[0].State = state
		s.Visible = func(c content.Cell) bool {
			if c == (content.Cell{0, 0}) {
				return true
			}
			for _, v := range visible {
				if c == v {
					return true
				}
			}
			return false
		}
		var out []Part
		for _, c := range Build(p, s, Options{}) {
			if c.Sprite == "floor" && c.X == 0 && (c.Y == 0 || c.Y == HalfHeight) {
				out = append(out, c.Part)
			}
		}
		return out
	}
	if got := parts(content.DoorClosed, content.Cell{0, 1}); !reflect.DeepEqual(got, []Part{UpperHalf}) {
		t.Errorf("closed door, (0, 1) visible: %v", got)
	}
	if got := parts(content.DoorClosed); len(got) != 0 {
		t.Errorf("closed door, nothing beyond visible: %v", got)
	}
	if got := parts(content.DoorOpen); !reflect.DeepEqual(got, []Part{Whole}) {
		t.Errorf("open door: %v", got)
	}
}
