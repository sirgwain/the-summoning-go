package viewer

import (
	"fmt"
	"image"
	"math/rand/v2"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The frontend (docs/spec/scenes.md, The frontend): the main menu, character creation, the
// premade characters, restoring a game, and the scenes before and after play. It runs on input
// snapshots and gives the viewer a character to start a new game with, a save to restore, or a
// request to quit. The shell draws what it describes: a title, lines, a portrait and buttons.

// Frontend screens.
const (
	ScreenSplash     = "splash"
	ScreenMenu       = "menu"
	ScreenPortrait   = "portrait"
	ScreenName       = "name"
	ScreenMethod     = "method"
	ScreenAttributes = "attributes"
	ScreenWeapon     = "weapon"
	ScreenSchool     = "school"
	ScreenPremade    = "premade"
	ScreenRestore    = "restore"
	ScreenScene      = "scene"
)

// NameLength is the longest name a character may have.
const NameLength = 25

// Attributes are the six attributes in order.
var Attributes = []string{sim.AttrStrength, sim.AttrAgility, sim.AttrEndurance, sim.AttrAccuracy, sim.AttrTalent, sim.AttrPower}

// AttributeLabel returns attribute name's short label.
func AttributeLabel(p *content.Pack, name string) string { return inspect.UI(p, "ui.attr."+name) }

// Frontend is the frontend's state.
type Frontend struct {
	pack *content.Pack
	rng  *rand.Rand
	// Screen is the screen showing; Title, Lines, Portrait and Buttons what it shows, laid out
	// for a screen of Size. Choice is the button the keys have chosen.
	Screen   string
	Title    string
	Lines    []string
	Portrait *content.Image
	Buttons  []PanelButton
	Choice   int
	Size     image.Point
	// Scene is the scene playing on ScreenScene.
	Scene *SceneRun

	// Saves are the saves to restore from.
	Saves []string

	// The character being made: its portrait's place, the attribute method, the attributes
	// rolled (each the least it may go to), the points left and the attribute chosen.
	char     sim.Character
	portrait int
	method   int
	least    map[string]int
	points   int
	attr     int
	// splashLeft is the number of updates remaining on the opening splash.
	splashLeft int
	// step is how far the update's − or + moves an attribute: shiftStep with Shift held, else 1.
	step int
	// Rolls counts the rolls of the last run of Shift and R, shown at RollsShown (1 while
	// rolling and for a while after, then fading to 0). rolling is set while the run goes on;
	// rollsLeft is how many updates the count has left to show once it is done.
	Rolls      int
	RollsShown float64
	rolling    bool
	rollsLeft  int
	// then is the screen after the scene, and following a scene to play before it.
	then, following string
	// music are the music cues since the presentation last took them (TakeMusic): a track to
	// start, or "" to fade the music out.
	music []string
	// disk is the menu showing over play (a content.Menu… name), empty for the main frontend;
	// typed the name being typed on it.
	disk  string
	typed string

	// Done is set when the frontend has finished: with Character a new game, with Restore a save
	// to load, with Quit nothing; otherwise back to play as it was.
	Done      bool
	Character *sim.Character
	Restore   string
	Quit      bool
	// A menu's results: the name to save as, the save to delete, the save to rename and its new
	// name, and whether a question was answered yes.
	SaveAs, Delete, Rename, RenameTo string
	Confirmed                        bool
}

// NewFrontend returns the frontend of pack p at its main menu, offering saves to restore.
func NewFrontend(p *content.Pack, seed uint64, saves []string, size image.Point) *Frontend {
	f := &Frontend{pack: p, rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), Saves: saves, Size: size}
	f.show(ScreenMenu)
	return f
}

// OverPlay reports whether the frontend is a menu over play rather than the main frontend.
func (f *Frontend) OverPlay() bool { return f.disk != "" }

// The splash (docs/spec/scenes.md, The frontend): the artwork fades in, then its title fills in
// from the left, then the prompt fades in; it goes on to the menu after SplashUpdates.
const (
	SplashUpdates      = 7 * sim.StepsPerSecond
	SplashArtUpdates   = sim.StepsPerSecond
	SplashTitleUpdates = 3 * sim.StepsPerSecond / 2
	SplashPromptUpdate = SplashArtUpdates + SplashTitleUpdates
	SplashPromptFade   = sim.StepsPerSecond / 2
)

