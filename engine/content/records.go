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
)

var behaviors = map[string]bool{
	BehaviorHostile: true, BehaviorNeverHostile: true, BehaviorLeavesCorpse: true,
	BehaviorResistsTeleport: true, BehaviorInvulnerable: true,
}

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
	MsgLeverStuck    = "lever_stuck"
	MsgSlotWrongItem = "slot_wrong_item"
	MsgSlotStuck     = "slot_stuck"
	MsgNoDropHere    = "no_drop_here"
	MsgNoRoom        = "no_room"
)

var messageKeys = map[string]bool{
	MsgLeverStuck: true, MsgSlotWrongItem: true, MsgSlotStuck: true, MsgNoDropHere: true, MsgNoRoom: true,
}
