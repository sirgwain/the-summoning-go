package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
)

// Palette is a 256-color palette, with the effects that animate it.
type Palette struct {
	ID      string
	Colors  [256]color.RGBA
	Effects []PaletteEffect
}

// Palette effect kinds.
const (
	// EffectRotate rotates groups of colors: Groups runs of Size colors from First. Each period,
	// every color in a run takes the value of the next, and the last takes the first's.
	EffectRotate = "rotate"
	// EffectCycle sets color Index to each of Colors in turn, one per period.
	EffectCycle = "cycle"
)

// PaletteEffect changes some of a palette's colors every PeriodMS milliseconds.
type PaletteEffect struct {
	Kind     string
	PeriodMS int
	First    int
	Size     int
	Groups   int
	Index    int
	Colors   []color.RGBA
}

// At returns the palette's colors after elapsedMS milliseconds of its effects.
func (p *Palette) At(elapsedMS int64) [256]color.RGBA {
	out := p.Colors
	for _, e := range p.Effects {
		if e.PeriodMS <= 0 {
			continue
		}
		step := int(elapsedMS / int64(e.PeriodMS))
		switch e.Kind {
		case EffectRotate:
			k := step % e.Size
			for g := range e.Groups {
				base := e.First + g*e.Size
				for i := range e.Size {
					out[base+i] = p.Colors[base+(i+k)%e.Size]
				}
			}
		case EffectCycle:
			out[e.Index] = e.Colors[step%len(e.Colors)]
		}
	}
	return out
}

// verify checks that the palette's effects stay inside the palette.
func (p *Palette) verify() []error {
	var errs []error
	for i, e := range p.Effects {
		fail := func(format string, args ...any) {
			errs = append(errs, fmt.Errorf("palette %s effect %d: "+format, append([]any{p.ID, i}, args...)...))
		}
		if e.PeriodMS <= 0 {
			fail("period %d ms", e.PeriodMS)
		}
		switch e.Kind {
		case EffectRotate:
			if e.Size <= 0 || e.Groups <= 0 || e.First < 0 || e.First+e.Size*e.Groups > len(p.Colors) {
				fail("rotates %d groups of %d from %d, outside the palette", e.Groups, e.Size, e.First)
			}
		case EffectCycle:
			if e.Index < 0 || e.Index >= len(p.Colors) || len(e.Colors) == 0 {
				fail("cycles color %d through %d colors", e.Index, len(e.Colors))
			}
		default:
			fail("unknown kind %q", e.Kind)
		}
	}
	return errs
}

// hexColor is a color written "#rrggbb".
type hexColor color.RGBA

func (c hexColor) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
}

func (c *hexColor) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	var r, g, b uint8
	if n, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); n != 3 || err != nil || len(s) != 7 {
		return fmt.Errorf("bad color %q", s)
	}
	*c = hexColor{r, g, b, 0xFF}
	return nil
}

func hexColors(cs []color.RGBA) []hexColor {
	out := make([]hexColor, len(cs))
	for i, c := range cs {
		out[i] = hexColor(c)
	}
	return out
}

func rgbaColors(cs []hexColor) []color.RGBA {
	out := make([]color.RGBA, len(cs))
	for i, c := range cs {
		out[i] = color.RGBA(c)
	}
	return out
}

type paletteJSON struct {
	ID      string       `json:"id"`
	Colors  []hexColor   `json:"colors"`
	Effects []effectJSON `json:"effects,omitempty"`
}

type effectJSON struct {
	Kind     string     `json:"kind"`
	PeriodMS int        `json:"period_ms"`
	First    *int       `json:"first,omitempty"`
	Size     int        `json:"size,omitempty"`
	Groups   int        `json:"groups,omitempty"`
	Index    *int       `json:"index,omitempty"`
	Colors   []hexColor `json:"colors,omitempty"`
}

func (p *Palette) MarshalJSON() ([]byte, error) {
	out := paletteJSON{ID: p.ID, Colors: hexColors(p.Colors[:])}
	for _, e := range p.Effects {
		ej := effectJSON{Kind: e.Kind, PeriodMS: e.PeriodMS}
		switch e.Kind {
		case EffectRotate:
			first := e.First
			ej.First, ej.Size, ej.Groups = &first, e.Size, e.Groups
		case EffectCycle:
			index := e.Index
			ej.Index, ej.Colors = &index, hexColors(e.Colors)
		}
		out.Effects = append(out.Effects, ej)
	}
	return json.Marshal(out)
}

func (p *Palette) UnmarshalJSON(data []byte) error {
	var in paletteJSON
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return err
	}
	if len(in.Colors) != len(p.Colors) {
		return fmt.Errorf("palette %s: %d colors, want %d", in.ID, len(in.Colors), len(p.Colors))
	}
	p.ID = in.ID
	copy(p.Colors[:], rgbaColors(in.Colors))
	p.Effects = nil
	for _, ej := range in.Effects {
		e := PaletteEffect{Kind: ej.Kind, PeriodMS: ej.PeriodMS, Size: ej.Size, Groups: ej.Groups, Colors: rgbaColors(ej.Colors)}
		if ej.First != nil {
			e.First = *ej.First
		}
		if ej.Index != nil {
			e.Index = *ej.Index
		}
		if len(e.Colors) == 0 {
			e.Colors = nil
		}
		p.Effects = append(p.Effects, e)
	}
	return nil
}
