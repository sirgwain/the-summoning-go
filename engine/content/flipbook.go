package content

import "fmt"

// Flipbooks: animations rendered ahead of time, played as a list of steps (docs/pack.md,
// Flipbooks; docs/spec/scenes.md, View effects and Sparkles).

const (
	flipbooksDir = "flipbooks"
	// FlipbookPrefix starts every flipbook's id.
	FlipbookPrefix = "flipbook."
	effectsFile    = "effects.json"
)

// EffectCues maps the engine's effect cues to flipbooks: the view effects and sparkles the
// simulation plays for what happens in the world, rather than for a level's or a spell's data.
type EffectCues struct {
	ID   string            `json:"id"`
	Cues map[string]string `json:"cues"`
}

// Engine effect cues.
const (
	// EffectClimbDown is the player stepping down into a pit or open trapdoor carefully, and
	// EffectFall falling into one any other way.
	EffectClimbDown = "climb_down"
	EffectFall      = "fall"
	// EffectCasterDies is a stationary caster dying, and EffectDeathBlow a death blow landing.
	EffectCasterDies = "caster_dies"
	EffectDeathBlow  = "death_blow"
	// SparkleCure and SparkleHeal are the sparkles of the curing and healing powers that show one.
	SparkleCure = "cure"
	SparkleHeal = "heal"
)

var effectCueKeys = map[string]bool{
	EffectClimbDown: true, EffectFall: true, EffectCasterDies: true, EffectDeathBlow: true,
	SparkleCure: true, SparkleHeal: true,
}

// Effect returns the flipbook an effect cue plays, or "" when the pack plays none.
func (p *Pack) Effect(cue string) string {
	if p.Effects == nil {
		return ""
	}
	return p.Effects.Cues[cue]
}

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

func (v *verifier) effectCues() {
	c := v.p.Effects
	if c == nil {
		return
	}
	for _, key := range sortedKeys(c.Cues) {
		if !effectCueKeys[key] {
			v.fail("%s: unknown effect cue %q", effectsFile, key)
		}
		v.flipbookRef(effectsFile+" "+key, c.Cues[key])
	}
}

// flipbookRef fails unless id is empty or a flipbook of the pack.
func (v *verifier) flipbookRef(owner, id string) {
	if id != "" && v.p.Flipbooks[id] == nil {
		v.fail("%s: flipbook %q not found", owner, id)
	}
}
