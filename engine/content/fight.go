package content

// What special weapons, breaking objects and animated creatures carry (docs/pack.md; docs/spec/
// combat.md, Special weapons; docs/spec/creatures.md, Animated objects).

// Grows is a counted weapon: the damage it deals goes into the player's weapon count.
type Grows struct {
	// Into replaces it at At; First and Grown are thing templates whose conversations the first
	// hit and the growing start; Counter is set to 1 when it grows.
	Into    string `json:"into"`
	At      int    `json:"at"`
	First   string `json:"first,omitempty"`
	Grown   string `json:"grown,omitempty"`
	Counter string `json:"counter,omitempty"`
}

// Levels is a grown weapon: every Every points of its count, while Counter is below Max, the
// counter rises, the count resets and Talk's conversation starts.
type Levels struct {
	Every   int    `json:"every"`
	Max     int    `json:"max"`
	Counter string `json:"counter"`
	Talk    string `json:"talk,omitempty"`
}

// Flying is an item's look in flight: a frame of Sprite for each direction.
type Flying struct {
	Sprite string `json:"sprite"`
	// FiredOnly shows it only when fired; thrown, the item shows its floor image.
	FiredOnly bool `json:"fired_only,omitempty"`
}

// Animate kinds.
const (
	AnimateFlicker = "flicker"
	AnimateToggle  = "toggle"
	AnimateRandom  = "random"
	AnimateCycle   = "cycle"
)

// Animate changes an animated creature's frame on each of its updates.
type Animate struct {
	Kind   string `json:"kind"`
	Frames []int  `json:"frames"`
	// Random flickers between random frames of the flicker frames.
	Random bool `json:"random,omitempty"`
}

// breakOps are the actions an item's on_break may use.
var breakOps = opSet("set_counter", "converse", "remove_all", "experience")

func (v *verifier) itemFight(owner string, i *Item) {
	if g := i.Grows; g != nil {
		v.item(owner, g.Into)
		for _, t := range []string{g.First, g.Grown} {
			if t != "" {
				v.thingRef(owner, t)
			}
		}
	}
	if l := i.Levels; l != nil {
		if l.Every <= 0 || l.Counter == "" {
			v.fail("%s: levels want every and a counter", owner)
		}
		if l.Talk != "" {
			v.thingRef(owner, l.Talk)
		}
	}
	if f := i.Flying; f != nil {
		if n := v.frames(owner+" flying", f.Sprite); n >= 0 && n < 8 {
			v.fail("%s: flying look has %d frames, want 8", owner, n)
		}
	}
	for n, a := range i.OnBreak {
		what := owner + " on_break " + itoa(n)
		if !v.actionKind(what, a, breakOps) {
			continue
		}
		switch {
		case a.Converse != nil:
			v.thingRef(what, *a.Converse)
		case a.RemoveAll != nil:
			v.thingRef(what, *a.RemoveAll)
		}
	}
}

func (v *verifier) thingFight(owner string, t *Thing) {
	if t.Name != "" {
		v.text(owner, t.Name)
	}
	if a := t.Animate; a != nil {
		switch a.Kind {
		case AnimateFlicker, AnimateToggle, AnimateRandom, AnimateCycle:
		default:
			v.fail("%s: unknown animate kind %q", owner, a.Kind)
		}
		if len(a.Frames) == 0 {
			v.fail("%s: animate has no frames", owner)
		}
		if n := v.frames(owner, t.Sprite); n >= 0 {
			for _, f := range a.Frames {
				if f < 0 || f >= n {
					v.fail("%s: animate frame %d of %d", owner, f, n)
				}
			}
		}
	}
}
