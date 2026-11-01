package content

// Spells, the spell UI's art, keywords and conversations (docs/pack.md; docs/spec/spells.md,
// docs/spec/dialogue.md).

// Schools is the number of schools of magic.
const Schools = 4

// GestureIcons is the number of gesture icons, named by the letters A to A + GestureIcons − 1.
const GestureIcons = 12

// MaxGesture is the longest gesture the player can enter.
const MaxGesture = 8

// Spell is a spell.
type Spell struct {
	ID string `json:"id"`
	// Name is a text key.
	Name   string `json:"name"`
	School int    `json:"school"`
	Cost   int    `json:"cost"`
	// Power is the dice of the spell's power; PowerBonus the sides of each extra die the caster
	// gets for its power attribute and school skill.
	Power      Dice `json:"power,omitzero"`
	PowerBonus int  `json:"power_bonus,omitempty"`
	// Gesture is its icons as letters A–L; a spell without one cannot be learned.
	Gesture string `json:"gesture,omitempty"`
	Icon    *Image `json:"icon,omitempty"`
	// Needs is what must hold before it can be cast, nil for nothing.
	Needs *SpellNeeds `json:"needs,omitempty"`
	// Practice is how much more than 1 a cast practices its school.
	Practice int `json:"practice,omitempty"`
	// Effects run when the spell takes effect.
	Effects []Action `json:"effects"`
}

// SpellNeeds is a spell's condition: one of its fields is set.
type SpellNeeds struct {
	// FloorAhead is how far ahead a walkable cell must be.
	FloorAhead int `json:"floor_ahead,omitempty"`
	// Mark is a rune a mark of the level must have.
	Mark string `json:"mark,omitempty"`
	// SpellNear is how near a spell in flight must be.
	SpellNear int `json:"spell_near,omitempty"`
	// Holding are two items held in the hands, in either order.
	Holding []string `json:"holding,omitempty"`
	// CreatureNear is how near a living creature must be.
	CreatureNear int  `json:"creature_near,omitempty"`
	Poisoned     bool `json:"poisoned,omitempty"`
	Fatigued     bool `json:"fatigued,omitempty"`
	Hurt         bool `json:"hurt,omitempty"`
	// Else is the message shown when it does not hold, in place of the condition's own.
	Else string `json:"else,omitempty"`
}

// Magic is what the spell UI draws, and the schools' names.
type Magic struct {
	ID string `json:"id"`
	// Icons are the gesture icons, Icons[i] for letter 'A' + i.
	Icons []Image `json:"icons"`
	// Panel is the empty gesture panel.
	Panel *Image `json:"panel,omitempty"`
	// Schools are the text keys of the schools' names.
	Schools []string `json:"schools"`
}

// Pointers are the pointer's images, each drawn with its top-left pixel at the mouse.
type Pointers struct {
	ID string `json:"id"`
	// Arrow is the pointer over the view; Talk over something a click talks to; Wait while the
	// game waits.
	Arrow Image  `json:"arrow"`
	Talk  *Image `json:"talk,omitempty"`
	Wait  *Image `json:"wait,omitempty"`
}

// Keywords is the pack's keywords.
type Keywords struct {
	ID       string    `json:"id"`
	Keywords []Keyword `json:"keywords"`
}

// Keyword is a word the player can ask about.
type Keyword struct {
	ID string `json:"id"`
	// Word and Answer are text keys: the word, and what asking it says.
	Word   string `json:"word"`
	Answer string `json:"answer,omitempty"`
}

// Conversation is what is said with a thing: clauses run in passes (docs/spec/dialogue.md).
type Conversation struct {
	ID      string   `json:"id"`
	Name    string   `json:"name,omitempty"`
	Clauses []Clause `json:"clauses"`
}

// Clause is a condition and the actions run when it holds.
type Clause struct {
	If Condition `json:"if"`
	Do []Action  `json:"do"`
}

// Condition is a clause's condition: one of its fields is set.
type Condition struct {
	FirstMeeting bool         `json:"first_meeting,omitempty"`
	Greeting     bool         `json:"greeting,omitempty"`
	Asked        []string     `json:"asked,omitempty"`
	Never        bool         `json:"never,omitempty"`
	Carries      []string     `json:"carries,omitempty"`
	Counter      *CounterTest `json:"counter,omitempty"`
	// Dead is a thing template one of whose creatures has died.
	Dead         string     `json:"dead,omitempty"`
	FullHealth   bool       `json:"full_health,omitempty"`
	CarriesCount *ItemCount `json:"carries_count,omitempty"`
	SlotEmpty    *int       `json:"slot_empty,omitempty"`
	SlotHolds    *SlotHolds `json:"slot_holds,omitempty"`
	Always       bool       `json:"always,omitempty"`
}

