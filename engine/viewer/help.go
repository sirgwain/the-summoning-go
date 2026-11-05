package viewer

import (
	"fmt"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// Binding is a key, or keys, and what it does: a line of the help the panel and the command show.
// What it does in play is a label key (docs/pack.md, Labels), in the pack's words; the tools'
// are words.
type Binding struct {
	Keys, Does string
}

// does returns what b does in pack p's words.
func (b Binding) does(p *content.Pack) string {
	if strings.HasPrefix(b.Does, "ui.") {
		return inspect.UI(p, b.Does)
	}
	return b.Does
}

// PlayBindings are what the keys and the mouse do while playing (playKeys, castKeys, nap, mouse).
var PlayBindings = []Binding{
	{"WASD/arrows", "ui.help.walk"},
	{"Home PgUp End PgDn", "ui.help.diagonals"},
	{"Shift+walk", "ui.help.run"},
	{"Space", "ui.help.right"},
	{"X", "ui.help.throw_r"},
	{"Q", "ui.help.left"},
	{"Z", "ui.help.throw_l"},
	{"F", "ui.help.take"},
	{"G", "ui.help.put"},
	{"Enter", "ui.help.underfoot"},
	{"E/C", "ui.help.talk"},
	{"F1-F4/1-4", "ui.help.cast"},
	{"F10/R", "ui.help.spells"},
	{"I/Tab", "ui.help.inventory"},
	{"L", "ui.help.log"},
	{"J", "ui.help.book"},
	{"M", "ui.help.map"},
	{"Esc", "ui.help.escape"},
	{"O", "ui.help.options"},
	{"F5", "ui.help.save"},
	{"F8", "ui.help.load"},
	{"F9", "ui.help.sound"},
	{"H", "ui.help.recenter"},
	{"P", "ui.help.pause"},
	{"N (held)", "ui.help.nap"},
	{"Click the player", "ui.help.click_me"},
	{"Click a box", "ui.help.click_box"},
	{"Right click", "ui.help.shelve"},
	{"Shift+click", "ui.help.read"},
	{"Click the portrait", "ui.help.portrait"},
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
	{"J", "the full book"},
	{"B/Shift+B", "choose a flipbook"},
	{"K", "play it"},
}

// CommonBindings are what the keys do in play and with the tools alike (Update).
var CommonBindings = []Binding{
	{"Wheel, + -", "ui.help.zoom"},
	{"V", "ui.help.see_all"},
	{"F12", "ui.help.mode"},
	{"~", "ui.help.debug"},
}

// helpLines packs bindings into lines of at most n characters, two spaces between bindings,
// each saying what it does in pack p's words.
func helpLines(p *content.Pack, bindings []Binding, n int) []string {
	var lines []string
	line := ""
	for _, b := range bindings {
		s := b.Keys + " " + b.does(p)
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
			fmt.Fprintf(&sb, "  %-*s  %s\n", w, b.Keys, b.does(nil))
		}
	}
	return sb.String()
}
