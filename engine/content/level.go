package content

import "fmt"

// Floor halves, as bits of Level.FloorHalves.
const (
	HalfUpper  = 1
	HalfLower  = 2
	HalvesBoth = HalfUpper | HalfLower
)

// Sight returns the directions of the neighbors whose sight shows the upper and lower halves of
// cell c's floor, NoFrame for none.
func (l *Level) Sight(c Cell) (upper, lower int) {
	upper, lower = NoFrame, NoFrame
	if l.UpperSight != nil {
		upper = l.UpperSight[c[0]][c[1]]
	}
	if l.LowerSight != nil {
		lower = l.LowerSight[c[0]][c[1]]
	}
	return upper, lower
}

// Halves returns which halves of cell c's floor are drawn.
func (l *Level) Halves(c Cell) int {
	if l.FloorHalves == nil {
		return HalvesBoth
	}
	return l.FloorHalves[c[0]][c[1]]
}

// DoorSight is a wall whose floor halves follow the doors beside it: while Doors[i] is closed
// exactly for the bits i set in m, a wall of shape s checks its halves by the sight of shape
// As[m][s] (docs/pack.md, door_sight).
type DoorSight struct {
	Cell  Cell    `json:"cell"`
	Doors []Cell  `json:"doors"`
	As    [][]int `json:"as"`
}

// Level is a level: its grids, the art fixed in it, its features and procedures, and the things
// and objects it places.
type Level struct {
	ID string `json:"id"`
	// Name is a text key, empty for none.
	Name string `json:"name"`
	// Map, if set, refuses the level map: a message (a text key) or a thing template's
	// conversation in its place.
	Map *MapRefusal `json:"map,omitempty"`
	// Footsteps is the sound of the player's steps here, in place of the footstep cue.
	Footsteps string `json:"footsteps,omitempty"`
	// TeleporterConfusion is what an armed teleporter adds to the player's confusion here, in
	// place of the usual 30.
	TeleporterConfusion *int `json:"teleporter_confusion,omitempty"`
	// Size is the number of rows and columns.
	Size   [2]int `json:"size"`
	Floors string `json:"floors"`
	Walls  string `json:"walls"`
	// FloorTypes are the level's kinds of floor; Floor holds an index into them.
	FloorTypes []FloorType `json:"floor_types"`
	// Floor is each cell's floor type; Wall each cell's wall shape. NoFrame for none.
	Floor Grid `json:"floor"`
	Wall  Grid `json:"wall"`
	// FloorHalves says which halves of each cell's floor are drawn (HalfUpper, HalfLower); the
	// others are blanked. Nil draws every floor whole.
	FloorHalves Grid `json:"floor_halves,omitempty"`
	// UpperSight and LowerSight give, for the floor under a closed door, or a wall whose shape has
	// no Sight, the direction of
	// the neighbor that must be visible for each half to show; NoFrame for none. Nil for none at
	// all.
	UpperSight Grid `json:"upper_sight,omitempty"`
	LowerSight Grid `json:"lower_sight,omitempty"`
	// DoorSight gives the walls whose floor halves follow the doors beside them the shape they
	// check as for each combination of those doors closed.
	DoorSight []DoorSight `json:"door_sight,omitempty"`
	// NoSight are the cells whose sight never shows the floor half of a wall beside them.
	NoSight []Cell `json:"no_sight,omitempty"`
	// Overlays are drawn after all floors and before anything standing.
	Overlays []Placed `json:"overlays,omitempty"`
	// Decorations are drawn on their cell's wall, right after it.
	Decorations []Placed `json:"decorations,omitempty"`
	// Props stand in their cell and are drawn in depth order with the walls.
	Props      []Placed            `json:"props,omitempty"`
	Features   []*Feature          `json:"features,omitempty"`
	Procedures map[string][]Action `json:"procedures,omitempty"`
	Things     []*PlacedThing      `json:"things,omitempty"`
	Objects    []*FloorObject      `json:"objects,omitempty"`
}

