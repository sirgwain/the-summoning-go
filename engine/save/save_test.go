package save

import (
	"path/filepath"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestWriteAndLoad(t *testing.T) {
	p := devpack.Build()
	w, err := sim.New(p, 3)
	if err != nil {
		t.Fatal(err)
	}
	for range 40 {
		w.Update([]input.Command{input.Walk{Direction: 0}})
	}
	path := filepath.Join(t.TempDir(), "saves", "quick.sav")
	if err := Write(path, "quick", w); err != nil {
		t.Fatal(err)
	}
	r, warning, err := Load(path, p)
	if err != nil || warning != "" {
		t.Fatalf("loading: %v %q", err, warning)
	}
	if r.Player().Cell() != w.Player().Cell() || r.Step() != w.Step() {
		t.Errorf("loaded at %v step %d, saved at %v step %d", r.Player().Cell(), r.Step(), w.Player().Cell(), w.Step())
	}
	p.Manifest.Version = "9"
	if _, warning, _ := Load(path, p); warning == "" {
		t.Error("no warning for another version")
	}
}

func TestWriteState(t *testing.T) {
	p := devpack.Build()
	w, err := sim.New(p, 3)
	if err != nil {
		t.Fatal(err)
	}
	s, err := w.State()
	if err != nil {
		t.Fatal(err)
	}
	s.Player.HitPoints = 1
	path := filepath.Join(t.TempDir(), "imported.sav")
	if err := WriteState(path, "imported", s); err != nil {
		t.Fatal(err)
	}
	f, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := Load(path, p)
	if err != nil || f.Name != "imported" || r.Player().HitPoints != 1 {
		t.Errorf("loaded %q with %d hit points: %v", f.Name, r.Player().HitPoints, err)
	}
}
