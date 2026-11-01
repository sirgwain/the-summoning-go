package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
)

func TestNewGame(t *testing.T) {
	p := devpack.Build()
	p.Creation.Counter = "var.new_game"
	p.Creation.Opening = devpack.ThingSage
	c := Character{Name: "Mira", Portrait: devpack.PortraitPlayer, Attributes: map[string]int{
		AttrStrength: 10, AttrAgility: 11, AttrEndurance: 12, AttrAccuracy: 13, AttrTalent: 14, AttrPower: 15}}
	w, err := NewGame(p, 1, c)
	if err != nil {
		t.Fatal(err)
	}
	pl := w.Player()
	if pl.Name != "Mira" || pl.Attributes[AttrTalent] != 14 || pl.MaxAttributes[AttrPower] != 15 {
		t.Errorf("name %q, attributes %v, maxima %v", pl.Name, pl.Attributes, pl.MaxAttributes)
	}
	if pl.MaxHitPoints != 10+12+20 || pl.HitPoints != pl.MaxHitPoints || pl.MaxMagicPoints != 14+15+20 {
		t.Errorf("hit points %d/%d, magic points %d", pl.HitPoints, pl.MaxHitPoints, pl.MaxMagicPoints)
	}
	if pl.Skills[SkillWeapon].Level != 1 || pl.Skills[SkillSchool].Level != 1 || pl.Skills[SkillWeapon+1].Level != 0 {
		t.Errorf("skills %+v", pl.Skills)
	}
	want := map[int]string{0: devpack.ItemPotion, 2: devpack.ItemScroll, SlotRightHand: devpack.ItemSword,
		SlotFeet: devpack.ItemBoots, SlotBody: devpack.ItemMail}
	for i, o := range pl.Slots {
		got := ""
		if o != nil {
			got = o.Item
		}
		if got != want[i] {
			t.Errorf("slot %d holds %q, want %q", i, got, want[i])
		}
	}
	if pl.Magic.Icons != "BDJ" {
		t.Errorf("knows icons %q", pl.Magic.Icons)
	}
	if w.Counters()["var.new_game"] != 1 {
		t.Errorf("counters %v", w.Counters())
	}
	// The opening conversation starts on the first update.
	if w.Talk() != nil {
		t.Fatal("a conversation before the first update")
	}
	w.Update(nil)
	if w.Talk() == nil || w.Talk().Def.ID != devpack.ConversationSage {
		t.Errorf("the opening conversation did not start: %+v", w.Talk())
	}
}

func TestNewGameNeedsCreation(t *testing.T) {
	p := devpack.Build()
	p.Creation = nil
	if _, err := NewGame(p, 1, Character{}); err == nil {
		t.Error("a new game on a pack without a creation record")
	}
}

func TestChangeSexTakesOtherPortrait(t *testing.T) {
	p := devpack.Build()
	other := *p.Portraits[devpack.PortraitPlayer]
	other.ID, other.Female, other.Other = "portrait.other", true, devpack.PortraitPlayer
	p.Portraits[other.ID] = &other
	p.Portraits[devpack.PortraitPlayer].Other = other.ID
	w, err := New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	pl := w.Player()
	pl.Portrait = devpack.PortraitPlayer
	w.changeSex()
	if !pl.Status.Female || pl.Portrait != other.ID {
		t.Errorf("female %v, portrait %q", pl.Status.Female, pl.Portrait)
	}
	w.changeSex()
	if pl.Status.Female || pl.Portrait != devpack.PortraitPlayer {
		t.Errorf("back: female %v, portrait %q", pl.Status.Female, pl.Portrait)
	}
}
