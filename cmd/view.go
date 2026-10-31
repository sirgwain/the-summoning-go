package cmd

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/spf13/cobra"
)

func init() {
	var level, screenshot string
	var scale, shotTicks int
	var magnify bool
	var at, sel []int
	var seed uint64
	viewCmd := &cobra.Command{
		Use:   "view [pack-dir]",
		Short: "Look around a content pack's levels and work their mechanisms",
		Long: `Opens a window showing a pack's levels in the game's projection, running the simulation
without a player.

Arrows or WASD (with Shift, faster) or dragging with the mouse move the view. Scrolling zooms
around the cursor, as do + and - around the center. Page Up and Page Down (or [ and ]) change
level, Home recenters, M magnifies objects on the floor.

Click a cell or a thing to inspect it. Then U uses it, T toggles its feature, 1–9 run its
hooks' procedures, E follows its exit, O drops the chosen object on it and X takes its top
object; I and Shift+I choose the object. P places the chosen thing, C and Shift+C choose it, and
Shift+click moves the selected thing to the clicked cell. Space pauses, N steps while paused.`,
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
			v, err := newViewer(pack, seed, level, scale, magnify)
			if err != nil {
				return err
			}
			v.screenshot, v.shotTicks = screenshot, int64(shotTicks)
			if center != nil {
				v.centerOn(*center)
			}
			if selected != nil {
				v.selected = selected
				if center == nil {
					v.centerOn(*selected)
				}
			}
			ebiten.SetWindowTitle("The Summoning: " + pack.Manifest.Name)
			ebiten.SetWindowSize(1280, 800)
			ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
			return ebiten.RunGame(v)
		},
	}
	viewCmd.Flags().StringVar(&level, "level", "", "level id to open (default: the start level)")
	viewCmd.Flags().IntVar(&scale, "scale", 2, "pixels per art pixel")
	viewCmd.Flags().IntSliceVar(&at, "at", nil, "center the view on this cell, as x,y")
	viewCmd.Flags().IntSliceVar(&sel, "select", nil, "select this cell, as x,y")
	viewCmd.Flags().BoolVar(&magnify, "magnify", false, "start with objects on the floor magnified")
	viewCmd.Flags().Uint64Var(&seed, "seed", 1, "seed for the simulation's random choices")
	viewCmd.Flags().StringVar(&screenshot, "screenshot", "", "save a frame to this PNG file and exit")
	viewCmd.Flags().IntVar(&shotTicks, "screenshot-ticks", 0, "updates to run before the screenshot")
	rootCmd.AddCommand(viewCmd)
}

const (
	maxScale    = 6
	panSpeed    = 6
	panSpeedFat = 24
	// clickSlop is how far, in screen pixels, the mouse may move between press and release and
	// still click rather than drag.
	clickSlop = 4
)

type viewer struct {
	pack      *content.Pack
	world     *sim.World
	sprites   *render.Sprites
	presenter *render.Presenter
	palette   *content.Palette
	// ticks counts updates, the clock palette effects run on.
	ticks  int64
	levels []string
	view   *render.LevelView
	// shown is the level the view draws.
	shown string
	// magnify draws floor objects with their inventory image.
	magnify bool
	// origin is the view pixel at the buffer's top-left.
	origin image.Point
	scale  int
	buffer *ebiten.Image
	// screen is the size of the screen in pixels.
	screen image.Point
	// press is where the mouse button went down; drag where the view was last dragged from.
	press, drag *image.Point
	// wheel accumulates scrolling not yet turned into zoom steps.
	wheel float64
	tools
	// screenshot, if set, is where to save the first frame before exiting.
	screenshot string
	shotTicks  int64
	shotTaken  bool
	err        error
}

func newViewer(p *content.Pack, seed uint64, level string, scale int, magnify bool) (*viewer, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	w, err := sim.New(p, seed)
	if err != nil {
		return nil, err
	}
	v := &viewer{
		pack:      p,
		world:     w,
		sprites:   render.NewSprites(p),
		presenter: pr,
		palette:   p.Palettes[p.Manifest.Palette],
		scale:     max(1, min(scale, maxScale)),
		magnify:   magnify,
		screen:    image.Pt(1280, 800),
		levels:    sortedIDs(p.Levels),
		tools:     newTools(p),
	}
	if level != "" && level != w.Current().ID {
		v.world.Update([]input.Command{input.GoTo{Level: level}})
		if w.Current().ID != level {
			return nil, fmt.Errorf("no level %q", level)
		}
	}
	if err := v.rebuild(); err != nil {
		return nil, err
	}
	return v, nil
}

