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

// Run runs a procedure of the current level.
type Run struct{ Procedure string }

// FollowExit makes the current level the one an exit, pit or trapdoor leads to.
type FollowExit struct{ Feature string }

// GoTo makes a level the current one.
type GoTo struct{ Level string }

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

func (Strike) command() {}
func (PutOn) command()  {}

func (Walk) command()        {}
func (PickUp) command()      {}
func (PutDown) command()     {}
func (Swap) command()        {}
func (Activate) command()    {}
func (PlacePlayer) command() {}

func (Use) command()        {}
func (Drop) command()       {}
func (Remove) command()     {}
func (Place) command()      {}
func (Move) command()       {}
func (Switch) command()     {}
func (Run) command()        {}
func (FollowExit) command() {}
func (GoTo) command()       {}
