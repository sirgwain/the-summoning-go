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
	templateSack           = 0x04
	templateBagOfLightness = 0x05
	// sackFillPer is how many objects in the Sack make each step of its fill pictures.
	sackFillPer      = 3
	templateFirstGem = 0xC0
	categoryRandom   = 6
)

// sealingSpheres are the templates that open sealed door stages 5, 6 and 7 (rules.md, Doors).
var sealingSpheres = [3]int{0xC9, 0xCB, 0xCD}

// categoryTags are the tags of the categories an item can be recognized by (items.md, Import).
var categoryTags = map[int]string{1: "body_armor", 4: "quiver", 7: "arrows", 8: "bottle"}

// engineMessages are the TEXT records the engine's own messages use, and the placeholders their
// format codes (%s, %d, %q, %N) become, in order.
var engineMessages = map[string]struct {
	text int
	args []string
}{
	content.MsgLeverStuck:    {0x86, nil},
	content.MsgSlotWrongItem: {0x8F, []string{"item"}},
	content.MsgSlotStuck:     {0x87, []string{"item"}},
	content.MsgNoDropHere:    {0x72, nil},
	content.MsgNoRoom:        {0x7F, nil},
	content.MsgNothingToTake: {0x80, nil},
	content.MsgIntoQuiver:    {0x99, []string{"item"}},
	content.MsgPaused:        {0x79, nil},
	content.MsgNotACreature:  {0x5A, nil},
	content.MsgCannotHurt:    {0x5D, nil},
	content.MsgMiss:          {0x5B, []string{"score"}},
	content.MsgHit:           {0x5C, []string{"score", "damage"}},
	content.MsgCreatureHits:  {0x7D, []string{"score", "damage"}},
	content.MsgCreatureMiss:  {0x7E, []string{"score"}},
	content.MsgWeaponBreaks:  {0x8A, []string{"item"}},
	content.MsgVanishes:      {0x32, []string{"item"}},
	content.MsgSkillUp:       {0x44, []string{"skill"}},
	content.MsgLevelUp:       {0xA5, nil},

	content.MsgFullHealth:            {0x3E, nil},
	content.MsgFullMagic:             {0x42, nil},
	content.MsgNotPoisoned:           {0x40, nil},
	content.MsgNotPoisonedOrConfused: {0x41, nil},
	content.MsgNotFatigued:           {0x3F, nil},
	content.MsgNoDestination:         {0x43, nil},
	content.MsgNoCreatures:           {0x88, nil},
	content.MsgAlreadyHave:           {0x21, nil},
	content.MsgHealed:                {0x4A, []string{"damage"}},
	content.MsgProtected:             {0x3C, nil},
	content.MsgBoostExtended:         {0x33, nil},
	content.MsgPoisoned:              {0x59, nil},
	content.MsgSlowed:                {0x60, nil},
	content.MsgLostWeaponSkills:      {0x64, nil},
	content.MsgLostMagicSkills:       {0x67, nil},
	content.MsgConfused:              {0x36, nil},
	content.MsgSexChanged:            {0x35, nil},

	content.MsgCannotCast:           {0x71, nil},
	content.MsgNotMemorized:         {0x46, []string{"school"}},
	content.MsgOnlySpell:            {0x47, []string{"school"}},
	content.MsgNoMagic:              {0x29, nil},
	content.MsgForgot:               {0x68, nil},
	content.MsgForgotMore:           {0x69, nil},
	content.MsgNoRoomForWall:        {0x6A, nil},
	content.MsgWallFailed:           {0x6B, nil},
	content.MsgNoRoomForPortal:      {0x6F, nil},
	content.MsgNoSpellNear:          {0x61, nil},
	content.MsgNothingToMend:        {0x6D, nil},
	content.MsgNoCreatureNear:       {0x89, nil},
	content.MsgNoBottle:             {0x6C, nil},
	content.MsgFilled:               {0x6E, []string{"item"}},
	content.MsgDoorResists:          {0x34, nil},
	content.MsgMissileWall:          {0x93, nil},
	content.MsgBattleRage:           {0x5E, nil},
	content.MsgBattleRageExtended:   {0x92, nil},
	content.MsgHaste:                {0x5F, nil},
	content.MsgFireShield:           {0x37, nil},
	content.MsgLightningShield:      {0x38, nil},
	content.MsgFrozen:               {0x39, nil},
	content.MsgSight:                {0x3B, nil},
	content.MsgWeaponSkillsUp:       {0x62, nil},
	content.MsgWeaponSkillsExtended: {0x63, nil},
	content.MsgMagicSkillsUp:        {0x65, nil},
	content.MsgMagicSkillsExtended:  {0x66, nil},
	content.MsgCured:                {0x7A, nil},
	content.MsgKilled:               {0x8B, []string{"name"}},
	content.MsgAbsorbed:             {0x3A, []string{"damage"}},
	content.MsgNoEnding:             {0x8E, nil},
	content.MsgMusicAll:             {0xA6, nil},
	content.MsgMusicEffects:         {0xA7, nil},
	content.MsgMusicSilent:          {0xA8, nil},
}

