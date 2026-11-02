package content

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func testPack() *Pack {
	p := NewPack(Manifest{
		ID: "test", Name: "Test", Version: "0.1.0", Format: Format, Depends: []string{},
		Palette: "palette.0",
		Start:   Start{Level: "level.00", Cell: Cell{1, 2}},
	})
	pal := &Palette{ID: "palette.0"}
	for i := range pal.Colors {
		pal.Colors[i] = color.RGBA{uint8(i), uint8(255 - i), 7, 255}
	}
	pal.Effects = []PaletteEffect{
		{Kind: EffectRotate, PeriodMS: 440, First: 0, Size: 4, Groups: 2},
		{Kind: EffectCycle, PeriodMS: 100, Index: 252, Colors: []color.RGBA{{1, 2, 3, 255}, {4, 5, 6, 255}}},
	}
	p.Palettes[pal.ID] = pal

	transparent := 9
	frames := [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 9, 9, 1}}
	sheet, cols := NewSheet(frames, 2, 2, 2, pal, &transparent, 9)
	p.Sprites["sprite.tiles"] = &SpriteBank{ID: "sprite.tiles", FrameSize: Offset{2, 2}, Columns: cols, Frames: 3, Transparent: &transparent, Sheet: sheet}

	p.Floors["floors.a"] = &FloorSet{ID: "floors.a", Sprite: "sprite.tiles", Variants: 2}
	p.Sprites["sprite.npc"] = &SpriteBank{ID: "sprite.npc", FrameSize: Offset{2, 2}, Columns: cols, Frames: 3, Transparent: &transparent, Sheet: sheet,
		Anchor: &Offset{1, 2},
		Animations: map[string]Animation{
			"walk.2": {Frames: []int{0, 1}, MS: Fixed(330)},
			"idle":   {Frames: []int{0, 1, 2}, MS: Duration{110, 440}, Mode: AnimRandom},
		}}

	p.Floors["floors.a"] = &FloorSet{ID: "floors.a", Sprite: "sprite.tiles", Variants: 2}
	p.Walls["walls.a"] = &WallSet{ID: "walls.a", Sprite: "sprite.tiles", Offset: Offset{0, -8},
		Shapes: []*WallShape{{Frame: 0, Diagonal: CornerBottom}, nil, {Frame: 2, Face: true}}}
	p.Looks["look.plate"] = &Look{ID: "look.plate", Layer: LayerFloor, States: map[string][]LookPart{
		"up":   {{Sprite: "sprite.tiles", Frame: 0, Offset: Offset{-16, 8}}},
		"down": {{Sprite: "sprite.tiles", Frame: 1, Offset: Offset{-16, 8}}},
	}}
	p.Looks["look.exit"] = &Look{ID: "look.exit", Layer: LayerWall, States: map[string][]LookPart{
		"exit": {{Sprite: "sprite.tiles", Frame: 2}},
	}}

	p.Texts["text.en"] = &Text{ID: "text.en", Strings: map[string]string{
		"item.bag.name": "Bag", "item.key.name": "Key", "msg.stuck": "It will not budge.",
	}}
	p.Messages = &Messages{ID: "messages", Messages: map[string]string{MsgLeverStuck: "msg.stuck"}}
	p.Items["item.bag"] = &Item{ID: "item.bag", Name: "item.bag.name", Weight: 2, Size: 3,
		Container: &Container{Capacity: 2, Largest: 3, Weightless: true}, Tags: []string{"bag"},
		Icon: Image{"sprite.tiles", 0}, Floor: Image{"sprite.tiles", 1}}
	p.Items["item.key"] = &Item{ID: "item.key", Name: "item.key.name", Weight: 1, Size: 1, Random: true,
		Icon: Image{"sprite.tiles", 2}, Floor: Image{"sprite.tiles", 2}}
	p.Things["thing.guard"] = &Thing{ID: "thing.guard", Kind: ThingCreature, Sprite: "sprite.npc", Facing: 2,
		Speed: 8, IntervalMS: 330, Weight: 100, HitPoints: Dice{3, 10, 8},
		Behaviors: []string{BehaviorHostile}, Stats: map[string]int{"accuracy": 10},
		Animations: map[string]Animation{"idle.4": {Frames: []int{2}}}}
	p.Things["thing.boulder"] = &Thing{ID: "thing.boulder", Kind: ThingBoulder, Sprite: "sprite.npc", Speed: 8, IntervalMS: 220, Weight: 100}

	l := &Level{ID: "level.00", Name: "", Size: [2]int{2, 3}, Floors: "floors.a", Walls: "walls.a"}
	l.FloorTypes = []FloorType{{Variant: 0}, {Variant: 1, Enter: &Effect{Damage: Dice{2, 2, 0}, Type: "fire"}, Sinks: true}}
	l.Floor = Grid{{0, 1, 0}, {NoFrame, 0, 0}}
	l.Wall = Grid{{NoFrame, NoFrame, 2}, {0, NoFrame, NoFrame}}
	l.Overlays = []Placed{{Cell: Cell{1, 1}, Sprite: "sprite.tiles", Frame: 1, Offset: Offset{-16, 8}}}
	l.Features = []*Feature{
		{ID: "plate.0.1", Kind: FeaturePlate, Cell: Cell{0, 1}, Look: "look.plate", Armed: true, Threshold: 50, OnPress: "p0", OnRelease: "p1"},
		{ID: "exit.0.2", Kind: FeatureExit, Cell: Cell{0, 2}, Look: "look.exit", To: &Destination{Level: "level.00", Cell: Cell{1, 2}}},
		{ID: "slot.1.1", Kind: FeatureSlot, Cell: Cell{1, 1}, Accepts: "item.key", Armed: true, OnInsert: "p0"},
	}
	none := (*string)(nil)
	l.Procedures = map[string][]Action{
		"p0": {{SetFloor: &SetFloor{Cell: Cell{1, 0}, Type: 1}}, {Roll: &Roll{Thing: "boulder.1.2", Direction: 2}}},
		"p1": {{SetHook: &SetHook{Feature: "plate.0.1", Hook: HookPress, Procedure: none}}, {Create: &Create{Cell: Cell{1, 2}, Item: "item.key"}}},
	}
	facing := 4
	l.Things = []*PlacedThing{
		{ID: "creature.1.1", Thing: "thing.guard", Cell: Cell{1, 1}, Offset: &Offset{-6, 12}, Facing: &facing, Items: []Object{{Item: "item.key"}}},
		{ID: "boulder.1.2", Thing: "thing.boulder", Cell: Cell{1, 2}, Rolling: &Rolling{Direction: 0, Shuttle: true}},
	}
	l.Objects = []*FloorObject{
		{Cell: Cell{0, 0}, Object: Object{Item: "item.bag", Contents: []Object{{Item: RandomItem}, {Item: "item.key"}}}},
	}
	p.Levels[l.ID] = l
	return p
}

