package summoning

import (
	"encoding/binary"
	"fmt"
)

// ResourceType is the kind of a RESOURCE record.
type ResourceType uint8

const (
	ResourceSprites ResourceType = 1
	ResourceFont    ResourceType = 2
	ResourceMusic   ResourceType = 3
)

// Resource is one RESOURCE record, split into its descriptor and payload.
type Resource struct {
	Type       ResourceType
	Descriptor []byte
	Payload    []byte
}

const resourceHeaderSize = 5

// ParseResource splits a RESOURCE record into its type, descriptor and payload.
func ParseResource(rec []byte) (Resource, error) {
	if len(rec) < resourceHeaderSize {
		return Resource{}, fmt.Errorf("resource: %d bytes, too short for a header", len(rec))
	}
	if rec[0] != 'E' || rec[1] != 'H' {
		return Resource{}, fmt.Errorf("resource: bad magic %q", rec[:2])
	}
	h := int(binary.LittleEndian.Uint16(rec[3:]))
	if resourceHeaderSize+h > len(rec) {
		return Resource{}, fmt.Errorf("resource: descriptor of %d bytes runs past the record", h)
	}
	return Resource{
		Type:       ResourceType(rec[2]),
		Descriptor: rec[resourceHeaderSize : resourceHeaderSize+h],
		Payload:    rec[resourceHeaderSize+h:],
	}, nil
}

// NoTransparency is SpriteBank.Transparent for an opaque bank.
const NoTransparency = -1

// maxSpriteBankPixels bounds a bank's decoded size, so a corrupt descriptor cannot ask for
// gigabytes.
const maxSpriteBankPixels = 64 << 20

const spriteDescriptorSize = 7

// SpriteBank is a decoded sprite bank: frames of the same size, one palette index per pixel.
type SpriteBank struct {
	Width, Height int
	// Transparent is the palette index drawn as transparent, or NoTransparency.
	Transparent int
	// Frames holds each frame's pixels, row-major.
	Frames [][]byte
}

// DecodeSpriteBank decodes a type 1 resource.
func DecodeSpriteBank(r Resource) (*SpriteBank, error) {
	if r.Type != ResourceSprites {
		return nil, fmt.Errorf("sprite bank: resource type %d", r.Type)
	}
	return decodeSprites(r.Descriptor, r.Payload)
}

// Font is a decoded type 2 resource: 128 glyphs, glyph n for character n.
type Font struct {
	Advance [128]uint8
	Glyphs  *SpriteBank
}

// DecodeFont decodes a type 2 resource.
func DecodeFont(r Resource) (*Font, error) {
	if r.Type != ResourceFont {
		return nil, fmt.Errorf("font: resource type %d", r.Type)
	}
	if len(r.Descriptor) < 128 {
		return nil, fmt.Errorf("font: descriptor of %d bytes", len(r.Descriptor))
	}
	f := &Font{}
	copy(f.Advance[:], r.Descriptor)
	glyphs, err := decodeSprites(r.Descriptor[128:], r.Payload)
	if err != nil {
		return nil, fmt.Errorf("font: %w", err)
	}
	f.Glyphs = glyphs
	return f, nil
}

func decodeSprites(desc, payload []byte) (*SpriteBank, error) {
	if len(desc) < spriteDescriptorSize {
		return nil, fmt.Errorf("sprite bank: descriptor of %d bytes", len(desc))
	}
	frames := int(binary.LittleEndian.Uint16(desc))
	b := &SpriteBank{
		Height:      int(desc[2]),
		Width:       int(desc[3]) * 8,
		Transparent: int(desc[4]),
	}
	if desc[4] == 0xFF {
		b.Transparent = NoTransparency
	}
	frameSize := b.Width * b.Height
	total := frames * frameSize
	if total > maxSpriteBankPixels {
		return nil, fmt.Errorf("sprite bank: %d pixels, over the limit", total)
	}
	pixels, err := decompress(desc[5], payload, total)
	if err != nil {
		return nil, fmt.Errorf("sprite bank: %w", err)
	}
	b.Frames = make([][]byte, frames)
	for i := range frames {
		b.Frames[i] = pixels[i*frameSize : (i+1)*frameSize : (i+1)*frameSize]
	}
	return b, nil
}