// rebuild redraws the current level as the world has it, recentering when the level changed.
func (v *viewer) rebuild() error {
	l := v.world.Current()
	scene := drawlist.SimScene(l)
	opts := drawlist.Options{Magnify: v.magnify}
	if v.view == nil {
		lv, err := render.NewLevelView(v.pack, v.sprites, scene, opts)
		if err != nil {
			return err
		}
		v.view = lv
	} else if err := v.view.Set(v.pack, v.sprites, scene, opts); err != nil {
		return err
	}
	if v.shown != l.ID {
		v.shown = l.ID
		v.selected, v.thing = nil, ""
		v.recenter()
	}
	return nil
}

// recenter centers the view on the start cell on the start level, else on the level.
func (v *viewer) recenter() {
	if s := v.pack.Manifest.Start; s.Level == v.shown {
		v.centerOn(s.Cell)
		return
	}
	center := v.view.Bounds().Min.Add(v.view.Bounds().Max).Div(2)
	v.origin = center.Sub(v.bufferSize().Div(2))
}

// centerOn centers the view on cell c.
func (v *viewer) centerOn(c content.Cell) {
	center := drawlist.TileBox(c).Add(image.Pt(drawlist.TileSize/2, drawlist.TileSize/2))
	v.origin = center.Sub(v.bufferSize().Div(2))
}

func (v *viewer) bufferSize() image.Point {
	return image.Pt((v.screen.X+v.scale-1)/v.scale, (v.screen.Y+v.scale-1)/v.scale)
}

// screenRect returns the screen's rectangle.
func (v *viewer) screenRect() image.Rectangle { return image.Rectangle{Max: v.screen} }

// viewPoint returns the view pixel under screen point pt.
func (v *viewer) viewPoint(pt image.Point) image.Point { return v.origin.Add(pt.Div(v.scale)) }

// screenPoint returns the screen point of view pixel pt.
func (v *viewer) screenPoint(pt image.Point) image.Point { return pt.Sub(v.origin).Mul(v.scale) }

func (v *viewer) Update() error {
	if v.err != nil {
		return v.err
	}
	if v.shotTaken {
		return ebiten.Termination
	}
	v.ticks++
	v.pan()
	x, y := ebiten.CursorPosition()
	cursor := image.Pt(x, y)
	v.scroll(cursor)
	cmds := v.mouse(cursor)

	switch {
	case justPressed(ebiten.KeyPageDown, ebiten.KeyBracketRight):
		cmds = append(cmds, input.GoTo{Level: v.levels[(v.levelIndex()+1)%len(v.levels)]})
	case justPressed(ebiten.KeyPageUp, ebiten.KeyBracketLeft):
		cmds = append(cmds, input.GoTo{Level: v.levels[(v.levelIndex()+len(v.levels)-1)%len(v.levels)]})
	case justPressed(ebiten.KeyEqual, ebiten.KeyNumpadAdd):
		v.zoom(1)
	case justPressed(ebiten.KeyMinus, ebiten.KeyNumpadSubtract):
		v.zoom(-1)
	case justPressed(ebiten.KeyHome):
		v.recenter()
	case justPressed(ebiten.KeyM):
		v.magnify = !v.magnify
		v.err = v.rebuild()
	}
	cmds = append(cmds, v.keys()...)

	if v.paused && len(cmds) == 0 && !justPressed(ebiten.KeyN) {
		return v.err
	}
	events := v.world.Update(cmds)
	v.log(events)
	if len(events) > 0 && v.err == nil {
		v.err = v.rebuild()
	}
	return v.err
}

func (v *viewer) levelIndex() int {
	for i, id := range v.levels {
		if id == v.shown {
			return i
		}
	}
	return 0
}

// pan moves the view by the arrow and WASD keys.
func (v *viewer) pan() {
	speed := panSpeed
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		speed = panSpeedFat
	}
	for _, k := range []struct {
		keys   []ebiten.Key
		dx, dy int
	}{
		{[]ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyA}, -1, 0},
		{[]ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyD}, 1, 0},
		{[]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyW}, 0, -1},
		{[]ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyS}, 0, 1},
	} {
		for _, key := range k.keys {
			if ebiten.IsKeyPressed(key) {
				v.origin = v.origin.Add(image.Pt(k.dx*speed, k.dy*speed))
				break
			}
		}
	}
}

