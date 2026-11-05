package window

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/frame"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The pack's skin (docs/pack.md, Skin): its frames, colors, letters and splash, over the
// engine's own.

// ui returns label key in words (inspect.UI).
func (s *Window) ui(key string, args ...any) string { return inspect.UI(s.v.Pack, key, args...) }

// textColor is the color of plain text.
var textColor = color.RGBA{255, 255, 255, 255}

// skinColors are the colors a skin can name, by name.
var skinColors = map[string]*color.RGBA{
	content.ColorText: &textColor, content.ColorTextLow: &lowToneColor, content.ColorTextHigh: &highToneColor,
	content.ColorPanel: &panelColor, content.ColorSlot: &slotColor, content.ColorSlotEdge: &slotEdgeColor,
	content.ColorHandEdge: &handEdgeColor, content.ColorOpenEdge: &openEdgeColor, content.ColorAction: &actionColor,
	content.ColorWait: &waitColor, content.ColorSelection: &selectionColor, content.ColorHover: &hoverColor,
	content.ColorButton: &plaqueColor, content.ColorButtonHover: &plaqueHoverColor,
	content.ColorButtonChosen: &plaqueChosenColor, content.ColorButtonLight: &plaqueLightColor,
	content.ColorButtonDark: &plaqueDarkColor, content.ColorGem: &sapphireLightColor,
	content.ColorGemDark: &sapphireDarkColor, content.ColorHitPoints: &gaugeColors[0],
	content.ColorMagicPoints: &gaugeColors[1], content.ColorLoad: &gaugeColors[2],
	content.ColorOverload: &overloadColor, content.ColorArmor: &armorColor,
	content.ColorSceneWords: &sceneWordsColor, content.ColorFront: &frontColor,
}

// engineColors are the engine's own colors, kept to put back before a skin applies.
var engineColors = func() map[string]color.RGBA {
	m := map[string]color.RGBA{}
	for name, c := range skinColors {
		m[name] = *c
	}
	return m
}()

// skinned is a window's skin worked out: each panel's frame and the pack's frame styles drawn.
type skinned struct {
	frames map[string]string
	styles map[string]*nineStyle
	splash *ebiten.Image
}

// nineStyle is a pack frame style: its image and how it is cut.
type nineStyle struct {
	img    *ebiten.Image
	corner int
	tile   bool
	inside *color.RGBA
}

// applySkin sets the window's colors, frames, letters and splash from the pack's skin, the
// engine's own where it says nothing.
func (s *Window) applySkin() {
	for name, c := range engineColors {
		*skinColors[name] = c
	}
	s.skin = skinned{frames: map[string]string{}, styles: map[string]*nineStyle{}}
	p := s.v.Pack
	sk := p.Skin
	if sk == nil {
		return
	}
	for name, hex := range sk.Colors {
		if c, err := content.ParseColor(hex); err == nil && skinColors[name] != nil {
			*skinColors[name] = c
		}
	}
	for panel, name := range sk.Frames {
		s.skin.frames[panel] = name
	}
	for name, st := range sk.Styles {
		img, ok := p.FrameRGBA(st.Image)
		if !ok {
			continue
		}
		ns := &nineStyle{img: ebiten.NewImageFromImage(img), corner: st.Corner, tile: st.Tile}
		if c, err := content.ParseColor(st.Inside); err == nil {
			ns.inside = &c
		}
		s.skin.styles[name] = ns
	}
	if f := sk.Font; f != nil {
		for face, id := range map[*font.Face]string{font.Small: f.Small, font.Large: f.Large} {
			if atlas := fontAtlas(p, face, id); atlas != nil {
				s.fonts[face] = ebiten.NewImageFromImage(atlas)
			}
		}
	}
	if sk.Splash != nil {
		if img, ok := p.FrameRGBA(*sk.Splash); ok {
			s.skin.splash = ebiten.NewImageFromImage(img)
		}
	}
}

