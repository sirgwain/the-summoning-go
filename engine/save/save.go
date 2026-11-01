// Package save writes and reads saved games: the world's state as gzipped JSON, with the pack it
// was made with, so loading on another pack or version can warn (docs/design.md, Saves).
package save

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// File is a saved game.
type File struct {
	// Name is the save's name; Saved when it was made.
	Name  string     `json:"name"`
	Saved time.Time  `json:"saved"`
	State *sim.State `json:"state"`
}

// Write saves world w as name to path, making its directory.
func Write(path, name string, w *sim.World) error {
	s, err := w.State()
	if err != nil {
		return err
	}
	return writeFile(path, &File{Name: name, Saved: time.Now().UTC(), State: s})
}

// WriteState saves state s as name to path, making its directory.
func WriteState(path, name string, s *sim.State) error {
	return writeFile(path, &File{Name: name, Saved: time.Now().UTC(), State: s})
}

// DefaultDir returns where saves of pack go when no directory is given, or "" when there is no
// configuration directory.
func DefaultDir(pack string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "summoning", "saves", pack)
}

// writeFile writes file to path, making its directory.
func writeFile(path string, file *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".save-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	z := gzip.NewWriter(f)
	if err := json.NewEncoder(z).Encode(file); err != nil {
		f.Close()
		return err
	}
	if err := z.Close(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// Read reads the save at path.
func Read(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var file File
	if err := json.NewDecoder(z).Decode(&file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if file.State == nil {
		return nil, fmt.Errorf("%s: no state", path)
	}
	return &file, nil
}

// Load reads the save at path and restores its world on pack p. The warning, if not empty, says
// the save was made with another pack or version.
func Load(path string, p *content.Pack) (w *sim.World, warning string, err error) {
	file, err := Read(path)
	if err != nil {
		return nil, "", err
	}
	s := file.State
	if s.Pack != p.Manifest.ID || s.Version != p.Manifest.Version {
		warning = fmt.Sprintf("saved with pack %s %s, loading on %s %s", s.Pack, s.Version, p.Manifest.ID, p.Manifest.Version)
	}
	w, err = sim.Restore(p, s)
	return w, warning, err
}

// Rename moves the save at from to to, naming it name.
func Rename(from, to, name string) error {
	f, err := Read(from)
	if err != nil {
		return err
	}
	f.Name = name
	if err := writeFile(to, f); err != nil {
		return err
	}
	if from == to {
		return nil
	}
	return os.Remove(from)
}