// Placed is a sprite frame placed relative to a cell's tile box.
type Placed struct {
	Cell   Cell   `json:"cell"`
	Sprite string `json:"sprite"`
	Frame  int    `json:"frame"`
	Offset Offset `json:"offset"`
}

// InBounds reports whether c is inside the level.
func (l *Level) InBounds(c Cell) bool {
	return c[0] >= 0 && c[1] >= 0 && c[0] < l.Size[0] && c[1] < l.Size[1]
}

// NewGrid returns a rows×columns grid filled with NoFrame.
func NewGrid(rows, columns int) Grid {
	g := make(Grid, rows)
	for x := range g {
		g[x] = make([]int, columns)
		for y := range g[x] {
			g[x][y] = NoFrame
		}
	}
	return g
}

// FloorType is a floor set variant and what the floor does.
type FloorType struct {
	Variant int `json:"variant"`
	// Enter acts on whoever enters the cell; Tick on whoever stands there, every status tick.
	Enter *Effect `json:"enter,omitempty"`
	Tick  *Effect `json:"tick,omitempty"`
	// Depth is water depth 1..4, 0 for none.
	Depth int `json:"depth,omitempty"`
	// Sinks loses objects put down here, except on a plate that is up.
	Sinks bool `json:"sinks,omitempty"`
	// Avoided floors are not walked onto by creatures.
	Avoided bool `json:"avoided,omitempty"`
}

// Effect is what a floor does to whoever is on it.
type Effect struct {
	Damage     Dice   `json:"damage,omitzero"`
	Type       string `json:"type,omitempty"`
	DrainMagic int    `json:"drain_magic,omitempty"`
	// Cue is the cue played when its damage hurts the player.
	Cue string `json:"cue,omitempty"`
}

// Feature kinds.
const (
	FeatureDoor       = "door"
	FeatureLever      = "lever"
	FeaturePlate      = "plate"
	FeatureSlot       = "slot"
	FeatureTeleporter = "teleporter"
	FeatureExit       = "exit"
	FeaturePit        = "pit"
	FeatureTrapdoor   = "trapdoor"
	FeatureSocket     = "socket"
	FeatureMouth      = "mouth"
	FeatureMark       = "mark"
)

// Door states.
const (
	DoorClosed  = "closed"
	DoorOpen    = "open"
	DoorOpening = "opening"
	DoorClosing = "closing"
)

// Hooks.
const (
	HookTurnOn   = "on_turn_on"
	HookTurnOff  = "on_turn_off"
	HookPress    = "on_press"
	HookRelease  = "on_release"
	HookInsert   = "on_insert"
	HookComplete = "on_complete"
)

// featureHooks are the hooks each kind of feature has.
var featureHooks = map[string][]string{
	FeatureDoor:       nil,
	FeatureLever:      {HookTurnOn, HookTurnOff},
	FeaturePlate:      {HookPress, HookRelease},
	FeatureSlot:       {HookInsert},
	FeatureTeleporter: nil,
	FeatureExit:       nil,
	FeaturePit:        nil,
	FeatureTrapdoor:   nil,
	FeatureSocket:     {HookComplete},
	FeatureMouth:      nil,
	FeatureMark:       nil,
}

