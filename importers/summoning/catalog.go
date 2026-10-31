package summoning

import (
	"fmt"
	"math"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Templates the rules name (items.md, rules.md).
const (
	templateRandom         = 0x00
	templateBagOfLightness = 0x05
	templateFirstGem       = 0xC0
	categoryRandom         = 6
)

// sealingSpheres are the templates that open sealed door stages 5, 6 and 7 (rules.md, Doors).
var sealingSpheres = [3]int{0xC9, 0xCB, 0xCD}

// categoryTags are the tags of the categories an item can be recognized by (items.md, Import).
var categoryTags = map[int]string{1: "body_armor", 4: "quiver", 7: "arrows", 8: "bottle"}

// engineMessages are the TEXT records the engine's own messages use, and whether the message
// takes an item's name.
var engineMessages = map[string]struct {
	text     int
	itemName bool
}{
	content.MsgLeverStuck:    {0x86, false},
	content.MsgSlotWrongItem: {0x8F, true},
	content.MsgSlotStuck:     {0x87, true},
	content.MsgNoDropHere:    {0x72, false},
	content.MsgNoRoom:        {0x7F, false},
}

// ms converts a duration in ticks to milliseconds.
func ms(ticks int) int { return int(math.Round(float64(ticks) * 1000 / ticksPerSecond)) }

func msgID(n int) string             { return fmt.Sprintf("msg.%d", n) }
func levelMsgID(level, n int) string { return fmt.Sprintf("msg.l%02d.%d", level, n) }

// text adds TEXT as text.en, every record keyed msg.<n>, and the engine's messages.
func (im *importer) text(strs []string) {
	t := &content.Text{ID: "text.en", Strings: make(map[string]string, len(strs))}
	for i, s := range strs {
		t.Strings[msgID(i)] = s
	}
	im.pack.Texts[t.ID] = t
	im.strings = t.Strings

	m := &content.Messages{ID: "messages", Messages: map[string]string{}}
	for key, msg := range engineMessages {
		id := msgID(msg.text)
		s, ok := t.Strings[id]
		if !ok {
			im.report.addf("message %s: no TEXT record %d", key, msg.text)
			continue
		}
		if msg.itemName {
			t.Strings[id] = strings.ReplaceAll(s, "%s", "{item}")
		}
		m.Messages[key] = id
	}
	im.pack.Messages = m
}

// items adds an item for every OBJECTS template, named after its name.
func (im *importer) items() error {
	o := im.objects
	used := map[string]int{}
	im.itemIDs = make([]string, len(o.Templates))
	for i := range o.Templates {
		tm := &o.Templates[i]
		name := msgID(o.NameBase + tm.NameIndex())
		s, ok := im.strings[name]
		if !ok {
			return fmt.Errorf("object template %02Xh: no name text %s", i, name)
		}
		id := "item." + slug(s)
		if id == "item." {
			id = fmt.Sprintf("item.template_%d", i)
		}
		if used[id]++; used[id] > 1 {
			id = fmt.Sprintf("%s.%d", id, used[id])
		}
		im.itemIDs[i] = id

		icon, err := im.image(int(im.init.Bases.SmallObject) + tm.Image())
		if err != nil {
			return fmt.Errorf("object template %02Xh icon: %w", i, err)
		}
		floor, err := im.image(int(im.init.Bases.TinyObject) + tm.Image())
		if err != nil {
			return fmt.Errorf("object template %02Xh floor image: %w", i, err)
		}
		it := &content.Item{
			ID: id, Name: name, Weight: tm.Weight(), Size: tm.Size(), Icon: icon, Floor: floor,
			Random: !tm.NeverRandom() && (tm.Weapon() || tm.Category() == categoryRandom),
		}
		if tm.Container() {
			it.Container = &content.Container{Capacity: tm.Capacity(), Largest: tm.Largest(), Weightless: i == templateBagOfLightness}
		}
		if tag, ok := categoryTags[tm.Category()]; ok {
			it.Tags = append(it.Tags, tag)
		}
		if tm.Weapon() {
			it.Tags = append(it.Tags, "weapon")
		}
		if tm.TwoHanded() {
			it.Tags = append(it.Tags, "two_handed")
		}
		im.pack.Items[id] = it
	}
	return nil
}

// slug lowercases s and joins its runs of letters and digits with underscores.
func slug(s string) string {
	var b strings.Builder
	gap := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if gap && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			gap = false
		} else {
			gap = true
		}
	}
	return b.String()
}

