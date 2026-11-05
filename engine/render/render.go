// Package render draws content with Ebitengine. The world is drawn in palette indexes into an
// offscreen buffer at the art's pixel scale; a Kage shader then looks each index up in the
// palette and scales the buffer to the screen. Palette effects change the lookup, not the art.
package render

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/render/light"
)

// Index images hold a palette index in the red channel. Alpha is 0 where nothing is drawn and
// 255 elsewhere, so drawing them with the default blend copies indexes exactly. In the world
// buffer, green is 255 where a fixed command drew (drawlist.Cmd.Fixed): those indexes are looked
// up in the palette's own colors, not its animated ones. Blue is how deep a shadow lies there
// (docs/spec/world.md, Light and shadow): a shadow raises it to its own, and anything drawn over
// it sets it back to 0.

// Sprites holds the pack's sprite banks as index images, one per frame, and each bank's density.
type Sprites struct {
	frames  map[string][]*ebiten.Image
	density map[string]int
}

// NewSprites uploads every sprite bank of p.
func NewSprites(p *content.Pack) *Sprites {
	s := &Sprites{frames: map[string][]*ebiten.Image{}, density: map[string]int{}}
	for id, b := range p.Sprites {
		s.frames[id] = uploadBank(b)
		s.density[id] = b.Scale()
	}
	return s
}

// Density returns sprite bank id's density, 1 for an unknown one.
func (s *Sprites) Density(id string) int { return max(1, s.density[id]) }

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
	sprites *Sprites
	cmds    []drawlist.Cmd
	// images holds each command's frame, or blank for a command that blanks a tile box, and
	// densities each image's density.
	images    []*ebiten.Image
	densities []int
	blank     *ebiten.Image
	bounds    image.Rectangle
	// shadows are the shadow ellipses, by width.
	shadows map[int]*ebiten.Image
}

// NewLevelView returns a view that draws nothing until Set.
func NewLevelView(s *Sprites) *LevelView {
	v := &LevelView{sprites: s, blank: ebiten.NewImage(drawlist.TileSize, drawlist.TileSize), shadows: map[int]*ebiten.Image{}}
	v.blank.Fill(color.White)
	return v
}

// Set replaces what the view draws with cmds, built by drawlist.Build, for a level covering
// bounds.
func (v *LevelView) Set(cmds []drawlist.Cmd, bounds image.Rectangle) error {
	images := make([]*ebiten.Image, len(cmds))
	densities := make([]int, len(cmds))
	for i, c := range cmds {
		img := v.blank
		densities[i] = 1
		if c.Shadow > 0 {
			images[i] = v.shadow(c.Shadow)
			continue
		}
		if c.Sprite != "" {
			var err error
			if img, err = v.sprites.Frame(c.Sprite, c.Frame); err != nil {
				return err
			}
			densities[i] = v.sprites.Density(c.Sprite)
		}
		images[i] = part(img, c.Part, densities[i])
	}
	v.cmds, v.images, v.densities, v.bounds = cmds, images, densities, bounds
	return nil
}

// shadowBlend raises the blue channel to the shadow's, leaving the rest as it is.
var shadowBlend = ebiten.Blend{
	BlendFactorSourceRGB: ebiten.BlendFactorOne, BlendFactorDestinationRGB: ebiten.BlendFactorOne,
	BlendOperationRGB:      ebiten.BlendOperationMax,
	BlendFactorSourceAlpha: ebiten.BlendFactorZero, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationAlpha: ebiten.BlendOperationAdd,
}

