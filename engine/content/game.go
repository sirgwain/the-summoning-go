package content

import "fmt"

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
	// Music is the track the frontend starts with its splash; StoryMusic the one Create
	// Character or a premade character starts, fading out as Story ends. Story plays when a
	// character is made or chosen, before Scene.
	Music      string `json:"music,omitempty"`
	Story      string `json:"story,omitempty"`
	StoryMusic string `json:"story_music,omitempty"`
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

// Scene is steps the presentation plays in order, on a picture of Size (SceneWidth ×
// SceneHeight when not given; one twice as tall shows each row half as tall).
type Scene struct {
	ID    string      `json:"id"`
	Size  [2]int      `json:"size,omitzero"`
	Steps []SceneStep `json:"steps"`
}

// Dims returns the size of the scene's picture.
func (s *Scene) Dims() (w, h int) {
	if s.Size[0] > 0 && s.Size[1] > 0 {
		return s.Size[0], s.Size[1]
	}
	return SceneWidth, SceneHeight
}

// SceneStep is one step of a scene: one of Picture, Part, Caption, Fade, Text, Say and ByWeapon,
// held MS. Sounds start as the step shows. Crop draws only that part (x, y, w, h) of Picture;
// Text is a page of words wrapped and centered in Box (x, y, w, h); ByWeapon plays the step at
// the character's weapon class.
type SceneStep struct {
	Picture  *Image      `json:"picture,omitempty"`
	At       Offset      `json:"at,omitzero"`
	Crop     *[4]int     `json:"crop,omitempty"`
	Clear    bool        `json:"clear,omitempty"`
	Part     *Part       `json:"part,omitempty"`
	Caption  string      `json:"caption,omitempty"`
	Banner   *Image      `json:"banner,omitempty"`
	Fade     string      `json:"fade,omitempty"`
	Text     string      `json:"text,omitempty"`
	Box      *[4]int     `json:"box,omitempty"`
	Say      *SceneSay   `json:"say,omitempty"`
	ByWeapon []SceneStep `json:"by_weapon,omitempty"`
	MS       int         `json:"ms,omitempty"`
	Sounds   []string    `json:"sounds,omitempty"`
}

// Waits reports whether a key or click ends the step early, rather than skipping the scene.
func (st *SceneStep) Waits() bool { return st.Text != "" || st.Say != nil }

// SceneSay is a line shown in a speaker's panel: Side "top" or "bottom", with Portrait (the
// character's own when empty).
type SceneSay struct {
	Portrait string `json:"portrait,omitempty"`
	Side     string `json:"side"`
	Text     string `json:"text"`
}

// Scene panel sides.
const (
	SayTop    = "top"
	SayBottom = "bottom"
)

// Part moves a picture's two halves apart (or together) over a background, a step at a time,
// each held StepMS, or the first ones LeadMS's. StepSounds start at each step.
type Part struct {
	Left       Image    `json:"left"`
	Right      Image    `json:"right"`
	Background *Image   `json:"background,omitempty"`
	From       int      `json:"from"`
	To         int      `json:"to"`
	StepMS     int      `json:"step_ms"`
	LeadMS     []int    `json:"lead_ms,omitempty"`
	StepSounds []string `json:"step_sounds,omitempty"`
}

// HoldMS returns how long step i shows.
func (p *Part) HoldMS(i int) int {
	if i >= 0 && i < len(p.LeadMS) {
		return p.LeadMS[i]
	}
	return p.StepMS
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
	v.music(owner+": music", c.Music)
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
	for _, id := range []string{c.Scene, c.Story} {
		if _, ok := v.p.Scenes[id]; id != "" && !ok {
			v.fail("%s: scene %q not found", owner, id)
		}
	}
	v.music(owner+": story_music", c.StoryMusic)
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
	if w, h := s.Dims(); w != SceneWidth || (h != SceneHeight && h != 2*SceneHeight) {
		v.fail("%s: size %dx%d", owner, w, h)
	}
	var check func(name string, st SceneStep, nested bool)
	check = func(name string, st SceneStep, nested bool) {
		n := 0
		for _, set := range []bool{st.Picture != nil, st.Part != nil, st.Caption != "", st.Fade != "",
			st.Text != "", st.Say != nil, len(st.ByWeapon) > 0} {
			if set {
				n++
			}
		}
		if n != 1 && !(n == 0 && st.MS > 0) {
			v.fail("%s: %s does %d things", owner, name, n)
		}
		img(st.Picture)
		img(st.Banner)
		for _, key := range []string{st.Caption, st.Text} {
			if key != "" {
				v.text(owner, key)
			}
		}
		if st.Text != "" && (st.Box == nil || st.Box[2] <= 0 || st.Box[3] <= 0) {
			v.fail("%s: %s has text but no box", owner, name)
		}
		if st.Crop != nil && (st.Picture == nil || st.Crop[2] <= 0 || st.Crop[3] <= 0) {
			v.fail("%s: %s crop", owner, name)
		}
		if sy := st.Say; sy != nil {
			v.text(owner, sy.Text)
			if sy.Side != SayTop && sy.Side != SayBottom {
				v.fail("%s: %s says on side %q", owner, name, sy.Side)
			}
			if _, ok := v.p.Portraits[sy.Portrait]; sy.Portrait != "" && !ok {
				v.fail("%s: %s: portrait %q not found", owner, name, sy.Portrait)
			}
		}
		if len(st.ByWeapon) > 0 {
			if nested {
				v.fail("%s: %s: by_weapon within by_weapon", owner, name)
			}
			for j, sub := range st.ByWeapon {
				check(fmt.Sprintf("%s weapon %d", name, j), sub, true)
			}
		}
		step := owner + " " + name
		for _, id := range st.Sounds {
			v.effect(step, id)
		}
		if p := st.Part; p != nil {
			img(&p.Left)
			img(&p.Right)
			img(p.Background)
			for _, id := range p.StepSounds {
				v.effect(step, id)
			}
		}
	}
	for i, st := range s.Steps {
		check(fmt.Sprintf("step %d", i), st, false)
	}
}
