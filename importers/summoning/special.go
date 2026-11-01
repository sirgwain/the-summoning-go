package summoning

import (
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Special weapons and objects, and creatures' names, kill practice, mirroring and animated
// patterns (combat.md, Special weapons; items.md; creatures.md).

// Templates, records and texts the special rules name.
const (
	templateSerpentStaff = 0x8A
	templateDecanter     = 0x90
	rustedBladeGrowsAt   = 250
	warMongerEvery       = 1000
	warMongerMaxLevel    = 9
	npcFirstHit          = 0x45
	npcGrown             = 0x46
	npcDecanter          = 0x2E
	npcMirrors           = 0x79
	decanterExperience   = 72903
	msgFirstName         = 0xAA
	killPracticeFrom     = 50
	varWarMonger         = 27
	varDecanter          = 18
	varQuietCombat       = 96
)

// thrownLookOffsets are the offsets from the held-item base of the thrown looks (items.md,
// Throwing).
var thrownLookOffsets = map[int]int{0x67: 0x27, 0x68: 0x28, 0x75: 0x29, 0x6D: 0x2A, 0x97: 0x2D, 0x98: 0x2C, 0x99: 0x2B}

// specialItem gives template i's item what the special rules say about it.
func (im *importer) specialItem(i int, it *content.Item) {
	talker := func(n int) string {
		id := npcID(n)
		im.talkers[id] = true
		return id
	}
	str := func(s string) *string { return &s }
	switch i {
	case templateSerpentStaff:
		it.Poisons = true
	case templateRustedBlade:
		if templateWarMonger < len(im.itemIDs) {
			it.Grows = &content.Grows{Into: im.itemIDs[templateWarMonger], At: rustedBladeGrowsAt,
				First: talker(npcFirstHit), Grown: talker(npcGrown), Counter: varID(varWarMonger)}
		}
	case templateWarMonger:
		it.Levels = &content.Levels{Every: warMongerEvery, Max: warMongerMaxLevel, Counter: varID(varWarMonger), Talk: talker(npcGrown)}
	case templateDecanter:
		xp, kind := decanterExperience, talker(npcDecanter)
		it.OnBreak = []content.Action{
			{SetCounter: &content.SetCounter{Counter: varID(varDecanter), Value: 1}},
			{Converse: str(kind)}, {RemoveAll: str(kind)}, {Experience: &xp},
		}
	}
	if off, ok := thrownLookOffsets[i]; ok {
		sid, b, err := im.rawSprite(int(im.init.Bases.HeldItemAnimation) + off)
		switch {
		case err != nil:
			im.report.addf("object template %02Xh: thrown look: %v", i, err)
		case len(b.Frames) < 8:
			im.report.addf("object template %02Xh: thrown look has %d frames", i, len(b.Frames))
		default:
			it.Flying = &content.Flying{Sprite: sid, FiredOnly: thrownLooks[i]}
		}
	}
}

// specialCreature gives creature template t of record n its name, kill practice, mirroring and
// animated pattern.
func (im *importer) specialCreature(n int, rec *NPCRecord, t *content.Thing) {
	if id := msgID(msgFirstName + n); im.strings[id] != "" && im.strings[id] != "x" {
		t.Name = id
	}
	if n >= killPracticeFrom && rec.Conversation() == 0 {
		t.KillPractice = 2
	}
	if n == npcMirrors {
		t.Behaviors = append(t.Behaviors, content.BehaviorMirrors)
	}
	switch rec.Frames() {
	case patternFixed + 1:
		t.Animate = &content.Animate{Kind: content.AnimateFlicker, Frames: []int{0, 1, 2}}
	case patternFixed + 2:
		t.Animate = &content.Animate{Kind: content.AnimateFlicker, Frames: []int{0, 1, 2, 3}, Random: true}
	case patternFixed + 3:
		t.Animate = &content.Animate{Kind: content.AnimateToggle, Frames: []int{0, 1}}
	case patternFixed + 4:
		t.Animate = &content.Animate{Kind: content.AnimateRandom, Frames: []int{0, 1, 2, 3}}
	}
	if a := t.Animate; a != nil {
		frames := 0
		if b := im.pack.Sprites[t.Sprite]; b != nil {
			frames = b.Frames
		}
		var keep []int
		for _, f := range a.Frames {
			if f < frames {
				keep = append(keep, f)
			}
		}
		if len(keep) < len(a.Frames) {
			im.report.addf("animated object pattern %d: sprite has %d frames", rec.Frames(), frames)
		}
		if a.Frames = keep; len(keep) < 2 {
			t.Animate = nil
		}
	}
}
