package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSavePath(t *testing.T) {
	dir := t.TempDir()
	saves := filepath.Join(dir, "saves")
	if err := os.MkdirAll(saves, 0o755); err != nil {
		t.Fatal(err)
	}
	inSaves := filepath.Join(saves, "quick.sav")
	if err := os.WriteFile(inSaves, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	here := filepath.Join(dir, "here.sav")
	if err := os.WriteFile(here, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, dir, want string }{
		{here, saves, here},                   // found as given
		{"quick.sav", saves, inSaves},         // falls back to the saves directory
		{"missing.sav", saves, "missing.sav"}, // neither: left as given, so the error names it
		{"quick.sav", "", "quick.sav"},        // no saves directory
	} {
		if got := savePath(tc.path, tc.dir); got != tc.want {
			t.Errorf("savePath(%q, %q) = %q, want %q", tc.path, tc.dir, got, tc.want)
		}
	}
}
