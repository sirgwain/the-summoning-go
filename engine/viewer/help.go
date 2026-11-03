package viewer

import (
	"fmt"
	"strings"
)

// Binding is a key, or keys, and what it does: a line of the help the panel and the command show.
type Binding struct {
	Keys, Does string
}

// PlayBindings are what the keys and the mouse do while playing (playKeys, castKeys, nap, mouse).
var PlayBindings = []Binding{
	{"WASD/arrows", "walk (two: diagonal)"},
	{"Home PgUp End PgDn", "walk the diagonals"},
	{"Space", "right hand (Shift: 2nd)"},
	{"X", "throw right"},
	{"Q", "left hand (Shift: 2nd)"},
	{"Z", "throw left"},
	{"F", "take"},
	{"G", "put down"},
	{"Enter", "lever/key underfoot"},
	{"E/C", "talk"},
	{"F1-F4/1-4", "cast (Shift: next spell)"},
	{"F10/R", "spells"},
	{"I/Tab", "inventory"},
	{"L", "conversation log"},
	{"M", "level map"},
	{"Esc", "close / options"},
	{"O", "options"},
	{"F5", "quick save"},
	{"F8", "quick load"},
	{"F9", "music, effects, silence"},
	{"H", "recenter"},
	{"P", "pause"},
	{"N (held)", "nap"},
	{"Click the player", "take / put down"},
	{"Click a box", "swap with the hand"},
	{"Right click", "hand to a shelf"},
	{"Shift+click", "read"},
	{"Click the portrait", "inventory"},
}

// ToolBindings are what the keys and the mouse do with the tools (keys, levelKeys, pan, mouse).
var ToolBindings = []Binding{
	{"WASD/arrows", "move the view (Shift: faster)"},
	{"[ ]/PgUp PgDn", "level"},
	{"Home", "recenter"},
	{"M", "magnify"},
	{"Drag", "move the view"},
	{"Click", "inspect a cell or thing"},
	{"Shift+click", "move the selected thing"},
	{"U/Enter", "use (plate: press/lift; slot: fill)"},
	{"T", "toggle"},
	{"R", "reset"},
	{"1-9", "run hook"},
	{"E", "follow exit"},
	{"X", "take object"},
	{"Space", "pause"},
	{"N", "step"},
}

// CommonBindings are what the keys do in play and with the tools alike (Update).
var CommonBindings = []Binding{
	{"Wheel, + -", "zoom"},
	{"V", "see all"},
	{"B/Shift+B", "choose a flipbook"},
	{"K", "play it"},
	{"F12", "play / tools"},
	{"~", "debug overlay"},
}

// helpLines packs bindings into lines of at most n characters, two spaces between bindings.
func helpLines(bindings []Binding, n int) []string {
	var lines []string
	line := ""
	for _, b := range bindings {
		s := b.Keys + " " + b.Does
		switch {
		case line == "":
			line = s
		case len(line)+2+len(s) <= n:
			line += "  " + s
		default:
			lines = append(lines, line)
			line = s
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// HelpText lays bindings out as a table for a command's help, a binding a line.
func HelpText(bindings ...[]Binding) string {
	w := 0
	for _, bs := range bindings {
		for _, b := range bs {
			w = max(w, len(b.Keys))
		}
	}
	var sb strings.Builder
	for _, bs := range bindings {
		for _, b := range bs {
			fmt.Fprintf(&sb, "  %-*s  %s\n", w, b.Keys, b.Does)
		}
	}
	return sb.String()
}
