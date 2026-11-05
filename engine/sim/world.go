// Package sim is the simulation: the state of every level the world has entered and the
// mechanisms that change it. It takes commands, advances on a fixed step and emits events; it
// never knows about the screen and does not import Ebitengine (docs/design.md, Simulation).
package sim

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"

	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/combat/summoning"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// StepsPerSecond is the simulation's fixed rate.
const StepsPerSecond = 60

// ticksPerSecond is the game's clock rate (docs/spec/README.md, Ticks).
const ticksPerSecond = 18.2

// Steps converts a duration in ticks to steps.
func Steps(ticks int) int64 {
	return int64(float64(ticks)*StepsPerSecond/ticksPerSecond + 0.5)
}

// World is the simulation's state.
type World struct {
	pack *content.Pack
	rng  *rand.Rand
	src  *rand.PCG
	// defenses are the creature templates' defenses, worked out once each.
	defenses map[*content.Thing]combat.CreatureDefense
	// step counts updates since the world began.
	step int64
	// levels holds every level entered so far, as it was left.
	levels  map[string]*Level
	current string
	// theme is the music track a creature in sight started, empty for none; music the track a
	// new game starts at its first update.
	theme, music string
	events       []Event
	// randomItems are the items a random object is chosen from.
	randomItems []string
	// player is the player, nil when the pack has none; status the step of the next status tick.
	player *Player
	status int64
	// rules is the combat rule set the pack names.
	rules combat.Rules
	// talk is the open conversation, nil when none is; records are what the world keeps about
	// thing templates; counters the story's counters, by name (absent: 0); shared the shared
	// charges of the items that have them, by item; log the conversation log, and heard what
	// each template has said in it, by template.
	talk     *Talk
	records  map[string]*Record
	counters map[string]int
	shared   map[string]int
	log      []*Logged
	heard    map[string]map[string]bool
	// book is what the player has come across (book.go).
	book Book
	// opening is the thing template whose conversation a new game starts on its first update.
	opening *content.Thing
	// ended is how play ended, nil while it goes on.
	ended *Ended
	// effectPlayed is set when a view effect played this update; after is what waits for it,
	// run at the start of the next (effects.go).
	effectPlayed bool
	after        []func()
	// links are the links the player has travelled, visited the levels it has been on in the
	// order first visited (links.go).
	links   []Link
	visited []string
	// mouths are the mouths the player has heard, talkers the creatures that talk it has seen.
	mouths  []HeardMouth
	talkers []SeenTalker
}

var summoningRules combat.Rules = summoning.Rules{}

// New returns a world for pack p at its start level. The seed makes its random choices
// reproducible.
func New(p *content.Pack, seed uint64) (*World, error) {
	src := rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)
	w := &World{pack: p, src: src, rng: rand.New(src), levels: map[string]*Level{}, rules: rulesFor(p),
		records: map[string]*Record{}, counters: map[string]int{}, shared: map[string]int{}}
	for _, id := range sortedKeys(p.Items) {
		if p.Items[id].Random {
			w.randomItems = append(w.randomItems, id)
		}
	}
	if err := w.enter(p.Manifest.Start.Level); err != nil {
		return nil, err
	}
	if w.player = w.newPlayer(); w.player != nil {
		w.place(p.Manifest.Start.Cell, true)
		w.startInventory()
	}
	w.status = Steps(statusTicks)
	w.events = nil
	return w, nil
}

// Pack returns the content the world runs.
func (w *World) Pack() *content.Pack { return w.pack }

// Step returns the number of updates run.
func (w *World) Step() int64 { return w.step }

// Current returns the current level.
func (w *World) Current() *Level { return w.levels[w.current] }

// Level returns a level the world has entered, or nil.
func (w *World) Level(id string) *Level { return w.levels[id] }

