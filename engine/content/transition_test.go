package content

import (
	"image/color"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestTransitions(t *testing.T) {
	p := testPack()
	p.Effects = &EffectCues{ID: "effects", Cues: map[string]string{},
		Transitions: map[string]*Transition{
			TransitionTeleporter: {Style: StyleIris, MS: 700, HoldMS: 150, Color: "#ff8000"},
			TransitionFall:       {Style: StyleSqueeze, From: SideBottom, MS: 300, Steps: 5},
			TransitionRipple:     {Style: StyleCut, MS: 300},
		},
		ItemLost: &LostItem{Style: LostShatter, MS: 450}}
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}
	if tr := p.Transition(TransitionTeleporter); tr == nil || tr.RGBA() != (color.RGBA{0xff, 0x80, 0, 0xff}) {
		t.Errorf("teleporter: %+v", tr)
	}
	if p.Transition(TransitionRipple) != nil || p.Transition(TransitionTravel) != nil {
		t.Error("a cut or a cue left out has a transition")
	}
	if l := p.Lost(); l == nil || l.Style != LostShatter {
		t.Errorf("lost: %+v", l)
	}
	dir := t.TempDir()
	if err := Write(p, dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Effects, p.Effects) {
		t.Errorf("effects %+v, want %+v", got.Effects, p.Effects)
	}

	for name, tc := range map[string]struct {
		tr   *Transition
		cue  string
		want string
	}{
		"cue":   {&Transition{Style: StyleFade, MS: 1}, "warp", "unknown transition cue"},
		"style": {&Transition{Style: "spin", MS: 1}, TransitionFall, "unknown style"},
		"side":  {&Transition{Style: StyleSlide, MS: 1, From: "up"}, TransitionFall, "unknown side"},
		"hold":  {&Transition{Style: StyleIris, MS: 1, HoldMS: 2}, TransitionFall, "longer than"},
		"color": {&Transition{Style: StyleIris, MS: 1, Color: "black"}, TransitionFall, "#rrggbb"},
	} {
		p := testPack()
		p.Effects = &EffectCues{Cues: map[string]string{}, Transitions: map[string]*Transition{tc.cue: tc.tr}}
		if err := p.Verify(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", name, err, tc.want)
		}
	}
	p = testPack()
	p.Effects = &EffectCues{Cues: map[string]string{}, ItemLost: &LostItem{Style: "melt", MS: 1}}
	if err := p.Verify(); err == nil || !strings.Contains(err.Error(), "unknown style") {
		t.Errorf("lost style: error %v", err)
	}
}