func TestWriteLoad(t *testing.T) {
	dir := t.TempDir()
	want := testPack()
	if err := Write(want, dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.Manifest, want.Manifest) {
		t.Errorf("manifest %+v, want %+v", got.Manifest, want.Manifest)
	}
	if !reflect.DeepEqual(got.Palettes["palette.0"], want.Palettes["palette.0"]) {
		t.Errorf("palette %+v, want %+v", got.Palettes["palette.0"].Effects, want.Palettes["palette.0"].Effects)
	}
	for name, eq := range map[string]bool{
		"levels":   reflect.DeepEqual(got.Levels, want.Levels),
		"floors":   reflect.DeepEqual(got.Floors, want.Floors),
		"walls":    reflect.DeepEqual(got.Walls, want.Walls),
		"looks":    reflect.DeepEqual(got.Looks, want.Looks),
		"items":    reflect.DeepEqual(got.Items, want.Items),
		"things":   reflect.DeepEqual(got.Things, want.Things),
		"texts":    reflect.DeepEqual(got.Texts, want.Texts),
		"messages": reflect.DeepEqual(got.Messages, want.Messages),
	} {
		if !eq {
			t.Errorf("%s differ", name)
		}
	}
	gn, wn := got.Sprites["sprite.npc"], want.Sprites["sprite.npc"]
	if !reflect.DeepEqual(gn.Anchor, wn.Anchor) || !reflect.DeepEqual(gn.Animations, wn.Animations) {
		t.Errorf("sprite anchor or animations %+v, want %+v", gn.Animations, wn.Animations)
	}
	gb, wb := got.Sprites["sprite.tiles"], want.Sprites["sprite.tiles"]
	if gb.Columns != wb.Columns || gb.Frames != wb.Frames || *gb.Transparent != *wb.Transparent {
		t.Errorf("sprite bank %+v, want %+v", gb, wb)
	}
	r := gb.FrameRect(2)
	if gb.Sheet.ColorIndexAt(r.Min.X+1, r.Min.Y+1) != 1 || gb.Sheet.ColorIndexAt(r.Min.X, r.Min.Y) != 9 {
		t.Error("frame 2 pixels differ")
	}
	if gb.Sheet.ColorIndexAt(3, 2) != 9 {
		t.Error("the unused frame cell is not filled")
	}
	if _, _, _, a := gb.Sheet.Palette[9].RGBA(); a != 0 {
		t.Error("transparent index is opaque in the PNG")
	}

	level, err := os.ReadFile(filepath.Join(dir, "levels", "00", "level.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`". 0 0"`, `"damage": "2d2"`, `"procedure": null`} {
		if !strings.Contains(string(level), want) {
			t.Errorf("level.json has no %s:\n%s", want, level)
		}
	}
	sprite, err := os.ReadFile(filepath.Join(dir, "sprites", "npc.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sprite), `"ms": [110, 440]`) || !strings.Contains(string(sprite), `"ms": 330`) {
		t.Errorf("durations not written as a number and a range:\n%s", sprite)
	}
}

func TestAnimation(t *testing.T) {
	p := testPack()
	guard := p.Things["thing.guard"]
	for _, tc := range []struct {
		names []string
		frame int
		ok    bool
	}{
		{[]string{"idle.4", "idle"}, 2, true}, // the thing's own
		{[]string{"idle.0", "idle"}, 0, true}, // falls back to the sprite's idle
		{[]string{"walk.2"}, 0, true},         // the sprite's
		{[]string{"attack.2"}, 0, false},
	} {
		a, ok := p.Animation(guard, tc.names...)
		if ok != tc.ok || (ok && a.Frames[0] != tc.frame) {
			t.Errorf("%v: %+v %v", tc.names, a, ok)
		}
	}
}

func TestDice(t *testing.T) {
	for s, want := range map[string]Dice{"3d10+8": {3, 10, 8}, "2d2": {2, 2, 0}, "5": {0, 0, 5}} {
		d, err := ParseDice(s)
		if err != nil || d != want || d.String() != s {
			t.Errorf("%s: %+v %v", s, d, err)
		}
	}
	for _, s := range []string{"", "d6", "3d", "3d0", "x", "1d6+"} {
		if _, err := ParseDice(s); err == nil {
			t.Errorf("%q: no error", s)
		}
	}
}

func TestGridJSON(t *testing.T) {
	g := Grid{{0, NoFrame, 12}, {}}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `["0 . 12",""]` {
		t.Errorf("got %s", data)
	}
	var back Grid
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, g) {
		t.Errorf("round trip %v, want %v", back, g)
	}
	if err := json.Unmarshal([]byte(`["0 x"]`), &back); err == nil {
		t.Error("bad value: no error")
	}
}

func TestVerify(t *testing.T) {
	tests := map[string]struct {
		break_ func(p *Pack)
		want   string
	}{
		"missing palette": {func(p *Pack) { p.Manifest.Palette = "palette.9" }, "palette"},
		"start outside":   {func(p *Pack) { p.Manifest.Start.Cell = Cell{5, 5} }, "start cell"},
		"missing sprite":  {func(p *Pack) { p.Floors["floors.a"].Sprite = "sprite.none" }, "not found"},
		"too many floors": {func(p *Pack) { p.Floors["floors.a"].Variants = 4 }, "variants"},
		"floor shape":     {func(p *Pack) { p.Floors["floors.a"].Shape = "hex" }, "unknown shape"},
		"diamond frames":  {func(p *Pack) { p.Floors["floors.a"].Shape = FloorDiamond }, "diamond set wants"},
		"bad floor value": {func(p *Pack) { p.Levels["level.00"].Floor[0][0] = 2 }, "out of range"},
		"missing shape":   {func(p *Pack) { p.Levels["level.00"].Wall[0][0] = 1 }, "out of range"},
		"short grid":      {func(p *Pack) { p.Levels["level.00"].Wall = p.Levels["level.00"].Wall[:1] }, "rows"},
		"overlay frame":   {func(p *Pack) { p.Levels["level.00"].Overlays[0].Frame = 3 }, "frame 3"},
		"sheet too small": {func(p *Pack) { p.Sprites["sprite.tiles"].Frames = 5 }, "do not fit"},
		"floor type":      {func(p *Pack) { p.Levels["level.00"].Floor[0][0] = 2 }, "out of range"},
		"floor variant":   {func(p *Pack) { p.Levels["level.00"].FloorTypes[1].Variant = 2 }, "variant 2"},
		"anim frame": {func(p *Pack) {
			p.Sprites["sprite.npc"].Animations["walk.2"] = Animation{Frames: []int{7}, MS: Fixed(1)}
		}, "frame 7"},
		"anim mode": {func(p *Pack) {
			p.Sprites["sprite.npc"].Animations["idle"] = Animation{Frames: []int{0}, Mode: "bounce"}
		}, "unknown mode"},
		"anchors":         {func(p *Pack) { p.Sprites["sprite.npc"].Anchors = []Offset{{0, 0}} }, "anchors"},
		"thing anim":      {func(p *Pack) { p.Things["thing.guard"].Animations = map[string]Animation{"idle": {Frames: []int{3}}} }, "frame 3 of 3"},
		"look layer":      {func(p *Pack) { p.Looks["look.plate"].Layer = "sky" }, "unknown layer"},
		"look frame":      {func(p *Pack) { p.Looks["look.plate"].States["up"][0].Frame = 9 }, "frame 9"},
		"item text":       {func(p *Pack) { p.Items["item.key"].Name = "item.nope" }, "not in text.en"},
		"item icon":       {func(p *Pack) { p.Items["item.key"].Icon.Sprite = "sprite.none" }, "not found"},
		"thing kind":      {func(p *Pack) { p.Things["thing.guard"].Kind = "ghost" }, "unknown kind"},
		"behavior":        {func(p *Pack) { p.Things["thing.guard"].Behaviors = []string{"sleepy"} }, "unknown behavior"},
		"message key":     {func(p *Pack) { p.Messages.Messages["hello"] = "msg.stuck" }, "unknown message"},
		"message text":    {func(p *Pack) { p.Messages.Messages[MsgLeverStuck] = "msg.nope" }, "not in text.en"},
		"feature kind":    {func(p *Pack) { p.Levels["level.00"].Features[0].Kind = "fountain" }, "unknown kind"},
		"feature cell":    {func(p *Pack) { p.Levels["level.00"].Features[1].Cell = Cell{0, 1} }, "share cell"},
		"feature dup":     {func(p *Pack) { p.Levels["level.00"].Features[1].ID = "plate.0.1" }, "duplicate feature"},
		"look state":      {func(p *Pack) { p.Levels["level.00"].Features[0].Look = "look.exit" }, `no state "up"`},
		"hook procedure":  {func(p *Pack) { p.Levels["level.00"].Features[0].OnPress = "p9" }, `procedure "p9"`},
		"wrong hook":      {func(p *Pack) { p.Levels["level.00"].Features[0].OnInsert = "p0" }, "no hook on_insert"},
		"slot item":       {func(p *Pack) { p.Levels["level.00"].Features[2].Accepts = "item.nope" }, "not found"},
		"exit level":      {func(p *Pack) { p.Levels["level.00"].Features[1].To.Level = "level.09" }, "not found"},
		"exit cell":       {func(p *Pack) { p.Levels["level.00"].Features[1].To.Cell = Cell{9, 9} }, "outside"},
		"two ops":         {func(p *Pack) { a := &p.Levels["level.00"].Procedures["p0"][0]; a.Stop = new(string) }, "2 operations"},
		"action target":   {func(p *Pack) { p.Levels["level.00"].Procedures["p1"][0].SetHook.Feature = "door.9" }, `feature "door.9"`},
		"action hook":     {func(p *Pack) { p.Levels["level.00"].Procedures["p1"][0].SetHook.Hook = HookInsert }, "no hook on_insert"},
		"roll creature":   {func(p *Pack) { p.Levels["level.00"].Procedures["p0"][1].Roll.Thing = "creature.1.1" }, "not a boulder"},
		"set floor type":  {func(p *Pack) { p.Levels["level.00"].Procedures["p0"][0].SetFloor.Type = 5 }, "floor type 5"},
		"thing template":  {func(p *Pack) { p.Levels["level.00"].Things[0].Thing = "thing.nope" }, "not found"},
		"creature rolls":  {func(p *Pack) { p.Levels["level.00"].Things[0].Rolling = &Rolling{} }, "only boulders"},
		"boulder carries": {func(p *Pack) { p.Levels["level.00"].Things[1].Items = []Object{{Item: "item.key"}} }, "only creatures"},
		"not a container": {func(p *Pack) { p.Levels["level.00"].Objects[0].Item = "item.key" }, "not a container"},
		"over capacity": {func(p *Pack) {
			o := p.Levels["level.00"].Objects[0]
			o.Contents = append(o.Contents, Object{Item: "item.key"})
		}, "capacity"},
		"rotate too far": {func(p *Pack) { p.Palettes["palette.0"].Effects[0].First = 250 }, "outside the palette"},
		"no period":      {func(p *Pack) { p.Palettes["palette.0"].Effects[1].PeriodMS = 0 }, "period"},
		"unknown effect": {func(p *Pack) { p.Palettes["palette.0"].Effects[1].Kind = "fade" }, "unknown kind"},
	}
	if err := testPack().Verify(); err != nil {
		t.Fatalf("valid pack: %v", err)
	}
	for name, tc := range tests {
		p := testPack()
		tc.break_(p)
		err := p.Verify()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error with %q", name, err, tc.want)
		}
	}
}