// Hold applies the commands a held world takes, those that only handle the player's objects
// and spells, without time passing: nothing else moves and the step does not advance. It
// refuses any other command. A conversation, and one about to open, are as Update has them,
// which takes only their own commands and passes no time either (docs/spec/items.md, Holding
// the world).
func (w *World) Hold(cmds []input.Command) []Event {
	if w.ended != nil || w.talk != nil || w.opening != nil {
		return w.Update(cmds)
	}
	w.events = nil
	w.runAfter()
	for _, c := range cmds {
		if !heldCommand(c) {
			w.emit(Event{Kind: EventRefused, Level: w.current, Detail: "the world is held"})
			continue
		}
		w.apply(c)
		if w.talk != nil {
			break
		}
	}
	w.see()
	w.look()
	events := w.events
	w.events = nil
	return events
}

// heldCommand reports whether a held world takes cmd.
func heldCommand(cmd input.Command) bool {
	switch cmd.(type) {
	case input.PickUp, input.PutDown, input.Swap, input.Open, input.PutOn, input.UseSlot, input.MoveSlot, input.Read,
		input.Learn, input.Memorize, input.NextSpell:
		return true
	}
	return false
}

// Update applies cmds, advances one step and returns what happened.
func (w *World) Update(cmds []input.Command) []Event {
	w.events = nil
	if w.ended != nil {
		return nil
	}
	w.runAfter()
	if w.music != "" {
		w.startMusic(w.music)
		w.music = ""
	}
	if w.opening != nil && w.talk == nil {
		t := w.opening
		w.opening = nil
		w.startTalk(t, nil)
	}
	if w.talk != nil {
		// A conversation pauses the world: only its own commands are taken.
		for _, c := range cmds {
			if w.talk == nil || !w.talkCommand(c) {
				w.emit(Event{Kind: EventRefused, Level: w.current, Detail: "in a conversation"})
			}
		}
		events := w.events
		w.events = nil
		return events
	}
	if p := w.player; p != nil && !runAsked(cmds) {
		// Letting go of the run ends it at once (docs/spec/player.md, Running).
		p.Running = false
	}
	for _, c := range cmds {
		w.apply(c)
		if w.talk != nil {
			break
		}
	}
	if w.talk != nil {
		events := w.events
		w.events = nil
		return events
	}
	if p := w.player; p != nil {
		// No walk asked for: the keys were let go. A walk held on from the last update is
		// dropped (docs/spec/entities.md, The player's update), and the player's glide finishes
		// at once (Drawn between updates).
		walking := slices.ContainsFunc(cmds, func(c input.Command) bool { _, ok := c.(input.Walk); return ok })
		if !walking {
			if p.held {
				p.want, p.held = -1, false
			}
			// What the player pushes glides with it, so stops with it.
			for _, t := range w.Current().Things {
				if t.glide.set && t.glide.at == p.glide.at && t.glide.end == p.glide.end {
					t.glide.hurry(t.U, t.V, w.step)
				}
			}
			p.glide.hurry(p.U, p.V, w.step)
		}
		// A walk asked for from a stand comes once half the interval has passed (The player's
		// update).
		if p.want >= 0 && !p.walked && p.Pose == "" && w.step < p.next {
			p.next = min(p.next, max(p.last+(w.PlayerInterval()+1)/2, w.step))
		}
		if w.step >= p.next {
			w.updatePlayer()
			p.held = walking
		}
	}
	w.rollBoulders(w.Current())
	w.updateCreatures(w.Current())
	w.updateFlights(w.Current())
	for _, id := range sortedKeys(w.levels) {
		w.advance(w.levels[id])
	}
	if w.player != nil && w.step >= w.status {
		w.status = w.step + Steps(statusTicks)
		w.statusTick()
	}
	w.see()
	w.look()
	w.updateTheme(w.Current())
	w.step++
	events := w.events
	w.events = nil
	return events
}

// enter makes level id current, loading it the first time.
func (w *World) enter(id string) error {
	if _, err := w.load(id); err != nil {
		return err
	}
	if w.current != id {
		changed := w.current != ""
		w.current = id
		w.visit(id)
		w.emit(Event{Kind: EventLevelChanged, Level: id})
		if changed {
			w.stopMusic()
		}
	}
	return nil
}

