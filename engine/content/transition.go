package content

import (
	"fmt"
	"image/color"
	"strconv"
)

// Transitions and the lost-item indicator: how the presentation shows a teleport or a change of
// level, and an object breaking or running out in its slot (docs/pack.md, Effects; docs/spec/
// scenes.md, Level transitions). Both are kept in effects.json, beside the effect cues.

// Transition cues: what moved the player.
const (
	// TransitionTeleporter is an armed teleporter, TransitionRipple a teleport to a rune mark.
	TransitionTeleporter = "teleporter"
	TransitionRipple     = "ripple"
	// TransitionExitWall is an exit in a wall, TransitionExitFloor an exit marker on the floor,
	// and TransitionFall a pit or an open trapdoor.
	TransitionExitWall  = "exit_wall"
	TransitionExitFloor = "exit_floor"
	TransitionFall      = "fall"
	// TransitionTravel is a conversation's travel or return, or a portal.
	TransitionTravel = "travel"
	// TransitionLoad is a saved game loaded during play.
	TransitionLoad = "load"
)

var transitionCues = map[string]bool{
	TransitionTeleporter: true, TransitionRipple: true, TransitionExitWall: true,
	TransitionExitFloor: true, TransitionFall: true, TransitionTravel: true, TransitionLoad: true,
}

// Transition styles.
const (
	// StyleCut shows the new view at once.
	StyleCut = "cut"
	// StyleIris closes a circle on the player in the old view, holds the color, and opens one on
	// the player in the new.
	StyleIris = "iris"
	// StyleSlide pushes the old view out as the new one comes in from a side.
	StyleSlide = "slide"
	// StyleSqueeze brings the new view in from a side, squeezed, as the old one squeezes away.
	StyleSqueeze = "squeeze"
	// StyleRipple shifts the view's rows sideways in a zigzag that grows and shrinks, the new
	// view taking over at its height.
	StyleRipple = "ripple"
	// StyleFade fades the old view to the color and the color to the new view.
	StyleFade = "fade"
	// StyleCurtains draws two pleated curtains of the color in from the sides over the old view,
	// holds them closed, and parts them on the new.
	StyleCurtains = "curtains"
)

var transitionStyles = map[string]bool{
	StyleCut: true, StyleIris: true, StyleSlide: true, StyleSqueeze: true, StyleRipple: true, StyleFade: true, StyleCurtains: true,
}

// Sides a sliding or squeezing view comes in from.
const (
	SideLeft   = "left"
	SideRight  = "right"
	SideTop    = "top"
	SideBottom = "bottom"
)

var sides = map[string]bool{SideLeft: true, SideRight: true, SideTop: true, SideBottom: true}

// Transition is how one cue shows the new view.
type Transition struct {
	Style string `json:"style"`
	// MS is how long the whole transition takes. HoldMS is, for an iris or curtains, the time
	// within it that the view is closed.
	MS     int `json:"ms,omitempty"`
	HoldMS int `json:"hold_ms,omitempty"`
	// From is the side the new view comes in from, for slide and squeeze: right or bottom when
	// empty.
	From string `json:"from,omitempty"`
	// Steps, when above 0, moves the transition in that many even steps instead of smoothly.
	Steps int `json:"steps,omitempty"`
	// Color is what an iris closes to, a fade passes through and curtains are, "#rrggbb"; black
	// when empty.
	Color string `json:"color,omitempty"`
	// Size is a ripple's largest sideways shift, in view pixels, and Rows how many rows each
	// zag of it takes. For curtains, Size is the width of a pleat, in view pixels.
	Size int `json:"size,omitempty"`
	Rows int `json:"rows,omitempty"`
}

// RGBA is the transition's color.
func (t *Transition) RGBA() color.RGBA {
	c, _ := parseColor(t.Color)
	return c
}

// Lost-item styles.
const (
	// LostShatter breaks the object's image into pieces that fly apart, fall and fade.
	LostShatter = "shatter"
	// LostFade fades the image out where it was.
	LostFade = "fade"
	// LostNone shows nothing.
	LostNone = "none"
)

var lostStyles = map[string]bool{LostShatter: true, LostFade: true, LostNone: true}

// LostItem is how an object that breaks or runs out in one of the player's slots goes.
type LostItem struct {
	Style string `json:"style"`
	MS    int    `json:"ms,omitempty"`
	// Inventory opens the inventory panel while the object goes, holding the world, and shows it
	// going there; the panel closes again after unless it was open.
	Inventory bool `json:"inventory,omitempty"`
}

// Transition returns how cue shows, or nil when the pack shows the new view at once.
func (p *Pack) Transition(cue string) *Transition {
	if p.Effects == nil {
		return nil
	}
	t := p.Effects.Transitions[cue]
	if t == nil || t.Style == StyleCut || t.MS <= 0 {
		return nil
	}
	return t
}

// Lost returns how an object lost from a slot goes, or nil when it shows nothing.
func (p *Pack) Lost() *LostItem {
	if p.Effects == nil || p.Effects.ItemLost == nil {
		return nil
	}
	l := p.Effects.ItemLost
	if l.Style == LostNone || l.MS <= 0 {
		return nil
	}
	return l
}

func parseColor(s string) (color.RGBA, error) {
	if s == "" {
		return color.RGBA{A: 255}, nil
	}
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{A: 255}, fmt.Errorf("color %q is not #rrggbb", s)
	}
	n, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.RGBA{A: 255}, fmt.Errorf("color %q is not #rrggbb", s)
	}
	return color.RGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}, nil
}

func (v *verifier) transitions() {
	c := v.p.Effects
	if c == nil {
		return
	}
	for _, cue := range sortedKeys(c.Transitions) {
		t := c.Transitions[cue]
		owner := effectsFile + " transition " + cue
		switch {
		case !transitionCues[cue]:
			v.fail("%s: unknown transition cue", owner)
		case t == nil:
			v.fail("%s: empty", owner)
			continue
		case !transitionStyles[t.Style]:
			v.fail("%s: unknown style %q", owner, t.Style)
		}
		if t.MS < 0 || t.HoldMS < 0 || t.Steps < 0 || t.Size < 0 || t.Rows < 0 {
			v.fail("%s: negative number", owner)
		}
		if t.HoldMS > t.MS {
			v.fail("%s: hold_ms %d longer than ms %d", owner, t.HoldMS, t.MS)
		}
		if t.From != "" && !sides[t.From] {
			v.fail("%s: unknown side %q", owner, t.From)
		}
		if _, err := parseColor(t.Color); err != nil {
			v.fail("%s: %v", owner, err)
		}
	}
	if l := c.ItemLost; l != nil {
		if !lostStyles[l.Style] {
			v.fail("%s item_lost: unknown style %q", effectsFile, l.Style)
		}
		if l.MS < 0 {
			v.fail("%s item_lost: negative ms", effectsFile)
		}
	}
}