func TestPaletteAt(t *testing.T) {
	p := testPack().Palettes["palette.0"]
	at := func(ms int64, i int) uint8 { c := p.At(ms); return c[i].R }

	// Rotation: two groups of four from 0; each step every color takes the next one's value.
	if at(0, 0) != 0 || at(439, 3) != 3 {
		t.Error("rotated before the first period")
	}
	if at(440, 0) != 1 || at(440, 3) != 0 || at(440, 4) != 5 || at(440, 7) != 4 {
		t.Error("one step of rotation wrong")
	}
	if at(4*440, 2) != 2 {
		t.Error("four steps do not come back around")
	}
	if at(440, 8) != 8 {
		t.Error("a color outside the groups changed")
	}

	// Cycle: color 252 through its two colors, one per 100 ms.
	if at(0, 252) != 1 || at(100, 252) != 4 || at(200, 252) != 1 {
		t.Error("cycle wrong")
	}
}

// TestFill shows a container's own look empty and a step for each Per objects, up to the last.
func TestFill(t *testing.T) {
	it := &Item{
		Icon: Image{Sprite: "s", Frame: 0}, Doll: &DollImage{Image: Image{Sprite: "d"}},
		Fill: &Fill{Per: 3, Icons: []Image{{Sprite: "s", Frame: 1}, {Sprite: "s", Frame: 2}},
			Dolls: []DollImage{{Image: Image{Sprite: "d1"}}}},
	}
	for n, want := range []int{0, 1, 1, 1, 2, 2, 2, 2, 2} {
		if got := it.IconFor(n).Frame; got != want {
			t.Errorf("%d objects: icon frame %d, want %d", n, got, want)
		}
	}
	if it.DollFor(0).Image.Sprite != "d" || it.DollFor(5).Image.Sprite != "d1" {
		t.Errorf("dolls %+v %+v", it.DollFor(0), it.DollFor(5))
	}
	if plain := (&Item{Icon: Image{Frame: 7}}); plain.IconFor(4).Frame != 7 || plain.DollFor(4) != nil {
		t.Error("an item without a fill changed its look")
	}
}

