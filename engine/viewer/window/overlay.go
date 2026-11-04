package window

import (
	"image"
	"image/color"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/render"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
)

// drawTools draws the selection, the inspector panel and the event log over the view.
func (s *Window) drawTools(screen *ebiten.Image, o viewer.Overlay) {
	v := s.v
	if v.HideDebug {
		return
	}
	// A selected thing glows in place of its cell's outline.
	if v.Selected != nil && v.Thing == "" {
		s.diamond(screen, *v.Selected, selectionColor)
	}
	text := func(r image.Rectangle, lines []inspect.Line) {
		vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), panelColor, false)
		for i, line := range lines {
			x, y := r.Min.X+viewer.Margin/2, r.Min.Y+i*viewer.LineHeight
			if c, ok := markColors[line.Kind]; ok && line.Kind != "" {
				s.printColored(screen, line.Text, x, y, c)
			} else {
				s.print(screen, line.Text, x, y)
			}
			if line.Item != "" {
				s.drawItemIcon(screen, line.Item, x+utf8.RuneCountInString(line.Text)*viewer.CharWidth+4, y, r.Max.X)
			}
		}
	}
	text(o.PanelRect, o.Panel)
	if !o.LogRect.Empty() {
		var lines []inspect.Line
		for _, l := range o.Log {
			lines = append(lines, inspect.Line{Text: l})
		}
		text(o.LogRect, lines)
	}
}

// drawItemIcon draws item's inventory icon with its left edge at x, ending by right, its
// middle on the line of text at y.
func (s *Window) drawItemIcon(screen *ebiten.Image, item string, x, y, right int) {
	it := s.v.Pack.Items[item]
	if it == nil {
		return
	}
	b := s.v.Pack.Sprites[it.Icon.Sprite]
	if b == nil {
		return
	}
	size := image.Pt(b.FrameSize[0], b.FrameSize[1])
	art := s.artImage(size)
	if err := s.sprites.DrawImage(art, it.Icon, 0, 0, false); err != nil {
		return
	}
	x = min(x, right-size.X-2)
	s.presenter.PresentAt(screen, art, 1, image.Pt(x, y+(viewer.LineHeight-size.Y)/2))
}

// printColored prints text at (x, y) on the screen in color c.
func (s *Window) printColored(screen *ebiten.Image, text string, x, y int, c color.RGBA) {
	s.printIn(screen, text, x, y, c)
}

// markColors are the colors of the glow and the links for the selection and each kind of link.
var markColors = map[inspect.LinkKind]color.RGBA{
	viewer.MarkSelected: selectionColor,
	viewer.MarkHover:    hoverColor,
	inspect.LinkOn:      {90, 220, 110, 255},
	inspect.LinkOff:     {240, 80, 70, 255},
	inspect.LinkToggle:  {255, 150, 40, 255},
	inspect.LinkHook:    {220, 110, 255, 255},
	inspect.LinkSet:     {100, 180, 255, 255},
	inspect.LinkCell:    {120, 240, 230, 255},
	inspect.LinkShoot:   {255, 90, 160, 255},
	inspect.LinkRoll:    {210, 170, 110, 255},
	inspect.LinkTravel:  {160, 140, 255, 255},
}

// windUpColor is the glow of a creature winding up a blow.
var windUpColor = color.RGBA{255, 50, 30, 255}

// drawWindUps glows around the creatures winding up a blow (docs/spec/combat.md, Wind-up).
func (s *Window) drawWindUps(screen *ebiten.Image) {
	v := s.v
	ws := v.WindUps()
	if len(ws) == 0 {
		return
	}
	marks := make([]render.Mark, 0, len(ws))
	for _, w := range ws {
		if w.Strength < 0.02 {
			continue
		}
		marks = append(marks, render.Mark{Cmd: w.Cmd, Color: windUpColor, Strength: w.Strength})
	}
	s.glow.DrawSteady(screen, s.view, marks, v.Origin, v.BufferSize(), v.Scale)
}

// drawGlow outlines the selection and what its links act on or come from with a glow.
func (s *Window) drawGlow(screen *ebiten.Image, vm []viewer.Mark) {
	v := s.v
	if len(vm) == 0 {
		return
	}
	marks := make([]render.Mark, len(vm))
	for i, m := range vm {
		marks[i] = render.Mark{Cmd: m.Cmd, Color: markColors[m.Kind]}
	}
	s.glow.Draw(screen, s.view, marks, v.Origin, v.BufferSize(), v.Scale, float64(v.Ticks)/float64(ebiten.TPS()))
}