// shadow returns the shadow ellipse w pixels wide and half as high: its depth in blue, 0
// outside it.
func (v *LevelView) shadow(w int) *ebiten.Image {
	if img, ok := v.shadows[w]; ok {
		return img
	}
	h := max(1, w/2)
	pix := make([]byte, 4*w*h)
	for y := range h {
		for x := range w {
			dx := (float64(x) + 0.5 - float64(w)/2) / (float64(w) / 2)
			dy := (float64(y) + 0.5 - float64(h)/2) / (float64(h) / 2)
			d2 := dx*dx + dy*dy
			if d2 >= 1 {
				continue
			}
			i := 4 * (y*w + x)
			pix[i+2] = byte(math.Round(255 * light.ShadowDepth * (1 - d2)))
			pix[i+3] = 0xFF
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pix)
	v.shadows[w] = img
	return img
}

// part returns the part of img, of density d, a command draws.
func part(img *ebiten.Image, p drawlist.Part, d int) *ebiten.Image {
	b := img.Bounds()
	switch p {
	case drawlist.UpperHalf:
		b.Max.Y = b.Min.Y + drawlist.HalfHeight*d
	case drawlist.LowerHalf:
		b.Min.Y += drawlist.HalfHeight * d
	default:
		return img
	}
	return img.SubImage(b).(*ebiten.Image)
}

// Cmds returns the commands the view draws, back to front.
func (v *LevelView) Cmds() []drawlist.Cmd { return v.cmds }

// Bounds returns the view-pixel rectangle the level covers.
func (v *LevelView) Bounds() image.Rectangle { return v.bounds }

// Draw draws the level into the index image dst, with view pixel origin at dst's top-left, at
// density d: d of dst's pixels to a view pixel each way. Each image is scaled from its own
// density to d, nearest pixel.
func (v *LevelView) Draw(dst *ebiten.Image, origin image.Point, d int) {
	d = max(1, d)
	view := image.Rectangle{origin, origin.Add(dst.Bounds().Size().Div(d))}
	var op ebiten.DrawImageOptions
	var fixedOp colorm.DrawImageOptions
	var fixed colorm.ColorM
	fixed.Translate(0, 1, 0, 0)
	for i, c := range v.cmds {
		img := v.images[i]
		k := float64(d) / float64(v.densities[i])
		r := image.Rectangle{Max: img.Bounds().Size().Div(v.densities[i])}.Add(image.Pt(c.X, c.Y))
		if c.Shadow > 0 {
			r = r.Sub(r.Size().Div(2))
		}
		if !r.Overlaps(view) {
			continue
		}
		if c.Shadow > 0 {
			op.GeoM.Reset()
			op.GeoM.Scale(k, k)
			op.GeoM.Translate(float64((r.Min.X-origin.X)*d), float64((r.Min.Y-origin.Y)*d))
			op.Blend = shadowBlend
			dst.DrawImage(img, &op)
			continue
		}
		if c.Fixed && c.Sprite != "" {
			fixedOp.GeoM.Reset()
			fixedOp.GeoM.Scale(k, k)
			fixedOp.GeoM.Translate(float64((c.X-origin.X)*d), float64((c.Y-origin.Y)*d))
			colorm.DrawImage(dst, img, fixed, &fixedOp)
			continue
		}
		op.GeoM.Reset()
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(float64((c.X-origin.X)*d), float64((c.Y-origin.Y)*d))
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
var Fixed [256]vec4
var Clear float

// Ambient is the light everywhere; 1 with no Lights draws the colors as they are. Each light is
// its center and radius in source pixels (x, y, radius, 0) and its color times its strength.
var Ambient float
var MaxFactor float
var Lights [32]vec4
var LightColors [32]vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos)
	if c.a == 0 {
		if Clear > 0.5 {
			return vec4(0)
		}
		return vec4(0, 0, 0, 1)
	}
	var out vec4
	if c.g > 0.5 {
		out = Fixed[int(c.r*255+0.5)]
	} else {
		out = Palette[int(c.r*255+0.5)]
	}
	p := srcPos - imageSrc0Origin()
	lit := vec3(Ambient)
	for i := 0; i < 32; i++ {
		l := Lights[i]
		if l.z > 0 {
			d := (p - l.xy) * vec2(1, 2) / l.z
			d2 := dot(d, d)
			if d2 < 1 {
				f := (1 - d2) * (1 - d2)
				lit += LightColors[i].rgb * f
			}
		}
	}
	lit = min(lit, vec3(MaxFactor)) * (1 - c.b)
	return vec4(min(out.rgb*lit, vec3(1)), 1)
}
`

// MaxLights is the most lights the view is lit by.
const MaxLights = 32

// Lighting is how the world buffer is lit: lights in buffer pixels, each with its strength now.
type Lighting struct {
	Lights    []light.Light
	Strengths []float64
}

// Presenter shows an index image on the screen through a palette.
type Presenter struct {
	shader         *ebiten.Shader
	palette, fixed []float32
	lights, colors []float32
}

// NewPresenter compiles the palette shader.
func NewPresenter() (*Presenter, error) {
	sh, err := ebiten.NewShader([]byte(paletteShaderSrc))
	if err != nil {
		return nil, fmt.Errorf("palette shader: %w", err)
	}
	return &Presenter{shader: sh, palette: make([]float32, 256*4), fixed: make([]float32, 256*4),
		lights: make([]float32, MaxLights*4), colors: make([]float32, MaxLights*4)}, nil
}

// SetColors sets the colors indexes are looked up in: colors, such as a palette's colors at a
// moment of its effects (content.Palette.At), and fixed for what fixed commands drew, such as the
// palette's own colors.
func (pr *Presenter) SetColors(colors, fixed [256]color.RGBA) {
	setUniform(pr.palette, colors)
	setUniform(pr.fixed, fixed)
}

func clear32(u []float32) {
	for i := range u {
		u[i] = 0
	}
}

func setUniform(u []float32, colors [256]color.RGBA) {
	for i, c := range colors {
		u[4*i] = float32(c.R) / 255
		u[4*i+1] = float32(c.G) / 255
		u[4*i+2] = float32(c.B) / 255
		u[4*i+3] = 1
	}
}

// Present draws src onto dst through the palette, scaled by a whole number. What src leaves
// undrawn shows black.
func (pr *Presenter) Present(dst, src *ebiten.Image, scale int) {
	pr.present(dst, src, float64(scale), float64(scale), image.Point{}, false, nil)
}

// PresentLit draws src, the world buffer, as Present does, lit by l (docs/spec/world.md, Light
// and shadow): the first MaxLights of its lights, and its shadows.
func (pr *Presenter) PresentLit(dst, src *ebiten.Image, scale int, l *Lighting) {
	pr.present(dst, src, float64(scale), float64(scale), image.Point{}, false, l)
}

// PresentWorld draws src, the world buffer drawn at density d, as Present does at scale screen
// pixels a view pixel, lit by l when it is not nil.
func (pr *Presenter) PresentWorld(dst, src *ebiten.Image, scale, d int, l *Lighting) {
	k := float64(scale) / float64(max(1, d))
	pr.present(dst, src, k, k, image.Point{}, false, l)
}

// PresentAt draws src onto dst through the palette at point at, scaled by a whole number. What
// src leaves undrawn stays as dst has it.
func (pr *Presenter) PresentAt(dst, src *ebiten.Image, scale int, at image.Point) {
	pr.present(dst, src, float64(scale), float64(scale), at, true, nil)
}

// PresentScaled draws src onto dst through the palette at point at, each pixel sx × sy screen
// pixels. What src leaves undrawn stays as dst has it.
func (pr *Presenter) PresentScaled(dst, src *ebiten.Image, sx, sy float64, at image.Point) {
	pr.present(dst, src, sx, sy, at, true, nil)
}

func (pr *Presenter) present(dst, src *ebiten.Image, sx, sy float64, at image.Point, clear bool, l *Lighting) {
	var op ebiten.DrawRectShaderOptions
	op.Images[0] = src
	c := float32(0)
	if clear {
		c = 1
	}
	clear32(pr.lights)
	ambient, most := float32(1), float32(1)
	if l != nil {
		ambient, most = light.Ambient, light.MaxFactor
		for i, li := range l.Lights[:min(len(l.Lights), MaxLights)] {
			pr.lights[4*i], pr.lights[4*i+1], pr.lights[4*i+2] = float32(li.X), float32(li.Y), float32(li.Radius)
			for j := range 3 {
				pr.colors[4*i+j] = float32(li.Color[j] * l.Strengths[i])
			}
		}
	}
	op.Uniforms = map[string]any{"Palette": pr.palette, "Fixed": pr.fixed, "Clear": c,
		"Ambient": ambient, "MaxFactor": most, "Lights": pr.lights, "LightColors": pr.colors}
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	b := src.Bounds()
	dst.DrawRectShader(b.Dx(), b.Dy(), pr.shader, &op)
}

// DrawImage draws the frame of image img into the index image dst at (x, y), flipped left to
// right when mirrored.
func (s *Sprites) DrawImage(dst *ebiten.Image, img content.Image, x, y int, mirrored bool) error {
	f, err := s.Frame(img.Sprite, img.Frame)
	if err != nil {
		return err
	}
	var op ebiten.DrawImageOptions
	if mirrored {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(f.Bounds().Dx()), 0)
	}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(f, &op)
	return nil
}