// fontAtlas lays a pack font's frames out as face's atlas, a cell a letter from ' ' to '~'; a
// letter it lacks keeps the engine's. It returns nil without such a sprite.
func fontAtlas(p *content.Pack, face *font.Face, id string) *image.RGBA {
	b := p.Sprites[id]
	if id == "" || b == nil {
		return nil
	}
	own := face.Atlas()
	atlas := image.NewRGBA(own.Rect)
	for r := ' '; r <= '~'; r++ {
		cell := face.Cell(r)
		src := image.Image(own)
		at := cell.Min
		if n := int(r - ' '); n < b.Frames {
			if img, ok := p.FrameRGBA(content.Image{Sprite: id, Frame: n}); ok {
				src, at = img, image.Point{}
			}
		}
		for y := range cell.Dy() {
			for x := range cell.Dx() {
				atlas.Set(cell.Min.X+x, cell.Min.Y+y, src.At(at.X+x, at.Y+y))
			}
		}
	}
	return atlas
}

// panelFrame draws panel's frame filling outer: the skin's style for it, else def.
func (s *Window) panelFrame(screen *ebiten.Image, panel string, def frame.Style, outer image.Rectangle, scale int, opacity float32) {
	name := s.skin.frames[panel]
	if st := s.skin.styles[name]; st != nil {
		s.drawNine(screen, st, outer, scale, opacity)
		return
	}
	style := def
	if b, ok := frame.Names[name]; ok {
		style = b
	}
	s.drawFrame(screen, style, outer, scale, opacity)
}

// drawNine draws pack frame style st filling outer at scale: its corners kept, its edges
// stretched or tiled, its middle stretched, or filled with its inside color at opacity.
func (s *Window) drawNine(screen *ebiten.Image, st *nineStyle, outer image.Rectangle, scale int, opacity float32) {
	b := st.img.Bounds()
	c := st.corner
	src := [3]int{0, c, b.Dx() - c}
	srcEnd := [3]int{c, b.Dx() - c, b.Dx()}
	srcY := [3]int{0, c, b.Dy() - c}
	srcYEnd := [3]int{c, b.Dy() - c, b.Dy()}
	k := c * scale
	dst := [3]int{outer.Min.X, outer.Min.X + k, outer.Max.X - k}
	dstEnd := [3]int{outer.Min.X + k, outer.Max.X - k, outer.Max.X}
	dstY := [3]int{outer.Min.Y, outer.Min.Y + k, outer.Max.Y - k}
	dstYEnd := [3]int{outer.Min.Y + k, outer.Max.Y - k, outer.Max.Y}
	for j := range 3 {
		for i := range 3 {
			from := image.Rect(src[i], srcY[j], srcEnd[i], srcYEnd[j])
			to := image.Rect(dst[i], dstY[j], dstEnd[i], dstYEnd[j])
			if from.Empty() || to.Empty() {
				continue
			}
			if i == 1 && j == 1 && st.inside != nil {
				in := *st.inside
				k := opacity * float32(in.A) / 255
				fill := color.RGBA{uint8(float32(in.R) * k), uint8(float32(in.G) * k), uint8(float32(in.B) * k), uint8(255 * k)}
				vector.FillRect(screen, float32(to.Min.X), float32(to.Min.Y), float32(to.Dx()), float32(to.Dy()), fill, false)
				continue
			}
			piece := st.img.SubImage(from).(*ebiten.Image)
			if st.tile && (i == 1) != (j == 1) {
				tileInto(screen, piece, to, scale)
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(to.Dx())/float64(from.Dx()), float64(to.Dy())/float64(from.Dy()))
			op.GeoM.Translate(float64(to.Min.X), float64(to.Min.Y))
			if i == 1 && j == 1 {
				op.ColorScale.ScaleAlpha(opacity)
			}
			screen.DrawImage(piece, op)
		}
	}
}

// tileInto repeats piece at scale across to, cut off at its end.
func tileInto(screen, piece *ebiten.Image, to image.Rectangle, scale int) {
	pw, ph := piece.Bounds().Dx()*scale, piece.Bounds().Dy()*scale
	sub := screen.SubImage(to).(*ebiten.Image)
	for y := to.Min.Y; y < to.Max.Y; y += ph {
		for x := to.Min.X; x < to.Max.X; x += pw {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(scale), float64(scale))
			op.GeoM.Translate(float64(x), float64(y))
			sub.DrawImage(piece, op)
		}
	}
}
