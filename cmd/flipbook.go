package cmd

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/audio/speaker"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
	"github.com/spf13/cobra"
)

func init() {
	var packDir string
	var mute bool
	cmd := &cobra.Command{
		Use:   "flipbook [id]",
		Short: "Play a pack's flipbooks (view effects, sparkles) in a window",
		Long: `Plays a flipbook of the pack, over a plain backdrop, at the largest whole scale that
fits the window, with its sounds and music. Without an id it starts at the first.

Left and Right (or [ and ]) step through the pack's flipbooks; Space pauses, and . steps a
frame while paused; R restarts; Escape or a right click cancels where the flipbook allows it, as
in play; Q quits. A finished flipbook starts again after a second.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pack, err := content.Load(os.DirFS(packDir))
			if err != nil {
				return err
			}
			ids := make([]string, 0, len(pack.Flipbooks))
			for id := range pack.Flipbooks {
				ids = append(ids, id)
			}
			if len(ids) == 0 {
				return fmt.Errorf("%s has no flipbooks", packDir)
			}
			sort.Strings(ids)
			at := 0
			if len(args) > 0 {
				id := args[0]
				if pack.Flipbooks[id] == nil {
					id = content.FlipbookPrefix + id
				}
				if at = slices.Index(ids, id); at < 0 {
					return fmt.Errorf("no flipbook %q", args[0])
				}
			}
			p, err := newFlipbookPlayer(pack, ids, at)
			if err != nil {
				return err
			}
			if !mute {
				p.mixer = audio.New(speaker.New(), pack)
			}
			ebiten.SetWindowTitle("Flipbooks: " + pack.Manifest.Name)
			ebiten.SetWindowSize(windowWidth, windowHeight)
			ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
			if err := ebiten.RunGame(p); err != nil && !errors.Is(err, ebiten.Termination) {
				return err
			}
			return p.err
		},
	}
	cmd.Flags().StringVar(&packDir, "pack", "packs/summoning", "the pack's directory")
	cmd.Flags().BoolVar(&mute, "mute", false, "play no music or sound effects")
	rootCmd.AddCommand(cmd)
}

var (
	flipbookBackdrop = color.RGBA{40, 44, 52, 255}
	flipbookFrame    = color.RGBA{120, 100, 70, 255}
)

// flipbookBorder is the width of a framed flipbook's border, in art pixels.
const flipbookBorder = 8

// restartMS is how long a finished flipbook waits before it starts again.
const restartMS = 1000

type flipbookPlayer struct {
	pack      *content.Pack
	ids       []string
	at        int
	run       *viewer.FlipbookRun
	sprites   *render.Sprites
	presenter *render.Presenter
	palette   *content.Palette
	mixer     *audio.Mixer
	art       *ebiten.Image
	paused    bool
	// clock runs the palette's effects; idle counts time since the run finished.
	clock, idle int
	err         error
	screen      image.Point
}

func newFlipbookPlayer(pack *content.Pack, ids []string, at int) (*flipbookPlayer, error) {
	pr, err := render.NewPresenter()
	if err != nil {
		return nil, err
	}
	p := &flipbookPlayer{pack: pack, ids: ids, at: at, sprites: render.NewSprites(pack), presenter: pr,
		palette: pack.Palettes[pack.Manifest.Palette]}
	p.restart()
	return p, nil
}

func (p *flipbookPlayer) flipbook() *content.Flipbook { return p.pack.Flipbooks[p.ids[p.at]] }

func (p *flipbookPlayer) restart() {
	if p.mixer != nil {
		p.mixer.HaltMusic()
	}
	p.run, p.idle = viewer.NewFlipbookRun(p.flipbook()), 0
	p.step(0)
}

func (p *flipbookPlayer) step(ms int) {
	p.run.Update(ms, func(s *content.FlipbookStep) { viewer.PlayFlipbookStep(p.mixer, s) })
}

func (p *flipbookPlayer) Update() error {
	ms := 1000 / ebiten.TPS()
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyQ):
		return ebiten.Termination
	case inpututil.IsKeyJustPressed(ebiten.KeyRight), inpututil.IsKeyJustPressed(ebiten.KeyBracketRight):
		p.at = (p.at + 1) % len(p.ids)
		p.restart()
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft), inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft):
		p.at = (p.at + len(p.ids) - 1) % len(p.ids)
		p.restart()
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		p.restart()
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		p.paused = !p.paused
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape), inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight):
		p.run.Cancel()
	}
	if p.paused {
		if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) && !p.run.Done {
			p.step(p.flipbook().Steps[p.run.Step].MS - p.run.MS)
		}
	} else {
		p.clock += ms
		if p.run.Done {
			if p.idle += ms; p.idle >= restartMS {
				p.restart()
			}
		} else {
			p.step(ms)
		}
	}
	if p.mixer != nil {
		p.mixer.Update(time.Duration(ms) * time.Millisecond)
	}
	return nil
}

func (p *flipbookPlayer) Draw(screen *ebiten.Image) {
	screen.Fill(flipbookBackdrop)
	f := p.flipbook()
	b := p.pack.Sprites[f.Sprite]
	if b == nil {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("%s: sprite %q missing", f.ID, f.Sprite))
		return
	}
	size := image.Pt(b.FrameSize[0], b.FrameSize[1])
	if p.art == nil || p.art.Bounds().Size() != size {
		p.art = ebiten.NewImage(size.X, size.Y)
	}
	p.art.Clear()
	if err := p.sprites.DrawImage(p.art, content.Image{Sprite: f.Sprite, Frame: p.run.Frame()}, 0, 0, false); err != nil && p.err == nil {
		p.err = err
	}
	outer := size.Add(image.Pt(2*flipbookBorder, 2*flipbookBorder))
	scale := max(1, min(p.screen.X/outer.X, (p.screen.Y-40)/outer.Y))
	at := p.screen.Sub(size.Mul(scale)).Div(2)
	if f.Framed {
		bw := float32(flipbookBorder * scale)
		vector.FillRect(screen, float32(at.X)-bw, float32(at.Y)-bw, float32(size.X*scale)+2*bw, float32(size.Y*scale)+2*bw, flipbookFrame, false)
		vector.FillRect(screen, float32(at.X), float32(at.Y), float32(size.X*scale), float32(size.Y*scale), flipbookBackdrop, false)
	}
	p.presenter.SetColors(p.palette.At(int64(p.clock)), p.palette.Colors)
	p.presenter.PresentAt(screen, p.art, scale, at)
	s := f.Steps[p.run.Step]
	state := ""
	switch {
	case p.run.Done:
		state = "  done"
	case p.paused:
		state = "  paused"
	}
	if s.Cancel && !p.run.Done {
		state += "  (Esc cancels)"
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%s  [%d/%d]  step %d/%d  frame %d  %d ms of %d%s\n%s",
		f.ID, p.at+1, len(p.ids), p.run.Step+1, len(f.Steps), s.Frame, p.run.MS, s.MS, state, stepSounds(s)))
}

func stepSounds(s content.FlipbookStep) string {
	out := ""
	if s.MusicStop {
		out += "music stops  "
	}
	if s.MusicFade {
		out += "music fades  "
	}
	if s.Music != "" {
		out += "music " + s.Music + "  "
	}
	for _, id := range s.Sounds {
		out += id + "  "
	}
	return out
}

func (p *flipbookPlayer) Layout(w, h int) (int, int) {
	p.screen = image.Pt(w, h)
	return w, h
}