// LoadLevel loads level id as it starts, if the world has not loaded it, without entering it.
func (w *World) LoadLevel(id string) error {
	_, err := w.load(id)
	return err
}

// load returns level id, placing what it holds the first time it is asked for.
func (w *World) load(id string) (*Level, error) {
	if l, ok := w.levels[id]; ok {
		return l, nil
	}
	def, ok := w.pack.Levels[id]
	if !ok {
		return nil, fmt.Errorf("no level %q", id)
	}
	l := newLevel(w, def)
	w.levels[id] = l
	return l, nil
}

// roll rolls dice.
func (w *World) roll(d content.Dice) int {
	n := d.Bonus
	for range d.Count {
		n += 1 + w.rng.IntN(d.Sides)
	}
	return n
}

// chance reports a 1 in n chance.
func (w *World) chance(in, n int) bool { return w.rng.IntN(n) < in }

func (w *World) emit(e Event) { w.events = append(w.events, e) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Event kinds.
const (
	// EventMessage shows a text: Text is its key, Item the item it names.
	EventMessage = "message"
	// EventSound plays a sound: Detail names an engine cue, or Item gives a sound id.
	EventSound = "sound"
	// EventEffect plays a view effect: Detail is its flipbook, placed by Thing (the player, a
	// conversation's speaker, a dying caster), or by Cell when the thing is gone or there is
	// none. The presentation holds the world while it plays.
	EventEffect = "effect"
	// EventSparkle plays a sparkle: Detail is its flipbook, placed by Thing, or by Feature at Cell.
	EventSparkle = "sparkle"
	// EventFeature reports a feature's state changed.
	EventFeature = "feature"
	// EventProcedure reports a procedure ran.
	EventProcedure = "procedure"
	// EventMap reports a cell's floor or wall changed.
	EventMap = "map"
	// EventObject reports an object landed on a cell (Detail "landed"), was taken (Detail
	// "taken"), was used up or lost (Detail "used", "sank"), or left for another level ("fell"),
	// or broke or ran out in the player's slot Slot ("worn_out").
	EventObject = "object"
	// EventThing reports a thing was placed, moved, hurt or removed: Detail says which.
	EventThing = "thing"
	// EventLevelChanged reports the current level changed.
	EventLevelChanged = "level"
	// EventTransition reports the player moved by a teleport or a change of level that the
	// presentation shows: Detail is the transition cue (content.Transition*), Level the level
	// it arrived on.
	EventTransition = "transition"
	// EventNotYet reports something the simulation cannot do yet: Detail says what.
	EventNotYet = "not_yet"
	// EventRefused reports a command that could not be carried out: Detail says why.
	EventRefused = "refused"
	// EventMagic reports the player's magic changed: a spell learned, memorized, selected, cast
	// (Item the spell, Damage its power) or an object read (Item the object's item).
	EventMagic = "magic"
	// EventTalk reports a conversation started or ended (Item the conversation), or a line said
	// (Text its text key, Detail its kind).
	EventTalk = "talk"
)

// Event is something that happened in an update, for the presentation to show.
type Event struct {
	Kind    string
	Level   string
	Cell    content.Cell
	Feature string
	Thing   string
	Text    string
	Item    string
	Detail  string
	// Score and Damage are a strike's numbers, for its message.
	Score, Damage int
	// Slot is the player's slot an object left, for "worn_out".
	Slot int
}

func (e Event) String() string {
	s := fmt.Sprintf("%s %s %v", e.Kind, e.Level, e.Cell)
	for _, f := range []struct{ name, v string }{{"feature", e.Feature}, {"thing", e.Thing}, {"text", e.Text}, {"item", e.Item}, {"", e.Detail}} {
		if f.v != "" {
			if f.name != "" {
				s += " " + f.name + "=" + f.v
			} else {
				s += " " + f.v
			}
		}
	}
	return s
}