// image returns frame 0 of resource id.
func (im *importer) image(id int) (content.Image, error) {
	sid, b, err := im.rawSprite(id)
	if err != nil {
		return content.Image{}, err
	}
	if len(b.Frames) == 0 {
		return content.Image{}, fmt.Errorf("resource %d has no frames", id)
	}
	return content.Image{Sprite: sid, Frame: 0}, nil
}

// item returns the item of template t, reporting a template OBJECTS does not have.
func (im *importer) item(t int, what string) (string, bool) {
	if t < 0 || t >= len(im.itemIDs) {
		im.report.addf("%s: no object template %02Xh", what, t)
		return "", false
	}
	return im.itemIDs[t], true
}

// Anchors of the things the importer places (entities.md, Engine model).
var (
	creatureAnchor = content.Offset{18, 19}
	blockAnchor    = content.Offset{24, 20}
	creatureOffset = content.Offset{-6, 12}
)

// NPC record values (creatures.md).
const (
	stationaryCaster     = 2
	firstPattern         = 100
	patternFixed         = 100
	slowBank             = 0x12
	slowSpeed            = 2
	creatureSpeed        = 8
	creatureWeight       = 100
	creatureFootprint    = 16
	trapFootprint        = 20
	behaviorNeverHostile = 0x0004
	behaviorCorpse       = 0x0010
	behaviorHostile      = 0x0100
	behaviorResists      = 0x0200
	behaviorInvulnerable = 0x1000
	boulderSpeed         = 8
	boulderInterval      = 4
	trapInterval         = 128
	boulderRollFrames    = 10
)

// creature returns the thing template of NPC record n, adding it the first time.
func (im *importer) creature(n int) (string, error) {
	id := fmt.Sprintf("thing.npc.%d", n)
	if _, ok := im.pack.Things[id]; ok {
		return id, nil
	}
	if n < 0 || n >= len(im.npcs) {
		return "", fmt.Errorf("no NPC record %d", n)
	}
	rec := &im.npcs[n]
	res := int(im.init.Bases.NPC) + rec.Bank()
	sid, b, err := im.rawSprite(res)
	if err != nil {
		return "", fmt.Errorf("NPC record %d: %w", n, err)
	}
	im.anchor(sid, creatureAnchor)

	t := &content.Thing{
		ID: id, Kind: content.ThingCreature, Sprite: sid, Speed: creatureSpeed,
		IntervalMS: ms(rec.Interval()), Weight: creatureWeight, Footprint: creatureFootprint,
		HitPoints: content.Dice{Count: rec.HitDice(), Sides: 10, Bonus: 4 * rec.HitBonus()},
		Stats:     map[string]int{"resistance": rec.Resistance(), "bonus": rec.Bonus(), "accuracy": rec.Accuracy()},
	}
	if t.HitPoints.Count == 0 {
		t.HitPoints.Sides = 0
	}
	if rec.Bank() == slowBank {
		t.Speed = slowSpeed
	}
	beh := rec.Behavior()
	for _, b := range []struct {
		bits uint16
		name string
	}{
		{behaviorHostile, content.BehaviorHostile},
		{behaviorNeverHostile | behaviorInvulnerable, content.BehaviorNeverHostile},
		{behaviorCorpse, content.BehaviorLeavesCorpse},
		{behaviorResists, content.BehaviorResistsTeleport},
		{behaviorInvulnerable, content.BehaviorInvulnerable},
	} {
		if beh&b.bits != 0 {
			t.Behaviors = append(t.Behaviors, b.name)
		}
	}

	frames, d0 := rec.Frames(), rec.Direction()
	anims := map[string]content.Animation{}
	still := func(f int) content.Animation { return content.Animation{Frames: []int{f}} }
	switch {
	case frames >= firstPattern:
		t.Speed = 0
		f := 0
		if frames == patternFixed {
			f = d0
		}
		anims["idle"] = still(f)
		if frames != patternFixed {
			im.report.addf("animated object pattern %d: drawn still", frames)
		}
	case frames == stationaryCaster:
		t.Facing = 2
		anims["idle.2"], anims["idle.6"] = still(0), still(1)
	default:
		t.Facing = 2 * (d0 % 4)
		wrap := func(f int) int { return ((f % frames) + frames) % frames }
		every := content.Fixed(t.IntervalMS)
		for d := range 4 {
			e := 2 * d
			walk := []int{wrap(d), wrap(d + 4)}
			if frames > 12 {
				walk = []int{wrap(d), wrap(d + 4), wrap(d), wrap(d + 8)}
			}
			anims[fmt.Sprintf("idle.%d", e)] = still(wrap(d))
			anims[fmt.Sprintf("walk.%d", e)] = content.Animation{Frames: walk, MS: every}
			anims[fmt.Sprintf("attack.%d", e)] = still(wrap(d + frames - 4))
		}
		anims["dead"] = still(frames - 1)
	}
	t.Animations = im.fitAnimations(fmt.Sprintf("NPC record %d", n), anims, len(b.Frames))
	im.pack.Things[id] = t
	return id, nil
}

