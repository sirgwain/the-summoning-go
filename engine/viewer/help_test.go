package viewer

import (
	"slices"
	"strings"
	"testing"
)

func TestHelpLinesPack(t *testing.T) {
	got := helpLines([]Binding{{"A", "one"}, {"B", "two"}, {"C", "three"}}, 12)
	if want := []string{"A one  B two", "C three"}; !slices.Equal(got, want) {
		t.Errorf("helpLines = %q, want %q", got, want)
	}
}

// TestPanelHelp checks the panel shows play's and the tools' bindings in their modes.
func TestPanelHelp(t *testing.T) {
	v := newViewer(t)
	has := func(s string) bool {
		return slices.ContainsFunc(v.PanelLines(), func(l string) bool { return strings.Contains(l, s) })
	}
	if !has("U/Enter use") || has("N (held) nap") {
		t.Errorf("the tools' panel does not show the tools' keys: %q", v.PanelLines())
	}
	v.SetPlay(true)
	if !has("N (held) nap") || !has("M level map") || has("U/Enter use") {
		t.Errorf("the play panel does not show play's keys: %q", v.PanelLines())
	}
}
