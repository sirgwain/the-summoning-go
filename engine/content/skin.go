package content

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
)

// Skin is ui.json: how the engine's panels look in this pack (docs/pack.md, Skin). Without it
// the panels keep the engine's own frames, colors and letters.
type Skin struct {
	ID string `json:"id"`
	// Frames gives each framed panel (Panel… names) a frame style: a built-in one by name
	// ("iron", "runic", "hull") or one of Styles.
	Frames map[string]string `json:"frames,omitempty"`
	// Styles are the pack's frame styles, each an image cut in nine.
	Styles map[string]*FrameStyle `json:"styles,omitempty"`
	// Colors override the engine's named colors (Color… names), each "#rrggbb" or "#rrggbbaa".
	Colors map[string]string `json:"colors,omitempty"`
	// Font replaces the engine's letters.
	Font *SkinFont `json:"font,omitempty"`
	// Splash is the artwork of the opening splash, in place of the engine's.
	Splash *Image `json:"splash,omitempty"`
}

// FrameStyle is a frame drawn from an image cut in nine: its corners (Corner square) kept, its
// edges and middle stretched, or tiled with Tile. Width is how thick it stands round a panel, in
// the panel's pixels: the engine's frame width (8), to which the panels' layout is made.
// Inside fills the panel within it.
type FrameStyle struct {
	Image  Image  `json:"image"`
	Corner int    `json:"corner"`
	Tile   bool   `json:"tile,omitempty"`
	Inside string `json:"inside,omitempty"`
}

// SkinFont is a pack's letters: a sprite of a frame each from ' ' to '~', each frame a letter's
// cell (the engine face's cell width by its line height, the letter one column in), lit white on
// clear for drawing tinted. A letter the sprite lacks draws as '?'.
type SkinFont struct {
	Small string `json:"small,omitempty"`
	Large string `json:"large,omitempty"`
}

// Panels a skin frames.
const (
	PanelHotbar    = "hotbar"
	PanelInventory = "inventory"
	PanelSpells    = "spells"
	PanelMenu      = "menu"
	PanelNotice    = "notice"
	PanelEffect    = "effect"
)

var skinPanels = map[string]bool{PanelHotbar: true, PanelInventory: true, PanelSpells: true, PanelMenu: true, PanelNotice: true, PanelEffect: true}

// BuiltinFrames are the engine's frame styles by name.
var BuiltinFrames = map[string]bool{"iron": true, "runic": true, "hull": true}

// Colors a skin names.
const (
	ColorText         = "text"
	ColorTextLow      = "text_low"
	ColorTextHigh     = "text_high"
	ColorPanel        = "panel"
	ColorSlot         = "slot"
	ColorSlotEdge     = "slot_edge"
	ColorHandEdge     = "hand_edge"
	ColorOpenEdge     = "open_edge"
	ColorAction       = "action"
	ColorWait         = "wait"
	ColorSelection    = "selection"
	ColorHover        = "hover"
	ColorButton       = "button"
	ColorButtonHover  = "button_hover"
	ColorButtonChosen = "button_chosen"
	ColorButtonLight  = "button_light"
	ColorButtonDark   = "button_dark"
	ColorGem          = "gem"
	ColorGemDark      = "gem_dark"
	ColorHitPoints    = "gauge.hit_points"
	ColorMagicPoints  = "gauge.magic_points"
	ColorLoad         = "gauge.load"
	ColorOverload     = "gauge.overload"
	ColorArmor        = "gauge.armor"
	ColorSceneWords   = "scene_words"
	ColorFront        = "front"
)