// ShowSplash displays the opening artwork before the main menu.
func (f *Frontend) ShowSplash() {
	f.splashLeft = SplashUpdates
	f.show(ScreenSplash)
}

// SplashShown is how many updates the splash has shown.
func (f *Frontend) SplashShown() int { return SplashUpdates - f.splashLeft }

// PlayScene shows scene id, then goes to screen then ("" to finish).
func (f *Frontend) PlayScene(id, then string) {
	f.then = then
	if s := f.pack.Scenes[id]; s != nil {
		f.Scene = NewSceneRun(s, f.viewer())
		f.show(ScreenScene)
		return
	}
	f.after()
}

// viewer returns who watches a scene: the character being begun, if any.
func (f *Frontend) viewer() SceneViewer {
	if ch := f.Character; ch != nil {
		return SceneViewer{Name: ch.Name, Portrait: ch.Portrait, WeaponClass: ch.WeaponClass}
	}
	return SceneViewer{}
}

// TakeMusic returns the music cues since it was last called: a track to start, or "" to fade
// the music out.
func (f *Frontend) TakeMusic() []string {
	m := f.music
	f.music = nil
	return m
}

// cue asks for track id to start ("" to fade the music out), when the pack names one.
func (f *Frontend) cue(id string) {
	if c := f.creation(); c != nil && c.StoryMusic != "" {
		f.music = append(f.music, id)
	}
}

// after leaves a scene: the story's music fades as the story ends, and the scene following
// plays.
func (f *Frontend) after() {
	if c := f.creation(); f.Scene != nil && c != nil && f.Scene.Def.ID == c.Story {
		f.cue("")
	}
	f.Scene = nil
	if id := f.following; id != "" {
		f.following = ""
		f.PlayScene(id, f.then)
		return
	}
	if f.then == "" {
		f.Done = true
		return
	}
	f.show(f.then)
}

func (f *Frontend) creation() *content.Creation { return f.pack.Creation }

// ui returns label key in words (inspect.UI).
func (f *Frontend) ui(key string, args ...any) string { return inspect.UI(f.pack, key, args...) }

// show goes to screen s.
func (f *Frontend) show(s string) {
	f.Screen, f.Choice = s, 0
	if s == ScreenAttributes && f.least == nil {
		f.roll()
	}
	f.layout()
}

// option is a button of a list: its name and label.
type option struct{ name, label string }

// layout fills in what the screen shows.
func (f *Frontend) layout() {
	f.Title, f.Lines, f.Portrait, f.Buttons = "", nil, nil, nil
	if f.layoutDisk() {
		return
	}
	c := f.creation()
	var opts []option
	switch f.Screen {
	case ScreenMenu:
		f.Title = f.pack.Manifest.Name
		if c != nil && len(c.Portraits) > 0 {
			opts = append(opts, option{"create", f.ui("ui.menu.create")})
		}
		if len(f.Saves) > 0 {
			opts = append(opts, option{"restore", f.ui("ui.menu.restore")})
		}
		switch {
		case c != nil && len(c.Portraits) == 0 && len(c.Premade) == 1:
			// Without character creation the one premade character is the new game.
			opts = append([]option{{"new", f.ui("ui.menu.new")}}, opts...)
		case c != nil && len(c.Premade) > 0:
			opts = append(opts, option{"premade", f.ui("ui.menu.premade")})
		}
		opts = append(opts, option{"quit", f.ui("ui.menu.quit")})
	case ScreenPortrait:
		f.Title = f.ui("ui.portrait.title")
		f.Lines = []string{f.ui("ui.portrait.help")}
		f.Portrait = f.portraitImage(c.Portraits[f.portrait])
		opts = []option{{"prev", "<"}, {"next", ">"}, {"random", f.ui("ui.random")}, {"ok", f.ui("ui.choose")}, {"back", f.ui("ui.back")}}
	case ScreenName:
		f.Title = f.ui("ui.name.title")
		f.Lines = []string{f.char.Name + "_", "", f.ui("ui.name.help", "n", NameLength)}
		opts = []option{{"ok", f.ui("ui.go_on")}, {"back", f.ui("ui.back")}}
	case ScreenMethod:
		f.Title = f.ui("ui.method.title")
		for i, m := range c.Methods {
			opts = append(opts, option{fmt.Sprintf("method.%d", i), m.Name})
		}
		opts = append(opts, option{"back", f.ui("ui.back")})
	case ScreenAttributes:
		f.layoutAttributes()
		return
	case ScreenWeapon, ScreenSchool:
		list := c.WeaponClasses
		f.Title = f.ui("ui.weapon.title")
		if f.Screen == ScreenSchool {
			list, f.Title = c.Schools, f.ui("ui.school.title")
		}
		for i, t := range list {
			opts = append(opts, option{fmt.Sprintf("train.%d", i), f.trainingName(t, f.Screen == ScreenSchool, i)})
		}
		opts = append(opts, option{"back", f.ui("ui.back")})
	case ScreenPremade:
		f.Title = f.ui("ui.menu.premade")
		for i, pm := range c.Premade {
			opts = append(opts, option{fmt.Sprintf("premade.%d", i), pm.Name})
		}
		opts = append(opts, option{"back", f.ui("ui.back")})
		if f.Choice < len(c.Premade) {
			pm := c.Premade[f.Choice]
			f.Portrait = f.portraitImage(pm.Portrait)
			f.Lines = []string{f.attributeLine(pm.Attributes)}
		}
	case ScreenRestore:
		f.Title = f.ui("ui.menu.restore")
		for i, s := range f.Saves {
			opts = append(opts, option{fmt.Sprintf("save.%d", i), s})
		}
		opts = append(opts, option{"back", f.ui("ui.back")})
	case ScreenScene:
		return
	}
	f.list(opts)
}

