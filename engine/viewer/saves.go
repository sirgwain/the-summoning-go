package viewer

import (
	"image"
	"path/filepath"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"

	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Saving and loading while playing (docs/design.md, Saves): F5 saves to the quick save, F8 loads
// it, and changing level writes the autosave. Saves go in SaveDir; without one there are none.

// Save names.
const (
	QuickSave = "quick"
	AutoSave  = "auto"
	// noticeUpdates is how long a notice shows.
	noticeUpdates = 3 * sim.StepsPerSecond
)

// savePath returns the path of save name.
func (v *Viewer) savePath(name string) string { return filepath.Join(v.SaveDir, name+".sav") }

// notice shows text on the message line for a while.
func (v *Viewer) notice(text string) {
	v.Notice, v.NoticeUntil = text, v.Ticks+noticeUpdates
}

// announce shows the game's messages among events on the message line, those said in one update
// together, leaving out what a mouth says (shown beside it).
func (v *Viewer) announce(events []sim.Event) {
	var lines []string
	for _, e := range events {
		if e.Kind != sim.EventMessage || v.isMouth(e) {
			continue
		}
		if s := inspect.Message(v.Pack, e); s != "" {
			lines = append(lines, strings.ReplaceAll(s, "@", v.playerName()))
		}
	}
	if len(lines) > 0 {
		v.notice(strings.Join(lines, " "))
	}
}

// NoticeRect is where the message line is drawn: centered just above the hotbar and the panels
// over it, as wide as its text.
func (v *Viewer) NoticeRect() image.Rectangle {
	w, h := len([]rune(v.Notice))*CharWidth+8, LineHeight+4
	bottom := v.Screen.Y - Margin
	if bar := v.Hotbar(); bar != nil {
		bottom = bar.Frame.Min.Y - Margin
	}
	if sp := v.SpellPanel(); sp != nil {
		bottom = min(bottom, sp.Frame.Min.Y-Margin)
	}
	if ip := v.Inventory(); ip != nil {
		bottom = min(bottom, ip.Frame.Min.Y-Margin)
	}
	x := (v.Screen.X - w) / 2
	return image.Rect(x, bottom-h, x+w, bottom)
}

// SaveGame writes the world to save name.
func (v *Viewer) SaveGame(name string) {
	if v.SaveDir == "" {
		v.notice(v.ui("ui.no_save_place"))
		return
	}
	if v.Holding() {
		v.notice(v.ui("ui.saving_held"))
		return
	}
	if err := save.Write(v.savePath(name), name, v.World); err != nil {
		v.notice(v.ui("ui.not_saved", "error", err))
		return
	}
	v.notice(v.ui("ui.saved"))
}

// LoadGame replaces the world with save name.
func (v *Viewer) LoadGame(name string) {
	if v.SaveDir == "" {
		v.notice(v.ui("ui.no_load_place"))
		return
	}
	playing := v.Play && v.Front == nil
	if err := v.LoadFile(v.savePath(name)); err != nil {
		v.notice(v.ui("ui.not_loaded", "error", err))
		return
	}
	// Loaded during play, the view as last drawn gives way to the loaded world.
	if d := v.Pack.Transition(content.TransitionLoad); playing && d != nil {
		v.transitions++
		v.Transition = &Transition{Def: d, From: v.playerMiddle(), To: v.playerMiddle(), Seq: v.transitions}
	}
}

// LoadFile replaces the world with the save at path.
func (v *Viewer) LoadFile(path string) error {
	w, warning, err := save.Load(path, v.Pack)
	if err != nil {
		return err
	}
	v.setWorld(w)
	if v.Audio != nil {
		v.Audio.StopMusic()
	}
	if warning != "" {
		v.notice(v.ui("ui.loaded_warning", "error", warning))
		return nil
	}
	v.notice(v.ui("ui.loaded"))
	return nil
}

// autosave writes the autosave when events change level while playing. A change of level that
// comes with a transition, a view effect or anything waiting for one is saved once they are over.
func (v *Viewer) autosave(events []sim.Event) {
	if !v.Play || v.SaveDir == "" {
		return
	}
	for _, e := range events {
		if e.Kind == sim.EventLevelChanged {
			v.autosaveDue = true
		}
	}
	if !v.autosaveDue || v.World.Talk() != nil || v.Holding() || v.World.Waiting() {
		return
	}
	v.autosaveDue = false
	if err := save.Write(v.savePath(AutoSave), AutoSave, v.World); err != nil {
		v.notice(v.ui("ui.not_saved", "error", err))
	}
}