// Feature is a cell's feature. Which fields apply depends on Kind (docs/pack.md, Features).
type Feature struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Cell Cell   `json:"cell"`
	Look string `json:"look,omitempty"`

	// Door. Sparkle is the flipbook a sealed door plays as it opens.
	State         string `json:"state,omitempty"`
	SealedBy      string `json:"sealed_by,omitempty"`
	OpenedBySpell bool   `json:"opened_by_spell,omitempty"`
	Sparkle       string `json:"sparkle,omitempty"`

	// Effect is the view effect (a flipbook) an exit plays as the player goes through it, the
	// level change waiting for it, or a socket plays when its last place is filled.
	Effect string `json:"effect,omitempty"`

	// Lever, plate, slot, teleporter.
	On        bool `json:"on,omitempty"`
	Armed     bool `json:"armed,omitempty"`
	Down      bool `json:"down,omitempty"`
	Threshold int  `json:"threshold,omitempty"`
	// Accepts is the item a slot takes. Keyhole marks a slot whose item is plainly a key: using
	// the slot puts it in from what the player carries. Other slots are object holes.
	Accepts string `json:"accepts,omitempty"`
	Keyhole bool   `json:"keyhole,omitempty"`
	// Full marks a slot that has taken its object and refuses any more: the world's state.
	Full bool `json:"full,omitempty"`

	// Teleporter.
	Destinations []Cell `json:"destinations,omitempty"`
	Next         int    `json:"next,omitempty"`
	Turn         int    `json:"turn,omitempty"`
	FlyOver      bool   `json:"fly_over,omitempty"`

	// Exit, pit, trapdoor.
	To   *Destination `json:"to,omitempty"`
	Open bool         `json:"open,omitempty"`

	// Socket: the item each place takes, and the places filled.
	Places []string `json:"places,omitempty"`
	Set    []int    `json:"set,omitempty"`

	// Mouth and mark.
	Message string `json:"message,omitempty"`
	Rune    string `json:"rune,omitempty"`

	// Hooks name procedures.
	OnTurnOn   string `json:"on_turn_on,omitempty"`
	OnTurnOff  string `json:"on_turn_off,omitempty"`
	OnPress    string `json:"on_press,omitempty"`
	OnRelease  string `json:"on_release,omitempty"`
	OnInsert   string `json:"on_insert,omitempty"`
	OnComplete string `json:"on_complete,omitempty"`
}

// Destination is a level and a cell in it.
type Destination struct {
	Level string `json:"level"`
	Cell  Cell   `json:"cell"`
}

// Hook returns the field holding hook name, or nil when the feature's kind has no such hook.
func (f *Feature) Hook(name string) *string {
	if !HasHook(f.Kind, name) {
		return nil
	}
	switch name {
	case HookTurnOn:
		return &f.OnTurnOn
	case HookTurnOff:
		return &f.OnTurnOff
	case HookPress:
		return &f.OnPress
	case HookRelease:
		return &f.OnRelease
	case HookInsert:
		return &f.OnInsert
	case HookComplete:
		return &f.OnComplete
	}
	return nil
}

// HasHook reports whether features of kind have hook name.
func HasHook(kind, name string) bool {
	for _, h := range featureHooks[kind] {
		if h == name {
			return true
		}
	}
	return false
}

// LookState returns the state of the feature's look that shows its state.
func (f *Feature) LookState() string {
	switch f.Kind {
	case FeatureDoor:
		switch {
		case f.SealedBy != "":
			return "sealed"
		case f.State == DoorOpen || f.State == DoorClosing:
			return DoorOpen
		}
		return DoorClosed
	case FeatureLever:
		return onOff(f.On, "on", "off")
	case FeaturePlate:
		return onOff(f.Down, "down", "up")
	case FeatureTeleporter:
		return onOff(f.Armed, "armed", "disarmed")
	case FeatureTrapdoor:
		return onOff(f.Open, "open", "closed")
	case FeatureSocket:
		if len(f.Set) == len(f.Places) {
			return "full"
		}
		return "empty"
	}
	return f.Kind
}

// SocketSetState is the look state drawn for a socket's filled place i.
func SocketSetState(i int) string { return fmt.Sprintf("set.%d", i) }

