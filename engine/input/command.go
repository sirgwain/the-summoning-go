// Package input defines the commands the simulation takes: what the player, or a tool acting for
// one, does. Commands are plain data; key, mouse and gamepad bindings that produce them belong to
// the presentation. The package does not import Ebitengine.
package input

import "github.com/sirgwain/the-summoning-go/engine/content"

// Command is one thing done to the world in an update.
type Command interface{ command() }

// Debug commands act on the current level directly, without a player (docs/design.md, Input).

// Use uses what is at a cell: pulls a lever, hears a mouth.
type Use struct{ Cell content.Cell }

// Drop puts a new object of an item down on a cell, as the player would.
type Drop struct {
	Cell content.Cell
	Item string
}

// Remove takes the top object off a cell's floor.
type Remove struct{ Cell content.Cell }

// Place puts a new thing of a template on a cell, running the cell's effects.
type Place struct {
	Cell  content.Cell
	Thing string
}

// Move moves a placed thing to a cell, running the effects of the cell it leaves and the one it
// enters.
type Move struct {
	Thing string
	Cell  content.Cell
}

// Switch modes.
const (
	SwitchOn     = "on"
	SwitchOff    = "off"
	SwitchToggle = "toggle"
)

// Switch switches a feature on, off or over, as a procedure would.
type Switch struct {
	Feature string
	Mode    string
}

// Weigh adds weight (negative to take it off) to the plate on a cell, as an object of that weight
// put down or taken would, without one.
type Weigh struct {
	Cell   content.Cell
	Weight int
}

// Reset puts a feature of the current level back as the level starts it, running nothing. A
// plate keeps the weight on it, less Unweigh: weight put on it by Weigh.
type Reset struct {
	Feature string
	Unweigh int
}

// Run runs a procedure of the current level.
type Run struct{ Procedure string }

// FollowExit makes the current level the one an exit, pit or trapdoor leads to.
type FollowExit struct{ Feature string }

// GoTo makes a level the current one.
type GoTo struct{ Level string }

// SetCounter sets a counter.
type SetCounter struct {
	Counter string
	Value   int
}

// Player commands: what the player does (docs/spec/entities.md, docs/spec/items.md).

// Walk asks the player to walk a step along a direction at its next update. A presentation
// sends it every update while the walk is held.
type Walk struct{ Direction int }

// PickUp takes the top object of the player's cell to the first empty shelf (an arrow into a
// worn quiver with room), as the take key does; with Hand, onto the pointer when the hand is
// empty, as clicking the player does.
type PickUp struct{ Hand bool }

// PutDown puts the hand's object down on the player's cell.
type PutDown struct{}

// Swap exchanges the hand with an inventory slot.
type Swap struct{ Slot int }

// Open opens the container in a fixed inventory slot as the pack, or closes it when it is.
type Open struct{ Slot int }

// Activate uses the lever in the player's cell, or the nearest mouth in reach.
type Activate struct{}

// PlacePlayer moves the player to a cell of the current level, running the cell's effects. It is
// a debug command.
type PlacePlayer struct{ Cell content.Cell }

// Strike acts with a hand (0 right, 1 left) by its object's action 0 or 1: strikes with a weapon,
// or with bare hands.
type Strike struct{ Hand, Action int }

// Throw throws the object in a hand (0 right, 1 left) along the player's facing.
type Throw struct{ Hand int }

func (Throw) command() {}

// PutOn puts the hand's object on, in the worn slot or hand its item goes in.
type PutOn struct{}

// UseSlot uses the object in a slot (a fixed slot or a pack slot) by its first use or use-up
// action, as the right hand would; in a hand, by that hand.
type UseSlot struct{ Slot int }

// MoveSlot moves the object in a slot, with nothing on the pointer, to a place of kind To: the
// pack (ToPack), the first empty shelf (ToShelf) or its worn slot (ToWorn), what that slot held
// taking its old place.
type MoveSlot struct {
	Slot int
	To   string
}

// The places MoveSlot moves to.
const (
	ToPack  = "pack"
	ToShelf = "shelf"
	ToWorn  = "worn"
)

func (Strike) command()   {}
func (PutOn) command()    {}
func (UseSlot) command()  {}
func (MoveSlot) command() {}

func (Walk) command()        {}
func (PickUp) command()      {}
func (PutDown) command()     {}
func (Swap) command()        {}
func (Open) command()        {}
func (Activate) command()    {}
func (PlacePlayer) command() {}

func (Use) command()        {}
func (Drop) command()       {}
func (Remove) command()     {}
func (Place) command()      {}
func (Move) command()       {}
func (Switch) command()     {}
func (Weigh) command()      {}
func (Reset) command()      {}
func (Run) command()        {}
func (FollowExit) command() {}
func (GoTo) command()       {}
func (SetCounter) command() {}

// Magic and talk commands (docs/spec/spells.md, docs/spec/dialogue.md).

// Learn enters a gesture, its icons as letters A–L: one that matches a spell puts the spell in
// the spellbook and memorizes it.
type Learn struct{ Gesture string }

// Memorize memorizes a spell of the spellbook once more and selects it.
type Memorize struct{ Spell string }

// Cast casts a school's selected spell.
type Cast struct{ School int }

// NextSpell selects a school's next memorized spell.
type NextSpell struct{ School int }

// SlotHand names the hand on the pointer where a command takes a slot.
const SlotHand = -1

// Read looks at the details of the object in a slot (SlotHand for the hand): a scroll teaches its
// icons.
type Read struct{ Slot int }

// Talk talks to a thing of the current level, or hears a mouth when Thing names one; with no
// Thing it talks to the nearest that talks, or hears the nearest mouth.
type Talk struct{ Thing string }

// Continue goes on past the text a conversation shows; Ask, if set, asks that word next.
type Continue struct{ Ask string }

// Ask asks a word in a conversation.
type Ask struct{ Word string }

// Leave leaves a conversation.
type Leave struct{}

// ShowMap shows the level map, or what the level shows instead.
type ShowMap struct{}

// Menu opens the game's menu: the pack's menu conversation.
type Menu struct{}

func (ShowMap) command() {}
func (Menu) command()    {}

func (Learn) command()     {}
func (Memorize) command()  {}
func (Cast) command()      {}
func (NextSpell) command() {}
func (Read) command()      {}
func (Talk) command()      {}
func (Continue) command()  {}
func (Ask) command()       {}
func (Leave) command()     {}