// list lays out opts as a column of buttons below the title, lines and portrait.
func (f *Frontend) list(opts []option) {
	w := 0
	for _, o := range opts {
		w = max(w, len(o.label))
	}
	w = (w + 6) * MenuCharWidth
	y := f.top()
	if f.Screen == ScreenMenu && f.disk == "" {
		// Under the splash's lettering.
		y = f.Size.Y * 3 / 8
	}
	for _, o := range opts {
		x := (f.Size.X - w) / 2
		f.Buttons = append(f.Buttons, PanelButton{Rect: image.Rect(x, y, x+w, y+MenuLineHeight+4), Name: o.name, Label: o.label})
		y += MenuLineHeight + 8
	}
}

// PortraitScale is how much the frontend magnifies portraits.
const PortraitScale = 3

// top returns where the buttons start: below the title, the lines and the portrait.
func (f *Frontend) top() int {
	y := f.Size.Y/5 + 2*MenuLineHeight + len(f.Lines)*MenuLineHeight + MenuLineHeight
	if f.Portrait != nil {
		y += f.portraitSize().Y*PortraitScale + MenuLineHeight
	}
	return y
}

// TitleAt is where the title's line is; LinesAt the first line; PortraitAt the portrait's corner.
func (f *Frontend) TitleAt() int { return f.Size.Y / 5 }
func (f *Frontend) LinesAt() int { return f.Size.Y/5 + 2*MenuLineHeight }
func (f *Frontend) PortraitAt() image.Point {
	s := f.portraitSize().Mul(PortraitScale)
	return image.Pt((f.Size.X-s.X)/2, f.LinesAt()+len(f.Lines)*MenuLineHeight+MenuLineHeight/2)
}

func (f *Frontend) portraitSize() image.Point {
	if f.Portrait == nil {
		return image.Point{}
	}
	if b := f.pack.Sprites[f.Portrait.Sprite]; b != nil {
		return image.Pt(b.FrameSize[0], b.FrameSize[1])
	}
	return image.Point{}
}

func (f *Frontend) portraitImage(id string) *content.Image {
	if pt := f.pack.Portraits[id]; pt != nil {
		img := pt.Image
		return &img
	}
	return nil
}

// beginPremade begins a new game as premade character pm.
func (f *Frontend) beginPremade(pm content.Premade) {
	f.cue(f.creation().StoryMusic)
	f.begin(sim.Character{Name: pm.Name, Portrait: pm.Portrait, Attributes: pm.Attributes,
		WeaponClass: pm.WeaponClass, School: pm.School})
}

// trainingName returns a weapon class's or school's name.
func (f *Frontend) trainingName(t content.Training, school bool, i int) string {
	if t.Name != "" {
		return inspect.Text(f.pack, t.Name)
	}
	if school {
		return f.ui("ui.school", "n", i+1)
	}
	return f.ui(fmt.Sprintf("ui.weapon.%d", min(i, 3)))
}