// mouse tracks the cell under the cursor, drags the view, and turns a left click into a
// selection, or with Shift into moving the selected thing to the clicked cell. A click is a
// release near its press; a press and release between two updates (a trackpad tap) counts too.
func (v *viewer) mouse(cursor image.Point) []input.Command {
	if cursor.In(v.screenRect()) && !cursor.In(v.panel) {
		c := drawlist.CellAt(v.viewPoint(cursor))
		v.hover = &c
	} else {
		v.hover = nil
	}

	left, right := ebiten.MouseButtonLeft, ebiten.MouseButtonRight
	if inpututil.IsMouseButtonJustPressed(left) || inpututil.IsMouseButtonJustPressed(right) {
		v.press, v.drag = &cursor, &cursor
	}
	if v.drag != nil && (ebiten.IsMouseButtonPressed(left) || ebiten.IsMouseButtonPressed(right)) {
		v.origin = v.origin.Sub(cursor.Sub(*v.drag).Div(v.scale))
		moved := cursor.Sub(*v.drag).Div(v.scale).Mul(v.scale)
		next := v.drag.Add(moved)
		v.drag = &next
	}
	if inpututil.IsMouseButtonJustReleased(right) {
		v.press, v.drag = nil, nil
	}
	if !inpututil.IsMouseButtonJustReleased(left) {
		return nil
	}
	press := cursor
	if v.press != nil {
		press = *v.press
	}
	v.press, v.drag = nil, nil
	if d := cursor.Sub(press); d.X*d.X+d.Y*d.Y > clickSlop*clickSlop || cursor.In(v.panel) {
		return nil
	}
	pt := v.viewPoint(cursor)
	c := drawlist.CellAt(pt)
	if ebiten.IsKeyPressed(ebiten.KeyShift) && v.thing != "" {
		return []input.Command{input.Move{Thing: v.thing, Cell: c}}
	}
	v.selected, v.thing = &c, ""
	if id, _, ok := drawlist.ThingAt(v.pack, v.view.Cmds(), pt); ok {
		v.thing = id
		tc := v.world.Current().Thing(id).Cell()
		v.selected = &tc
	}
	return nil
}

// scroll zooms around the cursor by the scroll wheel, a step per whole wheel unit.
func (v *viewer) scroll(cursor image.Point) {
	_, dy := ebiten.Wheel()
	v.wheel += dy
	for ; v.wheel >= 1; v.wheel-- {
		v.zoomAt(1, cursor)
	}
	for ; v.wheel <= -1; v.wheel++ {
		v.zoomAt(-1, cursor)
	}
}

// zoom changes the scale, keeping the view centered on the same point.
func (v *viewer) zoom(d int) {
	v.zoomAt(d, v.screen.Div(2))
}

// zoomAt changes the scale, keeping the art pixel under the screen point at fixed.
func (v *viewer) zoomAt(d int, at image.Point) {
	under := v.origin.Add(at.Div(v.scale))
	v.scale = max(1, min(v.scale+d, maxScale))
	v.origin = under.Sub(at.Div(v.scale))
}

func justPressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func (v *viewer) Draw(screen *ebiten.Image) {
	size := v.bufferSize()
	if v.buffer == nil || v.buffer.Bounds().Size() != size {
		if v.buffer != nil {
			v.buffer.Deallocate()
		}
		v.buffer = ebiten.NewImage(size.X, size.Y)
	}
	v.buffer.Clear()
	v.view.Draw(v.buffer, v.origin)
	v.presenter.SetColors(v.palette.At(v.ticks * 1000 / int64(ebiten.TPS())))
	v.presenter.Present(screen, v.buffer, v.scale)
	v.drawTools(screen)
	if v.screenshot != "" && !v.shotTaken && v.ticks >= v.shotTicks {
		v.shotTaken = true
		v.err = saveScreenshot(screen, v.screenshot)
	}
}

func saveScreenshot(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
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

func (v *viewer) Layout(outsideWidth, outsideHeight int) (int, int) {
	v.screen = image.Pt(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}