// placeholders replaces s's format codes, in order, with the named placeholders.
func placeholders(s string, args []string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+1 < len(s) && strings.IndexByte("sdqN", s[i+1]) >= 0 && len(args) > 0 {
			b.WriteString("{" + args[0] + "}")
			args = args[1:]
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
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
		if msg.args != nil {
			t.Strings[id] = placeholders(s, msg.args)
		}
		m.Messages[key] = id
	}
	// The transcript's headings are four records each, joined as lines (dialogue.md).
	for key, recs := range map[string][]int{
		content.MsgTranscriptHeading:   {0x74, 0x75, 0x76, 0x78},
		content.MsgTranscriptContinued: {0x77, 0x75, 0x76, 0x78},
	} {
		var lines []string
		for _, r := range recs {
			if s, ok := t.Strings[msgID(r)]; ok {
				lines = append(lines, s)
			}
		}
		if len(lines) == len(recs) {
			id := "msg." + key
			t.Strings[id] = strings.Join(lines, "\n")
			m.Messages[key] = id
		}
	}
	// Taking an object names it, with no record of its own (items.md, Inventory).
	t.Strings["msg."+content.MsgTaken] = "{item}"
	m.Messages[content.MsgTaken] = "msg." + content.MsgTaken
	im.pack.Messages = m
}

// itemIDs returns the id of every OBJECTS template's item, named after its name in strings, TEXT
// by message id (items.md, Import).
func itemIDs(o *Objects, strings map[string]string) ([]string, error) {
	used := map[string]int{}
	ids := make([]string, len(o.Templates))
	for i := range o.Templates {
		name := msgID(o.NameBase + o.Templates[i].NameIndex())
		s, ok := strings[name]
		if !ok {
			return nil, fmt.Errorf("object template %02Xh: no name text %s", i, name)
		}
		id := "item." + slug(s)
		if id == "item." {
			id = fmt.Sprintf("item.template_%d", i)
		}
		if used[id]++; used[id] > 1 {
			id = fmt.Sprintf("%s.%d", id, used[id])
		}
		ids[i] = id
	}
	return ids, nil
}