func (f *Frontend) attributeLine(a map[string]int) string {
	var parts []string
	for _, name := range Attributes {
		parts = append(parts, fmt.Sprintf("%s %d", AttributeLabel(f.pack, name), a[name]))
	}
	return strings.Join(parts, "  ")
}

// Attribute rows' buttons are laid out by layoutAttributes.
func (f *Frontend) layoutAttributes() {
	m := f.creation().Methods[f.method]
	f.Title = m.Name
	a := f.char.Attributes
	f.Lines = []string{
		fmt.Sprintf("%s %d   %s %d", f.ui("ui.hit_points"), a[sim.AttrStrength]+a[sim.AttrEndurance]+20,
			f.ui("ui.magic_points"), a[sim.AttrTalent]+a[sim.AttrPower]+20),
	}
	if len(m.Points) > 0 {
		f.Lines = append(f.Lines, f.ui("ui.points_to_spend", "n", f.points))
	}
	w := 24 * MenuCharWidth
	x := (f.Size.X - w) / 2
	y := f.top()
	for i, name := range Attributes {
		row := image.Rect(x, y, x+w, y+MenuLineHeight+4)
		f.Buttons = append(f.Buttons, PanelButton{Rect: row, Name: fmt.Sprintf("attr.%d", i),
			Label: fmt.Sprintf("%s %2d", AttributeLabel(f.pack, name), a[name])})
		if len(m.Points) > 0 {
			bw := 3 * MenuCharWidth
			f.Buttons = append(f.Buttons,
				PanelButton{Rect: image.Rect(row.Max.X+4, y, row.Max.X+4+bw, row.Max.Y), Name: fmt.Sprintf("dec.%d", i), Label: "-"},
				PanelButton{Rect: image.Rect(row.Max.X+8+bw, y, row.Max.X+8+2*bw, row.Max.Y), Name: fmt.Sprintf("inc.%d", i), Label: "+"})
		}
		y += MenuLineHeight + 8
	}
	var opts []option
	if len(m.Roll) > 0 && len(m.Points) == 0 {
		opts = append(opts, option{"reroll", f.ui("ui.reroll")})
	}
	if f.points == 0 {
		opts = append(opts, option{"ok", f.ui("ui.keep")})
	}
	opts = append(opts, option{"back", f.ui("ui.back")})
	for _, o := range opts {
		f.Buttons = append(f.Buttons, PanelButton{Rect: image.Rect(x, y, x+w, y+MenuLineHeight+4), Name: o.name, Label: o.label})
		y += MenuLineHeight + 8
	}
	f.Choice = min(f.Choice, len(f.Buttons)-1)
}

// roll makes the attributes by the chosen method.
func (f *Frontend) roll() {
	m := f.creation().Methods[f.method]
	f.char.Attributes, f.least = map[string]int{}, map[string]int{}
	for _, name := range Attributes {
		v := 0
		for _, d := range m.Roll {
			v += f.dice(d)
		}
		v = min(v, m.Max)
		f.char.Attributes[name], f.least[name] = v, v
	}
	f.points = 0
	for _, d := range m.Points {
		f.points += f.dice(d)
	}
}

func (f *Frontend) dice(d content.Dice) int {
	n := d.Bonus
	for range d.Count {
		if d.Sides > 0 {
			n += 1 + f.rng.IntN(d.Sides)
		}
	}
	return n
}

// shiftStep is how far Shift moves an attribute at once.
const shiftStep = 10

// A run of Shift and R rolls rollsPerUpdate times an update until the total is at least
// rollTarget; its count then shows rollsHold updates and fades over rollsFade.
const (
	rollTarget     = 100
	rollsPerUpdate = 100
	rollsHold      = 2 * sim.StepsPerSecond
	rollsFade      = 1 * sim.StepsPerSecond
)

// total is the attributes' sum with the points still to spend: what they come to once spent.
func (f *Frontend) total() int {
	n := f.points
	for _, name := range Attributes {
		n += f.char.Attributes[name]
	}
	return n
}

// rollDone reports whether a run of Shift and R can stop: the total reaches rollTarget, and
// every point can be spent.
func (f *Frontend) rollDone() bool {
	t := f.total()
	return t >= rollTarget && t <= len(Attributes)*f.creation().Methods[f.method].Max
}