// SkinColors are the colors a skin may name.
var SkinColors = map[string]bool{
	ColorText: true, ColorTextLow: true, ColorTextHigh: true, ColorPanel: true, ColorSlot: true,
	ColorSlotEdge: true, ColorHandEdge: true, ColorOpenEdge: true, ColorAction: true, ColorWait: true,
	ColorSelection: true, ColorHover: true, ColorButton: true, ColorButtonHover: true,
	ColorButtonChosen: true, ColorButtonLight: true, ColorButtonDark: true,
	ColorGem: true, ColorGemDark: true, ColorHitPoints: true, ColorMagicPoints: true, ColorLoad: true,
	ColorOverload: true, ColorArmor: true, ColorSceneWords: true, ColorFront: true,
}

// ParseColor reads "#rrggbb" or "#rrggbbaa".
func ParseColor(s string) (color.RGBA, error) {
	if len(s) != 7 && len(s) != 9 || s[0] != '#' {
		return color.RGBA{}, fmt.Errorf("color %q is not #rrggbb or #rrggbbaa", s)
	}
	n, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("color %q is not #rrggbb or #rrggbbaa", s)
	}
	if len(s) == 7 {
		return color.RGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}, nil
	}
	return color.RGBA{uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n)}, nil
}

// FrameRGBA returns img as true color through the pack's palette, its transparent index clear.
func (p *Pack) FrameRGBA(img Image) (*image.RGBA, bool) {
	b := p.Sprites[img.Sprite]
	pal := p.Palettes[p.Manifest.Palette]
	if b == nil || b.Sheet == nil || pal == nil || img.Frame < 0 || img.Frame >= b.Frames {
		return nil, false
	}
	colors := pal.At(0)
	r := b.FrameRect(img.Frame)
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := range r.Dy() {
		for x := range r.Dx() {
			i := int(b.Sheet.ColorIndexAt(r.Min.X+x, r.Min.Y+y))
			if b.Transparent != nil && i == *b.Transparent {
				continue
			}
			out.SetRGBA(x, y, colors[i])
		}
	}
	return out, true
}

// skin checks ui.json.
func (v *verifier) skin() {
	s := v.p.Skin
	if s == nil {
		return
	}
	image := func(owner string, img Image) {
		b, ok := v.p.Sprites[img.Sprite]
		switch {
		case !ok:
			v.fail("ui.json %s: unknown sprite %q", owner, img.Sprite)
		case img.Frame < 0 || img.Frame >= b.Frames:
			v.fail("ui.json %s: frame %d of %d", owner, img.Frame, b.Frames)
		}
	}
	for _, panel := range sortedKeys(s.Frames) {
		name := s.Frames[panel]
		if !skinPanels[panel] {
			v.fail("ui.json: unknown panel %q", panel)
		}
		if !BuiltinFrames[name] && s.Styles[name] == nil {
			v.fail("ui.json: panel %s has unknown frame style %q", panel, name)
		}
	}
	for _, name := range sortedKeys(s.Styles) {
		st := s.Styles[name]
		image("style "+name, st.Image)
		if b := v.p.Sprites[st.Image.Sprite]; b != nil && (st.Corner <= 0 || 2*st.Corner >= b.FrameSize[0] || 2*st.Corner >= b.FrameSize[1]) {
			v.fail("ui.json style %s: corner %d does not fit its %v image", name, st.Corner, b.FrameSize)
		}
		if st.Inside != "" {
			if _, err := ParseColor(st.Inside); err != nil {
				v.fail("ui.json style %s: %v", name, err)
			}
		}
	}
	for _, name := range sortedKeys(s.Colors) {
		if !SkinColors[name] {
			v.fail("ui.json: unknown color %q", name)
		}
		if _, err := ParseColor(s.Colors[name]); err != nil {
			v.fail("ui.json: %v", err)
		}
	}
	if f := s.Font; f != nil {
		for _, id := range []string{f.Small, f.Large} {
			if id != "" {
				if _, ok := v.p.Sprites[id]; !ok {
					v.fail("ui.json font: unknown sprite %q", id)
				}
			}
		}
	}
	if s.Splash != nil {
		image("splash", *s.Splash)
	}
}
