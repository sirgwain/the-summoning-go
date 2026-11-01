package content

// Character creation and scenes (docs/pack.md, Creation and Scenes; docs/spec/scenes.md).

// Creation is what character creation offers and how a new game starts.
type Creation struct {
	ID string `json:"id"`
	// Portraits are those a new character chooses among.
	Portraits     []string          `json:"portraits"`
	Methods       []AttributeMethod `json:"methods"`
	WeaponClasses []Training        `json:"weapon_classes"`
	Schools       []Training        `json:"schools"`
	// Inventory is what every character starts with; BodyArmor the items one is chosen among for
	// the body.
	Inventory []InventoryObject `json:"inventory,omitempty"`
	BodyArmor []string          `json:"body_armor,omitempty"`
	Premade   []Premade         `json:"premade,omitempty"`
	// Counter is set to 1 by a new game; Opening's conversation starts on arrival; Scene plays
	// first.
	Counter string `json:"counter,omitempty"`
	Opening string `json:"opening,omitempty"`
	Scene   string `json:"scene,omitempty"`
	// Ending is how the game ends.
	Ending *Ending `json:"ending,omitempty"`
}

// Ending is how the game ends (docs/pack.md, Creation). The ending power takes the first choice
// whose item the player holds in a hand, sets Counter to its value and runs Conversation (a thing
// template's); when that ends, play ends with the choice's scene. The conversations' `ending`
// action ends play at once with Scene.
type Ending struct {
	Scene        string         `json:"scene,omitempty"`
	Counter      string         `json:"counter,omitempty"`
	Conversation string         `json:"conversation,omitempty"`
	Choices      []EndingChoice `json:"choices,omitempty"`
}

// EndingChoice is one of the endings the ending power chooses among.
type EndingChoice struct {
	Name    string `json:"name"`
	Holding string `json:"holding"`
	Value   int    `json:"value"`
	Scene   string `json:"scene,omitempty"`
}

// AttributeMethod is a way of making the attributes: each the sum of Roll's dice, at most Max,
// then Points' dice spent by hand.
type AttributeMethod struct {
	Name   string `json:"name"`
	Roll   []Dice `json:"roll,omitempty"`
	Points []Dice `json:"points,omitempty"`
	Max    int    `json:"max"`
}

// Training is a weapon class or school a new character can choose.
type Training struct {
	Name      string            `json:"name,omitempty"`
	Icon      *Image            `json:"icon,omitempty"`
	Icons     string            `json:"icons,omitempty"`
	Inventory []InventoryObject `json:"inventory,omitempty"`
}

// Premade is a premade character.
type Premade struct {
	Name        string         `json:"name"`
	Portrait    string         `json:"portrait"`
	Attributes  map[string]int `json:"attributes"`
	WeaponClass int            `json:"weapon_class"`
	School      int            `json:"school"`
}

// SceneWidth and SceneHeight are the size of a scene's picture.
const (
	SceneWidth  = 320
	SceneHeight = 200
)

// Scene is steps the presentation plays in order.
type Scene struct {
	ID    string      `json:"id"`
	Steps []SceneStep `json:"steps"`
}

// SceneStep is one step of a scene: one of Picture, Part, Caption and Fade, held MS.
type SceneStep struct {
	Picture *Image `json:"picture,omitempty"`
	At      Offset `json:"at,omitzero"`
	Clear   bool   `json:"clear,omitempty"`
	Part    *Part  `json:"part,omitempty"`
	Caption string `json:"caption,omitempty"`
	Banner  *Image `json:"banner,omitempty"`
	Fade    string `json:"fade,omitempty"`
	MS      int    `json:"ms,omitempty"`
}

// Part moves a picture's two halves apart (or together) over a background, a step at a time.
type Part struct {
	Left       Image  `json:"left"`
	Right      Image  `json:"right"`
	Background *Image `json:"background,omitempty"`
	From       int    `json:"from"`
	To         int    `json:"to"`
	StepMS     int    `json:"step_ms"`
}

