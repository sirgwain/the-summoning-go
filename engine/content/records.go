package content

// Item is an item template.
type Item struct {
	ID string `json:"id"`
	// Name is a text key.
	Name string `json:"name"`
	// Weight and Size are the item's own, without contents.
	Weight    int        `json:"weight"`
	Size      int        `json:"size"`
	Container *Container `json:"container,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	// Random items can be picked for a random object.
	Random bool `json:"random,omitempty"`
	// Icon is the inventory image; Floor the image drawn on the floor.
	Icon  Image `json:"icon"`
	Floor Image `json:"floor"`
	// Armor is what the item adds worn. Durability is a weapon's or armor's starting value;
	// Charges anything else's. SharedCharges makes the charges one count for the whole item,
	// kept by the world and refilled whenever an object of the item is made; such an item may
	// have durability besides.
	Armor         int  `json:"armor,omitempty"`
	Durability    int  `json:"durability,omitempty"`
	Charges       int  `json:"charges,omitempty"`
	SharedCharges bool `json:"shared_charges,omitempty"`
	// Wear is the worn slot the item goes in, one of the Wear constants.
	Wear string `json:"wear,omitempty"`
	// WeaponClass is a weapon's class, nil for anything else.
	WeaponClass *int `json:"weapon_class,omitempty"`
	// Actions are what the item does in hand.
	Actions []ItemAction `json:"actions,omitempty"`
	// Power is the power the item starts when used, 0 for none; PowerWorn starts it worn too.
	Power     int  `json:"power,omitempty"`
	PowerWorn bool `json:"power_worn,omitempty"`
	// Spent is the item a used-up object becomes.
	Spent string `json:"spent,omitempty"`
	// Held is how it looks in the player's hands.
	Held *Held `json:"held,omitempty"`
	// Missile is how it hurts in flight; BreaksInto what a landing bottle becomes.
	Missile    *Missile `json:"missile,omitempty"`
	BreaksInto string   `json:"breaks_into,omitempty"`
	// Casts is a spell using it casts; Conversation a thing template whose conversation using it
	// starts, used only when UseResult and the pack's use_result counter is then nonzero.
	Casts        *Cast  `json:"casts,omitempty"`
	Conversation string `json:"conversation,omitempty"`
	UseResult    bool   `json:"use_result,omitempty"`
	// Teaches are the gesture icons reading it teaches; Picture what reading it shows.
	Teaches string `json:"teaches,omitempty"`
	Picture *Image `json:"picture,omitempty"`
	// Effect is a text key saying what its power does, and Lore one about it, both for the book.
	Effect string `json:"effect,omitempty"`
	Lore   string `json:"lore,omitempty"`
	// Doll is how it shows on the inventory's figure when worn.
	Doll *DollImage `json:"doll,omitempty"`
	// Looks are the looks it puts on the player's figure layers while worn, by layer name.
	Looks map[string]FigureLook `json:"looks,omitempty"`
	// Fill gives a container a look for how full it is.
	Fill *Fill `json:"fill,omitempty"`
	// Poisons makes its blows poison; Grows and Levels make it a counted weapon; OnBreak runs
	// when it breaks; Flying is its look in flight.
	Poisons bool     `json:"poisons,omitempty"`
	Grows   *Grows   `json:"grows,omitempty"`
	Levels  *Levels  `json:"levels,omitempty"`
	OnBreak []Action `json:"on_break,omitempty"`
	Flying  *Flying  `json:"flying,omitempty"`
}

// PowerLasts reports whether power lasts while worn, ending when taken off (docs/spec/items.md,
// Powers).
func PowerLasts(power int) bool {
	switch power {
	case 0x0B, 0x0F, 0x16, 0x17, 0x47, 0x4C:
		return true
	}
	return power >= 0x30 && power <= 0x41
}

// Fill is how a container looks as it fills: step k, for k from 1, is shown while it holds more
// than (k−1)·Per objects; empty, it shows its own icon and worn look.
type Fill struct {
	Per int `json:"per"`
	// Icons and Dolls are steps 1, 2, …'s icons and worn looks.
	Icons []Image     `json:"icons"`
	Dolls []DollImage `json:"dolls,omitempty"`
}

// step returns the fill step of n contents, 0 for none, at most last.
func (f *Fill) step(n, last int) int {
	if n <= 0 || f.Per <= 0 || last <= 0 {
		return 0
	}
	return min((n+f.Per-1)/f.Per, last)
}

// IconFor returns the item's icon holding n objects (docs/spec/items.md, Templates).
func (i *Item) IconFor(n int) Image {
	if f := i.Fill; f != nil {
		if k := f.step(n, len(f.Icons)); k > 0 {
			return f.Icons[k-1]
		}
	}
	return i.Icon
}

// DollFor returns the item's worn look holding n objects, nil for none.
func (i *Item) DollFor(n int) *DollImage {
	if i.Fill != nil {
		if k := i.Fill.step(n, len(i.Fill.Dolls)); k > 0 {
			return &i.Fill.Dolls[k-1]
		}
	}
	return i.Doll
}

// Held is how an item looks in each of the player's hands.
type Held struct {
	Right *HeldLayers `json:"right,omitempty"`
	Left  *HeldLayers `json:"left,omitempty"`
}

// HeldLayers are the sprites drawn behind and in front of the player for an item in one hand,
// frame for frame with the player's sprite.
type HeldLayers struct {
	Behind string `json:"behind,omitempty"`
	Front  string `json:"front,omitempty"`
}

// Hand returns the layers for hand 0 (right) or 1 (left), or nil.
func (h *Held) Hand(i int) *HeldLayers {
	if h == nil {
		return nil
	}
	if i == 0 {
		return h.Right
	}
	return h.Left
}

// Missile is how an item hurts in flight.
type Missile struct {
	Damage Dice `json:"damage"`
	// FiredOnly items do 1 when thrown rather than fired.
	FiredOnly bool `json:"fired_only,omitempty"`
}

// InventoryObject is an object in an inventory slot.
type InventoryObject struct {
	Slot int `json:"slot"`
	Object
}

// Worn slots an item can go in.
const (
	WearHead   = "head"
	WearBody   = "body"
	WearFeet   = "feet"
	WearArms   = "arms"
	WearQuiver = "quiver"
	WearNeck   = "neck"
	WearHand   = "hand"
)

var wears = map[string]bool{WearHead: true, WearBody: true, WearFeet: true, WearArms: true, WearQuiver: true, WearNeck: true, WearHand: true}

// Item action kinds.
const (
	ActionAttack     = "attack"
	ActionAttackBoth = "attack_both"
	ActionFire       = "fire"
	ActionUseUp      = "use_up"
	ActionUse        = "use"
)

var actionKinds = map[string]bool{ActionAttack: true, ActionAttackBoth: true, ActionFire: true, ActionUseUp: true, ActionUse: true}

// ItemAction is something an item does in hand.
type ItemAction struct {
	Kind   string `json:"kind"`
	Damage Dice   `json:"damage,omitzero"`
	// Delay is the player updates the hand waits afterwards.
	Delay int `json:"delay"`
	// Skill is the weapon class skill level an attack needs.
	Skill int `json:"skill,omitempty"`
	// Icon is the image its box shows.
	Icon *Image `json:"icon,omitempty"`
	// Sound is the sound a use-up action makes when it uses its object up.
	Sound string `json:"sound,omitempty"`
	// Swings makes a use or use-up action also show the swing pose, without a blow: of both
	// hands for a weapon, else of its hand.
	Swings bool `json:"swings,omitempty"`
}

// Attack reports whether the action strikes.
func (a ItemAction) Attack() bool {
	return a.Kind == ActionAttack || a.Kind == ActionAttackBoth || a.Kind == ActionFire
}

// Container says what a container item holds.
type Container struct {
	Capacity int `json:"capacity"`
	Largest  int `json:"largest"`
	// Weightless contents add no weight.
	Weightless bool `json:"weightless,omitempty"`
}

// Image is one frame of a sprite.
type Image struct {
	Sprite string `json:"sprite"`
	Frame  int    `json:"frame"`
}

// Thing kinds.
const (
	ThingCreature = "creature"
	ThingBoulder  = "boulder"
	ThingTrap     = "trap"
	ThingPlayer   = "player"
	// ThingConjured things are put up by spells: they block, can be pushed and last a while.
	ThingConjured = "conjured"
	// ThingPortal things take the player who walks into them where the spell said.
	ThingPortal = "portal"
)

// DefaultFootprint is the footprint of a thing whose template gives none.
const DefaultFootprint = 16

// Behaviors.
const (
	BehaviorHostile         = "hostile"
	BehaviorNeverHostile    = "never_hostile"
	BehaviorLeavesCorpse    = "leaves_corpse"
	BehaviorResistsTeleport = "resists_teleport"
	BehaviorInvulnerable    = "invulnerable"
	BehaviorPoisonous       = "poisonous"
	BehaviorHurtsOnContact  = "hurts_on_contact"
	BehaviorDeathBlow       = "death_blow"
	BehaviorPhysicalOnly    = "physical_only"
	BehaviorResistsFire     = "resists_fire"
	BehaviorResistsLight    = "resists_lightning"
	BehaviorResistsPoison   = "resists_poison"
	// BehaviorAnimated creatures only animate in place and cannot be struck; BehaviorStationary
	// ones stand still and cast.
	BehaviorAnimated   = "animated"
	BehaviorStationary = "stationary"
	// BehaviorTalks creatures can be talked to, until they turn hostile.
	BehaviorTalks = "talks"
	// BehaviorMirrors creatures pass the damage they take to the player.
	BehaviorMirrors = "mirrors"
)

var behaviors = map[string]bool{
	BehaviorHostile: true, BehaviorNeverHostile: true, BehaviorLeavesCorpse: true,
	BehaviorResistsTeleport: true, BehaviorInvulnerable: true, BehaviorPoisonous: true,
	BehaviorHurtsOnContact: true, BehaviorDeathBlow: true, BehaviorPhysicalOnly: true,
	BehaviorResistsFire: true, BehaviorResistsLight: true, BehaviorResistsPoison: true, BehaviorAnimated: true, BehaviorStationary: true,
	BehaviorTalks: true, BehaviorMirrors: true,
}

// Behaviors returns the behaviors the engine knows, sorted.
func Behaviors() []string { return sortedKeys(behaviors) }

// ArrowClass is the weapon class of bows; arrows hurt creatures vulnerable to it more.
const ArrowClass = 3

// Thing is a template for things that stand in a level.
type Thing struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Sprite string `json:"sprite"`
	Facing int    `json:"facing"`
	// Speed is the step in world units; HostileSpeed, if set, replaces it once a conversation
	// turns the creature hostile.
	Speed        int `json:"speed,omitempty"`
	HostileSpeed int `json:"hostile_speed,omitempty"`
	IntervalMS   int `json:"interval_ms,omitempty"`
	// Footprint is the side of the square, in world units around the ground point, the thing
	// takes up; 0 means DefaultFootprint.
	Footprint int `json:"footprint,omitempty"`
	// Weight is what the thing adds to a plate it stands on.
	Weight      int            `json:"weight"`
	HitPoints   Dice           `json:"hit_points,omitzero"`
	MagicPoints Dice           `json:"magic_points,omitzero"`
	Behaviors   []string       `json:"behaviors,omitempty"`
	Stats       map[string]int `json:"stats,omitempty"`
	// Attack is a creature's blow; Experience what killing it is worth.
	Attack     Dice `json:"attack,omitzero"`
	Experience Dice `json:"experience,omitzero"`
	// Vulnerability is the weapon class that hurts it more, nil for none.
	Vulnerability *int `json:"vulnerability,omitempty"`
	// Spell is the spell a creature casts, empty for none, at the power SpellPower rolls.
	Spell      string `json:"spell,omitempty"`
	SpellPower Dice   `json:"spell_power,omitzero"`
	// Conversation is the conversation held with it; TalkPortrait the portrait it shows.
	Conversation string `json:"conversation,omitempty"`
	TalkPortrait string `json:"talk_portrait,omitempty"`
	// Theme is a creature's music track, which plays while the player sees one.
	Theme string `json:"theme,omitempty"`
	// Icons are the gesture icons a player starts knowing.
	Icons string `json:"icons,omitempty"`
	// Name is a creature's name, a text key; KillPractice the practice a player's blow that kills
	// it gives; Animate how an animated creature's frame changes.
	Name         string   `json:"name,omitempty"`
	KillPractice int      `json:"kill_practice,omitempty"`
	Animate      *Animate `json:"animate,omitempty"`
	// Lore is a text key about a creature, for the book.
	Lore string `json:"lore,omitempty"`
	// Inventory is what a player starts with.
	Inventory []InventoryObject `json:"inventory,omitempty"`
	// Portrait is a player's portrait, which also gives its figure's colors.
	Portrait string `json:"portrait,omitempty"`
	// Figure draws a player in layers, bottom up, in place of its sprite.
	Figure []FigureLayer `json:"figure,omitempty"`
	// Animations are the thing's own; a name it does not have comes from its sprite.
	Animations map[string]Animation `json:"animations,omitempty"`
}

// FigureLayer is a layer of a player's figure: its name, which items' looks name, and its own look.
type FigureLayer struct {
	Name string `json:"name"`
	FigureLook
}

// FigureLook is a sprite drawn as a layer of a player's figure, frame for frame with the player's
// sprite, and the sprite for a female player when it differs.
type FigureLook struct {
	Sprite string `json:"sprite"`
	Female string `json:"female,omitempty"`
}

// For returns the look's sprite for a player, female or not.
func (l FigureLook) For(female bool) string {
	if female && l.Female != "" {
		return l.Female
	}
	return l.Sprite
}

// Animation returns the animation the thing shows under name, falling back as docs/pack.md
// says: the thing's own, then its sprite's, for name, then for each fallback in turn. It
// returns false when neither has any of them.
func (p *Pack) Animation(t *Thing, names ...string) (Animation, bool) {
	sprite := p.Sprites[t.Sprite]
	for _, n := range names {
		if a, ok := t.Animations[n]; ok {
			return a, true
		}
		if sprite != nil {
			if a, ok := sprite.Animations[n]; ok {
				return a, true
			}
		}
	}
	return Animation{}, false
}

// Size returns the thing's footprint.
func (t *Thing) Size() int {
	if t.Footprint > 0 {
		return t.Footprint
	}
	return DefaultFootprint
}

// Has reports whether the thing has behavior b.
func (t *Thing) Has(b string) bool {
	for _, x := range t.Behaviors {
		if x == b {
			return true
		}
	}
	return false
}

// Look layers.
const (
	LayerFloor = "floor"
	LayerStand = "stand"
	LayerWall  = "wall"
)

// Look is how a feature is drawn in each of its states.
type Look struct {
	ID     string                `json:"id"`
	Layer  string                `json:"layer"`
	States map[string][]LookPart `json:"states"`
}

// LookPart is a sprite frame offset from a cell's tile box.
type LookPart struct {
	Sprite string `json:"sprite"`
	Frame  int    `json:"frame"`
	Offset Offset `json:"offset"`
}

// Text is every string of the pack in one language, by key.
type Text struct {
	ID      string            `json:"id"`
	Strings map[string]string `json:"strings"`
}

// Messages maps the engine's message keys to text keys.
type Messages struct {
	ID       string            `json:"id"`
	Messages map[string]string `json:"messages"`
}

// Engine message keys.
const (
	MsgLeverStuck            = "lever_stuck"
	MsgSlotWrongItem         = "slot_wrong_item"
	MsgSlotStuck             = "slot_stuck"
	MsgNoDropHere            = "no_drop_here"
	MsgNoRoom                = "no_room"
	MsgNothingToTake         = "nothing_to_take"
	MsgIntoQuiver            = "into_quiver"
	MsgTaken                 = "taken"
	MsgPaused                = "paused"
	MsgNotACreature          = "not_a_creature"
	MsgCannotHurt            = "cannot_hurt"
	MsgMiss                  = "miss"
	MsgHit                   = "hit"
	MsgCreatureHits          = "creature_hits"
	MsgCreatureMiss          = "creature_misses"
	MsgWeaponBreaks          = "weapon_breaks"
	MsgVanishes              = "vanishes"
	MsgSkillUp               = "skill_up"
	MsgLevelUp               = "level_up"
	MsgFullHealth            = "full_health"
	MsgFullMagic             = "full_magic"
	MsgNotPoisoned           = "not_poisoned"
	MsgNotPoisonedOrConfused = "not_poisoned_or_confused"
	MsgNotFatigued           = "not_fatigued"
	MsgNoDestination         = "no_destination"
	MsgNoCreatures           = "no_creatures"
	MsgAlreadyHave           = "already_have"
	MsgHealed                = "healed"
	MsgProtected             = "protected"
	MsgBoostExtended         = "boost_extended"
	MsgPoisoned              = "poisoned"
	MsgSlowed                = "slowed"
	MsgLostWeaponSkills      = "lost_weapon_skills"
	MsgLostMagicSkills       = "lost_magic_skills"
	MsgNoQuiver              = "no_quiver"
	MsgNoArrows              = "no_arrows"
	MsgNoRoomToShoot         = "no_room_to_shoot"
	MsgCannotThrow           = "cannot_throw"
	MsgCannotAct             = "cannot_act"

	MsgCannotCast           = "cannot_cast"
	MsgNotMemorized         = "not_memorized"
	MsgOnlySpell            = "only_spell"
	MsgNoMagic              = "no_magic"
	MsgForgot               = "forgot"
	MsgForgotMore           = "forgot_more"
	MsgNoRoomForWall        = "no_room_for_wall"
	MsgWallFailed           = "wall_failed"
	MsgNoRoomForPortal      = "no_room_for_portal"
	MsgNoSpellNear          = "no_spell_near"
	MsgNothingToMend        = "nothing_to_mend"
	MsgNoCreatureNear       = "no_creature_near"
	MsgNoBottle             = "no_bottle"
	MsgFilled               = "filled"
	MsgDoorResists          = "door_resists"
	MsgMissileWall          = "missile_wall"
	MsgBattleRage           = "battle_rage"
	MsgBattleRageExtended   = "battle_rage_extended"
	MsgHaste                = "haste"
	MsgFireShield           = "fire_shield"
	MsgLightningShield      = "lightning_shield"
	MsgFrozen               = "frozen"
	MsgSight                = "sight"
	MsgWeaponSkillsUp       = "weapon_skills_up"
	MsgWeaponSkillsExtended = "weapon_skills_extended"
	MsgMagicSkillsUp        = "magic_skills_up"
	MsgMagicSkillsExtended  = "magic_skills_extended"
	MsgCured                = "cured"
	MsgKilled               = "killed"
	MsgAbsorbed             = "absorbed"
	MsgNoEnding             = "no_ending"
	// MsgMusicAll, MsgMusicEffects and MsgMusicSilent announce the music key's settings.
	MsgMusicAll     = "music_all"
	MsgMusicEffects = "music_effects"
	MsgMusicSilent  = "music_silent"
	// MsgTranscriptHeading and MsgTranscriptContinued head the transcript.
	MsgTranscriptHeading   = "transcript_heading"
	MsgTranscriptContinued = "transcript_continued"
)

var messageKeys = map[string]bool{
	MsgLeverStuck: true, MsgSlotWrongItem: true, MsgSlotStuck: true, MsgNoDropHere: true, MsgNoRoom: true, MsgNothingToTake: true, MsgIntoQuiver: true, MsgTaken: true, MsgPaused: true,
	MsgNotACreature: true, MsgCannotHurt: true, MsgMiss: true, MsgHit: true, MsgCreatureHits: true,
	MsgCreatureMiss: true, MsgWeaponBreaks: true, MsgVanishes: true, MsgSkillUp: true, MsgLevelUp: true,
	MsgFullHealth:            true,
	MsgFullMagic:             true,
	MsgNotPoisoned:           true,
	MsgNotPoisonedOrConfused: true,
	MsgNotFatigued:           true,
	MsgNoDestination:         true,
	MsgNoCreatures:           true,
	MsgAlreadyHave:           true,
	MsgNoEnding:              true,
	MsgMusicAll:              true,
	MsgMusicEffects:          true,
	MsgMusicSilent:           true,
	MsgTranscriptHeading:     true,
	MsgTranscriptContinued:   true,
	MsgHealed:                true,
	MsgProtected:             true,
	MsgBoostExtended:         true,
	MsgPoisoned:              true,
	MsgSlowed:                true,
	MsgLostWeaponSkills:      true,
	MsgLostMagicSkills:       true,
	MsgNoQuiver:              true,
	MsgNoArrows:              true,
	MsgNoRoomToShoot:         true,
	MsgCannotThrow:           true,
	MsgCannotAct:             true,
	MsgCannotCast:            true, MsgNotMemorized: true, MsgOnlySpell: true, MsgNoMagic: true,
	MsgForgot: true, MsgForgotMore: true, MsgNoRoomForWall: true, MsgWallFailed: true,
	MsgNoRoomForPortal: true, MsgNoSpellNear: true, MsgNothingToMend: true, MsgNoCreatureNear: true,
	MsgNoBottle: true, MsgFilled: true, MsgDoorResists: true, MsgMissileWall: true,
	MsgBattleRage: true, MsgBattleRageExtended: true, MsgHaste: true, MsgFireShield: true,
	MsgLightningShield: true, MsgFrozen: true, MsgSight: true, MsgWeaponSkillsUp: true,
	MsgWeaponSkillsExtended: true, MsgMagicSkillsUp: true, MsgMagicSkillsExtended: true, MsgCured: true, MsgKilled: true, MsgAbsorbed: true,
}

// FigureColor is the first of the colors a player's figure draws in its portrait's FigureColors.
const FigureColor = 224

// Portrait is a character's face: the picture shown in panels and conversations, and the colors
// a player with it is drawn in.
type Portrait struct {
	ID    string `json:"id"`
	Image Image  `json:"image"`
	// Mirrored draws the image flipped left to right in the player's panel.
	Mirrored bool `json:"mirrored,omitempty"`
	// FigureColors are the palette colors the player's figure draws its colors FigureColor,
	// FigureColor+1, … in (hair and skin).
	FigureColors []int `json:"figure_colors,omitempty"`
	Female       bool  `json:"female,omitempty"`
	// Other is the portrait of the other sex; Talk the one a conversation shows for the player
	// when the creature talked to has this portrait too. Empty means this one.
	Other string `json:"other,omitempty"`
	Talk  string `json:"talk,omitempty"`
}