// items adds an item for every OBJECTS template, named after its name.
func (im *importer) items() error {
	o := im.objects
	var err error
	if im.itemIDs, err = itemIDs(o, im.strings); err != nil {
		return err
	}
	for i := range o.Templates {
		tm := &o.Templates[i]
		name := msgID(o.NameBase + tm.NameIndex())
		id := im.itemIDs[i]

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
		if i == templateSack && tm.Container() {
			it.Fill = &content.Fill{Per: sackFillPer}
			for k := 1; k <= (tm.Capacity()+sackFillPer-1)/sackFillPer; k++ {
				fi, err := im.image(int(im.init.Bases.SmallObject) + tm.Image() + k)
				if err != nil {
					return fmt.Errorf("sack fill icon %d: %w", k, err)
				}
				it.Fill.Icons = append(it.Fill.Icons, fi)
			}
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
		if i == templateSilverSword {
			it.Tags = append(it.Tags, "silver")
		}
		im.fighting(i, tm, it)
		im.scroll(i, tm, it)
		im.casting(tm, it)
		im.pack.Items[id] = it
	}
	im.spentAndBroken()
	for i, id := range im.itemIDs {
		im.specialItem(i, im.pack.Items[id])
	}
	im.powerTexts()
	return nil
}

// Item templates and categories the fighting fields name (items.md, combat.md).
const (
	templateRustedBlade = 0x63
	templateWarMonger   = 0x64
	templateSilverSword = 0xBF
	categoryBodyArmor   = 1
	categoryFirstWeapon = 9
	categoryLastWeapon  = 12
	unwearingValue      = 30000
	// delayUpdates is the player updates in each point of a hand's delay.
	delayUpdates   = 3
	bodyArmorValue = 7500
)

// wearByCategory are the worn slots of categories 0–5; the rest go in a hand (items.md,
// Engine model).
var wearByCategory = []string{content.WearHead, content.WearBody, content.WearFeet, content.WearArms, content.WearQuiver, content.WearNeck}

// dice returns damage code c as dice: (c >> 4) + 1 dice of (c & 0Fh) + 1 sides; 00h is none.
func dice(c int) content.Dice {
	if c == 0 {
		return content.Dice{}
	}
	return content.Dice{Count: c>>4 + 1, Sides: c&0x0F + 1}
}

// fighting sets item it's armor, wear, weapon class, value, actions and power from template i.
func (im *importer) fighting(i int, tm *ObjectTemplate, it *content.Item) {
	cat := tm.Category()
	it.Armor = tm.Armor()
	it.Wear = content.WearHand
	if cat < len(wearByCategory) {
		it.Wear = wearByCategory[cat]
	}
	switch {
	case tm.Weapon():
		if cat >= categoryFirstWeapon && cat <= categoryLastWeapon {
			class := cat - categoryFirstWeapon
			it.WeaponClass = &class
		}
		switch {
		case i == templateRustedBlade || i == templateWarMonger || i == templateSilverSword:
			it.Durability = unwearingValue
		case tm.Weight() == 10:
			it.Durability = 6000
		default:
			it.Durability = tm.Weight() * 250
		}
	case cat == categoryBodyArmor:
		it.Durability = bodyArmorValue
	default:
		it.Charges = tm.Charges()
	}
	// Templates 72h–7Bh keep their charges as one shared count (items.md, Object values).
	if i >= firstSharedTemplate && i < firstSharedTemplate+shareCount {
		it.Charges, it.SharedCharges = tm.Charges(), true
	}
	for a := range 2 {
		icon := tm.ActionIcon(a)
		// A delay's points are three player updates each (items.md, Hands and actions).
		act := content.ItemAction{Damage: dice(tm.DamageCode(a)), Delay: delayUpdates * ((tm.DelayCode()>>4)&0x0F + 3)}
		if a == 1 {
			act.Delay = delayUpdates * (tm.DelayCode()&0x0F + 3)
		}
		switch {
		case icon >= 100 && icon < 255:
			act.Kind, act.Skill = content.ActionAttack, icon%10
			switch icon / 10 * 10 {
			case 110:
				act.Kind = content.ActionAttackBoth
			case 120:
				act.Kind, act.Skill = content.ActionFire, 0
			}
		case icon == 9 || icon == 10:
			act.Kind = content.ActionUseUp
			if im.useUpTracks == nil {
				im.useUpTracks = map[string]map[int]int{}
			}
			if im.useUpTracks[it.ID] == nil {
				im.useUpTracks[it.ID] = map[int]int{}
			}
			im.useUpTracks[it.ID][len(it.Actions)] = useUpTrack[icon]
		case icon >= 11 && icon <= 20:
			act.Kind = content.ActionUse
			// Icon 11h also swings the object, without a blow (items.md, Using objects).
			act.Swings = icon == iconUseAndSwing
		default:
			continue
		}
		it.Actions = append(it.Actions, act)
	}
	if p, worn := tm.Power(); p != 0 {
		it.Power, it.PowerWorn = p, worn
	}
	if look, ok := thrownLooks[i]; ok {
		it.Missile = &content.Missile{Damage: dice(tm.DamageCode(1)), FiredOnly: look}
	}
}

// iconUseAndSwing is the use action that also shows the swing pose.
const iconUseAndSwing = 0x11

// thrownLooks are the templates with a look of their own in flight, which roll their second
// action's damage there; true for those that roll it only when fired (items.md, Throwing).
var thrownLooks = map[int]bool{0x67: false, 0x68: false, 0x75: false, 0x6D: false, 0x97: true, 0x98: true, 0x99: true}

// Templates objects become (items.md, Using up; Throwing).
const (
	templateEmptyBottle = 0x07
	templateBrokenGlass = 0x0C
	categoryBottle      = 8
)

// spentAndBroken sets what each item leaves when used up, and what bottles break into, once
// every item has its id. Only an object used up with a quantity leaves anything (items.md, Using
// up).
func (im *importer) spentAndBroken() {
	for i := range im.objects.Templates {
		tm := &im.objects.Templates[i]
		it := im.pack.Items[im.itemIDs[i]]
		if tm.Category() == categoryBottle {
			it.BreaksInto = im.itemIDs[templateBrokenGlass]
		}
		usedUp := false
		for _, a := range it.Actions {
			usedUp = usedUp || a.Kind == content.ActionUseUp
		}
		switch {
		case !usedUp || it.Charges == 0:
		case tm.Category() == categoryBottle:
			it.Spent = im.itemIDs[templateEmptyBottle]
		case i+1 < len(im.itemIDs):
			it.Spent = im.itemIDs[i+1]
		}
	}
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
	stationaryCaster      = 2
	firstPattern          = 100
	patternFixed          = 100
	slowBank              = 0x12
	slowSpeed             = 2
	creatureSpeed         = 8
	hostileSpeed          = 16
	creatureWeight        = 100
	creatureFootprint     = 16
	trapFootprint         = 20
	behaviorNeverHostile  = 0x0004
	behaviorTalks         = 0x0008
	behaviorCorpse        = 0x0010
	behaviorHostile       = 0x0100
	behaviorResists       = 0x0200
	behaviorInvulnerable  = 0x1000
	behaviorPoisonous     = 0x0040
	behaviorContact       = 0x0080
	behaviorPhysicalOnly  = 0x0400
	behaviorDeathBlow     = 0x0800
	behaviorResistsFire   = 0x2000
	behaviorResistsLight  = 0x4000
	behaviorResistsPoison = 0x8000
	boulderSpeed          = 8
	boulderInterval       = 4
	trapInterval          = 128
	boulderRollFrames     = 10
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

	expBonus := 4*rec.HitBonus() + 13*rec.Bonus() + rec.Accuracy() + rec.Resistance()
	if rec.Spell() != 0 {
		expBonus += 247
	}
	if rec.Behavior()&behaviorPoisonous != 0 {
		expBonus += 123
	}
	t := &content.Thing{
		Attack:     dice(rec.Attack()),
		Experience: content.Dice{Count: rec.HitDice(), Sides: 10, Bonus: expBonus},
		ID:         id, Kind: content.ThingCreature, Sprite: sid, Speed: creatureSpeed,
		IntervalMS: ms(rec.Interval()), Weight: creatureWeight, Footprint: creatureFootprint,
		HitPoints: content.Dice{Count: rec.HitDice(), Sides: 10, Bonus: 4 * rec.HitBonus()},
		Stats:     map[string]int{"resistance": rec.Resistance(), "bonus": rec.Bonus(), "accuracy": rec.Accuracy()},
	}
	if t.HitPoints.Count == 0 {
		t.HitPoints.Sides = 0
	}
	if t.Experience.Count == 0 {
		t.Experience.Sides = 0
	}
	if v := rec.Vulnerability(); v >= categoryFirstWeapon && v <= categoryLastWeapon {
		class := v - categoryFirstWeapon
		t.Vulnerability = &class
	}
	if rec.Bank() == slowBank {
		t.Speed = slowSpeed
	}
	if n == 0x23 || n == 0x2B {
		t.HostileSpeed = hostileSpeed
	}
	if sp, ok := spellID(rec.Spell()); ok && rec.Spell() != 0 {
		t.Spell, t.SpellPower = sp, content.Dice{Count: 5, Sides: 5}
		if rec.Frames() == stationaryCaster {
			t.Spell, _ = spellID(1)
			t.SpellPower = content.Dice{Count: 15, Sides: 10}
		}
	}
	// Only a record that talks holds a conversation and a portrait (creatures.md): the others'
	// bytes 0 and 1 are 0, which would name the disk menu's script and portrait 0.
	if rec.Behavior()&behaviorTalks != 0 {
		t.Behaviors = append(t.Behaviors, content.BehaviorTalks)
		if id := conversationID(rec.Conversation()); im.pack.Conversations[id] != nil {
			t.Conversation = id
		}
		if p := im.npcPortrait(rec.Portrait()); p != "" {
			t.TalkPortrait = p
		}
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
		{behaviorPoisonous, content.BehaviorPoisonous},
		{behaviorContact, content.BehaviorHurtsOnContact},
		{behaviorDeathBlow, content.BehaviorDeathBlow},
		{behaviorPhysicalOnly, content.BehaviorPhysicalOnly},
		{behaviorResistsFire, content.BehaviorResistsFire},
		{behaviorResistsLight, content.BehaviorResistsLight},
		{behaviorResistsPoison, content.BehaviorResistsPoison},
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
		t.Behaviors = append(t.Behaviors, content.BehaviorAnimated)
		f := 0
		if frames == patternFixed {
			f = d0
		}
		anims["idle"] = still(f)
	case frames == stationaryCaster:
		t.Facing = 2
		t.Behaviors = append(t.Behaviors, content.BehaviorStationary)
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
		anims["dead"] = still(len(b.Frames) - 1)
	}
	t.Animations = im.fitAnimations(fmt.Sprintf("NPC record %d", n), anims, len(b.Frames))
	im.specialCreature(n, rec, t)
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