// Steps returns how many steps the part takes.
func (p *Part) Steps() int {
	if p.To >= p.From {
		return p.To - p.From + 1
	}
	return p.From - p.To + 1
}

// Spread returns the spread of step i.
func (p *Part) Spread(i int) int {
	if p.To >= p.From {
		return p.From + i
	}
	return p.From - i
}

func (v *verifier) creation() {
	c := v.p.Creation
	if c == nil {
		return
	}
	const owner = "creation.json"
	for _, pt := range c.Portraits {
		if _, ok := v.p.Portraits[pt]; !ok {
			v.fail("%s: portrait %q not found", owner, pt)
		}
	}
	if len(c.Methods) == 0 {
		v.fail("%s: no attribute methods", owner)
	}
	inv := func(list []InventoryObject) {
		for _, io := range list {
			if io.Slot < 0 || io.Slot >= 28 {
				v.fail("%s: inventory slot %d", owner, io.Slot)
			}
			v.item(owner, io.Item)
		}
	}
	inv(c.Inventory)
	for _, list := range [][]Training{c.WeaponClasses, c.Schools} {
		for _, t := range list {
			if t.Name != "" {
				v.text(owner, t.Name)
			}
			if t.Icon != nil {
				v.frame(owner, t.Icon.Sprite, t.Icon.Frame)
			}
			v.icons(owner, t.Icons)
			inv(t.Inventory)
		}
	}
	if len(c.WeaponClasses) > 4 || len(c.Schools) > Schools {
		v.fail("%s: at most 4 weapon classes and %d schools", owner, Schools)
	}
	for _, it := range c.BodyArmor {
		v.item(owner, it)
	}
	for _, pm := range c.Premade {
		if _, ok := v.p.Portraits[pm.Portrait]; !ok {
			v.fail("%s: premade %s: portrait %q not found", owner, pm.Name, pm.Portrait)
		}
		if pm.WeaponClass < 0 || pm.WeaponClass >= len(c.WeaponClasses) || pm.School < 0 || pm.School >= len(c.Schools) {
			v.fail("%s: premade %s: no such weapon class or school", owner, pm.Name)
		}
	}
	if c.Opening != "" {
		if t := v.thingRef(owner, c.Opening); t != nil && t.Conversation == "" {
			v.fail("%s: %s holds no conversation", owner, c.Opening)
		}
	}
	if _, ok := v.p.Scenes[c.Scene]; c.Scene != "" && !ok {
		v.fail("%s: scene %q not found", owner, c.Scene)
	}
	if e := c.Ending; e != nil {
		scene := func(id string) {
			if _, ok := v.p.Scenes[id]; id != "" && !ok {
				v.fail("%s: ending scene %q not found", owner, id)
			}
		}
		scene(e.Scene)
		if e.Conversation != "" {
			if t := v.thingRef(owner, e.Conversation); t != nil && t.Conversation == "" {
				v.fail("%s: %s holds no conversation", owner, e.Conversation)
			}
		}
		for _, ch := range e.Choices {
			v.item(owner, ch.Holding)
			scene(ch.Scene)
		}
	}
}

func (v *verifier) scene(s *Scene) {
	owner := "scene " + s.ID
	img := func(im *Image) {
		if im != nil {
			v.frame(owner, im.Sprite, im.Frame)
		}
	}
	for i, st := range s.Steps {
		n := 0
		for _, set := range []bool{st.Picture != nil, st.Part != nil, st.Caption != "", st.Fade != ""} {
			if set {
				n++
			}
		}
		if n != 1 && !(n == 0 && st.MS > 0) {
			v.fail("%s: step %d does %d things", owner, i, n)
		}
		img(st.Picture)
		img(st.Banner)
		if st.Caption != "" {
			v.text(owner, st.Caption)
		}
		if p := st.Part; p != nil {
			img(&p.Left)
			img(&p.Right)
			img(p.Background)
		}
	}
}