// canReach reports whether the method's dice and maximum can make a total of rollTarget.
func (f *Frontend) canReach() bool {
	m := f.creation().Methods[f.method]
	most := func(ds []content.Dice) int {
		n := 0
		for _, d := range ds {
			n += d.Count*d.Sides + d.Bonus
		}
		return n
	}
	best := min(len(Attributes)*min(most(m.Roll), m.Max)+most(m.Points), len(Attributes)*m.Max)
	return best >= rollTarget
}

// tickRolls runs a run of Shift and R for an update, or fades its count.
func (f *Frontend) tickRolls() {
	if f.Screen != ScreenAttributes {
		f.rolling, f.rollsLeft = false, 0
	}
	if f.rolling {
		for range rollsPerUpdate {
			f.roll()
			f.Rolls++
			if f.rollDone() {
				f.rolling, f.rollsLeft = false, rollsHold+rollsFade
				break
			}
		}
	} else if f.rollsLeft > 0 {
		f.rollsLeft--
	}
	switch {
	case f.rolling:
		f.RollsShown = 1
	default:
		f.RollsShown = min(1, float64(f.rollsLeft)/rollsFade)
	}
}

// spend moves attribute i by up to delta, as far as its least, the method's maximum and the
// points allow.
func (f *Frontend) spend(i, delta int) {
	m := f.creation().Methods[f.method]
	name := Attributes[i]
	v := f.char.Attributes[name]
	to := max(f.least[name], min(v+delta, m.Max, v+f.points))
	f.char.Attributes[name] = to
	f.points -= to - v
}

// Update runs the frontend for one update.
func (f *Frontend) Update(in Input) {
	if f.Done {
		return
	}
	if f.Screen == ScreenSplash {
		f.splashLeft--
		if in.anyPressed() || in.Left.Released || in.Right.Released || f.splashLeft <= 0 {
			f.show(ScreenMenu)
		}
		return
	}
	if f.Screen == ScreenScene {
		r := f.Scene
		pressed := in.anyPressed() || in.Left.Released || in.Right.Released
		if (pressed && r.Press(in.pressed(KeyEscape))) || (!pressed && r.Update()) {
			f.after()
		}
		return
	}
	f.step = 1
	if in.held(KeyShift) {
		f.step = shiftStep
	}
	pick := ""
	if in.Left.Released {
		for i, b := range f.Buttons {
			if in.Cursor.In(b.Rect) {
				// A save clicked again while chosen is saved over.
				if f.Screen == ScreenSave && i == f.Choice && strings.HasPrefix(b.Name, "save.") && f.typed == b.Label {
					pick = "ok"
				} else {
					f.Choice, pick = i, b.Name
				}
			}
		}
	}
	n := len(f.Buttons)
	switch {
	case n == 0:
	case in.pressed(KeyArrowDown), in.pressed(KeyArrowUp):
		if in.pressed(KeyArrowDown) {
			f.Choice = (f.Choice + 1) % n
		} else {
			f.Choice = (f.Choice + n - 1) % n
		}
		// Moving onto a save to replace takes its name.
		if b := f.Buttons[f.Choice]; f.Screen == ScreenSave && strings.HasPrefix(b.Name, "save.") {
			pick = b.Name
		}
	case in.pressed(KeyEnter):
		switch {
		case (f.Screen == ScreenName || f.typing()) && f.Buttons[f.Choice].Name == "back":
			pick = "back"
		case f.Screen == ScreenName || f.typing():
			pick = "ok"
		default:
			pick = f.Buttons[f.Choice].Name
		}
	case in.pressed(KeyEscape):
		pick = "back"
	}
	switch f.Screen {
	case ScreenPortrait:
		switch {
		case in.pressed(KeyArrowLeft):
			pick = "prev"
		case in.pressed(KeyArrowRight):
			pick = "next"
		case in.pressed("R"):
			pick = "random"
		}
	case ScreenName:
		f.typeName(in)
	case ScreenSave, ScreenRenameTo:
		f.typeInto(in)
	case ScreenAttributes:
		if f.Choice < len(Attributes)*f.attrButtons() {
			i := f.Choice / f.attrButtons()
			switch {
			case in.pressed(KeyArrowLeft):
				pick = fmt.Sprintf("dec.%d", i)
			case in.pressed(KeyArrowRight):
				pick = fmt.Sprintf("inc.%d", i)
			}
		}
		if in.pressed("R") {
			if in.held(KeyShift) && f.canReach() {
				f.rolling, f.Rolls = true, 0
			} else {
				pick = "reroll"
			}
		}
	}
	if pick != "" {
		f.pick(pick)
	}
	f.tickRolls()
	f.layout()
}