// fitAnimations drops, and reports, animations using frames the sprite does not have.
func (im *importer) fitAnimations(what string, anims map[string]content.Animation, n int) map[string]content.Animation {
	for name, a := range anims {
		for _, f := range a.Frames {
			if f >= n {
				im.report.addf("%s: animation %s needs frame %d of %d; left out", what, name, f, n)
				delete(anims, name)
				break
			}
		}
	}
	return anims
}

// anchor gives sprite sid its anchor, reporting a sprite used with two different anchors.
func (im *importer) anchor(sid string, a content.Offset) {
	b := im.pack.Sprites[sid]
	switch {
	case b.Anchor == nil:
		b.Anchor = &a
	case *b.Anchor != a:
		im.report.addf("%s: used with anchors %v and %v; keeps the first", sid, *b.Anchor, a)
	}
}

// boulder returns the thing template of boulders drawn from resource res.
func (im *importer) boulder(res int) (string, error) {
	id := fmt.Sprintf("thing.boulder.%d", res)
	if _, ok := im.pack.Things[id]; ok {
		return id, nil
	}
	sid, b, err := im.rawSprite(res)
	if err != nil {
		return "", fmt.Errorf("boulder: %w", err)
	}
	im.anchor(sid, blockAnchor)
	every := content.Fixed(ms(boulderInterval))
	anims := map[string]content.Animation{"idle": {Frames: []int{0}}}
	if len(b.Frames) >= boulderRollFrames {
		anims["roll.0"] = content.Animation{Frames: []int{0, 1, 2, 3, 4}, MS: every}
		anims["roll.2"] = content.Animation{Frames: []int{5, 6, 7, 8, 9}, MS: every}
		anims["roll.4"] = content.Animation{Frames: []int{4, 3, 2, 1, 0}, MS: every}
		anims["roll.6"] = content.Animation{Frames: []int{9, 8, 7, 6, 5}, MS: every}
	} else {
		im.report.addf("boulder resource %d: %d frames, not enough to roll", res, len(b.Frames))
	}
	im.pack.Things[id] = &content.Thing{
		ID: id, Kind: content.ThingBoulder, Sprite: sid, Speed: boulderSpeed, IntervalMS: ms(boulderInterval),
		Weight: creatureWeight, Footprint: creatureFootprint, Animations: im.fitAnimations(id, anims, len(b.Frames)),
	}
	return id, nil
}

// trap returns the thing template of traps drawn from resource res.
func (im *importer) trap(res int) (string, error) {
	id := fmt.Sprintf("thing.trap.%d", res)
	if _, ok := im.pack.Things[id]; ok {
		return id, nil
	}
	sid, b, err := im.rawSprite(res)
	if err != nil {
		return "", fmt.Errorf("trap: %w", err)
	}
	im.anchor(sid, blockAnchor)
	im.pack.Things[id] = &content.Thing{
		ID: id, Kind: content.ThingTrap, Sprite: sid, IntervalMS: ms(trapInterval), Weight: creatureWeight,
		Footprint:  trapFootprint,
		Animations: im.fitAnimations(id, map[string]content.Animation{"idle": {Frames: []int{0}}}, len(b.Frames)),
	}
	return id, nil
}
