package viewer

import (
	"image"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Playing a scene (docs/pack.md, Scenes; docs/spec/scenes.md): its steps by the clock, the
// pictures drawn since the last clear, a caption, a fade, a page of words and the speech panels.
// The frontend plays scenes and passes it keys; the shell draws what it describes.

// SceneRun is a scene playing (docs/pack.md, Scenes).
type SceneRun struct {
	Def *content.Scene
	// Viewer is who watches: the name the words' @ stands for, the portrait the character's
	// lines show and the weapon class that picks a by_weapon step.
	Viewer SceneViewer
	// Step is the step showing; Left the updates it has left; PartStep the step of its part.
	Step, Left, PartStep int
	// Pictures are the pictures drawn since the last clear, in order.
	Pictures []ScenePicture
	// Caption and Banner show at the bottom; Fade is the color faded to and Faded how far, 0–1.
	Caption string
	Banner  *content.Image
	Fade    string
	Faded   float64
	// Text is the page of words showing, in Box (from its top when FromTop); Top and Bottom the
	// lines in the panels, Talking whether the panels show (the picture under them darkened).
	Text        string
	Box         image.Rectangle
	FromTop     bool
	Top, Bottom *SceneLine
	Talking     bool
	// Sounds are the sounds started since the presentation last took them (TakeSounds).
	Sounds []string
	// cur is the step showing, a by_weapon step resolved.
	cur content.SceneStep
}

// SceneViewer is who watches a scene.
type SceneViewer struct {
	Name, Portrait string
	WeaponClass    int
}

// SceneLine is a line showing in a panel, with its speaker's portrait.
type SceneLine struct {
	Portrait string
	Text     string
}

// TakeSounds returns the sounds started since it was last called.
func (r *SceneRun) TakeSounds() []string {
	s := r.Sounds
	r.Sounds = nil
	return s
}

// ScenePicture is a picture drawn at a point of the scene.
type ScenePicture struct {
	Image content.Image
	At    image.Point
}

// NewSceneRun starts scene s, watched by w.
func NewSceneRun(s *content.Scene, w SceneViewer) *SceneRun {
	r := &SceneRun{Def: s, Viewer: w, Step: -1}
	r.next()
	return r
}

// Words returns text with @ standing for the viewer's name.
func (r *SceneRun) Words(text string) string { return strings.ReplaceAll(text, "@", r.Viewer.Name) }

func updates(ms int) int { return (ms*sim.StepsPerSecond + 999) / 1000 }

func rect(b [4]int) image.Rectangle { return image.Rect(b[0], b[1], b[0]+b[2], b[1]+b[3]) }

// next goes to the next step and shows it, and on past pictures that do not hold, so a picture
// drawn in parts shows whole.
func (r *SceneRun) next() {
	for r.show() {
	}
}

// show goes to the next step and shows it, and reports whether it is a picture that does not
// hold.
func (r *SceneRun) show() bool {
	r.Step++
	r.PartStep = 0
	if r.Step >= len(r.Def.Steps) {
		return false
	}
	st := r.Def.Steps[r.Step]
	if n := len(st.ByWeapon); n > 0 {
		st = st.ByWeapon[min(max(r.Viewer.WeaponClass, 0), n-1)]
	}
	r.cur = st
	r.Left = updates(st.MS)
	r.Sounds = append(r.Sounds, st.Sounds...)
	if st.Clear {
		r.Pictures, r.Caption, r.Banner, r.Fade, r.Faded = nil, "", nil, "", 0
		r.Top, r.Bottom, r.Talking = nil, nil, false
		if !st.KeepText {
			r.Text = ""
		}
	}
	switch {
	case st.Picture != nil:
		r.Pictures = append(r.Pictures, ScenePicture{*st.Picture, image.Pt(st.At[0], st.At[1])})
	case st.Caption != "":
		r.Caption, r.Banner = st.Caption, st.Banner
	case st.Fade != "":
		r.Fade, r.Faded = st.Fade, 0
	case st.Part != nil:
		r.Left = updates(st.Part.HoldMS(0))
		r.Sounds = append(r.Sounds, st.Part.StepSounds...)
	case st.Text != "":
		r.Text, r.Box, r.FromTop = st.Text, rect(*st.Box), st.FromTop
	case st.Say != nil:
		line := &SceneLine{Portrait: st.Say.Portrait, Text: st.Say.Text}
		if line.Portrait == "" {
			line.Portrait = r.Viewer.Portrait
		}
		r.Text, r.Talking = "", true
		if st.Say.Side == content.SayTop {
			r.Top, r.Bottom = line, nil
		} else {
			r.Top, r.Bottom = nil, line
		}
	}
	return st.Picture != nil && st.MS <= 0
}

// Current returns the step showing, or nil when the scene has ended.
func (r *SceneRun) Current() *content.SceneStep {
	if r.Step < 0 || r.Step >= len(r.Def.Steps) {
		return nil
	}
	return &r.cur
}

// Spread returns the spread of the part showing, and whether one is.
func (r *SceneRun) Spread() (int, bool) {
	st := r.Current()
	if st == nil || st.Part == nil || r.PartStep >= st.Part.Steps() {
		if st != nil && st.Part != nil {
			return st.Part.Spread(st.Part.Steps() - 1), true
		}
		return 0, false
	}
	return st.Part.Spread(r.PartStep), true
}

// Press passes the scene a key or click, Escape when skip, and reports whether the scene has
// ended: Escape skips it; another key or a click ends a page of words or a line early, and
// skips the scene from any other step.
func (r *SceneRun) Press(skip bool) bool {
	if st := r.Current(); !skip && st != nil && st.Waits() {
		r.next()
		return r.Current() == nil
	}
	r.Step = len(r.Def.Steps)
	return true
}

// Update advances the scene one update and reports whether it has ended.
func (r *SceneRun) Update() bool {
	st := r.Current()
	if st == nil {
		return true
	}
	if st.Fade != "" {
		if total := updates(st.MS); total > 0 {
			r.Faded = min(1, float64(total-r.Left+1)/float64(total))
		} else {
			r.Faded = 1
		}
	}
	if r.Left--; r.Left > 0 {
		return false
	}
	if p := st.Part; p != nil && r.PartStep < p.Steps() {
		if r.PartStep++; r.PartStep < p.Steps() {
			r.Left = updates(p.HoldMS(r.PartStep))
			r.Sounds = append(r.Sounds, p.StepSounds...)
		} else {
			r.Left = updates(st.MS)
		}
		if r.Left > 0 {
			return false
		}
	}
	if st.Fade != "" {
		r.Faded = 1
	}
	r.next()
	return r.Current() == nil
}