// TestWallSetTaken follows a wall's unseen pairs from shape to shape until none applies.
func TestWallSetTaken(t *testing.T) {
	w := &WallSet{Shapes: []*WallShape{
		{Unseen: [][2]int{{2, 1}}},
		{Unseen: [][2]int{{0, 2}, {4, 3}}},
		{},
		{},
	}}
	dirs := [8][2]int{{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}}
	in := func(c Cell) bool { return c[0] >= 0 && c[1] >= 0 && c[0] < 3 && c[1] < 3 }
	for _, c := range []struct {
		hidden []Cell
		want   int
	}{
		{nil, 0},
		{[]Cell{{1, 2}}, 1},         // (1, 1) + direction 2 unseen: shape 1
		{[]Cell{{1, 2}, {2, 1}}, 3}, // then direction 4 unseen: shape 3
		{[]Cell{{1, 2}, {0, 1}}, 2}, // direction 0 first
	} {
		visible := func(v Cell) bool { return !slices.Contains(c.hidden, v) }
		if got := w.Taken(0, Cell{1, 1}, dirs, in, visible); got != c.want {
			t.Errorf("hidden %v: shape %d, want %d", c.hidden, got, c.want)
		}
	}
	// A neighbor outside the level does not count as unseen.
	if got := w.Taken(0, Cell{1, 2}, dirs, in, func(Cell) bool { return false }); got != 0 {
		t.Errorf("outside: shape %d", got)
	}
}