// attrButtons is how many buttons each attribute row has.
func (f *Frontend) attrButtons() int {
	if len(f.creation().Methods[f.method].Points) > 0 {
		return 3
	}
	return 1
}

// typeName takes the characters typed into the name.
func (f *Frontend) typeName(in Input) {
	for _, r := range in.Chars {
		if r >= ' ' && r < 0x7F && len(f.char.Name) < NameLength {
			f.char.Name += string(r)
		}
	}
	if in.pressed(KeyBackspace) && f.char.Name != "" {
		f.char.Name = f.char.Name[:len(f.char.Name)-1]
	}
}

// pick acts on button name.
func (f *Frontend) pick(name string) {
	c := f.creation()
	var i int
	num := func(prefix string) bool {
		if !strings.HasPrefix(name, prefix) {
			return false
		}
		_, err := fmt.Sscanf(name[len(prefix):], "%d", &i)
		return err == nil
	}
	if f.pickDisk(name, func(prefix string) (int, bool) { ok := num(prefix); return i, ok }) {
		return
	}
	switch f.Screen {
	case ScreenMenu:
		switch name {
		case "create":
			f.char, f.least, f.portrait = sim.Character{}, nil, 0
			f.cue(f.creation().StoryMusic)
			f.show(ScreenPortrait)
		case "restore":
			f.show(ScreenRestore)
		case "premade":
			f.show(ScreenPremade)
		case "new":
			f.beginPremade(c.Premade[0])
		case "quit":
			f.Done, f.Quit = true, true
		}
	case ScreenPortrait:
		n := len(c.Portraits)
		switch name {
		case "prev":
			f.portrait = (f.portrait + n - 1) % n
		case "next":
			f.portrait = (f.portrait + 1) % n
		case "random":
			f.portrait = f.rng.IntN(n)
		case "ok":
			f.char.Portrait = c.Portraits[f.portrait]
			f.show(ScreenName)
		case "back":
			f.cue("")
			f.show(ScreenMenu)
		}
	case ScreenName:
		switch name {
		case "ok":
			if strings.TrimSpace(f.char.Name) == "" {
				f.char.Name = f.ui("ui.stranger")
			}
			f.show(ScreenMethod)
		case "back":
			f.show(ScreenPortrait)
		}
	case ScreenMethod:
		switch {
		case num("method."):
			f.method, f.least = i, nil
			f.show(ScreenAttributes)
		case name == "back":
			f.show(ScreenName)
		}
	case ScreenAttributes:
		switch {
		case num("dec."):
			f.spend(i, -f.step)
		case num("inc."):
			f.spend(i, f.step)
		case name == "reroll":
			f.roll()
		case name == "ok" && f.points == 0:
			f.show(ScreenWeapon)
		case name == "back":
			f.least = nil
			f.show(ScreenMethod)
		}
	case ScreenWeapon:
		switch {
		case num("train."):
			f.char.WeaponClass = i
			f.show(ScreenSchool)
		case name == "back":
			f.show(ScreenAttributes)
		}
	case ScreenSchool:
		switch {
		case num("train."):
			f.char.School = i
			f.begin(f.char)
		case name == "back":
			f.show(ScreenWeapon)
		}
	case ScreenPremade:
		switch {
		case num("premade."):
			f.beginPremade(c.Premade[i])
		case name == "back":
			f.show(ScreenMenu)
		}
	case ScreenRestore:
		switch {
		case num("save."):
			f.Restore, f.Done = f.Saves[i], true
		case name == "back" && f.disk != "":
			f.Done = true
		case name == "back":
			f.show(ScreenMenu)
		}
	}
}

// begin finishes with character ch, after the story and the new game's scene.
func (f *Frontend) begin(ch sim.Character) {
	f.Character = &ch
	c := f.creation()
	if c.Story != "" {
		if _, ok := f.pack.Scenes[c.Story]; ok {
			f.following = c.Scene
			f.PlayScene(c.Story, "")
			return
		}
	}
	f.cue("")
	f.PlayScene(c.Scene, "")
}
