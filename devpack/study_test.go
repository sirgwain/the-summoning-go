package devpack

import (
	"image"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// TestStudy plays the study by its script (scripts/study.json) through the viewer, as a person
// would: the player talks to the sage (E), asks about magic by clicking the marked word, reads on
// with keys, types "gift" and takes the potion, leaves (Escape); opens the spell panel (F10),
// clicks in the three icons the sage taught and learns Fireball; turns toward the target and
// casts by clicking Wizardry's box on the hotbar. Watch it with
//
//	summoning view packs/dev --level level.study --script devpack/scripts/study.json --out /tmp/study
func TestStudy(t *testing.T) { forPacks(t, testStudy) }

func testStudy(t *testing.T, build func() *content.Pack) {
	script, err := viewer.LoadScript("scripts/study.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := viewer.New(build(), 1, Study, 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	r := viewer.NewRunner(script, t.TempDir())
	var sawTalk, sawPanel, sawBolt bool
	bolt := v.Pack.Spells[SpellFireball].Effects[0].Missile.Sprite
	for !r.Done() {
		in, err := r.Before(v)
		if err != nil {
			t.Fatal(err)
		}
		v.Update(in)
		if _, err := r.After(v); err != nil {
			t.Fatal(err)
		}
		sawTalk = sawTalk || v.Talk() != nil
		sawPanel = sawPanel || v.SpellPanel() != nil
		for _, c := range v.Cmds {
			sawBolt = sawBolt || c.Sprite == bolt
		}
	}
	if !sawBolt {
		t.Error("the fireball was never drawn in flight")
	}
	w, p := v.World, v.World.Player()
	switch {
	case !sawTalk || !sawPanel:
		t.Fatalf("talked %v, opened the spell panel %v", sawTalk, sawPanel)
	case w.Talk() != nil || v.SpellsOpen:
		t.Fatal("the conversation or the spell panel is still open")
	case !w.Records()[ThingSage].Talked:
		t.Error("the sage was not talked to")
	case p.Slots[0] == nil || p.Slots[0].Item != ItemPotion:
		t.Errorf("no potion on shelf 0: %+v", p.Slots[0])
	case !p.Magic.Learned(SpellFireball):
		t.Errorf("fireball not learned: %+v", p.Magic)
	}
	if tgt := w.Current().Thing("target"); tgt == nil || tgt.HitPoints != 0 {
		t.Errorf("the target lives: %+v", tgt)
	}
}
