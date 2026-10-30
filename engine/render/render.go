// Package render draws content with Ebitengine. The world is drawn in palette indexes into an
// offscreen buffer at the art's pixel scale; a Kage shader then looks each index up in the
// palette and scales the buffer to the screen. Palette effects change the lookup, not the art.
package render

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
)

// Index images hold a palette index in the red channel. Alpha is 0 where nothing is drawn and
// 255 elsewhere, so drawing them with the default blend copies indexes exactly.

// Sprites holds the pack's sprite banks as index images, one per frame.
type Sprites struct {
	frames map[string][]*ebiten.Image
}

// NewSprites uploads every sprite bank of p.
func NewSprites(p *content.Pack) *Sprites {
	s := &Sprites{frames: map[string][]*ebiten.Image{}}
	for id, b := range p.Sprites {
		s.frames[id] = uploadBank(b)
	}
	return s
}

func uploadBank(b *content.SpriteBank) []*ebiten.Image {
	bounds := b.Sheet.Bounds()
	pix := make([]byte, 4*bounds.Dx()*bounds.Dy())
	for i, idx := range b.Sheet.Pix {
		if b.Transparent != nil && int(idx) == *b.Transparent {
			continue
		}
		pix[4*i] = idx
		pix[4*i+3] = 0xFF
	}
	sheet := ebiten.NewImage(bounds.Dx(), bounds.Dy())
	sheet.WritePixels(pix)
	frames := make([]*ebiten.Image, b.Frames)
	for i := range frames {
		frames[i] = sheet.SubImage(b.FrameRect(i)).(*ebiten.Image)
	}
	return frames
}

// Frame returns frame i of a sprite bank.
func (s *Sprites) Frame(sprite string, i int) (*ebiten.Image, error) {
	frames, ok := s.frames[sprite]
	if !ok || i < 0 || i >= len(frames) {
		return nil, fmt.Errorf("no frame %d in sprite bank %q", i, sprite)
	}
	return frames[i], nil
}

// LevelView draws one level.
type LevelView struct {
	cmds []drawlist.Cmd
	// images holds each command's frame, or blank for a command that blanks a tile box.
	images []*ebiten.Image
	blank  *ebiten.Image
	bounds image.Rectangle
}

// NewLevelView prepares level l for drawing.
func NewLevelView(p *content.Pack, s *Sprites, l *content.Level) (*LevelView, error) {
	v := &LevelView{
		cmds:   drawlist.Build(p, l),
		blank:  ebiten.NewImage(drawlist.TileSize, drawlist.TileSize),
		bounds: drawlist.Bounds(l),
	}
	v.blank.Fill(color.White)
	v.images = make([]*ebiten.Image, len(v.cmds))
	for i, c := range v.cmds {
		img := v.blank
		if c.Sprite != "" {
			var err error
			if img, err = s.Frame(c.Sprite, c.Frame); err != nil {
				return nil, fmt.Errorf("level %s: %w", l.ID, err)
			}
		}
		v.images[i] = part(img, c.Part)
	}
	return v, nil
}

// part returns the part of img a command draws.
func part(img *ebiten.Image, p drawlist.Part) *ebiten.Image {
	b := img.Bounds()
	switch p {
	case drawlist.UpperHalf:
		b.Max.Y = b.Min.Y + drawlist.HalfHeight
	case drawlist.LowerHalf:
		b.Min.Y += drawlist.HalfHeight
	default:
		return img
	}
	return img.SubImage(b).(*ebiten.Image)
}

// Bounds returns the view-pixel rectangle the level covers.
func (v *LevelView) Bounds() image.Rectangle { return v.bounds }

// Draw draws the level into the index image dst, with view pixel origin at dst's top-left.
func (v *LevelView) Draw(dst *ebiten.Image, origin image.Point) {
	view := dst.Bounds().Add(origin)
	var op ebiten.DrawImageOptions
	for i, c := range v.cmds {
		img := v.images[i]
		r := img.Bounds().Sub(img.Bounds().Min).Add(image.Pt(c.X, c.Y))
		if !r.Overlaps(view) {
			continue
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(c.X-origin.X), float64(c.Y-origin.Y))
		op.Blend = ebiten.BlendSourceOver
		if c.Sprite == "" {
			op.Blend = ebiten.BlendClear
		}
		dst.DrawImage(img, &op)
	}
}

const paletteShaderSrc = `//kage:unit pixels
package main

var Palette [256]vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos)
	if c.a == 0 {
		return vec4(0, 0, 0, 1)
	}
	return Palette[int(c.r*255+0.5)]
}
`

// Presenter shows an index image on the screen through a palette.
type Presenter struct {
	shader  *ebiten.Shader
	palette []float32
}

// NewPresenter compiles the palette shader.
func NewPresenter() (*Presenter, error) {
	sh, err := ebiten.NewShader([]byte(paletteShaderSrc))
	if err != nil {
		return nil, fmt.Errorf("palette shader: %w", err)
	}
	return &Presenter{shader: sh, palette: make([]float32, 256*4)}, nil
}

// SetColors sets the colors indexes are looked up in, such as a palette's colors at a moment of
// its effects (content.Palette.At).
func (pr *Presenter) SetColors(colors [256]color.RGBA) {
	for i, c := range colors {
		pr.palette[4*i] = float32(c.R) / 255
		pr.palette[4*i+1] = float32(c.G) / 255
		pr.palette[4*i+2] = float32(c.B) / 255
		pr.palette[4*i+3] = 1
	}
}

// Present draws src onto dst through the palette, scaled by a whole number.
func (pr *Presenter) Present(dst, src *ebiten.Image, scale int) {
	var op ebiten.DrawRectShaderOptions
	op.Images[0] = src
	op.Uniforms = map[string]any{"Palette": pr.palette}
	op.GeoM.Scale(float64(scale), float64(scale))
	b := src.Bounds()
	dst.DrawRectShader(b.Dx(), b.Dy(), pr.shader, &op)
}
