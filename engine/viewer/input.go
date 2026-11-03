// Package viewer is the level viewer's controls: a camera over a level, a selection, and tools
// that send debug commands to the simulation. It reads its input as a snapshot per update and
// does not import Ebitengine, so tests and scripts drive it as a person would (docs/design.md,
// Testing). The window, drawing and reading the real keyboard are the command's.
package viewer

import (
	"fmt"
	"image"
	"strings"
)

// Key names a key, as Ebitengine's key names do ("A", "ArrowLeft", "Digit1", "Shift").
type Key string

// The keys the viewer reads.
const (
	KeyArrowLeft      Key = "ArrowLeft"
	KeyArrowRight     Key = "ArrowRight"
	KeyArrowUp        Key = "ArrowUp"
	KeyArrowDown      Key = "ArrowDown"
	KeyShift          Key = "Shift"
	KeyPageUp         Key = "PageUp"
	KeyPageDown       Key = "PageDown"
	KeyBracketLeft    Key = "BracketLeft"
	KeyBracketRight   Key = "BracketRight"
	KeyEqual          Key = "Equal"
	KeyMinus          Key = "Minus"
	KeyNumpadAdd      Key = "NumpadAdd"
	KeyNumpadSubtract Key = "NumpadSubtract"
	KeyHome           Key = "Home"
	KeySpace          Key = "Space"
	KeyTab            Key = "Tab"
	KeyBackquote      Key = "Backquote"
	KeyEnd            Key = "End"
	KeyEnter          Key = "Enter"
	KeyControl        Key = "Control"
	KeyF5             Key = "F5"
	KeyF8             Key = "F8"
	KeyF9             Key = "F9"
	KeyF10            Key = "F10"
	KeyF12            Key = "F12"
	KeyEscape         Key = "Escape"
	KeyBackspace      Key = "Backspace"
)

// Keys are every key the viewer reads; a shell need only report these.
var Keys = func() []Key {
	keys := []Key{
		KeyArrowLeft, KeyArrowRight, KeyArrowUp, KeyArrowDown, KeyShift, KeyPageUp, KeyPageDown,
		KeyBracketLeft, KeyBracketRight, KeyEqual, KeyMinus, KeyNumpadAdd, KeyNumpadSubtract,
		KeyHome, KeySpace, KeyTab, KeyBackquote, KeyEnd, KeyEnter, KeyControl, KeyF9, KeyF10, KeyF12,
		KeyEscape, KeyBackspace,
	}
	for _, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		keys = append(keys, Key(c))
	}
	for d := '0'; d <= '9'; d++ {
		keys = append(keys, Key("Digit"+string(d)))
	}
	for f := 1; f <= functionKeys; f++ {
		keys = append(keys, Key(fmt.Sprintf("F%d", f)))
	}
	return keys
}()

// ParseKey returns the viewer's key named s, ignoring case.
func ParseKey(s string) (Key, error) {
	for _, k := range Keys {
		if strings.EqualFold(string(k), s) {
			return k, nil
		}
	}
	return "", fmt.Errorf("unknown key %q", s)
}

// Button is a mouse button's state in an update.
type Button struct {
	// Held is whether it is down; Pressed and Released whether it went down or up since the last
	// update. A tap can press and release between two updates.
	Held, Pressed, Released bool
}

// Input is what the person did since the last update.
type Input struct {
	// Cursor is the mouse's screen point.
	Cursor image.Point
	// Held are the keys down; Pressed those that went down since the last update.
	Held, Pressed map[Key]bool
	Left, Right   Button
	// Wheel is the vertical scroll since the last update.
	Wheel float64
	// Chars are the characters typed since the last update.
	Chars []rune
}

func (in Input) held(keys ...Key) bool {
	for _, k := range keys {
		if in.Held[k] {
			return true
		}
	}
	return false
}

// anyPressed reports whether any key went down, other than Shift and Control.
func (in Input) anyPressed() bool {
	for k, down := range in.Pressed {
		if down && k != KeyShift && k != KeyControl {
			return true
		}
	}
	return false
}

// without returns the input with key k neither held nor pressed.
func (in Input) without(k Key) Input {
	held, pressed := make(map[Key]bool, len(in.Held)), make(map[Key]bool, len(in.Pressed))
	for key, down := range in.Held {
		held[key] = down && key != k
	}
	for key, down := range in.Pressed {
		pressed[key] = down && key != k
	}
	in.Held, in.Pressed = held, pressed
	return in
}

func (in Input) pressed(keys ...Key) bool {
	for _, k := range keys {
		if in.Pressed[k] {
			return true
		}
	}
	return false
}
