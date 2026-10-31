// Package sim is the simulation: the state of every level the world has entered and the
// mechanisms that change it. It takes commands, advances on a fixed step and emits events; it
// never knows about the screen and does not import Ebitengine (docs/design.md, Simulation).
package sim

import (
	"fmt"
	"math/rand/v2"
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
	// step counts updates since the world began.
	step int64
	// levels holds every level entered so far, as it was left.
	levels  map[string]*Level
	current string
	events  []Event
	// randomItems are the items a random object is chosen from.
	randomItems []string
	// player is the player, nil when the pack has none; status the step of the next status tick.
	player *Player
	status int64
	// rules is the combat rule set the pack names.
	rules combat.Rules
}

var summoningRules combat.Rules = summoning.Rules{}

// New returns a world for pack p at its start level. The seed makes its random choices
// reproducible.
func New(p *content.Pack, seed uint64) (*World, error) {
	w := &World{pack: p, rng: rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)), levels: map[string]*Level{}, rules: rulesFor(p)}
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

// Update applies cmds, advances one step and returns what happened.
func (w *World) Update(cmds []input.Command) []Event {
	w.events = nil
	for _, c := range cmds {
		w.apply(c)
	}
	if p := w.player; p != nil && w.step >= p.next {
		w.updatePlayer()
	}
	w.rollBoulders(w.Current())
	w.updateCreatures(w.Current())
	for _, id := range sortedKeys(w.levels) {
		w.advance(w.levels[id])
	}
	if w.player != nil && w.step >= w.status {
		w.status = w.step + Steps(statusTicks)
		w.statusTick()
	}
	w.see()
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
		w.current = id
		w.emit(Event{Kind: EventLevelChanged, Level: id})
	}
	return nil
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
	// EventSound plays a sound: Detail names it.
	EventSound = "sound"
	// EventEffect starts a view effect: Detail names it.
	EventEffect = "effect"
	// EventFeature reports a feature's state changed.
	EventFeature = "feature"
	// EventProcedure reports a procedure ran.
	EventProcedure = "procedure"
	// EventMap reports a cell's floor or wall changed.
	EventMap = "map"
	// EventObject reports an object landed on a cell (Detail "landed"), was taken (Detail
	// "taken"), was used up or lost (Detail "used", "sank"), or left for another level ("fell").
	EventObject = "object"
	// EventThing reports a thing was placed, moved, hurt or removed: Detail says which.
	EventThing = "thing"
	// EventLevelChanged reports the current level changed.
	EventLevelChanged = "level"
	// EventNotYet reports something the simulation cannot do yet: Detail says what.
	EventNotYet = "not_yet"
	// EventRefused reports a command that could not be carried out: Detail says why.
	EventRefused = "refused"
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
