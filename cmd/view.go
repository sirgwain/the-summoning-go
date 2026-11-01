package cmd

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/audio/speaker"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newViewCmd(false), newViewCmd(true))
}

// newViewCmd returns the view command, or with play the play command, which opens the same
// window driving the player.
func newViewCmd(play bool) *cobra.Command {
	var level, screenshot, script, out, saves, load string
	var scale, shotTicks int
	var magnify, menu, mute bool
	var at, sel []int
	var seed uint64
	use, short, long := "view [pack-dir]", "Look around a content pack's levels and work their mechanisms",
		`Opens a window showing a pack's levels in the game's projection, running the simulation.
F12 switches to playing (see summoning play --help).
`
	if play {
		use, short, long = "play [pack-dir]", "Play a content pack's levels",
			`Opens a window playing a pack: the player starts where the pack says and the view follows it,
showing what the player sees.

A pack with character creation starts at the main menu (--menu=false skips it): create a
character (the descendant, the name, the attributes rolled or spent, the weapon and the school),
restore a saved game, or select a premade character. Arrows and Enter or the mouse choose;
Escape goes back. A new game shows the pack's opening scene (a key or click skips it). When the
player dies, or the game ends, play returns to the main menu.

The keys are laid out as the game's. WASD or the arrows walk, two together along a diagonal;
Home, PgUp, End and PgDn walk the diagonals. Space acts with the right hand and Shift+Space does
its second action, X throws what it holds; Q, Shift+Q and Z do the same with the left hand. F
takes the top object of the player's cell to the first empty shelf, G puts down what is in hand,
Enter pulls the lever underfoot or puts the key a keyhole underfoot takes into it, E or C talks to
the nearest creature that talks (or hears the nearest mouth), H recenters the view, P pauses.
Holding N naps: the world runs 32 times as fast, to get magic points back, until N is released,
the magic points are full, a creature comes near, the player is hurt, a message or conversation
starts, or another key or a click comes; it will not start with full magic points or a creature
near.
F5 saves the game and F8 loads it again; F9 cycles music and effects, effects only, and silence; changing level saves it too (the autosave). O opens the
disk menu: save under a name, restore, delete or rename a save, leave for the main menu, or quit. M shows the
level's map, with a legend (a key or click closes it); the minimap in the top-left corner shows it small. Some
levels refuse the map.

F1-F4 (or 1-4) cast the selected spell of Wizardry, Sorcery, Enchantment and Healing; with Shift
they select the school's next memorized spell. F10 opens the spell panel: type the letters A-L of
a gesture (or click its icons), Backspace takes one off, Enter or Learn tries it; clicking a spell
of the spellbook memorizes it once more; Escape closes it.

In a conversation, a key or click goes on past each line, and clicking a marked word, or typing
its first letter, asks it next. Then click an offered keyword, or type a word and press Enter;
Escape, a right click or Bye leaves.

The hotbar along the bottom shows the portrait, hit points, magic points and load, the hands
with their action boxes (each hand's actions, then throwing), and the twenty shelves. Clicking
the player takes the top object underfoot onto the pointer, or puts the pointer's object down
(stand on a keyhole and click the player to put a key in); clicking a hand or shelf swaps it with
the pointer; clicking an action box acts with that hand. The right button puts the pointer's object on the first empty shelf, or takes the
object underfoot straight there. Shift and a click reads an object (a shelf, a hand, or the
pointer's on the player): a scroll shows its picture and teaches its icons, as selecting it
(taking it from a shelf onto the pointer) does. The four boxes after
the shelves are the schools' selected spells: click to cast, right click for the next; the box at
the end shows the gesture playing, and opens the spell panel. Over a creature that talks, or a
mouth (a skull), the pointer becomes a speech bubble: a click talks to the creature, or shows the
mouth's message beside it. Taking a scroll onto the pointer reads it.

I or Tab, or clicking the portrait, opens the inventory panel (Escape closes it): the figure
showing what it wears, with a box for each worn slot (head, body, arms, feet on the left;
quiver, neck, right hand, left hand on the right), the pack (an opened container, or a bag held in the left hand), the details
of the object under the pointer or on it, and the character's attributes, points, armor, level,
experience and skills. Its boxes work as the hotbar's: a click swaps a box with the pointer.
A right click on a bag or chest (on the hotbar or the panel, with nothing on the pointer) opens
it as the pack, where its contents can be taken and put; a second right click closes it, and
the bag in the left hand is the pack again. Resting the pointer on a container shows what it
holds.

V shows the whole level, F12 switches to the viewer's tools (see summoning view --help),
clicking a cell inspects it, ~ shows or hides the debug overlay (hidden at first; --debug shows
it): the text, and the outlines of the selection and the cell under the mouse.
`
	}
	viewCmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long: long + `
The tools:

Arrows or WASD (with Shift, faster) or dragging with the mouse move the view. Scrolling zooms
around the cursor, as do + and - around the center. Page Up and Page Down (or [ and ]) change
level, Home recenters, M magnifies objects on the floor.

Click a cell or a thing to inspect it. Then U uses it, T toggles its feature, 1–9 run its
hooks' procedures, E follows its exit, O drops the chosen object on it and X takes its top
object; I and Shift+I choose the object. P places the chosen thing, C and Shift+C choose it, and
Shift+click moves the selected thing to the clicked cell. Space pauses, N steps while paused.

--script runs a script of input, screenshots and state dumps (docs/design.md, Testing) in place
of live input, and exits when it ends. Its files are written under --out.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "packs/summoning"
			if len(args) > 0 {
				dir = args[0]
			}
			pack, err := content.Load(os.DirFS(dir))
			if err != nil {
				return err
			}
			cell := func(name string, xy []int) (*content.Cell, error) {
				switch len(xy) {
				case 0:
					return nil, nil
				case 2:
					return &content.Cell{xy[0], xy[1]}, nil
				}
				return nil, fmt.Errorf("--%s wants a cell as x,y", name)
			}
			center, err := cell("at", at)
			if err != nil {
				return err
			}
			selected, err := cell("select", sel)
			if err != nil {
				return err
			}
			var steps viewer.Script
			if script != "" {
				if steps, err = viewer.LoadScript(script); err != nil {
					return err
				}
			}
			if screenshot != "" {
				steps = append(steps, viewer.Step{At: int64(shotTicks), Shot: screenshot})
			}
			v, err := viewer.New(pack, seed, level, scale, magnify, image.Pt(windowWidth, windowHeight))
			if err != nil {
				return err
			}
			v.SaveDir = saves
			v.HideDebug = play && !debugEnabled
			if !mute && script == "" && screenshot == "" {
				v.Audio = audio.New(speaker.New(), pack)
			}
			if v.SaveDir == "" && script == "" {
				v.SaveDir = save.DefaultDir(pack.Manifest.ID)
			}
			if load != "" {
				if err := v.LoadFile(savePath(load, v.SaveDir)); err != nil {
					return err
				}
				menu = false
			}
			if play {
				if v.World.Player() == nil {
					return fmt.Errorf("pack %s has no player", pack.Manifest.ID)
				}
				v.SetPlay(true)
				if menu && script == "" && pack.Creation != nil {
					v.OpenFrontend()
				}
			}
			if center != nil {
				v.CenterOn(*center)
			}
			if selected != nil {
				v.Select(*selected)
				if center == nil {
					v.CenterOn(*selected)
				}
			}
			s, err := newShell(v)
			if err != nil {
				return err
			}
			if steps != nil {
				s.script = viewer.NewRunner(steps, out)
			}
			ebiten.SetWindowTitle("The Summoning: " + pack.Manifest.Name)
			ebiten.SetWindowSize(windowWidth, windowHeight)
			ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
			if err := ebiten.RunGame(s); err != nil && !errors.Is(err, ebiten.Termination) {
				return err
			}
			return nil
		},
	}
	viewCmd.Flags().StringVar(&level, "level", "", "level id to open (default: the start level)")
	viewCmd.Flags().IntVar(&scale, "scale", 2, "pixels per art pixel")
	viewCmd.Flags().IntSliceVar(&at, "at", nil, "center the view on this cell, as x,y")
	viewCmd.Flags().IntSliceVar(&sel, "select", nil, "select this cell, as x,y")
	viewCmd.Flags().BoolVar(&magnify, "magnify", false, "start with objects on the floor magnified")
	viewCmd.Flags().Uint64Var(&seed, "seed", 1, "seed for the simulation's random choices")
	viewCmd.Flags().StringVar(&script, "script", "", "run this script instead of live input, then exit")
	viewCmd.Flags().StringVar(&out, "out", ".", "directory for a script's screenshots and dumps")
	viewCmd.Flags().StringVar(&load, "load", "", "start from the saved game in this file, looked for in the saves directory when not found as given")
	viewCmd.Flags().StringVar(&saves, "saves", "", "directory for saved games (default: the user's config directory; none while a script runs)")
	if play {
		viewCmd.Flags().BoolVar(&menu, "menu", true, "start at the main menu when the pack has character creation (not while a script runs)")
	}
	viewCmd.Flags().BoolVar(&mute, "mute", false, "play no music or sound effects (none while a script runs)")
	viewCmd.Flags().StringVar(&screenshot, "screenshot", "", "save a frame to this PNG file and exit")
	viewCmd.Flags().IntVar(&shotTicks, "screenshot-ticks", 0, "updates to run before the screenshot")
	return viewCmd
}

const windowWidth, windowHeight = 1280, 800

var (
	panelColor     = color.RGBA{0, 0, 0, 200}
	selectionColor = color.RGBA{255, 220, 0, 255}
	hoverColor     = color.RGBA{255, 255, 255, 160}
	thingColor     = color.RGBA{0, 220, 255, 255}
	damageColor    = color.RGBA{170, 20, 20, 230}
	hotbarColor    = color.RGBA{12, 10, 8, 240}
	slotColor      = color.RGBA{40, 36, 32, 230}
	slotEdgeColor  = color.RGBA{150, 130, 100, 255}
	handEdgeColor  = color.RGBA{220, 190, 120, 255}
	// openEdgeColor marks the box of the container open as the pack.
	openEdgeColor  = color.RGBA{90, 220, 120, 255}
	actionColor    = color.RGBA{70, 50, 30, 230}
	waitColor      = color.RGBA{40, 40, 40, 230}
	gaugeColors    = [...]color.RGBA{{200, 30, 30, 255}, {40, 90, 220, 255}, {200, 160, 40, 255}}
	overloadColor  = color.RGBA{255, 80, 0, 255}
	spellEdgeColor = color.RGBA{120, 140, 220, 255}
	talkColor      = color.RGBA{16, 12, 10, 235}
	keywordColor   = color.RGBA{230, 200, 90, 255}
	waitShade      = color.RGBA{0, 0, 0, 140}
)

// shell runs a viewer in a window: it reads Ebitengine's input into snapshots, or takes them
// from a script, and draws the viewer's state.
type shell struct {
	v         *viewer.Viewer
	sprites   *render.Sprites
	view      *render.LevelView
	presenter *render.Presenter
	palette   *content.Palette
	// version is the viewer's draw commands the view holds.
	version int
	buffer  *ebiten.Image
	keys    map[viewer.Key]ebiten.Key
	// script, if set, gives the input; shots are screenshots due at the next draw.
	script *viewer.Runner
	shots  []string
	err    error
	// art holds index images for the hotbar's and the pointer's art, by size.
	art map[image.Point]*ebiten.Image
	// talk, said and inv are the conversation window, a mouth's message and the inventory panel,
	// drawn at their own pixels and scaled onto the screen.
	talk, said, inv *ebiten.Image
}

func newShell(v *viewer.Viewer) (*shell, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	s := &shell{
		v: v, sprites: render.NewSprites(v.Pack), presenter: pr,
		palette: v.Pack.Palettes[v.Pack.Manifest.Palette], keys: map[viewer.Key]ebiten.Key{},
		art: map[image.Point]*ebiten.Image{},
	}
	s.view = render.NewLevelView(s.sprites)
	names := map[string]ebiten.Key{}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		names[k.String()] = k
	}
	for _, k := range viewer.Keys {
		ek, ok := names[string(k)]
		if !ok {
			return nil, fmt.Errorf("no Ebitengine key %q", k)
		}
		s.keys[k] = ek
	}
	return s, s.sync()
}

// sync reloads the view when the viewer has rebuilt its draw commands.
func (s *shell) sync() error {
	if s.version == s.v.Version {
		return nil
	}
	s.version = s.v.Version
	if err := s.view.Set(s.v.Cmds, s.v.Bounds); err != nil {
		return fmt.Errorf("level %s: %w", s.v.Shown, err)
	}
	return nil
}

// input reads Ebitengine's input.
func (s *shell) input() viewer.Input {
	in := viewer.Input{Held: map[viewer.Key]bool{}, Pressed: map[viewer.Key]bool{}}
	x, y := ebiten.CursorPosition()
	in.Cursor = image.Pt(x, y)
	for k, ek := range s.keys {
		in.Held[k] = ebiten.IsKeyPressed(ek)
		in.Pressed[k] = inpututil.IsKeyJustPressed(ek)
	}
	button := func(b ebiten.MouseButton) viewer.Button {
		return viewer.Button{Held: ebiten.IsMouseButtonPressed(b),
			Pressed: inpututil.IsMouseButtonJustPressed(b), Released: inpututil.IsMouseButtonJustReleased(b)}
	}
	in.Left, in.Right = button(ebiten.MouseButtonLeft), button(ebiten.MouseButtonRight)
	_, in.Wheel = ebiten.Wheel()
	in.Chars = ebiten.AppendInputChars(nil)
	return in
}

func (s *shell) Update() error {
	if s.err != nil {
		return s.err
	}
	if len(s.shots) > 0 {
		// Wait for the draw that takes them, so they show this update.
		return nil
	}
	if s.v.Quit {
		return ebiten.Termination
	}
	if s.v.Audio != nil {
		s.v.Audio.Update(time.Second / time.Duration(ebiten.TPS()))
	}
	if s.script == nil {
		s.v.Update(s.input())
		return s.sync()
	}
	if s.script.Done() {
		return ebiten.Termination
	}
	in, err := s.script.Before(s.v)
	if err != nil {
		return err
	}
	s.v.Update(in)
	if s.shots, err = s.script.After(s.v); err != nil {
		return err
	}
	return s.sync()
}

func (s *shell) Draw(screen *ebiten.Image) {
	v := s.v
	if v.Front != nil {
		s.drawFront(screen)
		s.takeShots(screen)
		return
	}
	size := v.BufferSize()
	if s.buffer == nil || s.buffer.Bounds().Size() != size {
		if s.buffer != nil {
			s.buffer.Deallocate()
		}
		s.buffer = ebiten.NewImage(size.X, size.Y)
	}
	s.buffer.Clear()
	s.view.Draw(s.buffer, v.Origin)
	s.presenter.SetColors(s.palette.At(v.Ticks*1000/int64(ebiten.TPS())), v.FixedColors(s.palette))
	s.presenter.Present(screen, s.buffer, v.Scale)
	s.drawLabels(screen)
	s.drawHotbar(screen)
	s.drawSpellPanel(screen)
	s.drawInventory(screen)
	s.drawSaid(screen)
	s.drawMinimap(screen)
	s.drawTalk(screen)
	s.drawReading(screen)
	s.drawTools(screen)
	s.drawNotice(screen)
	s.drawFullMap(screen)
	s.drawPointer(screen)
	s.takeShots(screen)
}

// takeShots saves the screenshots due.
func (s *shell) takeShots(screen *ebiten.Image) {
	for _, path := range s.shots {
		if err := saveScreenshot(screen, path); err != nil && s.err == nil {
			s.err = err
		}
	}
	s.shots = nil
}

// drawTools draws the selection, the inspector panel and the event log over the view.
func (s *shell) drawTools(screen *ebiten.Image) {
	v := s.v
	if v.HideDebug {
		return
	}
	if v.Hover != nil && (v.Selected == nil || *v.Hover != *v.Selected) {
		s.diamond(screen, *v.Hover, hoverColor)
	}
	if v.Selected != nil {
		s.diamond(screen, *v.Selected, selectionColor)
	}
	if v.Thing != "" {
		for _, c := range v.Cmds {
			if c.Thing == v.Thing {
				size := v.Pack.Sprites[c.Sprite].FrameSize
				at := image.Pt(c.X, c.Y)
				s.outline(screen, image.Rectangle{at, at.Add(image.Pt(size[0], size[1]))}, thingColor)
			}
		}
	}
	text := func(r image.Rectangle, lines []string) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), panelColor, false)
		for i, line := range lines {
			ebitenutil.DebugPrintAt(screen, line, r.Min.X+viewer.Margin/2, r.Min.Y+i*viewer.LineHeight)
		}
	}
	text(v.PanelRect(), v.PanelLines())
	if r := v.LogRect(); !r.Empty() {
		text(r, v.LogLines())
	}
}

// artImage returns a cleared index image of size, kept from frame to frame.
func (s *shell) artImage(size image.Point) *ebiten.Image {
	img, ok := s.art[size]
	if !ok {
		img = ebiten.NewImage(size.X, size.Y)
		s.art[size] = img
	}
	img.Clear()
	return img
}

// drawHotbar draws the hotbar: boxes, gauges and action buttons at native resolution, then the
// portrait and the objects' icons through the palette at the hotbar's scale.
func (s *shell) drawHotbar(screen *ebiten.Image) {
	v := s.v
	h := v.Hotbar()
	if h == nil {
		return
	}
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	stroke := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, c, false)
	}
	fill(h.Rect.Inset(-viewer.Margin/2), hotbarColor)
	for i, g := range h.Gauges {
		r := h.Screen(g.Rect)
		fill(r, slotColor)
		if g.Max > 0 {
			full := r
			c := gaugeColors[i]
			if g.Value >= g.Max && i == viewer.GaugeLoad {
				c = overloadColor
			}
			full.Min.Y = r.Max.Y - r.Dy()*min(g.Value, g.Max)/g.Max
			fill(full, c)
		}
		stroke(r, slotEdgeColor)
	}
	for _, b := range h.Slots {
		r := h.Screen(b.Rect).Inset(1)
		fill(r, slotColor)
		edge := slotEdgeColor
		if b.Slot >= sim.Shelves {
			edge = handEdgeColor
		}
		if s, ok := v.World.Player().OpenSlot(v.Pack); ok && s == b.Slot {
			edge = openEdgeColor
		}
		stroke(r, edge)
	}
	for _, b := range h.Spells {
		r := h.Screen(b.Rect).Inset(1)
		fill(r, slotColor)
		stroke(r, spellEdgeColor)
	}
	fill(h.Screen(h.Gesture.Rect).Inset(1), slotColor)
	stroke(h.Screen(h.Gesture.Rect).Inset(1), spellEdgeColor)
	for _, a := range h.Actions {
		r := h.Screen(a.Rect).Inset(1)
		c := actionColor
		if !a.Ready {
			c = waitColor
		}
		fill(r, c)
		stroke(r, slotEdgeColor)
		if a.Icon == nil {
			ebitenutil.DebugPrintAt(screen, a.Label, r.Min.X+(r.Dx()-len(a.Label)*viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
		}
	}

	art := s.artImage(h.Rect.Size().Div(h.Scale))
	if h.Has {
		if err := s.sprites.DrawImage(art, h.Portrait, h.PortraitRect.Min.X, h.PortraitRect.Min.Y, h.Mirrored); err != nil && s.err == nil {
			s.err = err
		}
	}
	centered := func(img content.Image, r image.Rectangle) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		at := r.Min.Add(image.Pt((r.Dx()-b.FrameSize[0])/2, (r.Dy()-b.FrameSize[1])/2))
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, b := range h.Slots {
		if img, ok := v.SlotImage(b.Slot); ok {
			centered(img, b.Rect)
		}
	}
	for _, b := range h.Spells {
		if b.Has && b.Count > 0 {
			centered(b.Icon, b.Rect)
		}
	}
	// An action box shows the action's icon alone; the hand's box shows the object.
	for _, a := range h.Actions {
		if a.Icon != nil {
			centered(*a.Icon, a.Rect)
		}
	}
	if h.Gesture.Has {
		centered(h.Gesture.Icon, h.Gesture.Rect)
	}
	s.presenter.PresentAt(screen, art, h.Scale, h.Rect.Min)
	for _, a := range h.Actions {
		if !a.Ready {
			fill(h.Screen(a.Rect).Inset(1), waitShade)
		}
	}
	for _, b := range h.Spells {
		if b.Count > 1 {
			r := h.Screen(b.Rect)
			ebitenutil.DebugPrintAt(screen, fmt.Sprint(min(b.Count, 9)), r.Max.X-viewer.CharWidth-2, r.Max.Y-viewer.LineHeight)
		}
	}
}

// drawInventory draws the inventory panel: the figure with what it wears, the worn slots' and
// the pack's boxes with their objects, and the lines of details and stats. It draws the panel at
// its own pixels, then scales it onto the screen.
func (s *shell) drawInventory(screen *ebiten.Image) {
	v := s.v
	ip := v.Inventory()
	if ip == nil {
		return
	}
	size := ip.Rect.Size().Div(ip.Scale)
	if s.inv == nil || s.inv.Bounds().Size() != size {
		if s.inv != nil {
			s.inv.Deallocate()
		}
		s.inv = ebiten.NewImage(size.X, size.Y)
	}
	win := s.inv
	win.Fill(talkColor)
	vector.StrokeRect(win, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	for _, b := range ip.Slots {
		r := b.Rect.Inset(1)
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotColor, false)
		edge := slotEdgeColor
		if b.Slot >= sim.Shelves && b.Slot < sim.SlotPack {
			edge = handEdgeColor
		}
		if s, ok := v.World.Player().OpenSlot(v.Pack); ok && s == b.Slot {
			edge = openEdgeColor
		}
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, edge, false)
	}
	for _, b := range ip.Preview {
		r := b.Rect.Inset(1)
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), slotColor, false)
		vector.StrokeRect(win, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, slotEdgeColor, false)
	}
	art := s.artImage(size)
	draw := func(img content.Image, at image.Point, mirrored bool) {
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, mirrored); err != nil && s.err == nil {
			s.err = err
		}
	}
	if ip.HasDoll {
		draw(ip.Doll, ip.DollAt, false)
		for _, part := range ip.Parts {
			draw(part.Image, ip.DollAt.Add(part.At), part.Mirrored)
		}
	}
	for _, b := range ip.Slots {
		if img, ok := v.SlotImage(b.Slot); ok {
			fs := v.Pack.Sprites[img.Sprite].FrameSize
			draw(img, b.Rect.Min.Add(image.Pt((b.Rect.Dx()-fs[0])/2, (b.Rect.Dy()-fs[1])/2)), false)
		}
	}
	for _, b := range ip.Preview {
		if b.Has {
			fs := v.Pack.Sprites[b.Icon.Sprite].FrameSize
			draw(b.Icon, b.Rect.Min.Add(image.Pt((b.Rect.Dx()-fs[0])/2, (b.Rect.Dy()-fs[1])/2)), false)
		}
	}
	textAt := map[int]int{}
	for i, l := range ip.Lines {
		if l.Icon == nil {
			continue
		}
		draw(*l.Icon, l.At.Add(image.Pt(0, 4)), false)
		if b := v.Pack.Sprites[l.Icon.Sprite]; b != nil {
			textAt[i] = b.FrameSize[0] + 4
		}
	}
	s.presenter.PresentAt(win, art, 1, image.Point{})
	for i, l := range ip.Lines {
		ebitenutil.DebugPrintAt(win, l.Text, l.At.X+textAt[i], l.At.Y)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(ip.Scale), float64(ip.Scale))
	op.GeoM.Translate(float64(ip.Rect.Min.X), float64(ip.Rect.Min.Y))
	screen.DrawImage(win, op)
}

// drawSpellPanel draws the spell panel: the icons known, the gesture entered, the buttons and the
// spellbook.
func (s *shell) drawSpellPanel(screen *ebiten.Image) {
	v := s.v
	sp := v.SpellPanel()
	if sp == nil {
		return
	}
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	stroke := func(r image.Rectangle, c color.Color) {
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, c, false)
	}
	fill(sp.Rect.Inset(-viewer.Margin/2), hotbarColor)
	for _, b := range sp.Icons {
		fill(sp.Screen(b.Rect), slotColor)
		stroke(sp.Screen(b.Rect), slotEdgeColor)
	}
	for _, b := range sp.Buttons {
		r := sp.Screen(b.Rect)
		fill(r, actionColor)
		stroke(r, slotEdgeColor)
		ebitenutil.DebugPrintAt(screen, b.Label, r.Min.X+(r.Dx()-len(b.Label)*viewer.CharWidth)/2, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
	for _, e := range sp.Book {
		r := sp.Screen(e.Rect)
		fill(r, slotColor)
	}
	art := s.artImage(sp.Rect.Size().Div(sp.Scale))
	draw := func(img content.Image, r image.Rectangle) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		at := r.Min.Add(image.Pt((r.Dx()-b.FrameSize[0])/2, (r.Dy()-b.FrameSize[1])/2))
		if err := s.sprites.DrawImage(art, img, at.X, at.Y, false); err != nil && s.err == nil {
			s.err = err
		}
	}
	for _, b := range append(sp.Icons, sp.Entered...) {
		if b.Known {
			draw(b.Icon, b.Rect)
		}
	}
	for _, e := range sp.Book {
		if e.Icon != nil {
			draw(*e.Icon, image.Rect(e.Rect.Min.X, e.Rect.Min.Y, e.Rect.Min.X+e.Rect.Dy(), e.Rect.Max.Y))
		}
	}
	s.presenter.PresentAt(screen, art, sp.Scale, sp.Rect.Min)
	for _, e := range sp.Book {
		r := sp.Screen(e.Rect)
		x := r.Min.X + r.Dy() + 2
		ebitenutil.DebugPrintAt(screen, viewer.Clip(e.Label, (r.Max.X-x)/viewer.CharWidth), x, r.Min.Y+(r.Dy()-viewer.LineHeight)/2)
	}
}

// drawTalk draws the open conversation: the line beside its portrait, its keywords underlined,
// or the offered keywords and the field. It draws the window at its own pixels, then scales it
// onto the screen.
func (s *shell) drawTalk(screen *ebiten.Image) {
	v := s.v
	tv := v.Talk()
	if tv == nil {
		return
	}
	size := tv.Rect.Size().Div(tv.Scale)
	if s.talk == nil || s.talk.Bounds().Size() != size {
		if s.talk != nil {
			s.talk.Deallocate()
		}
		s.talk = ebiten.NewImage(size.X, size.Y)
	}
	win := s.talk
	win.Fill(talkColor)
	vector.StrokeRect(win, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	fill := func(r image.Rectangle, c color.Color) {
		vector.FillRect(win, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
	}
	draw := func(img content.Image, at image.Point, scale int, mirrored bool) {
		b := v.Pack.Sprites[img.Sprite]
		if b == nil {
			return
		}
		art := s.artImage(image.Pt(b.FrameSize[0], b.FrameSize[1]))
		if err := s.sprites.DrawImage(art, img, 0, 0, mirrored); err != nil && s.err == nil {
			s.err = err
		}
		s.presenter.PresentAt(win, art, scale, at)
	}
	if tv.HasPortrait {
		draw(tv.Portrait, tv.PortraitAt, tv.PortraitScale, tv.Mirrored)
	}
	for i, img := range tv.Icons {
		draw(img, tv.IconsAt[i], 2, false)
	}
	for _, w := range tv.Words {
		ebitenutil.DebugPrintAt(win, w.Text, w.Rect.Min.X, w.Rect.Min.Y)
		if w.Keyword {
			vector.StrokeLine(win, float32(w.Rect.Min.X), float32(w.Rect.Max.Y)-0.5, float32(w.Rect.Max.X), float32(w.Rect.Max.Y)-0.5, 1, keywordColor, false)
		}
	}
	for _, b := range tv.Buttons {
		fill(b.Rect, actionColor)
		vector.StrokeRect(win, float32(b.Rect.Min.X)+0.5, float32(b.Rect.Min.Y)+0.5, float32(b.Rect.Dx()-1), float32(b.Rect.Dy()-1), 1, slotEdgeColor, false)
		ebitenutil.DebugPrintAt(win, b.Label, b.Rect.Min.X+(b.Rect.Dx()-len(b.Label)*viewer.CharWidth)/2, b.Rect.Min.Y+2)
	}
	if !tv.Waiting {
		fill(tv.Field, slotColor)
		cursor := ""
		if v.Ticks/30%2 == 0 {
			cursor = "_"
		}
		ebitenutil.DebugPrintAt(win, tv.Typed+cursor, tv.Field.Min.X+4, tv.Field.Min.Y+2)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(tv.Scale), float64(tv.Scale))
	op.GeoM.Translate(float64(tv.Rect.Min.X), float64(tv.Rect.Min.Y))
	screen.DrawImage(win, op)
}

// drawSaid draws a mouth's message beside the mouth: wrapped lines on a dark backing, hanging
// above the mouth and kept on the screen, at twice the debug font's size.
func (s *shell) drawSaid(screen *ebiten.Image) {
	v := s.v
	said := v.Said
	if said == nil {
		return
	}
	const scale, width = 2, 36
	lines := wrap(said.Text, width)
	size := image.Pt(width*viewer.CharWidth+8, len(lines)*viewer.LineHeight+8)
	if s.said == nil || s.said.Bounds().Size() != size {
		if s.said != nil {
			s.said.Deallocate()
		}
		s.said = ebiten.NewImage(size.X, size.Y)
	}
	s.said.Fill(talkColor)
	vector.StrokeRect(s.said, 0.5, 0.5, float32(size.X-1), float32(size.Y-1), 1, slotEdgeColor, false)
	for i, l := range lines {
		ebitenutil.DebugPrintAt(s.said, l, 4, 4+i*viewer.LineHeight)
	}
	at := v.ScreenPoint(said.At)
	shown := size.Mul(scale)
	x := min(max(at.X-shown.X/2, viewer.Margin), v.Screen.X-shown.X-viewer.Margin)
	y := max(at.Y-shown.Y-viewer.Margin, viewer.Margin)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(s.said, op)
}

// wrap breaks text into lines of at most n characters, at spaces.
func wrap(text string, n int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= n:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// mapColors are the level map's colors by cell kind.
var mapColors = map[int]color.RGBA{
	viewer.MapFloor: {120, 110, 95, 255}, viewer.MapWall: {55, 50, 48, 255}, viewer.MapDoor: {150, 100, 50, 255},
	viewer.MapExit: {80, 200, 90, 255}, viewer.MapWater: {50, 90, 190, 255}, viewer.MapHole: {15, 15, 15, 255},
	viewer.MapTeleporter: {170, 80, 200, 255}, viewer.MapMouth: {220, 220, 220, 255},
	viewer.MapPlate: {220, 190, 60, 255}, viewer.MapMark: {60, 200, 210, 255},
	viewer.MapTrapdoor: {120, 75, 40, 255}, viewer.MapSocket: {230, 110, 40, 255},
}

// mapPlayerColor is the player's cell on the level map.
var mapPlayerColor = color.RGBA{230, 40, 40, 255}

// drawLevelMap draws map m's cells as diamonds of half-width s around the middle of r, rows and
// columns turned as the view turns them, and the player's cell in red.
func drawLevelMap(screen *ebiten.Image, m *viewer.LevelMap, r image.Rectangle, s float32) {
	rows, cols := float32(m.Size[0]), float32(m.Size[1])
	// The map's corners: (0, 0) at the top, (rows, 0) at the left, (0, cols) at the right.
	ox := float32(r.Min.X+r.Max.X)/2 - (cols-rows)*s/2
	oy := float32(r.Min.Y+r.Max.Y)/2 - (cols+rows)*s/4
	paths := map[int]*vector.Path{}
	diamond := func(path *vector.Path, c content.Cell) {
		x := ox + float32(c[1]-c[0])*s
		y := oy + float32(c[0]+c[1])*s/2
		path.MoveTo(x, y)
		path.LineTo(x+s, y+s/2)
		path.LineTo(x, y+s)
		path.LineTo(x-s, y+s/2)
		path.Close()
	}
	for _, c := range m.Cells {
		if paths[c.Kind] == nil {
			paths[c.Kind] = &vector.Path{}
		}
		diamond(paths[c.Kind], c.Cell)
	}
	for kind, path := range paths {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(mapColors[kind])
		vector.FillPath(screen, path, nil, op)
	}
	player := &vector.Path{}
	diamond(player, m.Player)
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(mapPlayerColor)
	vector.FillPath(screen, player, nil, op)
}

// drawMinimap draws the level map small in the top-left corner.
func (s *shell) drawMinimap(screen *ebiten.Image) {
	v := s.v
	if !v.MinimapShown() {
		return
	}
	m := v.Map()
	const size = 200
	r := image.Rect(viewer.Margin, viewer.Margin, viewer.Margin+size, viewer.Margin+size/2)
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), panelColor, false)
	cell := float32(size) / float32(m.Size[0]+m.Size[1])
	drawLevelMap(screen, m, r, cell)
}

// drawFullMap draws the level map over the whole screen, with the level's name.
func (s *shell) drawFullMap(screen *ebiten.Image) {
	v := s.v
	if !v.MapOpen {
		return
	}
	m := v.Map()
	if m == nil {
		return
	}
	screen.Fill(color.RGBA{8, 8, 12, 255})
	legendW := s.drawMapLegend(screen, m)
	r := image.Rect(legendW, 3*viewer.LineHeight, v.Screen.X, v.Screen.Y-viewer.Margin)
	cell := min(float32(r.Dx())/float32(m.Size[0]+m.Size[1]), 2*float32(r.Dy())/float32(m.Size[0]+m.Size[1]))
	drawLevelMap(screen, m, r, cell)
	name := m.Name
	if name == "" {
		name = v.World.Current().ID
	}
	ebitenutil.DebugPrintAt(screen, name, (v.Screen.X-len(name)*viewer.CharWidth)/2, viewer.LineHeight)
}

// drawMapLegend draws the full map's legend down its left side, a swatch and a name for the
// player and for each kind the map shows, and returns the width it takes.
func (s *shell) drawMapLegend(screen *ebiten.Image, m *viewer.LevelMap) int {
	const swatch = 10
	x, y := viewer.Margin, 3*viewer.LineHeight
	width := 0
	line := func(c color.Color, name string) {
		vector.FillRect(screen, float32(x), float32(y+(viewer.LineHeight-swatch)/2), swatch, swatch, c, false)
		vector.StrokeRect(screen, float32(x), float32(y+(viewer.LineHeight-swatch)/2), swatch, swatch, 1, slotEdgeColor, false)
		ebitenutil.DebugPrintAt(screen, name, x+swatch+viewer.CharWidth, y)
		width = max(width, swatch+viewer.CharWidth*(1+len(name)))
		y += viewer.LineHeight + 4
	}
	line(mapPlayerColor, "You")
	for _, k := range m.Legend {
		line(mapColors[k], viewer.MapKindNames[k])
	}
	return x + width + viewer.Margin
}

// drawNotice draws the viewer's notice at the top middle of the screen while it lasts.
func (s *shell) drawNotice(screen *ebiten.Image) {
	v := s.v
	if v.Notice == "" || v.Ticks > v.NoticeUntil {
		return
	}
	w := len(v.Notice)*viewer.CharWidth + 8
	x, y := (v.Screen.X-w)/2, viewer.Margin
	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(viewer.LineHeight+4), panelColor, false)
	ebitenutil.DebugPrintAt(screen, v.Notice, x+4, y+2)
}

// drawReading draws the picture of a scroll being read in a small window centered above the
// hotbar, at the hotbar's scale.
func (s *shell) drawReading(screen *ebiten.Image) {
	v := s.v
	img := v.Reading
	if img == nil {
		return
	}
	b := v.Pack.Sprites[img.Sprite]
	if b == nil {
		return
	}
	scale, bottom := v.Scale, v.Screen.Y
	if h := v.Hotbar(); h != nil {
		scale, bottom = h.Scale, h.Rect.Min.Y-viewer.Margin
	}
	size := image.Pt(b.FrameSize[0], b.FrameSize[1])
	at := image.Pt((v.Screen.X-size.X*scale)/2, bottom-size.Y*scale-viewer.Margin)
	r := image.Rectangle{at, at.Add(size.Mul(scale))}.Inset(-viewer.Margin)
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), hotbarColor, false)
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, *img, 0, 0, false); err != nil && s.err == nil {
		s.err = err
	}
	s.presenter.PresentAt(screen, art, scale, at)
}

// drawPointer names what the cursor is over on the hotbar, and draws the pointer while playing:
// the arrow, the speech bubble over something a click talks to, or the object in hand, at the
// view's scale, in place of the system's pointer.
func (s *shell) drawPointer(screen *ebiten.Image) {
	v := s.v
	cursor := v.Cursor
	if lines := v.HoverLines(cursor); len(lines) > 0 {
		w := 0
		for _, l := range lines {
			w = max(w, utf8.RuneCountInString(l))
		}
		w = w*viewer.CharWidth + 4
		h := len(lines) * viewer.LineHeight
		// Above the pointer, kept on the screen.
		at := cursor.Add(image.Pt(12, -h-4))
		at.X = min(at.X, v.Screen.X-w)
		at.Y = max(at.Y, 0)
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), panelColor, false)
		for i, l := range lines {
			ebitenutil.DebugPrintAt(screen, l, at.X+2, at.Y+i*viewer.LineHeight)
		}
	}
	img, centered, ok := v.Pointer()
	ok = ok && cursor.In(image.Rectangle{Max: v.Screen})
	mode := ebiten.CursorModeVisible
	if ok && !s.scripted() {
		mode = ebiten.CursorModeHidden
	}
	if ebiten.CursorMode() != mode {
		ebiten.SetCursorMode(mode)
	}
	if !ok {
		return
	}
	scale := v.Scale
	if h := v.Hotbar(); h != nil && centered {
		scale = h.Scale
	}
	fs := v.Pack.Sprites[img.Sprite].FrameSize
	size := image.Pt(fs[0], fs[1])
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, img, 0, 0, false); err != nil && s.err == nil {
		s.err = err
	}
	at := cursor
	if centered {
		at = cursor.Sub(size.Mul(scale).Div(2))
	}
	s.presenter.PresentAt(screen, art, scale, at)
}

// scripted reports whether a script, not the mouse, moves the cursor.
func (s *shell) scripted() bool { return s.script != nil }

// drawLabels draws the damage shown over things: white numbers on red, centered over each
// sprite's top. A stand-in for the game's own until the UI has its font.
func (s *shell) drawLabels(screen *ebiten.Image) {
	for _, l := range s.v.Labels {
		at := s.v.ScreenPoint(l.At)
		w := len(l.Text)*viewer.CharWidth + 4
		x, y := at.X-w/2, at.Y-viewer.LineHeight
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(viewer.LineHeight), damageColor, false)
		ebitenutil.DebugPrintAt(screen, l.Text, x+2, y)
	}
}

// diamond strokes the outline of cell c's ground on the screen.
func (s *shell) diamond(screen *ebiten.Image, c content.Cell, clr color.Color) {
	d := drawlist.Diamond(c)
	for i := range d {
		a, b := s.v.ScreenPoint(d[i]), s.v.ScreenPoint(d[(i+1)%len(d)])
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 2, clr, true)
	}
}

// outline strokes view rectangle r on the screen.
func (s *shell) outline(screen *ebiten.Image, r image.Rectangle, c color.Color) {
	lo, hi := s.v.ScreenPoint(r.Min), s.v.ScreenPoint(r.Max)
	vector.StrokeRect(screen, float32(lo.X), float32(lo.Y), float32(hi.X-lo.X), float32(hi.Y-lo.Y), 2, c, false)
}

func saveScreenshot(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (s *shell) Layout(outsideWidth, outsideHeight int) (int, int) {
	s.v.Screen = image.Pt(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

// savePath returns path when a file is there, or else the file of that name in the saves
// directory dir when one is there, so --load takes a bare save name like quick.sav.
func savePath(path, dir string) string {
	if fileExists(path) || dir == "" || filepath.IsAbs(path) {
		return path
	}
	if alt := filepath.Join(dir, path); fileExists(alt) {
		return alt
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