// lookStates returns the states a look used by f must have.
func (f *Feature) lookStates() []string {
	switch f.Kind {
	case FeatureDoor:
		s := []string{DoorClosed, DoorOpen}
		if f.SealedBy != "" {
			s = append(s, "sealed")
		}
		return s
	case FeatureLever:
		return []string{"off", "on"}
	case FeaturePlate:
		return []string{"up", "down"}
	case FeatureTeleporter:
		return []string{"armed", "disarmed"}
	case FeatureTrapdoor:
		return []string{"open", "closed"}
	case FeatureSocket:
		s := []string{"empty", "full"}
		for i := range f.Places {
			s = append(s, SocketSetState(i))
		}
		return s
	}
	return []string{f.Kind}
}

func onOff(on bool, yes, no string) string {
	if on {
		return yes
	}
	return no
}

// Action is one step of a procedure. Exactly one field is set.
type Action struct {
	On           *string       `json:"on,omitempty"`
	Off          *string       `json:"off,omitempty"`
	Toggle       *string       `json:"toggle,omitempty"`
	SetHook      *SetHook      `json:"set_hook,omitempty"`
	SetThreshold *SetThreshold `json:"set_threshold,omitempty"`
	SetAccepts   *SetAccepts   `json:"set_accepts,omitempty"`
	SetMessage   *SetMessage   `json:"set_message,omitempty"`
	Roll         *Roll         `json:"roll,omitempty"`
	Stop         *string       `json:"stop,omitempty"`
	ShootItem    *ShootItem    `json:"shoot_item,omitempty"`
	ShootSpell   *ShootSpell   `json:"shoot_spell,omitempty"`
	Create       *Create       `json:"create,omitempty"`
	SetFloor     *SetFloor     `json:"set_floor,omitempty"`
	SetWall      *SetWall      `json:"set_wall,omitempty"`

	// Spell effects, run with a cast's caster and power.
	Missile        *MissileSpell `json:"missile,omitempty"`
	FireItem       *string       `json:"fire_item,omitempty"`
	Timed          *string       `json:"timed,omitempty"`
	Conjure        *Conjure      `json:"conjure,omitempty"`
	Portal         *Portal       `json:"portal,omitempty"`
	TeleportToMark *string       `json:"teleport_to_mark,omitempty"`
	Dispel         *int          `json:"dispel,omitempty"`
	Join           *Join         `json:"join,omitempty"`
	Freeze         *int          `json:"freeze,omitempty"`
	ShapeChange    *int          `json:"shape_change,omitempty"`
	Transmute      *Transmute    `json:"transmute,omitempty"`
	CurePoison     bool          `json:"cure_poison,omitempty"`
	Restore        bool          `json:"restore,omitempty"`
	Heal           *Dice         `json:"heal,omitempty"`
	// Sparkle plays a sparkle (a flipbook) at the player.
	Sparkle *string `json:"sparkle,omitempty"`
	// Converse starts the conversation of a thing template once the spell is done.
	Converse *string `json:"converse,omitempty"`
	// RemoveAll removes the creatures of a thing template from the current level; Experience
	// gives the player experience.
	RemoveAll  *string `json:"remove_all,omitempty"`
	Experience *int    `json:"experience,omitempty"`

	// Conversation actions, run in a conversation.
	Say           *string     `json:"say,omitempty"`
	SayAs         *SayAs      `json:"say_as,omitempty"`
	Offer         *string     `json:"offer,omitempty"`
	ForgetOffers  bool        `json:"forget_offers,omitempty"`
	Take          *string     `json:"take,omitempty"`
	Give          *Object     `json:"give,omitempty"`
	SetCounter    *SetCounter `json:"set_counter,omitempty"`
	AddCounter    *string     `json:"add_counter,omitempty"`
	End           bool        `json:"end,omitempty"`
	RemoveSpeaker bool        `json:"remove_speaker,omitempty"`
	Hostile       bool        `json:"hostile,omitempty"`
	SetFrame      *int        `json:"set_frame,omitempty"`
	HealFully     bool        `json:"heal_fully,omitempty"`
	Run           *string     `json:"run,omitempty"`
	Teach         *string     `json:"teach,omitempty"`
	SetPortrait   *string     `json:"set_portrait,omitempty"`
	// Effect plays a view effect (a flipbook) at the speaker; in a spell's effects, at the player.
	Effect *string      `json:"effect,omitempty"`
	Travel *Destination `json:"travel,omitempty"`
	Return bool         `json:"return,omitempty"`
	Jump   *int         `json:"jump,omitempty"`
	Ending bool         `json:"ending,omitempty"`
	Menu   *string      `json:"menu,omitempty"`
	// Transcript prints the transcript's heading: TranscriptHeading or TranscriptContinued.
	Transcript *string `json:"transcript,omitempty"`
}

