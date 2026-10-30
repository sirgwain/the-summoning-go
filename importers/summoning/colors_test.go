package summoning

import (
	"image/color"
	"testing"
)

func TestReadColors(t *testing.T) {
	// Twenty blocks: one full palette and four left over.
	f := fixture{}.u8(20)
	rgb := make([]byte, 20*16*3)
	copy(rgb, []byte{63, 32, 0})
	copy(rgb[255*3:], []byte{1, 2, 3})
	f = append(f, rgb...)

	pals, err := ReadColors(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(pals) != 1 {
		t.Fatalf("got %d palettes, want 1", len(pals))
	}
	if want := (color.RGBA{255, 130, 0, 255}); pals[0][0] != want {
		t.Errorf("color 0 = %v, want %v", pals[0][0], want)
	}
	if want := (color.RGBA{4, 8, 12, 255}); pals[0][255] != want {
		t.Errorf("color 255 = %v, want %v", pals[0][255], want)
	}
}

func TestReadColorsShort(t *testing.T) {
	if _, err := ReadColors(fixture{}.u8(16).zeros(100)); err == nil {
		t.Error("no error")
	}
}
