package cmd

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newViewCmd(false), newViewCmd(true))
}

// newViewCmd returns the view command, or with play the play command, which opens the same
// window driving the player.
func newViewCmd(play bool) *cobra.Command {
	var level, screenshot, script, out string
	var scale, shotTicks int
	var magnify bool
	var at, sel []int
	var seed uint64
	use, short, long := "view [pack-dir]", "Look around a content pack's levels and work their mechanisms",
		`Opens a window showing a pack's levels in the game's projection, running the simulation.
Tab switches to playing (see summoning play --help).
`
	if play {
		use, short, long = "play [pack-dir]", "Play a content pack's levels",
			`Opens a window playing a pack: the player starts where the pack says and the view follows it,
showing what the player sees.

Arrows or WASD walk, two together along a diagonal. G picks up the top object of the player's
cell, F puts down what is in hand, E uses the lever underfoot or a mouth beside the player, Q
puts what is in hand on (worn, or in the right hand). J and K strike with the right and left
hands, with Shift their second action; T throws what the right hand holds, Shift+T the left's. 1-9
and 0 swap the hand with shelves 0-9 (with Shift, 10-19), R and L with the right and left hands,
F1-F8 with the pack (the bag in the left hand). V shows the whole level, Tab switches to the
viewer's tools (see summoning view --help), clicking a cell inspects it.
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
			if play {
				if v.World.Player() == nil {
					return fmt.Errorf("pack %s has no player", pack.Manifest.ID)
				}
				v.SetPlay(true)
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
}

func newShell(v *viewer.Viewer) (*shell, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	s := &shell{
		v: v, sprites: render.NewSprites(v.Pack), presenter: pr,
		palette: v.Pack.Palettes[v.Pack.Manifest.Palette], keys: map[viewer.Key]ebiten.Key{},
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
	size := v.BufferSize()
	if s.buffer == nil || s.buffer.Bounds().Size() != size {
		if s.buffer != nil {
			s.buffer.Deallocate()
		}
		s.buffer = ebiten.NewImage(size.X, size.Y)
	}
	s.buffer.Clear()
	s.view.Draw(s.buffer, v.Origin)
	s.presenter.SetColors(s.palette.At(v.Ticks * 1000 / int64(ebiten.TPS())))
	s.presenter.Present(screen, s.buffer, v.Scale)
	s.drawTools(screen)
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