// drawLinks draws the selection's links as arrows from the feature or creature acting to what it
// acts on, dashed for a link through a hook a set_hook would point at the procedure, or through a
// conversation. What each does is
// labeled at its far end from the selection, the labels of a cell stacked. A cell acted on has
// its ground outlined, and a direction shot or rolled in is an arrow from it.
func (s *Window) drawLinks(screen *ebiten.Image, links []viewer.SelectionLink) {
	v := s.v
	type pair struct{ a, b content.Cell }
	seen := map[pair]int{}
	type line struct {
		text string
		c    color.RGBA
	}
	labels := map[content.Cell][]line{}
	var order []content.Cell
	for _, lk := range links {
		end := lk.To
		if !lk.Out {
			end = lk.FromCell
		}
		if end != *v.Selected {
			text := lk.Text()
			if !slices.ContainsFunc(labels[end], func(l line) bool { return l.text == text }) {
				if labels[end] == nil {
					order = append(order, end)
				}
				labels[end] = append(labels[end], line{text, markColors[lk.Kind]})
			}
		}
		c := markColors[lk.Kind]
		if lk.Feature == "" && lk.Thing == "" {
			s.diamond(screen, lk.To, c)
		}
		to := v.CellPoint(lk.To)
		if lk.Direction >= 0 && lk.Direction < len(sim.Dirs) {
			d := sim.Dirs[lk.Direction]
			ahead := v.CellPoint(content.Cell{lk.To[0] + 2*d[0], lk.To[1] + 2*d[1]})
			arrow(screen, to, to.Add(ahead.Sub(to).Div(2)), c, false)
		}
		if lk.FromCell == lk.To {
			continue
		}
		key := pair{lk.FromCell, lk.To}
		if key.b[0] < key.a[0] || key.b[0] == key.a[0] && key.b[1] < key.a[1] {
			key.a, key.b = key.b, key.a
		}
		n := seen[key]
		seen[key]++
		from := v.CellPoint(lk.FromCell)
		// Links between the same two cells lie side by side.
		dx, dy := float64(to.X-from.X), float64(to.Y-from.Y)
		length := math.Hypot(dx, dy)
		off := image.Pt(int(math.Round(-dy/length*float64(6*n))), int(math.Round(dx/length*float64(6*n))))
		arrow(screen, from.Add(off), to.Add(off), c, lk.Potential || lk.Conversation != "")
	}
	for _, cell := range order {
		lines := labels[cell]
		at := v.CellPoint(cell)
		w := 0
		for _, l := range lines {
			w = max(w, utf8.RuneCountInString(l.text)*viewer.CharWidth+10)
		}
		x, y := at.X-w/2, at.Y-8-len(lines)*viewer.LineHeight
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(len(lines)*viewer.LineHeight), panelColor, false)
		for i, l := range lines {
			ly := y + i*viewer.LineHeight
			vector.FillRect(screen, float32(x), float32(ly), 4, float32(viewer.LineHeight), l.c, false)
			s.print(screen, l.text, x+7, ly)
		}
	}
}

// arrow draws a line from a to b with a head at b, dashed if asked.
func arrow(screen *ebiten.Image, a, b image.Point, c color.RGBA, dashed bool) {
	ax, ay, bx, by := float64(a.X), float64(a.Y), float64(b.X), float64(b.Y)
	dx, dy := bx-ax, by-ay
	length := math.Hypot(dx, dy)
	if length < 1 {
		return
	}
	ux, uy := dx/length, dy/length
	line := func(x0, y0, x1, y1 float64) {
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 2, c, true)
	}
	if dashed {
		for t := 0.0; t < length; t += 12 {
			e := min(t+7, length)
			line(ax+ux*t, ay+uy*t, ax+ux*e, ay+uy*e)
		}
	} else {
		line(ax, ay, bx, by)
	}
	const head = 10
	for _, side := range []float64{-1, 1} {
		// The head's sides, 25 degrees off the line.
		cs, sn := math.Cos(25*math.Pi/180), side*math.Sin(25*math.Pi/180)
		hx, hy := -(ux*cs-uy*sn)*head, -(uy*cs+ux*sn)*head
		line(bx, by, bx+hx, by+hy)
	}
}

// drawLabels draws the damage shown over things: white numbers on red, centered over each
// sprite's top. A stand-in for the game's own until the UI has its font.
func (s *Window) drawLabels(screen *ebiten.Image) {
	for _, l := range s.v.Labels {
		at := s.v.ScreenPoint(l.At)
		w := len(l.Text)*viewer.CharWidth + 4
		x, y := at.X-w/2, at.Y-viewer.LineHeight
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(viewer.LineHeight), damageColor, false)
		s.print(screen, l.Text, x+2, y)
	}
}

// diamond strokes the outline of cell c's ground on the screen.
func (s *Window) diamond(screen *ebiten.Image, c content.Cell, clr color.Color) {
	d := drawlist.Diamond(c)
	for i := range d {
		a, b := s.v.ScreenPoint(d[i]), s.v.ScreenPoint(d[(i+1)%len(d)])
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 2, clr, true)
	}
}
