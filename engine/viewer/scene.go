package viewer

import (
	"image"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Playing a scene (docs/pack.md, Scenes; docs/spec/scenes.md): its steps by the clock, the
// pictures drawn since the last clear and the frame over them, a caption, the palette and its
// fades, a page of words, the speech panels and the music.
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
	// Over are the pictures drawn over them, the last frame's.
	Over []content.ScenePicture
	// Caption and Banner show at the bottom, or the caption in CaptionBox when it has one.
	Caption    string
	CaptionBox *image.Rectangle
	Banner     *content.Image
	// Fade is the color faded to (in from, with FadeIn) and Faded how far, 0–1; FadeOver the
	// color the scene was covered in when the fade began, faded from.
	Fade     string
	Faded    float64
	FadeIn   bool
	FadeOver string
	// Dims are the palette colors faded alone, each by how far.
	Dims []SceneDim
	// Palette is the palette the scene draws in ("" for the pack's), its effects running from
	// update PaletteFrom; Elapsed counts the updates played.
	Palette     string
	PaletteFrom int
	Elapsed     int
	// Text is the page of words showing, in Box (from its top when FromTop); Top and Bottom the
	// lines in the panels, Talking whether the panels show (the picture under them darkened).
	Text        string
	Box         image.Rectangle
	FromTop     bool
	Top, Bottom *SceneLine
	Talking     bool
	// Sounds are the sounds started since the presentation last took them (TakeSounds), and
	// Music the music cues (TakeMusic).
	Sounds []string
	Music  []SceneMusic
	// cur is the step showing, a by_weapon step resolved.
	cur content.SceneStep
}

// SceneViewer is who watches a scene.
type SceneViewer struct {
	Name, Portrait string
	WeaponClass    int
}

// SceneDim is palette colors First–Last faded toward Color, Amount of the way.
type SceneDim struct {
	Color       string
	First, Last int
	Amount      float64
}

// SceneMusic is a music cue: stop the track playing at once, fade it, then start Track.
type SceneMusic struct {
	Stop, Fade bool
	Track      string
}

// TakeMusic returns the music cues since it was last called.
func (r *SceneRun) TakeMusic() []SceneMusic {
	m := r.Music
	r.Music = nil
	return m
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

// ScenePicture is a picture drawn at a point of the scene, mirrored left to right with Mirror.
type ScenePicture struct {
	Image  content.Image
	At     image.Point
	Mirror bool
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

// next goes to the next step and shows it, and on past steps that do not hold, so a picture
// drawn in parts shows whole.
func (r *SceneRun) next() {
	for r.show() {
	}
}

// show goes to the next step and shows it, and reports whether it does not hold.
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
	if st.Music != "" || st.MusicFade || st.MusicStop {
		r.Music = append(r.Music, SceneMusic{Stop: st.MusicStop, Fade: st.MusicFade, Track: st.Music})
	}
	if st.Palette != "" {
		r.Palette, r.PaletteFrom = st.Palette, r.Elapsed
	}
	if st.Clear {
		r.Pictures, r.Over, r.Caption, r.CaptionBox, r.Banner = nil, nil, "", nil, nil
		r.Fade, r.Faded, r.FadeIn, r.FadeOver, r.Dims = "", 0, false, "", nil
		r.Top, r.Bottom, r.Talking = nil, nil, false
		if !st.KeepText {
			r.Text = ""
		}
	}
	switch {
	case st.Picture != nil:
		r.Pictures = append(r.Pictures, ScenePicture{*st.Picture, image.Pt(st.At[0], st.At[1]), st.Mirror})
	case st.Over != nil:
		r.Over = *st.Over
	case st.Caption != "":
		r.Caption, r.Banner, r.CaptionBox = st.Caption, st.Banner, nil
		if st.Box != nil {
			b := rect(*st.Box)
			r.CaptionBox = &b
		}
	case st.NoCaption:
		r.Caption, r.Banner, r.CaptionBox = "", nil, nil
	case st.Fade != "" && st.FadeColors != nil:
		r.Dims = append(r.Dims, SceneDim{Color: st.Fade, First: st.FadeColors[0], Last: st.FadeColors[1]})
		if st.MS <= 0 {
			r.Dims[len(r.Dims)-1].Amount = 1
		}
	case st.Fade != "":
		r.FadeOver = ""
		if r.Fade != "" && r.Faded >= 1 && !r.FadeIn && !st.FadeIn {
			r.FadeOver = r.Fade
		}
		r.Fade, r.Faded, r.FadeIn = st.Fade, 0, st.FadeIn
		if st.MS <= 0 {
			r.Faded = 1
		}
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
	return !st.Holds()
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
// ended: Escape skips it; another key or a click ends a page of words or a line early, skips a
// run of cancel steps, and skips the scene from any other step unless only Escape skips it.
func (r *SceneRun) Press(skip bool) bool {
	st := r.Current()
	switch {
	case st == nil:
		return true
	case !skip && st.Waits():
		r.next()
		return r.Current() == nil
	case !skip && st.Cancel:
		for {
			r.finish(st)
			r.next()
			if st = r.Current(); st == nil || !st.Cancel {
				return st == nil
			}
		}
	case !skip && r.Def.EscapeOnly:
		return false
	}
	r.Step = len(r.Def.Steps)
	return true
}

// finish brings the step showing to its end: its fade done.
func (r *SceneRun) finish(st *content.SceneStep) {
	if st.Fade == "" {
		return
	}
	if st.FadeColors != nil && len(r.Dims) > 0 {
		r.Dims[len(r.Dims)-1].Amount = 1
	} else {
		r.Faded = 1
	}
}

// Update advances the scene one update and reports whether it has ended.
func (r *SceneRun) Update() bool {
	st := r.Current()
	if st == nil {
		return true
	}
	r.Elapsed++
	if st.Fade != "" {
		f := 1.0
		if total := updates(st.MS); total > 0 {
			f = min(1, float64(total-r.Left+1)/float64(total))
		}
		if st.FadeColors != nil && len(r.Dims) > 0 {
			r.Dims[len(r.Dims)-1].Amount = f
		} else {
			r.Faded = f
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
	r.finish(st)
	r.next()
	return r.Current() == nil
}
