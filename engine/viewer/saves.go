package viewer

import (
	"path/filepath"

	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// Saving and loading while playing (docs/design.md, Saves): F5 saves to the quick save, F9 loads
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

// notice shows text at the top of the screen for a while.
func (v *Viewer) notice(text string) {
	v.Notice, v.NoticeUntil = text, v.Ticks+noticeUpdates
}

// SaveGame writes the world to save name.
func (v *Viewer) SaveGame(name string) {
	if v.SaveDir == "" {
		v.notice("No place to save")
		return
	}
	if err := save.Write(v.savePath(name), name, v.World); err != nil {
		v.notice("Not saved: " + err.Error())
		return
	}
	v.notice("Saved")
}

// LoadGame replaces the world with save name.
func (v *Viewer) LoadGame(name string) {
	if v.SaveDir == "" {
		v.notice("No place to load from")
		return
	}
	w, warning, err := save.Load(v.savePath(name), v.Pack)
	if err != nil {
		v.notice("Not loaded: " + err.Error())
		return
	}
	v.setWorld(w)
	if v.Audio != nil {
		v.Audio.StopMusic()
	}
	if warning != "" {
		v.notice("Loaded; " + warning)
		return
	}
	v.notice("Loaded")
}

// autosave writes the autosave when events change level while playing.
func (v *Viewer) autosave(events []sim.Event) {
	if !v.Play || v.SaveDir == "" || v.World.Talk() != nil {
		return
	}
	for _, e := range events {
		if e.Kind == sim.EventLevelChanged {
			if err := save.Write(v.savePath(AutoSave), AutoSave, v.World); err != nil {
				v.notice("Not saved: " + err.Error())
			}
			return
		}
	}
}
