package content

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
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
	p.Walls["walls.a"] = &WallSet{ID: "walls.a", Sprite: "sprite.tiles", Offset: Offset{0, -8}, Shapes: []int{0, NoFrame, 2}}

	l := &Level{ID: "level.00", Name: "", Size: [2]int{2, 3}, Floors: "floors.a", Walls: "walls.a", ExitWalls: "walls.a"}
	l.Floor = Grid{{0, 1, 0}, {NoFrame, 0, 0}}
	l.Wall = Grid{{NoFrame, NoFrame, 2}, {0, NoFrame, NoFrame}}
	l.Exits = []Cell{{0, 2}}
	l.Overlays = []Placed{{Cell: Cell{1, 1}, Sprite: "sprite.tiles", Frame: 1, Offset: Offset{-16, 8}}}
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
	if !reflect.DeepEqual(got.Levels, want.Levels) || !reflect.DeepEqual(got.Floors, want.Floors) || !reflect.DeepEqual(got.Walls, want.Walls) {
		t.Error("tile sets or levels differ")
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
	if !strings.Contains(string(level), `". 0 0"`) {
		t.Errorf("floor grid not written one row per line:\n%s", level)
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
		"missing palette":  {func(p *Pack) { p.Manifest.Palette = "palette.9" }, "palette"},
		"start outside":    {func(p *Pack) { p.Manifest.Start.Cell = Cell{5, 5} }, "start cell"},
		"missing sprite":   {func(p *Pack) { p.Floors["floors.a"].Sprite = "sprite.none" }, "not found"},
		"too many floors":  {func(p *Pack) { p.Floors["floors.a"].Variants = 4 }, "variants"},
		"bad floor value":  {func(p *Pack) { p.Levels["level.00"].Floor[0][0] = 2 }, "out of range"},
		"missing shape":    {func(p *Pack) { p.Levels["level.00"].Wall[0][0] = 1 }, "out of range"},
		"short grid":       {func(p *Pack) { p.Levels["level.00"].Wall = p.Levels["level.00"].Wall[:1] }, "rows"},
		"exit not a wall":  {func(p *Pack) { p.Levels["level.00"].Exits = []Cell{{0, 0}} }, "not a wall"},
		"overlay frame":    {func(p *Pack) { p.Levels["level.00"].Overlays[0].Frame = 3 }, "frame 3"},
		"sheet too small":  {func(p *Pack) { p.Sprites["sprite.tiles"].Frames = 5 }, "do not fit"},
		"exit without set": {func(p *Pack) { p.Levels["level.00"].ExitWalls = "" }, "no exit wall set"},
		"rotate too far":   {func(p *Pack) { p.Palettes["palette.0"].Effects[0].First = 250 }, "outside the palette"},
		"no period":        {func(p *Pack) { p.Palettes["palette.0"].Effects[1].PeriodMS = 0 }, "period"},
		"unknown effect":   {func(p *Pack) { p.Palettes["palette.0"].Effects[1].Kind = "fade" }, "unknown kind"},
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
