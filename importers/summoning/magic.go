package summoning

import (
	"fmt"
	"maps"
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Spells, the magic record, scrolls and objects that cast (spells.md, items.md).

// Resources the spells draw with (spells.md, Missiles and Effects).
const (
	resFireball    = 41
	resPoisonGas   = 65
	resKano        = 73
	resZapAway     = 61
	resAlteration  = 66
	resFireCloud   = 57
	resLightCloud  = 58
	resMagicWall   = 59
	resGateway     = 60
	resGestures    = 54
	resSchoolIcons = 0x45
	// heldFlameArrow and heldLightning are offsets from the held-item base.
	heldFlameArrow = 0x2E
	heldLightning  = 0x2F

	cellUnits      = 32
	magicWallTicks = 18
	gatewayTicks   = 9
	// npcTransformed is the NPC bank whose look Shape Change sets a counter for.
	npcTransformed = 0x4D
	// templateArrow is what spell 9 fires; the staff halves and staff Mending joins; the bottle
	// Liquify fills and what it fills it with.
	templateArrow     = 0x99
	templateStaffTop  = 0xD8
	templateStaffEnd  = 0xD9
	templateStaff     = 0xDA
	templateJera      = 0x81
	npcAlterationBane = 0x31
	npcMending        = 0x79
)

// Interaction variables the engine itself uses (dialogue.md), and the counter Mending sets.
const (
	varTransformed       = 14
	varUseResult         = 19
	varEnding            = 32
	varAfterConversation = 97
	varMagnify           = 98
	varTranscript        = 99
)

// Where the Gateway leads (spells.md, Effects).
var gatewayTo = content.Destination{Level: levelID(30), Cell: content.Cell{60, 26}}

// transmuteTable is what Liquify makes from the object in the other hand (spells.md, Effects).
var transmuteTable = map[int]int{0xAB: 0x80, 0xAC: 0x82, 0xAD: 0x83, 0xAE: 0xDD, 0x0B: 0x82}

// scrollSpells are the spells whose gesture icons each scroll 1..25 teaches; −1 teaches none
// (spells.md, Scrolls).
var scrollSpells = [25]int{38, 39, -1, 3, 0, 11, 10, 4, 1, 2, 5, 30, 31, 12, 13, 14, 15, 16, 23, 22, 21, 20, 32, 36, 37}

// spellNames are the spells' names by number (spells.md, Effects); a number without one does
// nothing and is not imported.
var spellNames = map[int]string{
	0: "Flame Arrow", 1: "Four-way Fireball", 2: "Bounce Fireball", 3: "Poison Gas", 4: "Fireball",
	5: "Lightning", 6: "Battle Rage", 9: "Arrow", 10: "Sight", 11: "Kano", 12: "Magic Wall",
	13: "Swiftness", 14: "Zap Away", 15: "Teleport", 16: "Invisibility", 17: "Dispel", 18: "Gateway",
	19: "Mending", 20: "Freeze", 21: "Lightning Shield", 22: "Fire Shield", 23: "Weapon Skill",
	24: "Magic Skill", 25: "Shape Change", 26: "Alteration", 30: "Liquify", 31: "Cure Poison",
	32: "Restore", 33: "Heal",
}

// spellID returns the id of spell n, and false for a number that is no spell.
func spellID(n int) (string, bool) {
	name, ok := spellNames[n]
	if !ok {
		return "", false
	}
	return "spell." + slug(name), true
}

// Things spells make.
const (
	magicWallID = "thing.magic_wall"
	gatewayID   = "thing.gateway"
)

// gesture returns spell n's gesture from INIT.
func (im *importer) gesture(n int) string {
	if n < 0 || n >= SpellSchools*SpellsPerSchool {
		return ""
	}
	return im.init.SpellGestures[n/SpellsPerSchool][n%SpellsPerSchool]
}

// learnable reports whether gesture is a gesture the player can enter: one to nine of A–L.
func learnable(gesture string) bool {
	if gesture == "" {
		return false
	}
	for _, c := range gesture {
		if c < 'A' || c >= 'A'+content.GestureIcons {
			return false
		}
	}
	return true
}

// icons returns the letters of gesture that are icons, which is what showing it teaches.
func icons(gesture string) string {
	var b []byte
	for i := 0; i < len(gesture); i++ {
		if c := gesture[i]; c >= 'A' && c < 'A'+content.GestureIcons {
			b = append(b, c)
		}
	}
	return string(b)
}

// spells adds the spells, the things they make and the magic record.
func (im *importer) spells() error {
	sprite := func(res int) (string, error) {
		sid, _, err := im.rawSprite(res)
		return sid, err
	}
	sprites := map[int]string{}
	held := int(im.init.Bases.HeldItemAnimation)
	for _, res := range []int{resFireball, resPoisonGas, resKano, resZapAway, resAlteration, resFireCloud, resLightCloud, held + heldFlameArrow, held + heldLightning} {
		sid, err := sprite(res)
		if err != nil {
			im.report.addf("spell sprite %d: %v; spells using it left out", res, err)
			continue
		}
		sprites[res] = sid
	}
	im.spellThings()
	strs := im.strings
	for _, n := range sortedInts(spellNames) {
		id, _ := spellID(n)
		missing := ""
		item := func(t int) string {
			if t < 0 || t >= len(im.itemIDs) {
				missing = fmt.Sprintf("object template %02Xh", t)
				return ""
			}
			return im.itemIDs[t]
		}
		thing := func(id string) string {
			if _, ok := im.pack.Things[id]; !ok {
				missing = id
			}
			return id
		}
		school, slot := n/SpellsPerSchool, n%SpellsPerSchool
		missile := func(res int, m content.MissileSpell) []content.Action {
			sid, ok := sprites[res]
			if !ok {
				return nil
			}
			m.Sprite = sid
			return []content.Action{{Missile: &m}}
		}
		str := func(s string) *string { return &s }
		num := func(n int) *int { return &n }
		sp := &content.Spell{
			ID: id, Name: id + ".name", School: school, Cost: int(im.init.SpellCosts[school][slot]),
			PowerBonus: int(im.init.SpellPower[school][slot].BonusSides),
		}
		if pw := im.init.SpellPower[school][slot]; pw.Dice > 0 && pw.Sides > 0 {
			sp.Power = content.Dice{Count: int(pw.Dice), Sides: int(pw.Sides)}
		}
		if g := im.gesture(n); learnable(g) {
			sp.Gesture = g
		}
		var effects []content.Action
		switch n {
		case 0:
			effects = missile(held+heldFlameArrow, content.MissileSpell{Hurts: content.HurtsFire, Directional: true, Cloud: sprites[resFireCloud]})
		case 1:
			effects = missile(resFireball, content.MissileSpell{Hurts: content.HurtsFire, Around: true})
		case 2:
			effects = missile(resFireball, content.MissileSpell{Hurts: content.HurtsFire, Bounces: true})
		case 3:
			effects = missile(resPoisonGas, content.MissileSpell{Hurts: content.HurtsPoison})
		case 4:
			effects = missile(resFireball, content.MissileSpell{Hurts: content.HurtsFire})
		case 5:
			effects = missile(held+heldLightning, content.MissileSpell{Hurts: content.HurtsLightning, Directional: true, Cloud: sprites[resLightCloud]})
		case 6:
			effects = []content.Action{{Timed: str(content.TimedBattleRage)}}
		case 9:
			effects = []content.Action{{FireItem: str(item(templateArrow))}}
		case 10:
			effects = []content.Action{{Timed: str(content.TimedSight)}}
		case 11:
			effects = missile(resKano, content.MissileSpell{Opens: true})
		case 12:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{FloorAhead: cellUnits / 2, Else: content.MsgNoRoomForWall}
			effects = []content.Action{{Conjure: &content.Conjure{Thing: thing(magicWallID), Distance: cellUnits / 2, LookPerSkill: 3}}}
		case 13:
			effects = []content.Action{{Timed: str(content.TimedHaste)}}
		case 14:
			effects = missile(resZapAway, content.MissileSpell{Scatters: true})
		case 15:
			sp.Needs = &content.SpellNeeds{Mark: "2"}
			effects = []content.Action{{TeleportToMark: str("2")}}
		case 16:
			sp.Practice = 3
			effects = []content.Action{{Timed: str(content.TimedInvisibility)}}
		case 17:
			sp.Needs = &content.SpellNeeds{SpellNear: 96}
			effects = []content.Action{{Dispel: num(96)}}
		case 18:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{FloorAhead: cellUnits, Else: content.MsgNoRoomForPortal}
			if im.levelExists(gatewayTo.Level) {
				effects = []content.Action{{Portal: &content.Portal{Thing: thing(gatewayID), Distance: cellUnits, To: gatewayTo}}}
			}
		case 19:
			halves := []string{item(templateStaffTop), item(templateStaffEnd)}
			sp.Practice, sp.Needs = 20, &content.SpellNeeds{Holding: halves}
			effects = []content.Action{
				{Join: &content.Join{Items: halves, Into: item(templateStaff)}},
				{Effect: str(effectID(viewMending))},
				{SetCounter: &content.SetCounter{Counter: varID(varEnding), Value: 1}},
			}
			if conv, err := im.creature(npcMending); err == nil {
				effects = append(effects, content.Action{Converse: &conv})
			}
		case 20:
			sp.Practice = 1
			effects = []content.Action{{Freeze: num(10)}}
		case 21:
			sp.Practice = 1
			effects = []content.Action{{Timed: str(content.TimedLightningShield)}}
		case 22:
			sp.Practice = 1
			effects = []content.Action{{Timed: str(content.TimedFireShield)}}
		case 23:
			effects = []content.Action{{Timed: str(content.TimedWeaponSkills)}}
		case 24:
			effects = []content.Action{{Timed: str(content.TimedMagicSkills)}}
		case 25:
			sp.Needs = &content.SpellNeeds{CreatureNear: 128}
			effects = []content.Action{{ShapeChange: num(128)}}
		case 26:
			if bane, err := im.creature(npcAlterationBane); err == nil {
				effects = missile(resAlteration, content.MissileSpell{Bane: bane, BaneDamage: 3000})
			}
		case 30:
			table := map[string]string{}
			for from, to := range transmuteTable {
				table[item(from)] = item(to)
			}
			sp.Practice, sp.Needs = 1, &content.SpellNeeds{Holding: []string{item(templateEmptyBottle)}, Else: content.MsgNoBottle}
			effects = []content.Action{{Transmute: &content.Transmute{Holder: item(templateEmptyBottle), Table: table, Otherwise: item(templateJera)}}}
		case 31:
			sp.Practice, sp.Needs = 3, &content.SpellNeeds{Poisoned: true}
			effects = []content.Action{{CurePoison: true}, {Sparkle: str(sparkleID("cure"))}}
		case 32:
			sp.Needs = &content.SpellNeeds{Fatigued: true}
			effects = []content.Action{{Restore: true}, {Sparkle: str(sparkleID("restore"))}}
		case 33:
			sp.Practice, sp.Needs = 5, &content.SpellNeeds{Hurt: true}
			effects = []content.Action{{Heal: &content.Dice{Bonus: 3000}}, {Sparkle: str(sparkleID("heal"))}}
		}
		if effects == nil || missing != "" {
			im.report.addf("spell %d: what it does could not be imported (%s); left out", n, missing)
			continue
		}
		sp.Effects = effects
		if sid, b, err := im.rawSprite(resSchoolIcons + school); err == nil && slot < len(b.Frames) {
			sp.Icon = &content.Image{Sprite: sid, Frame: slot}
		}
		strs[sp.Name] = spellNames[n]
		if text, ok := spellEffects[n]; ok {
			sp.Effect = id + ".effect"
			strs[sp.Effect] = text
		}
		im.pack.Spells[id] = sp
	}
	return im.magicRecord()
}

// spellThings adds the magic wall and the gateway.
func (im *importer) spellThings() {
	if sid, _, err := im.rawSprite(resMagicWall); err == nil {
		im.anchor(sid, blockAnchor)
		im.pack.Things[magicWallID] = &content.Thing{
			ID: magicWallID, Kind: content.ThingConjured, Sprite: sid, IntervalMS: ms(magicWallTicks),
			Weight: creatureWeight, Footprint: creatureFootprint, Box: boxOf(creatureBox),
		}
	} else {
		im.report.addf("magic wall: %v", err)
	}
	if sid, b, err := im.rawSprite(resGateway); err == nil {
		im.anchor(sid, content.Offset{b.Width / 2, b.Height - 4})
		every := content.Fixed(ms(gatewayTicks))
		im.pack.Things[gatewayID] = &content.Thing{
			ID: gatewayID, Kind: content.ThingPortal, Sprite: sid, IntervalMS: ms(gatewayTicks), Footprint: creatureFootprint, Box: boxOf(portalBox),
			Animations: im.fitAnimations(gatewayID, map[string]content.Animation{"idle": {Frames: []int{0, 1, 2, 1}, MS: every}}, len(b.Frames)),
		}
	} else {
		im.report.addf("gateway: %v", err)
	}
}

// magicRecord adds the gesture icons, the empty gesture panel and the schools' names.
func (im *importer) magicRecord() error {
	sid, b, err := im.rawSprite(resGestures)
	if err != nil {
		im.report.addf("gesture icons: %v; no spell UI art", err)
		return nil
	}
	if len(b.Frames) < content.GestureIcons+1 {
		im.report.addf("gesture icons: %d frames; no spell UI art", len(b.Frames))
		return nil
	}
	m := &content.Magic{ID: "magic", Panel: &content.Image{Sprite: sid, Frame: 0}}
	for i := range content.GestureIcons {
		m.Icons = append(m.Icons, content.Image{Sprite: sid, Frame: i + 1})
	}
	for s := range SpellSchools {
		m.Schools = append(m.Schools, msgID(msgFirstSchool+s))
	}
	m.Holds = gestureHolds
	im.pack.Magic = m
	return nil
}

// gestureHolds are the icons after which the gesture panel shows empty before each icon
// (spells.md, The gesture playing).
var gestureHolds = map[string]string{
	"A": "BDEIJK", "B": "AJL", "D": "AL", "E": "AFGHKL", "F": "E", "G": "BDEI",
	"H": "L", "I": "AGJKL", "J": "ABI", "K": "EL", "L": "BDEHIK",
}

// msgFirstSchool is the TEXT record of school 0's name.
const msgFirstSchool = 0x2A

func varID(n int) string { return fmt.Sprintf("var.%d", n) }

// scroll gives a scroll item its picture and the icons reading it teaches (spells.md, Scrolls).
func (im *importer) scroll(t int, tm *ObjectTemplate, it *content.Item) {
	s := tm.Scroll()
	if s == 0 {
		return
	}
	if s > len(scrollSpells) {
		im.report.addf("object template %02Xh: scroll %d; teaches nothing", t, s)
		return
	}
	if n := scrollSpells[s-1]; n >= 0 {
		it.Teaches = icons(im.gesture(n))
	}
	sid, err := im.stripSprite(int(im.init.Bases.Scroll)+s-1, "sprite.scroll")
	if err != nil {
		im.report.addf("object template %02Xh: scroll picture: %v", t, err)
		return
	}
	it.Picture = &content.Image{Sprite: sid}
}

// casting gives item it what its power casts or the conversation it starts (items.md, Engine
// model). Conversations name NPC records, whose templates are added with the conversations.
func (im *importer) casting(tm *ObjectTemplate, it *content.Item) {
	p, _ := tm.Power()
	cast := func(n int, power content.Dice, msg string) {
		if id, ok := spellID(n); ok {
			it.Casts = &content.Cast{Spell: id, Power: power, Message: msg}
		}
	}
	switch {
	case p == 0x13:
		cast(3, content.Dice{Bonus: 20}, "")
	case p == 0x19:
		cast(4, dice(tm.DamageCode(1)), "")
	case p == 0x1E:
		cast(11, content.Dice{}, "")
	case p == 0x29:
		cast(14, content.Dice{Bonus: 20}, "")
	case p == 0x4E:
		cast(10, content.Dice{Bonus: 10}, content.MsgSight)
	case p == 0x5C:
		cast(5, dice(tm.DamageCode(1)), "")
	case p == 0x5D:
		cast(6, content.Dice{}, "")
	case p == 0x51:
		it.Conversation, it.UseResult = npcID(0x30), true
	case p == 0x52:
		it.Conversation = npcID(0x46)
	case p >= 0x53 && p <= 0x5A:
		it.Conversation = npcID(p - 0x0B)
	}
	if it.Conversation != "" {
		im.talkers[it.Conversation] = true
	}
}

func npcID(n int) string { return fmt.Sprintf("thing.npc.%d", n) }

func sortedInts[V any](m map[int]V) []int { return slices.Sorted(maps.Keys(m)) }

func sortedKeys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// The pointer resource and its frames (dialogue.md, The pointer).
const (
	resPointer   = 2
	pointerArrow = 0
	pointerTalk  = 9
	pointerWait  = 10
)

// pointers adds the pointer's images.
func (im *importer) pointers() {
	sid, b, err := im.rawSprite(resPointer)
	if err != nil {
		im.report.addf("pointers: %v; the system pointer is used", err)
		return
	}
	pt := &content.Pointers{ID: "pointers", Arrow: content.Image{Sprite: sid, Frame: pointerArrow}}
	if len(b.Frames) > pointerTalk {
		pt.Talk = &content.Image{Sprite: sid, Frame: pointerTalk}
	}
	if len(b.Frames) > pointerWait {
		pt.Wait = &content.Image{Sprite: sid, Frame: pointerWait}
	}
	im.pack.Pointers = pt
}
