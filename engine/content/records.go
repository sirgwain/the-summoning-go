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
	// Charges anything else's.
	Armor      int `json:"armor,omitempty"`
	Durability int `json:"durability,omitempty"`
	Charges    int `json:"charges,omitempty"`
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
)

var behaviors = map[string]bool{
	BehaviorHostile: true, BehaviorNeverHostile: true, BehaviorLeavesCorpse: true,
	BehaviorResistsTeleport: true, BehaviorInvulnerable: true, BehaviorPoisonous: true,
	BehaviorHurtsOnContact: true, BehaviorDeathBlow: true, BehaviorPhysicalOnly: true,
	BehaviorResistsFire: true, BehaviorResistsLight: true, BehaviorResistsPoison: true, BehaviorAnimated: true, BehaviorStationary: true,
}

// ArrowClass is the weapon class of bows; arrows hurt creatures vulnerable to it more.
const ArrowClass = 3

// Thing is a template for things that stand in a level.
type Thing struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Sprite string `json:"sprite"`
	Facing int    `json:"facing"`
	// Speed is the step in world units.
	Speed      int `json:"speed,omitempty"`
	IntervalMS int `json:"interval_ms,omitempty"`
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
	// Spell is a creature's magic attack, 0 for none.
	Spell int `json:"spell,omitempty"`
	// Inventory is what a player starts with.
	Inventory []InventoryObject `json:"inventory,omitempty"`
	// Animations are the thing's own; a name it does not have comes from its sprite.
	Animations map[string]Animation `json:"animations,omitempty"`
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
	MsgNotACreature          = "not_a_creature"
	MsgCannotHurt            = "cannot_hurt"
	MsgMiss                  = "miss"
	MsgHit                   = "hit"
	MsgCreatureHits          = "creature_hits"
	MsgCreatureMiss          = "creature_misses"
	MsgWeaponBreaks          = "weapon_breaks"
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
)

var messageKeys = map[string]bool{
	MsgLeverStuck: true, MsgSlotWrongItem: true, MsgSlotStuck: true, MsgNoDropHere: true, MsgNoRoom: true,
	MsgNotACreature: true, MsgCannotHurt: true, MsgMiss: true, MsgHit: true, MsgCreatureHits: true,
	MsgCreatureMiss: true, MsgWeaponBreaks: true, MsgSkillUp: true, MsgLevelUp: true,
	MsgFullHealth:            true,
	MsgFullMagic:             true,
	MsgNotPoisoned:           true,
	MsgNotPoisonedOrConfused: true,
	MsgNotFatigued:           true,
	MsgNoDestination:         true,
	MsgNoCreatures:           true,
	MsgAlreadyHave:           true,
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
}
