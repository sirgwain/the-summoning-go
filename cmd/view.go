package cmd

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/spf13/cobra"
)

func init() {
	var level, screenshot string
	var scale int
	viewCmd := &cobra.Command{
		Use:   "view [pack-dir]",
		Short: "Look around a content pack's levels",
		Long: `Opens a window showing a pack's levels in the game's projection.

Arrows or WASD (with Shift, faster), dragging with the mouse or scrolling move the view. Page
Up and Page Down (or [ and ]) change level, + and - (or Ctrl or Cmd with the scroll wheel) zoom,
Home recenters.`,
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
			v, err := newViewer(pack, level, scale)
			if err != nil {
				return err
			}
			v.screenshot = screenshot
			ebiten.SetWindowTitle("The Summoning: " + pack.Manifest.Name)
			ebiten.SetWindowSize(1280, 800)
			ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
			return ebiten.RunGame(v)
		},
	}
	viewCmd.Flags().StringVar(&level, "level", "", "level id to open (default: the start level)")
	viewCmd.Flags().IntVar(&scale, "scale", 2, "pixels per art pixel")
	viewCmd.Flags().StringVar(&screenshot, "screenshot", "", "save the first frame to this PNG file and exit")
	rootCmd.AddCommand(viewCmd)
}

const (
	maxScale    = 6
	panSpeed    = 6
	panSpeedFat = 24
	// wheelSpeed is how far one step of the scroll wheel pans, in art pixels.
	wheelSpeed = 16
)

type viewer struct {
	pack      *content.Pack
	sprites   *render.Sprites
	presenter *render.Presenter
	levels    []string
	current   int
	view      *render.LevelView
	// origin is the view pixel at the buffer's top-left.
	origin image.Point
	scale  int
	buffer *ebiten.Image
	// screen is the size of the screen in pixels.
	screen image.Point
	drag   *image.Point
	// wheel accumulates scrolling not yet turned into whole pixels or zoom steps.
	wheel [2]float64
	// screenshot, if set, is where to save the first frame before exiting.
	screenshot string
	shotTaken  bool
	err        error
}

func newViewer(p *content.Pack, level string, scale int) (*viewer, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	pr.SetPalette(p.Palettes[p.Manifest.Palette])
	v := &viewer{
		pack:      p,
		sprites:   render.NewSprites(p),
		presenter: pr,
		scale:     max(1, min(scale, maxScale)),
		screen:    image.Pt(1280, 800),
	}
	for id := range p.Levels {
		v.levels = append(v.levels, id)
	}
	sort.Strings(v.levels)
	if level == "" {
		level = p.Manifest.Start.Level
	}
	v.current = sort.SearchStrings(v.levels, level)
	if v.current >= len(v.levels) || v.levels[v.current] != level {
		return nil, fmt.Errorf("no level %q", level)
	}
	if err := v.open(v.current); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *viewer) open(i int) error {
	l := v.pack.Levels[v.levels[i]]
	lv, err := render.NewLevelView(v.pack, v.sprites, l)
	if err != nil {
		return err
	}
	v.current, v.view = i, lv
	v.recenter()
	return nil
}

// recenter centers the view on the start cell on the start level, else on the level.
func (v *viewer) recenter() {
	center := v.view.Bounds().Min.Add(v.view.Bounds().Max).Div(2)
	if s := v.pack.Manifest.Start; s.Level == v.levels[v.current] {
		center = drawlist.TileBox(s.Cell).Add(image.Pt(16, 16))
	}
	v.origin = center.Sub(v.bufferSize().Div(2))
}

func (v *viewer) bufferSize() image.Point {
	return image.Pt((v.screen.X+v.scale-1)/v.scale, (v.screen.Y+v.scale-1)/v.scale)
}

func (v *viewer) Update() error {
	if v.err != nil {
		return v.err
	}
	if v.shotTaken {
		return ebiten.Termination
	}
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

	x, y := ebiten.CursorPosition()
	cursor := image.Pt(x, y)
	v.scroll(cursor)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if v.drag != nil {
			v.origin = v.origin.Sub(cursor.Sub(*v.drag).Div(v.scale))
			moved := cursor.Sub(*v.drag).Div(v.scale).Mul(v.scale)
			next := v.drag.Add(moved)
			v.drag = &next
		} else {
			v.drag = &cursor
		}
	} else {
		v.drag = nil
	}

	switch {
	case justPressed(ebiten.KeyPageDown, ebiten.KeyBracketRight):
		v.err = v.open((v.current + 1) % len(v.levels))
	case justPressed(ebiten.KeyPageUp, ebiten.KeyBracketLeft):
		v.err = v.open((v.current + len(v.levels) - 1) % len(v.levels))
	case justPressed(ebiten.KeyEqual, ebiten.KeyNumpadAdd):
		v.zoom(1)
	case justPressed(ebiten.KeyMinus, ebiten.KeyNumpadSubtract):
		v.zoom(-1)
	case justPressed(ebiten.KeyHome):
		v.recenter()
	}
	return v.err
}

// scroll pans by the scroll wheel, or with Ctrl or Cmd held zooms around the cursor.
func (v *viewer) scroll(cursor image.Point) {
	dx, dy := ebiten.Wheel()
	if dx == 0 && dy == 0 {
		return
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta) {
		v.wheel[0] = 0
		v.wheel[1] += dy
		for ; v.wheel[1] >= 1; v.wheel[1]-- {
			v.zoomAt(1, cursor)
		}
		for ; v.wheel[1] <= -1; v.wheel[1]++ {
			v.zoomAt(-1, cursor)
		}
		return
	}
	// Scrolling up or left shows what is above or to the left.
	v.wheel[0] -= dx * wheelSpeed
	v.wheel[1] -= dy * wheelSpeed
	step := image.Pt(int(v.wheel[0]), int(v.wheel[1]))
	v.wheel[0] -= float64(step.X)
	v.wheel[1] -= float64(step.Y)
	v.origin = v.origin.Add(step)
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
	v.presenter.Present(screen, v.buffer, v.scale)
	if v.screenshot != "" && !v.shotTaken {
		v.shotTaken = true
		v.err = saveScreenshot(screen, v.screenshot)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%s (%d/%d)  scale %d  [ ] level  +/- zoom  scroll to pan  Home recenter",
		v.levels[v.current], v.current+1, len(v.levels), v.scale))
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
