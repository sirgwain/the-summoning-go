package content_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// modFS is a mod over the dev pack replacing its floor sprite with one at density d, of frames
// frames, its sheet a row of frames.
func modFS(t *testing.T, manifest map[string]any, d, frames int) fstest.MapFS {
	t.Helper()
	base := devpack.Build().Sprites["sprite.floors"]
	w, h := base.FrameSize[0]*d, base.FrameSize[1]*d
	b := map[string]any{"id": "sprite.floors", "frame_size": []int{w, h}, "columns": frames, "frames": frames, "density": d}
	sheet := image.NewPaletted(image.Rect(0, 0, w*frames, h), base.Sheet.Palette)
	var buf bytes.Buffer
	if err := png.Encode(&buf, sheet); err != nil {
		t.Fatal(err)
	}
	js := func(v any) []byte { data, _ := json.Marshal(v); return data }
	return fstest.MapFS{
		"pack.json":           {Data: js(manifest)},
		"sprites/floors.json": {Data: js(b)},
		"sprites/floors.png":  {Data: buf.Bytes()},
		"text/en.json":        {Data: js(map[string]any{"id": "text.en", "strings": map[string]string{"ui.paused": "Held"}})},
	}
}

func modManifest() map[string]any {
	return map[string]any{"id": "hires", "format": content.Format, "kind": content.ModKind, "depends": []string{"dev"}}
}

// TestLoadMod replaces a sprite with one at density 2 and merges text, reporting each change.
func TestLoadMod(t *testing.T) {
	p := devpack.Build()
	frames := p.Sprites["sprite.floors"].Frames
	changes, err := content.LoadMod(p, modFS(t, modManifest(), 2, frames))
	if err != nil {
		t.Fatal(err)
	}
	if b := p.Sprites["sprite.floors"]; b.Scale() != 2 || p.Density() != 2 {
		t.Errorf("density %d, pack %d", b.Scale(), p.Density())
	}
	if p.Texts["text.en"].Strings["ui.paused"] != "Held" || p.Texts["text.en"].Strings["msg.paused"] == "" {
		t.Error("the mod's text did not merge into the pack's")
	}
	if len(changes) != 2 || !strings.Contains(changes[0], "replaces sprite sprite.floors") {
		t.Errorf("changes %q", changes)
	}
}

// TestLoadModRefuses refuses a mod that is not one, depends on another pack, or replaces a
// sprite with one of other frames.
func TestLoadModRefuses(t *testing.T) {
	frames := devpack.Build().Sprites["sprite.floors"].Frames
	for name, c := range map[string]struct {
		edit   func(m map[string]any)
		frames int
		want   string
	}{
		"not a mod":    {func(m map[string]any) { delete(m, "kind") }, frames, "not a mod"},
		"other base":   {func(m map[string]any) { m["depends"] = []string{"summoning"} }, frames, "does not depend"},
		"other format": {func(m map[string]any) { m["format"] = 1 }, frames, "format"},
		"frames":       {func(map[string]any) {}, frames + 1, "frames"},
	} {
		m := modManifest()
		c.edit(m)
		_, err := content.LoadMod(devpack.Build(), modFS(t, m, 2, c.frames))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}