// SetHook points a feature's hook at a procedure, or at none.
type SetHook struct {
	Feature   string  `json:"feature"`
	Hook      string  `json:"hook"`
	Procedure *string `json:"procedure"`
}

// SetThreshold sets a plate's threshold.
type SetThreshold struct {
	Feature string `json:"feature"`
	Value   int    `json:"value"`
}

// SetAccepts sets the item a slot takes.
type SetAccepts struct {
	Feature string `json:"feature"`
	Item    string `json:"item"`
}

// SetMessage sets what a mouth says.
type SetMessage struct {
	Feature string `json:"feature"`
	Message string `json:"message"`
}

// Roll starts a boulder rolling.
type Roll struct {
	Thing     string `json:"thing"`
	Direction int    `json:"direction"`
	Shuttle   bool   `json:"shuttle,omitempty"`
}

// ShootItem fires a new object from a cell.
type ShootItem struct {
	Cell      Cell   `json:"cell"`
	Item      string `json:"item"`
	Direction int    `json:"direction"`
}

// ShootSpell casts a spell from a cell along a direction, its power rolled from Power.
type ShootSpell struct {
	Cell      Cell   `json:"cell"`
	Spell     string `json:"spell"`
	Direction int    `json:"direction"`
	Power     Dice   `json:"power"`
}

// Create puts a new object on a cell's floor.
type Create struct {
	Cell Cell   `json:"cell"`
	Item string `json:"item"`
}

// SetFloor makes a cell floor of a type, removing any wall.
type SetFloor struct {
	Cell Cell `json:"cell"`
	Type int  `json:"type"`
}

// SetWall makes a cell a wall of a shape.
type SetWall struct {
	Cell  Cell `json:"cell"`
	Shape int  `json:"shape"`
}

// PlacedThing is a thing a level places when it is first entered.
type PlacedThing struct {
	ID    string `json:"id"`
	Thing string `json:"thing"`
	Cell  Cell   `json:"cell"`
	// Offset moves the ground point from the cell's center, in world units.
	Offset *Offset `json:"offset,omitempty"`
	// Facing overrides the template's.
	Facing  *int     `json:"facing,omitempty"`
	Items   []Object `json:"items,omitempty"`
	Rolling *Rolling `json:"rolling,omitempty"`
}

// Rolling is a boulder's starting roll.
type Rolling struct {
	Direction int  `json:"direction"`
	Shuttle   bool `json:"shuttle,omitempty"`
}

// RandomItem is the item of an object chosen at random when its level is first entered.
const RandomItem = "random"

// Object is an object: its item, and its contents if it is a container.
type Object struct {
	Item     string   `json:"item"`
	Contents []Object `json:"contents,omitempty"`
}

// FloorObject is an object lying on a cell's floor.
type FloorObject struct {
	Cell Cell `json:"cell"`
	Object
}

// MapRefusal is what a level shows instead of its map: one of its fields is set.
type MapRefusal struct {
	Message      string `json:"message,omitempty"`
	Conversation string `json:"conversation,omitempty"`
}

// The transcript action's headings.
const (
	TranscriptHeading   = "heading"
	TranscriptContinued = "continued"
)
