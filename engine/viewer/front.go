package viewer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The viewer's side of the frontend: opening it, and starting or restoring play from it.

// deathUpdates is how long the dead player shows before the main menu (docs/spec/scenes.md,
// Death).
const deathUpdates = 3 * sim.StepsPerSecond

// OpenFrontend shows the main menu.
func (v *Viewer) OpenFrontend() {
	v.Front = NewFrontend(v.Pack, v.Seed+uint64(v.Ticks), v.SaveNames(), v.Screen)
}

// SaveNames returns the saves in SaveDir, the newest first.
func (v *Viewer) SaveNames() []string {
	if v.SaveDir == "" {
		return nil
	}
	entries, err := os.ReadDir(v.SaveDir)
	if err != nil {
		return nil
	}
	type save struct {
		name string
		at   int64
	}
	var saves []save
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sav" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		saves = append(saves, save{strings.TrimSuffix(e.Name(), ".sav"), info.ModTime().UnixNano()})
	}
	sort.SliceStable(saves, func(i, j int) bool { return saves[i].at > saves[j].at })
	names := make([]string, len(saves))
	for i, s := range saves {
		names[i] = s.name
	}
	return names
}

// frontUpdate runs the frontend, and leaves it when it has finished.
func (v *Viewer) frontUpdate(in Input) {
	f := v.Front
	f.Size = v.Screen
	f.Update(in)
	if !f.Done {
		return
	}
	v.Front = nil
	switch {
	case f.Quit:
		v.Quit = true
	case f.Restore != "":
		v.LoadGame(f.Restore)
		v.SetPlay(true)
	case f.Character != nil:
		w, err := sim.NewGame(v.Pack, v.Seed+uint64(v.Ticks), *f.Character)
		if err != nil {
			v.notice("No new game: " + err.Error())
			v.OpenFrontend()
			return
		}
		v.setWorld(w)
		v.SetPlay(true)
	}
}

// setWorld replaces the world, clearing what showed of the old one.
func (v *Viewer) setWorld(w *sim.World) {
	v.World = w
	v.Shown, v.Said, v.Reading, v.Typed, v.SpellsOpen, v.InventoryOpen, v.MapOpen = "", nil, nil, "", false, false, false
	v.deadFor = 0
	v.rebuild()
	v.Recenter()
}

// afterPlay opens the frontend when play has ended: the player dead for a while, or an ending,
// whose scene plays first.
func (v *Viewer) afterPlay() {
	if !v.Play || v.Front != nil {
		return
	}
	if e := v.World.Ended(); e != nil {
		v.OpenFrontend()
		v.Front.PlayScene(e.Scene, ScreenMenu)
		return
	}
	if p := v.World.Player(); p != nil && p.Dead {
		if v.deadFor++; v.deadFor >= deathUpdates {
			v.OpenFrontend()
		}
	}
}
