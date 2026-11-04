package devpack

import (
	"math"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The dev pack's magic and talk: gesture icons, spells and their art, keywords, and a sage in a
// study who talks, teaches a gesture and gives a gift.

// Sprite ids of the magic art.
const (
	spriteGestures   = "sprite.gestures"
	spriteSpellIcons = "sprite.spell_icons"
	spriteBolt       = "sprite.bolt"
	spriteArrowBolt  = "sprite.arrow_bolt"
	spriteCloud      = "sprite.cloud"
	spriteMagicWall  = "sprite.magic_wall"
	spritePortal     = "sprite.portal"
	spriteScroll     = "sprite.scroll"
	spriteSagePic    = "sprite.portrait.sage"
	spritePointer    = "sprite.pointer"
)

// Spell ids.
const (
	SpellFlameArrow = "spell.flame_arrow"
	SpellFireball   = "spell.fireball"
	SpellKano       = "spell.kano"
	SpellSwiftness  = "spell.swiftness"
	SpellMagicWall  = "spell.magic_wall"
	SpellGateway    = "spell.gateway"
	SpellHeal       = "spell.heal"
)

// Thing, portrait, conversation and counter ids.
const (
	ThingSage        = "thing.sage"
	ThingTarget      = "thing.target"
	ThingMagicWall   = "thing.magic_wall"
	ThingPortal      = "thing.portal"
	PortraitSage     = "portrait.sage"
	ConversationSage = "conversation.sage"
	// CounterGift counts the sage's gifts; CounterAfter is cleared after every conversation.
	CounterGift  = "var.gift"
	CounterAfter = "var.after"
	ItemScroll   = "item.scroll"
)

// Study is a level for magic and talk: the sage, a target, a scroll on the floor and a door only
// a Kano spell opens.
const Study = "level.study"

// The study's cells.
var (
	StudyStart  = content.Cell{2, 4}
	SageCell    = content.Cell{1, 1}
	TargetCell  = content.Cell{2, 9}
	StudyScroll = content.Cell{3, 2}
	KanoDoor    = content.Cell{4, 6}
	KanoFrom    = content.Cell{3, 6}
)

// iconCount is the number of gesture icons, frame 0 of the gesture sprite being the empty panel.
const iconCount = content.GestureIcons

// magicArt draws the gesture icons, the spells' icons and missiles, the magic wall and the portal,
// the scroll's picture and the sage's portrait.
func magicArt(p *content.Pack) {
	gestures := []*canvas{gesturePanel(-1)}
	for i := range iconCount {
		gestures = append(gestures, gesturePanel(i))
	}
	bank(p, spriteGestures, gestures, nil)
	var icons []*canvas
	for _, sp := range devSpells {
		icons = append(icons, spellIcon(sp.color))
	}
	bank(p, spriteSpellIcons, icons, nil)
	bank(p, spriteBolt, bolt(red+5, false), nil)
	bank(p, spriteArrowBolt, bolt(gold+6, true), nil)
	bank(p, spriteCloud, bolt(gray+12, false), nil)
	bank(p, spriteMagicWall, magicWall(), &content.Offset{16, 28})
	b := bank(p, spritePortal, portal(), &content.Offset{16, 40})
	b.Animations = map[string]content.Animation{"idle": {Frames: []int{0, 1, 2, 1}, MS: content.Fixed(500)}}
	bank(p, spriteScroll, []*canvas{scrollPicture()}, nil)
	bank(p, spriteSagePic, []*canvas{sagePortrait()}, nil)
	bank(p, spritePointer, pointers(), nil)
	p.Pointers = &content.Pointers{ID: "pointers", Arrow: content.Image{Sprite: spritePointer},
		Talk: &content.Image{Sprite: spritePointer, Frame: 1}, Wait: &content.Image{Sprite: spritePointer, Frame: 2}}
	p.Portraits[PortraitSage] = &content.Portrait{ID: PortraitSage, Image: content.Image{Sprite: spriteSagePic}}
}

// gesturePanel draws gesture icon i, a ring with spokes turned by its number, or with i < 0 the
// empty panel.
func gesturePanel(i int) *canvas {
	c := newCanvas(32, 32, stone+2)
	c.outline(func(x, y float64) bool { return x >= 1 && x < 31 && y >= 1 && y < 31 }, stone+5)
	if i < 0 {
		return c
	}
	c.outline(ellipse(16, 16, 11, 11), gold+5)
	a := float64(i) * math.Pi / 6
	for k := range 1 + i%3 {
		b := a + float64(k)*2*math.Pi/3
		c.line(16, 16, 16+int(math.Round(10*math.Cos(b))), 16+int(math.Round(10*math.Sin(b))), gold+6)
	}
	c.fill(ellipse(16, 16, 2, 2), solid(red+5))
	return c
}

// spellIcon draws a spell's 16×16 icon, a star of its color.
func spellIcon(color byte) *canvas {
	c := newCanvas(16, 16, clear)
	c.fill(diamond(8, 8, 14), solid(color))
	c.fill(ellipse(8, 8, 2, 2), solid(gray+15))
	return c
}

// bolt draws a missile: frame 0 flying, frames 1–4 bursting; directional ones have a frame for
// each direction instead of the first.
func bolt(color byte, directional bool) []*canvas {
	var out []*canvas
	if directional {
		for d := range 8 {
			c := newCanvas(16, 16, clear)
			dir := directions[d]
			c.line(8-int(dir[0]*6), 8-int(dir[1]*6), 8+int(dir[0]*6), 8+int(dir[1]*6), color)
			c.fill(ellipse(8+dir[0]*5, 8+dir[1]*5, 2, 2), solid(red+6))
			out = append(out, c)
		}
		return out
	}
	c := newCanvas(16, 16, clear)
	c.fill(ellipse(8, 8, 4, 4), solid(color))
	out = append(out, c)
	for f := 1; f <= 4; f++ {
		c := newCanvas(16, 16, clear)
		r := float64(2 + 2*f)
		c.outline(ellipse(8, 8, r, r), color)
		out = append(out, c)
	}
	return out
}

// magicWall draws the conjured wall in three strengths of glow.
func magicWall() []*canvas {
	var out []*canvas
	for f := range 3 {
		c := newCanvas(32, 32, clear)
		c.fill(func(x, y float64) bool { return x >= 6 && x < 26 && y >= 4 && y < 28 }, solid(byte(blue+3+2*f)))
		c.outline(func(x, y float64) bool { return x >= 6 && x < 26 && y >= 4 && y < 28 }, blue+1)
		out = append(out, c)
	}
	return out
}

// portal draws a spinning gateway in three frames.
func portal() []*canvas {
	var out []*canvas
	for f := range 3 {
		c := newCanvas(32, 44, clear)
		c.fill(ellipse(16, 22, 10, 18), solid(purple+2))
		c.outline(ellipse(16, 22, 10, 18), purple+6)
		a := float64(f) * math.Pi / 3
		c.line(16, 22, 16+int(8*math.Cos(a)), 22+int(14*math.Sin(a)), gray+15)
		out = append(out, c)
	}
	return out
}

// pointers draws the pointer's frames, 16 pixels square, each pointing from its top-left: an
// arrow, a speech bubble and an hourglass.
func pointers() []*canvas {
	arrow := newCanvas(16, 16, clear)
	arrow.fill(func(x, y float64) bool { return x >= 0 && y >= 0 && x+y < 11 && y < 2*x+3 && x < 2*y+3 }, solid(gray+15))
	arrow.line(5, 5, 12, 12, gray+15)
	bubble := newCanvas(16, 16, clear)
	bubble.fill(ellipse(9, 6, 6.5, 5), solid(gray+15))
	bubble.outline(ellipse(9, 6, 6.5, 5), gray+4)
	bubble.line(5, 10, 2, 14, gray+15)
	wait := newCanvas(16, 16, clear)
	wait.fill(func(x, y float64) bool { return y >= 1 && y < 15 && abs(int(x)-8) <= abs(int(y)-8) }, solid(gold+6))
	return []*canvas{arrow, bubble, wait}
}

// scrollPicture draws a scroll's picture: parchment with three marks.
func scrollPicture() *canvas {
	c := newCanvas(112, 54, tan+6)
	c.outline(func(x, y float64) bool { return x >= 0 && x < 112 && y >= 0 && y < 54 }, tan+2)
	for i := range 3 {
		c.outline(ellipse(float64(28+28*i), 27, 9, 9), wood+2)
	}
	return c
}

// sagePortrait draws the sage's face: a gray beard and a hood.
func sagePortrait() *canvas {
	c := newCanvas(48, 48, clear)
	c.fill(ellipse(24, 50, 22, 12), solid(purple+2))
	c.fill(ellipse(24, 22, 11, 14), solid(tan+5))
	c.fill(func(x, y float64) bool { return ellipse(24, 30, 10, 9)(x, y) && y > 26 }, solid(gray+12))
	c.fill(func(x, y float64) bool { return ellipse(24, 16, 14, 12)(x, y) && y < 14 }, solid(purple+3))
	for _, x := range []int{19, 28} {
		c.rect(x, 20, x+1, 21, black)
	}
	return c
}

// devSpells are the dev pack's spells, in the order of their icons.
var devSpells = []struct {
	id, name, gesture string
	school, cost      int
	power             content.Dice
	bonus             int
	color             byte
}{
	{SpellFlameArrow, "Flame Arrow", "DJ", 0, 5, content.Dice{Count: 1, Sides: 8}, 4, red + 5},
	{SpellFireball, "Fireball", "IBF", 0, 15, content.Dice{Count: 3, Sides: 10}, 6, red + 7},
	{SpellKano, "Kano", "DB", 1, 5, content.Dice{}, 0, gold + 5},
	{SpellSwiftness, "Swiftness", "FACF", 1, 20, content.Dice{Count: 1, Sides: 20}, 4, green + 5},
	{SpellMagicWall, "Magic Wall", "BIKG", 1, 15, content.Dice{Count: 2, Sides: 6}, 4, blue + 5},
	{SpellGateway, "Gateway", "ELADJKJC", 1, 20, content.Dice{}, 0, purple + 5},
	{SpellHeal, "Heal", "FHBEHL", 3, 10, content.Dice{}, 0, green + 7},
}

// The keywords of the sage's conversation, by word.
var devKeywords = []struct{ word, answer string }{
	{"fireball", "How is a fireball cast?"},
	{"gift", "Have you a gift for me?"},
	{"magic", "Tell me of magic."},
	{"name", "What is your name?"},
	{"sage", "Are you truly wise?"},
}

func keywordID(word string) string { return "keyword." + word }

// magicRecords adds the spells, the magic record, the keywords, the sage and his conversation,
// the target, the magic wall, the portal and the scroll, with their texts and messages.
func magicRecords(p *content.Pack) {
	s := p.Texts[textID].Strings
	m := &content.Magic{ID: "magic", Panel: &content.Image{Sprite: spriteGestures}}
	for i := range iconCount {
		m.Icons = append(m.Icons, content.Image{Sprite: spriteGestures, Frame: i + 1})
	}
	for i, name := range []string{"Wizardry", "Sorcery", "Enchantment", "Healing"} {
		key := "school." + string(rune('0'+i))
		s[key] = name
		m.Schools = append(m.Schools, key)
	}
	p.Magic = m

	str := func(v string) *string { return &v }
	for i, d := range devSpells {
		sp := &content.Spell{ID: d.id, Name: d.id + ".name", School: d.school, Cost: d.cost, Power: d.power,
			PowerBonus: d.bonus, Gesture: d.gesture, Icon: &content.Image{Sprite: spriteSpellIcons, Frame: i}}
		s[sp.Name] = d.name
		switch d.id {
		case SpellFlameArrow:
			sp.Effects = []content.Action{{Missile: &content.MissileSpell{Hurts: content.HurtsFire, Sprite: spriteArrowBolt, Directional: true, Cloud: spriteCloud}}}
		case SpellFireball:
			sp.Effects = []content.Action{{Missile: &content.MissileSpell{Hurts: content.HurtsFire, Sprite: spriteBolt}}}
			sp.Effect, s[sp.ID+".effect"] = sp.ID+".effect", "Hurls a ball of fire."
		case SpellKano:
			sp.Effects = []content.Action{{Missile: &content.MissileSpell{Opens: true, Sprite: spriteBolt}}}
		case SpellSwiftness:
			sp.Effects = []content.Action{{Timed: str(content.TimedHaste)}}
		case SpellMagicWall:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{FloorAhead: 16, Else: content.MsgNoRoomForWall}
			sp.Effects = []content.Action{{Conjure: &content.Conjure{Thing: ThingMagicWall, Distance: 16, LookPerSkill: 3}}}
		case SpellGateway:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{FloorAhead: 32, Else: content.MsgNoRoomForPortal}
			sp.Effects = []content.Action{{Portal: &content.Portal{Thing: ThingPortal, Distance: 32, To: content.Destination{Level: Hall, Cell: HallStart}}}}
		case SpellHeal:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{Hurt: true}
			sp.Effects = []content.Action{{Heal: &content.Dice{Bonus: 3000}}}
		}
		p.Spells[sp.ID] = sp
	}

	k := &content.Keywords{ID: "keywords"}
	for _, kw := range devKeywords {
		id := keywordID(kw.word)
		s[id+".word"], s[id+".answer"] = kw.word, kw.answer
		k.Keywords = append(k.Keywords, content.Keyword{ID: id, Word: id + ".word", Answer: id + ".answer"})
	}
	p.Keywords = k

	for key, text := range map[string]string{
		"talk.sage.hello":    "Welcome to my study, @. I am the *sage who keeps it. Ask me my *name, or of *magic.",
		"talk.sage.again":    "Back again, @? There is always more *magic to learn.",
		"talk.sage.name":     "My name does not matter. What I know of *magic does.",
		"talk.sage.magic":    "Watch my hands. These are the signs of the *fireball.",
		"talk.sage.fireball": "Make the three signs in order, and the fire will answer you.",
		"talk.sage.sage":     "Wise enough to keep a *gift for a careful student.",
		"talk.sage.gift":     "Take this potion. You will need it.",
		"talk.sage.nomore":   "I have given you all I had to give.",
	} {
		s[key] = text
	}
	say := func(key string) content.Action { return content.Action{Say: str(key)} }
	offer := func(word string) content.Action { return content.Action{Offer: str(keywordID(word))} }
	asked := func(words ...string) content.Condition {
		var ids []string
		for _, w := range words {
			ids = append(ids, keywordID(w))
		}
		return content.Condition{Asked: ids}
	}
	zero, gift := 0, 8
	p.Conversations[ConversationSage] = &content.Conversation{ID: ConversationSage, Name: "Sage", Clauses: []content.Clause{
		{If: content.Condition{FirstMeeting: true}, Do: []content.Action{say("talk.sage.hello"), offer("name"), offer("magic")}},
		{If: content.Condition{Greeting: true}, Do: []content.Action{say("talk.sage.again"), offer("magic"), offer("gift")}},
		{If: asked("name"), Do: []content.Action{say("talk.sage.name"), offer("magic")}},
		{If: asked("magic"), Do: []content.Action{say("talk.sage.magic"), {Teach: str("IBF")}, offer("fireball")}},
		{If: asked("fireball"), Do: []content.Action{say("talk.sage.fireball")}},
		{If: asked("sage"), Do: []content.Action{say("talk.sage.sage"), offer("gift")}},
		{If: asked("gift"), Do: []content.Action{{Jump: &gift}}},
		// A pass that reaches here asks about something the sage has nothing to say about.
		{If: content.Condition{Never: true}},
		{If: content.Condition{Counter: &content.CounterTest{Counter: CounterGift, Equals: &zero}},
			Do: []content.Action{say("talk.sage.gift"), {Give: &content.Object{Item: ItemPotion}}, {SetCounter: &content.SetCounter{Counter: CounterGift, Value: 1}}}},
		{If: content.Condition{Always: true}, Do: []content.Action{say("talk.sage.nomore")}},
	}}
	p.Manifest.Counters = &content.Counters{AfterConversation: CounterAfter}

	p.Things[ThingSage] = &content.Thing{ID: ThingSage, Kind: content.ThingCreature, Sprite: "sprite.sage",
		Facing: 3, IntervalMS: 330, Weight: 100, HitPoints: content.Dice{Bonus: 30},
		Behaviors:    []string{content.BehaviorNeverHostile, content.BehaviorStationary, content.BehaviorTalks},
		Conversation: ConversationSage, TalkPortrait: PortraitSage}
	p.Things[ThingTarget] = &content.Thing{ID: ThingTarget, Kind: content.ThingCreature, Sprite: "sprite.brute",
		Facing: 6, IntervalMS: 330, Weight: 100, HitPoints: content.Dice{Bonus: 6},
		Behaviors: []string{content.BehaviorStationary, content.BehaviorLeavesCorpse},
		Stats:     map[string]int{"accuracy": 1, "bonus": 0, "resistance": 0}}
	p.Things[ThingMagicWall] = &content.Thing{ID: ThingMagicWall, Kind: content.ThingConjured, Sprite: spriteMagicWall,
		IntervalMS: 1000, Weight: 100, Footprint: 16}
	p.Things[ThingPortal] = &content.Thing{ID: ThingPortal, Kind: content.ThingPortal, Sprite: spritePortal,
		IntervalMS: 500, Footprint: 16}

	scroll := p.Items[ItemScroll]
	scroll.Teaches, scroll.Picture = "IBF", &content.Image{Sprite: spriteScroll}

	for key, text := range map[string]string{
		content.MsgCannotCast: "Your hands are wet; you cannot cast.", content.MsgNotMemorized: "No {school} spells prepared.",
		content.MsgOnlySpell: "Only one {school} spell prepared.", content.MsgNoMagic: "Not enough magic.",
		content.MsgForgot: "In learning this spell,", content.MsgForgotMore: "you forgot another.",
		content.MsgNoRoomForWall: "There is no room for a wall.", content.MsgWallFailed: "The wall would not form.",
		content.MsgNoRoomForPortal: "There is no room for a gateway.", content.MsgHaste: "You are moving faster.",
		content.MsgFullHealth: "You are not hurt.", content.MsgDoorResists: "The door resists the spell.",
		content.MsgMissileWall: "The spell strikes the wall.", content.MsgHealed: "You are healed {damage}.",
		content.MsgNoQuiver: "You wear no quiver.", content.MsgNoArrows: "Your quiver is empty.",
		content.MsgNoRoomToShoot: "No room to draw the bow.", content.MsgTooCloseToWall: "Too near the wall to throw.",
		content.MsgCannotThrow: "You cannot throw in this water.", content.MsgCannotAct: "You cannot do that in this water.",
	} {
		s["msg."+key] = text
		p.Messages.Messages[key] = "msg." + key
	}
}

// study lays out the study.
func study(p *content.Pack) {
	l := &content.Level{ID: Study, Name: levelName(p, Study, "The Study"), Floors: floorSet, Walls: wallSet, FloorTypes: floorTypes()}
	grid(l, []string{
		"############",
		"#..........#",
		"#..........#",
		"#..........#",
		"######.#####",
		"#..........#",
		"############",
	})
	l.Features = []*content.Feature{{ID: "door.kano", Kind: content.FeatureDoor, Cell: KanoDoor, Look: lookDoor,
		State: content.DoorClosed, OpenedBySpell: true}}
	l.Things = []*content.PlacedThing{
		{ID: "sage", Thing: ThingSage, Cell: SageCell},
		{ID: "target", Thing: ThingTarget, Cell: TargetCell},
	}
	l.Objects = []*content.FloorObject{{Cell: StudyScroll, Object: content.Object{Item: ItemScroll}}}
	p.Levels[l.ID] = l
}
