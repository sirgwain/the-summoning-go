package viewer

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The game's menus (docs/spec/dialogue.md, the disk menu): a conversation's `menu` action
// opens one as frontend screens over play; when it closes, the conversation goes on.

// Menu screens.
const (
	ScreenSave     = "save"
	ScreenDelete   = "delete"
	ScreenRename   = "rename"
	ScreenRenameTo = "rename_to"
	ScreenConfirm  = "confirm"
)

// openMenu opens menu kind over play.
func (v *Viewer) openMenu(kind string) {
	f := NewFrontend(v.Pack, v.Seed+uint64(v.Ticks), v.SaveNames(), v.Screen)
	f.disk = kind
	switch kind {
	case content.MenuSave:
		f.show(ScreenSave)
	case content.MenuRestore:
		f.show(ScreenRestore)
	case content.MenuDelete:
		f.show(ScreenDelete)
	case content.MenuRename:
		f.show(ScreenRename)
	default:
		f.show(ScreenConfirm)
	}
	v.Front = f
}

// menuDone carries out what a menu chose and goes on with the conversation that opened it.
func (v *Viewer) menuDone(f *Frontend) {
	switch {
	case f.SaveAs != "":
		v.SaveGame(f.SaveAs)
	case f.Delete != "":
		if err := os.Remove(v.savePath(f.Delete)); err != nil {
			v.notice("Not deleted: " + err.Error())
		} else {
			v.notice("Deleted")
		}
	case f.RenameTo != "":
		if err := save.Rename(v.savePath(f.Rename), v.savePath(f.RenameTo), f.RenameTo); err != nil {
			v.notice("Not renamed: " + err.Error())
		} else {
			v.notice("Renamed")
		}
	case f.Restore != "":
		// The restored world has no conversation open.
		v.LoadGame(f.Restore)
		return
	case f.Confirmed && f.disk == content.MenuQuit:
		v.Quit = true
		return
	case f.Confirmed && f.disk == content.MenuLeave:
		v.OpenFrontend()
		return
	}
	if t := v.World.Talk(); t != nil && t.Line != nil && t.Line.Kind == sim.LineMenu {
		v.updateWorld([]input.Command{input.Continue{}})
	}
}

// SaveName turns a name typed into one a save's file can have.
func SaveName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// layoutDisk lays out the menu screens, reporting whether screen is one.
func (f *Frontend) layoutDisk() bool {
	var opts []option
	saves := func() {
		for i, s := range f.Saves {
			opts = append(opts, option{fmt.Sprintf("save.%d", i), s})
		}
	}
	switch f.Screen {
	case ScreenSave:
		f.Title = "Save the game"
		f.Lines = []string{"Name: " + f.typed + "_", "Type a name, or choose a save to replace."}
		saves()
		opts = append(opts, option{"ok", "Save"}, option{"back", "Cancel"})
	case ScreenDelete:
		f.Title = "Delete a saved game"
		saves()
		opts = append(opts, option{"back", "Cancel"})
	case ScreenRename:
		f.Title = "Rename a saved game"
		saves()
		opts = append(opts, option{"back", "Cancel"})
	case ScreenRenameTo:
		f.Title = "Rename " + f.Rename
		f.Lines = []string{"New name: " + f.typed + "_"}
		opts = append(opts, option{"ok", "Rename"}, option{"back", "Cancel"})
	case ScreenConfirm:
		f.Title = "Leave the game for the main menu?"
		if f.disk == content.MenuQuit {
			f.Title = "Quit the game?"
		}
		opts = append(opts, option{"yes", "Yes"}, option{"back", "No"})
	default:
		return false
	}
	f.list(opts)
	return true
}

// typing reports whether the screen takes a typed name.
func (f *Frontend) typing() bool { return f.Screen == ScreenSave || f.Screen == ScreenRenameTo }

// typeInto takes the characters typed into the menu's name.
func (f *Frontend) typeInto(in Input) {
	for _, r := range in.Chars {
		if r >= ' ' && r < 0x7F && len(f.typed) < NameLength {
			f.typed += string(r)
		}
	}
	if in.pressed(KeyBackspace) && f.typed != "" {
		f.typed = f.typed[:len(f.typed)-1]
	}
}

// pickDisk acts on button name of a menu screen, reporting whether the screen is one.
func (f *Frontend) pickDisk(name string, num func(string) (int, bool)) bool {
	i, isSave := num("save.")
	switch f.Screen {
	case ScreenSave:
		switch {
		case isSave:
			f.typed = f.Saves[i]
		case name == "ok":
			if n := SaveName(f.typed); n != "" {
				f.SaveAs, f.Done = n, true
			}
		case name == "back":
			f.Done = true
		}
	case ScreenDelete:
		switch {
		case isSave:
			f.Delete, f.Done = f.Saves[i], true
		case name == "back":
			f.Done = true
		}
	case ScreenRename:
		switch {
		case isSave:
			f.Rename, f.typed = f.Saves[i], f.Saves[i]
			f.show(ScreenRenameTo)
		case name == "back":
			f.Done = true
		}
	case ScreenRenameTo:
		switch name {
		case "ok":
			if n := SaveName(f.typed); n != "" {
				f.RenameTo, f.Done = n, true
			}
		case "back":
			f.show(ScreenRename)
		}
	case ScreenConfirm:
		f.Confirmed = name == "yes"
		f.Done = name == "yes" || name == "back"
	default:
		return false
	}
	return true
}
