package summoning

import (
	"bytes"
	"testing"
)

func TestDecompressRLE(t *testing.T) {
	// A literal run of three, then a repeat of four.
	src := []byte{0x02, 'a', 'b', 'c', 0xFD, 'x'}
	got, err := decompress(CompressionRLE, src, 7)
	if err != nil {
		t.Fatal(err)
	}
	if want := "abcxxxx"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Output stops when full, mid-run.
	got, err = decompress(CompressionRLE, src, 5)
	if err != nil {
		t.Fatal(err)
	}
	if want := "abcxx"; string(got) != want {
		t.Errorf("truncated: got %q, want %q", got, want)
	}

	if _, err := decompress(CompressionRLE, src, 8); err == nil {
		t.Error("short input: no error")
	}
}

func TestDecompressLZSS(t *testing.T) {
	src := []byte{
		// Flags: three literals, then two references.
		0b00000111,
		'A', 'B', 'C',
		// Position FEEh (where writing started), length 6: overlaps what it writes.
		0xEE, 0xF3,
		// Position 0, length 3: the ring's initial fill.
		0x00, 0x00,
	}
	got, err := decompress(CompressionLZSS, src, 12)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("ABCABCABC\xFE\xFE\xFE")
	if !bytes.Equal(got, want) {
		t.Errorf("got % X, want % X", got, want)
	}

	if _, err := decompress(CompressionLZSS, src, 13); err == nil {
		t.Error("short input: no error")
	}
}

func TestDecompressRejectsMode2(t *testing.T) {
	if _, err := decompress(2, []byte{0}, 1); err == nil {
		t.Error("mode 2: no error")
	}
}

// spriteResource builds a sprite bank resource record.
func spriteResource(frames uint16, height, widthUnits, transparent, mode uint8, payload []byte) []byte {
	f := fixture{}.str("EH").u8(uint8(ResourceSprites)).u16(spriteDescriptorSize)
	f = f.u16(frames).u8(height, widthUnits, transparent, mode, 0)
	return append(f, payload...)
}

func TestDecodeSpriteBank(t *testing.T) {
	// Two 8×1 frames from one RLE stream whose repeat run crosses the frame boundary.
	payload := []byte{0x04, 1, 2, 3, 4, 5, 0xF6, 9}
	r, err := ParseResource(spriteResource(2, 1, 1, 0, CompressionRLE, payload))
	if err != nil {
		t.Fatal(err)
	}
	b, err := DecodeSpriteBank(r)
	if err != nil {
		t.Fatal(err)
	}
	if b.Width != 8 || b.Height != 1 || b.Transparent != 0 || len(b.Frames) != 2 {
		t.Fatalf("got %d×%d transparent %d, %d frames", b.Width, b.Height, b.Transparent, len(b.Frames))
	}
	if want := []byte{1, 2, 3, 4, 5, 9, 9, 9}; !bytes.Equal(b.Frames[0], want) {
		t.Errorf("frame 0 = %v, want %v", b.Frames[0], want)
	}
	if want := []byte{9, 9, 9, 9, 9, 9, 9, 9}; !bytes.Equal(b.Frames[1], want) {
		t.Errorf("frame 1 = %v, want %v", b.Frames[1], want)
	}
}

func TestDecodeSpriteBankOpaque(t *testing.T) {
	r, err := ParseResource(spriteResource(1, 1, 1, 0xFF, CompressionRaw, []byte{0, 1, 2, 3, 4, 5, 6, 7}))
	if err != nil {
		t.Fatal(err)
	}
	b, err := DecodeSpriteBank(r)
	if err != nil {
		t.Fatal(err)
	}
	if b.Transparent != NoTransparency {
		t.Errorf("transparent = %d, want none", b.Transparent)
	}
}

func TestDecodeFont(t *testing.T) {
	f := fixture{}.str("EH").u8(uint8(ResourceFont)).u16(128 + spriteDescriptorSize)
	advance := make([]byte, 128)
	advance['A'] = 6
	f = append(f, advance...)
	f = f.u16(128).u8(1, 1, 0, CompressionRLE, 0)
	// 128 frames of 8 zero pixels: eight repeats of 128.
	for range 8 {
		f = f.u8(0x81, 0)
	}
	r, err := ParseResource(f)
	if err != nil {
		t.Fatal(err)
	}
	font, err := DecodeFont(r)
	if err != nil {
		t.Fatal(err)
	}
	if font.Advance['A'] != 6 || len(font.Glyphs.Frames) != 128 {
		t.Errorf("advance %d, %d glyphs", font.Advance['A'], len(font.Glyphs.Frames))
	}
}

func TestParseResourceRejects(t *testing.T) {
	tests := map[string][]byte{
		"short":           []byte("EH"),
		"bad magic":       fixture{}.str("XX").u8(1).u16(0),
		"long descriptor": fixture{}.str("EH").u8(1).u16(10).u8(1, 2),
	}
	for name, rec := range tests {
		if _, err := ParseResource(rec); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
