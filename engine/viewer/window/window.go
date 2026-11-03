// Package window runs a viewer in a window through Ebitengine: it reads the keyboard and the
// mouse into the viewer's input snapshots, or takes them from a script, and draws the viewer's
// state. The viewer itself (engine/viewer) decides everything shown; this package only draws it.
package window

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/font"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// Window runs a viewer in a window: it reads Ebitengine's input into snapshots, or takes them
// from a script, and draws the viewer's state. It is an ebiten.Game.
type Window struct {
	v         *viewer.Viewer
	sprites   *render.Sprites
	view      *render.LevelView
	presenter *render.Presenter
	glow      *render.Glow
	tr        *transitions
	palette   *content.Palette
	// version is the viewer's draw commands the view holds.
	version int
	buffer  *ebiten.Image
	splash  *ebiten.Image
	// fonts are the faces' letters, made when first drawn.
	fonts map[*font.Face]*ebiten.Image
	// backdrop is how bright the splash artwork shows behind the frontend now.
	backdrop float32
	keys     map[viewer.Key]ebiten.Key
	// script, if set, gives the input; shots are screenshots due at the next draw.
	script *viewer.Runner
	shots  []string
	err    error
	// art holds index images for the hotbar's and the pointer's art, by size.
	art map[image.Point]*ebiten.Image
	// talk, said, inv and log are the conversation window, a mouth's message, the inventory
	// panel and the conversation log, drawn at their own pixels and scaled onto the screen.
	talk, said, inv, log, book *ebiten.Image
	// frames are the panels' frames, by style and size; drawn counts the draws, the frames'
	// clock.
	frames map[frameKey]*framePics
	drawn  int64
}

// New returns a window drawing viewer v.
func New(v *viewer.Viewer) (*Window, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	glow, err := render.NewGlow()
	if err != nil {
		return nil, err
	}
	tr, err := newTransitions()
	if err != nil {
		return nil, err
	}
	s := &Window{
		v: v, sprites: render.NewSprites(v.Pack), presenter: pr, glow: glow, tr: tr,
		palette: v.Pack.Palettes[v.Pack.Manifest.Palette], keys: map[viewer.Key]ebiten.Key{},
		art: map[image.Point]*ebiten.Image{}, frames: map[frameKey]*framePics{},
		fonts: map[*font.Face]*ebiten.Image{},
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

// SetScript has the window take its input from script r rather than the keyboard and mouse, and
// end when the script does.
func (s *Window) SetScript(r *viewer.Runner) { s.script = r }

// sync reloads the view when the viewer has rebuilt its draw commands.
func (s *Window) sync() error {
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
func (s *Window) input() viewer.Input {
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

func (s *Window) Update() error {
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

func (s *Window) Draw(screen *ebiten.Image) {
	v := s.v
	s.drawn++
	if v.Front != nil {
		// The frontend's screens use the system's pointer, which play hides for its own.
		if ebiten.CursorMode() != ebiten.CursorModeVisible {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		}
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
	world := s.tr.world(screen.Bounds().Size(), v.Transition)
	s.presenter.Present(world, s.buffer, v.Scale)
	s.tr.draw(screen, v.Transition, v.Scale)
	overlay := v.Overlay()
	// What is drawn over the world belongs to the new view, so waits for a transition to end.
	if !v.Transitioning() {
		s.drawGlow(screen, overlay.Marks)
		s.drawLinks(screen, overlay.Links)
		s.drawLabels(screen)
	}
	s.drawHotbar(screen)
	s.drawSpellPanel(screen)
	s.drawInventory(screen)
	s.drawLost(screen)
	s.drawSaid(screen)
	s.drawMinimap(screen)
	s.drawTalk(screen)
	s.drawLog(screen)
	s.drawBook(screen)
	s.drawShowing(screen)
	s.drawReading(screen)
	s.drawTools(screen, overlay)
	s.drawNotice(screen)
	s.drawFullMap(screen)
	s.drawPointer(screen)
	s.takeShots(screen)
}

// takeShots saves the screenshots due.
func (s *Window) takeShots(screen *ebiten.Image) {
	for _, path := range s.shots {
		if err := saveScreenshot(screen, path); err != nil && s.err == nil {
			s.err = err
		}
	}
	s.shots = nil
}

func (s *Window) Layout(outsideWidth, outsideHeight int) (int, int) {
	s.v.Screen = image.Pt(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

// scripted reports whether a script, not the mouse, moves the cursor.
func (s *Window) scripted() bool { return s.script != nil }

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

var (
	panelColor     = color.RGBA{0, 0, 0, 200}
	selectionColor = color.RGBA{255, 220, 0, 255}
	hoverColor     = color.RGBA{255, 255, 255, 255}
	damageColor    = color.RGBA{170, 20, 20, 230}
	slotColor      = color.RGBA{22, 27, 36, 230}
	slotEdgeColor  = color.RGBA{86, 102, 126, 255}
	handEdgeColor  = color.RGBA{143, 194, 255, 255}
	// openEdgeColor marks the box of the container open as the pack.
	openEdgeColor  = color.RGBA{90, 220, 120, 255}
	actionColor    = color.RGBA{30, 42, 64, 230}
	waitColor      = color.RGBA{40, 40, 40, 230}
	gaugeColors    = [...]color.RGBA{{200, 30, 30, 255}, {40, 90, 220, 255}, {200, 160, 40, 255}}
	overloadColor  = color.RGBA{255, 80, 0, 255}
	spellEdgeColor = color.RGBA{120, 140, 220, 255}
	talkColor      = color.RGBA{11, 14, 20, 235}
	bookLevelColor = color.RGBA{30, 70, 40, 235}
	keywordColor   = color.RGBA{230, 200, 90, 255}
	// talkSelectedColor fills the highlighted keyword button.
	talkSelectedColor = color.RGBA{150, 105, 40, 245}
	waitShade         = color.RGBA{0, 0, 0, 140}
)