// CounterTest compares a counter with a value: one of Above, Equals and Below is set.
type CounterTest struct {
	Counter string `json:"counter"`
	Above   *int   `json:"above,omitempty"`
	Equals  *int   `json:"equals,omitempty"`
	Below   *int   `json:"below,omitempty"`
}

// ItemCount is a number of objects of an item.
type ItemCount struct {
	Item  string `json:"item"`
	Count int    `json:"count"`
}

// SlotHolds is an inventory slot holding an object of an item.
type SlotHolds struct {
	Slot int    `json:"slot"`
	Item string `json:"item"`
}

// Counters names the counters the engine itself reads and writes.
type Counters struct {
	// AfterConversation is cleared when every conversation ends.
	AfterConversation string `json:"after_conversation,omitempty"`
	// UseResult decides whether an object whose conversation has UseResult is used.
	UseResult string `json:"use_result,omitempty"`
	// Transformed is set to 1 by a Shape Change into the sprite TransformedLook, and cleared
	// when the change ends.
	Transformed     string `json:"transformed,omitempty"`
	TransformedLook string `json:"transformed_look,omitempty"`
}

// Cast is a spell an item casts when used.
type Cast struct {
	Spell string `json:"spell"`
	// Power is the cast's power; none rolls the spell's own.
	Power Dice `json:"power,omitzero"`
	// Message is an engine message shown as it casts.
	Message string `json:"message,omitempty"`
}

// Spell effects (docs/pack.md, Procedures).

// Damage kinds a missile hurts by.
const (
	HurtsFire      = "fire"
	HurtsLightning = "lightning"
	HurtsPoison    = "poison"
)

// MissileSpell flies a spell.
type MissileSpell struct {
	Hurts    string `json:"hurts,omitempty"`
	Opens    bool   `json:"opens,omitempty"`
	Scatters bool   `json:"scatters,omitempty"`
	// Bane is a thing template the missile hurts by BaneDamage; it does nothing to others.
	Bane       string `json:"bane,omitempty"`
	BaneDamage int    `json:"bane_damage,omitempty"`
	Sprite     string `json:"sprite"`
	// Directional draws the frame of the direction it flies.
	Directional bool `json:"directional,omitempty"`
	// Cloud is the sprite it ends as, if not its own.
	Cloud   string `json:"cloud,omitempty"`
	Bounces bool   `json:"bounces,omitempty"`
	// Around casts four, along the diagonals of the caster's facing.
	Around bool `json:"around,omitempty"`
}

// Timed effects a spell starts.
const (
	TimedBattleRage      = "battle_rage"
	TimedSight           = "sight"
	TimedHaste           = "haste"
	TimedInvisibility    = "invisibility"
	TimedFireShield      = "fire_shield"
	TimedLightningShield = "lightning_shield"
	TimedWeaponSkills    = "weapon_skills"
	TimedMagicSkills     = "magic_skills"
)

var timedEffects = map[string]bool{
	TimedBattleRage: true, TimedSight: true, TimedHaste: true, TimedInvisibility: true,
	TimedFireShield: true, TimedLightningShield: true, TimedWeaponSkills: true, TimedMagicSkills: true,
}

// Conjure puts a new thing ahead of the caster.
type Conjure struct {
	Thing    string `json:"thing"`
	Distance int    `json:"distance"`
	// LookPerSkill picks its frame: the school's skill divided by it.
	LookPerSkill int `json:"look_per_skill,omitempty"`
}

// Portal puts a new portal ahead of the caster.
type Portal struct {
	Thing    string      `json:"thing"`
	Distance int         `json:"distance"`
	To       Destination `json:"to"`
}

// Join joins the two items held in the hands into one.
type Join struct {
	Items []string `json:"items"`
	Into  string   `json:"into"`
}

// Transmute fills the Holder held in a hand from the other hand's object.
type Transmute struct {
	Holder string            `json:"holder"`
	Table  map[string]string `json:"table"`
	// Otherwise is made from anything else, valued at the cast's power.
	Otherwise string `json:"otherwise"`
}

// Conversation actions.

// SayAs says a text beside a portrait; none is the player's.
type SayAs struct {
	Portrait string `json:"portrait,omitempty"`
	Text     string `json:"text"`
}

// SetCounter sets a counter.
type SetCounter struct {
	Counter string `json:"counter"`
	Value   int    `json:"value"`
}

// Menus a conversation can open.
const (
	MenuQuit    = "quit"
	MenuSave    = "save"
	MenuRestore = "restore"
	MenuDelete  = "delete"
	MenuRename  = "rename"
	MenuLeave   = "leave"
)

var menus = map[string]bool{MenuQuit: true, MenuSave: true, MenuRestore: true, MenuDelete: true, MenuRename: true, MenuLeave: true}
