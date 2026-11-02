package content

import "fmt"

// Flipbooks: animations rendered ahead of time, played as a list of steps (docs/pack.md,
// Flipbooks; docs/spec/scenes.md, View effects and Sparkles).

const (
	flipbooksDir = "flipbooks"
	// FlipbookPrefix starts every flipbook's id.
	FlipbookPrefix = "flipbook."
)

// Flipbook is a sprite's frames shown in the order and for the times its steps give.
type Flipbook struct {
	ID string `json:"id"`
	// Sprite holds the frames, all of the flipbook's size. Its transparent pixels show what is
	// behind the flipbook.
	Sprite string `json:"sprite"`
	// Framed draws a border around the flipbook.
	Framed bool           `json:"framed,omitempty"`
	Steps  []FlipbookStep `json:"steps"`
}

// FlipbookStep shows one frame for MS milliseconds, with the sounds that start as it shows.
type FlipbookStep struct {
	Frame int `json:"frame"`
	MS    int `json:"ms"`
	// Cancel lets the player cut the flipbook short during this step.
	Cancel bool `json:"cancel,omitempty"`
	// Sounds start as the step shows. Music starts a track; MusicFade fades out the track playing
	// and MusicStop stops it at once, both before Music starts.
	Sounds    []string `json:"sounds,omitempty"`
	Music     string   `json:"music,omitempty"`
	MusicFade bool     `json:"music_fade,omitempty"`
	MusicStop bool     `json:"music_stop,omitempty"`
}

// MS returns how long the flipbook plays.
func (f *Flipbook) MS() int {
	total := 0
	for _, s := range f.Steps {
		total += s.MS
	}
	return total
}

func (v *verifier) flipbook(f *Flipbook) {
	owner := "flipbook " + f.ID
	if len(f.Steps) == 0 {
		v.fail("%s: no steps", owner)
	}
	for i, s := range f.Steps {
		step := fmt.Sprintf("%s step %d", owner, i)
		v.frame(step, f.Sprite, s.Frame)
		if s.MS < 0 {
			v.fail("%s: negative ms %d", step, s.MS)
		}
		for _, id := range s.Sounds {
			v.effect(step, id)
		}
		v.music(step, s.Music)
	}
}
