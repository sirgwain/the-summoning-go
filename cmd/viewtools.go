package cmd

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/render/drawlist"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// tools is the viewer's state for working a level without a player: what is selected, the
// object and thing chosen to drop and place, and what has happened lately.
type tools struct {
	// hover is the cell under the cursor; panel is where the inspector panel was last drawn.
	hover    *content.Cell
	panel    image.Rectangle
	selected *content.Cell
	thing    string
	items    []string
	item     int
	things   []string
	template int
	paused   bool
	events   []string
}

const (
	logLines   = 12
	panelWidth = 456
	// Debug text is 6 by 16 pixels a character.
	charWidth  = 6
	lineHeight = 16
	margin     = 8
)

var (
	panelColor     = color.RGBA{0, 0, 0, 200}
	selectionColor = color.RGBA{255, 220, 0, 255}
	hoverColor     = color.RGBA{255, 255, 255, 160}
	thingColor     = color.RGBA{0, 220, 255, 255}
)

func newTools(p *content.Pack) tools {
	return tools{items: sortedIDs(p.Items), things: sortedIDs(p.Things)}
}

func sortedIDs[V any](m map[string]V) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// cycle steps index i through n entries, backward with Shift.
func cycle(i, n int) int {
	if n == 0 {
		return 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		return (i + n - 1) % n
	}
	return (i + 1) % n
}

var digitKeys = []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9}

// keys turns the tool keys into commands on the selection.
func (v *viewer) keys() []input.Command {
	switch {
	case justPressed(ebiten.KeySpace):
		v.paused = !v.paused
	case justPressed(ebiten.KeyI):
		v.item = cycle(v.item, len(v.items))
	case justPressed(ebiten.KeyC):
		v.template = cycle(v.template, len(v.things))
	}
	if v.selected == nil {
		return nil
	}
	c := *v.selected
	l := v.world.Current()
	f := l.FeatureAt(c)
	switch {
	case justPressed(ebiten.KeyU):
		return []input.Command{input.Use{Cell: c}}
	case justPressed(ebiten.KeyT) && f != nil:
		return []input.Command{input.Switch{Feature: f.ID, Mode: input.SwitchToggle}}
	case justPressed(ebiten.KeyE) && f != nil:
		return []input.Command{input.FollowExit{Feature: f.ID}}
	case justPressed(ebiten.KeyO) && len(v.items) > 0:
		return []input.Command{input.Drop{Cell: c, Item: v.items[v.item]}}
	case justPressed(ebiten.KeyX):
		return []input.Command{input.Remove{Cell: c}}
	case justPressed(ebiten.KeyP) && len(v.things) > 0:
		return []input.Command{input.Place{Cell: c, Thing: v.things[v.template]}}
	}
	if f != nil {
		hooks := inspect.Hooks(f)
		for i, k := range digitKeys {
			if i < len(hooks) && justPressed(k) {
				return []input.Command{input.Run{Procedure: hooks[i]}}
			}
		}
	}
	return nil
}

// log adds what happened to the event log, leaving out sounds.
func (v *viewer) log(events []sim.Event) {
	for _, e := range events {
		var line string
		switch e.Kind {
		case sim.EventSound:
			continue
		case sim.EventMessage:
			line = fmt.Sprintf("%v says: %s", e.Cell, inspect.Message(v.pack, e))
		default:
			line = strings.TrimPrefix(e.String(), e.Kind+" "+e.Level+" ")
			line = e.Kind + " " + line
		}
		v.events = append(v.events, fmt.Sprintf("%5d %s", v.world.Step(), line))
	}
	if n := len(v.events); n > logLines {
		v.events = v.events[n-logLines:]
	}
}

// drawTools draws the selection, the inspector panel and the event log over the view.
func (v *viewer) drawTools(screen *ebiten.Image) {
	if v.hover != nil && (v.selected == nil || *v.hover != *v.selected) {
		v.diamond(screen, *v.hover, hoverColor)
	}
	if v.selected != nil {
		v.diamond(screen, *v.selected, selectionColor)
	}
	if v.thing != "" {
		for _, c := range v.view.Cmds() {
			if c.Thing == v.thing {
				size := v.pack.Sprites[c.Sprite].FrameSize
				at := image.Pt(c.X, c.Y)
				v.outline(screen, image.Rectangle{at, at.Add(image.Pt(size[0], size[1]))}, thingColor)
			}
		}
	}

	b := screen.Bounds()
	width := min(panelWidth, b.Dx()/2)
	lines := v.panelLines(width/charWidth - 1)
	x := b.Max.X - width
	v.panel = image.Rect(x, 0, b.Max.X, len(lines)*lineHeight+margin)
	vector.FillRect(screen, float32(x), 0, float32(width), float32(v.panel.Dy()), panelColor, false)
	for i, s := range lines {
		ebitenutil.DebugPrintAt(screen, s, x+margin/2, i*lineHeight)
	}

	if len(v.events) > 0 {
		h := len(v.events)*lineHeight + margin
		vector.FillRect(screen, 0, float32(b.Max.Y-h), float32(b.Dx()-width), float32(h), panelColor, false)
		for i, s := range v.events {
			ebitenutil.DebugPrintAt(screen, clip(s, (b.Dx()-width)/charWidth-1), margin/2, b.Max.Y-h+i*lineHeight)
		}
	}
}

// diamond strokes the outline of cell c's ground on the screen.
func (v *viewer) diamond(screen *ebiten.Image, c content.Cell, clr color.Color) {
	d := drawlist.Diamond(c)
	for i := range d {
		a, b := v.screenPoint(d[i]), v.screenPoint(d[(i+1)%len(d)])
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 2, clr, true)
	}
}

// outline strokes view rectangle r on the screen.
func (v *viewer) outline(screen *ebiten.Image, r image.Rectangle, c color.Color) {
	lo, hi := v.screenPoint(r.Min), v.screenPoint(r.Max)
	vector.StrokeRect(screen, float32(lo.X), float32(lo.Y), float32(hi.X-lo.X), float32(hi.Y-lo.Y), 2, c, false)
}

// panelLines are the inspector panel's lines, clipped to width characters.
func (v *viewer) panelLines(width int) []string {
	l := v.world.Current()
	status := fmt.Sprintf("%s (%d/%d)  step %d  scale %d", l.ID, v.levelIndex()+1, len(v.levels), v.world.Step(), v.scale)
	if v.paused {
		status += "  PAUSED"
	}
	lines := []string{status}
	if v.hover != nil {
		lines = append(lines, fmt.Sprintf("under the mouse: (%d, %d)", v.hover[0], v.hover[1]))
	}
	if len(v.items) > 0 {
		lines = append(lines, "object (I): "+inspect.ItemName(v.pack, v.items[v.item])+" ["+v.items[v.item]+"]")
	}
	if len(v.things) > 0 {
		lines = append(lines, "thing (C):  "+v.things[v.template])
	}
	lines = append(lines, "")
	switch {
	case v.selected == nil:
		lines = append(lines, "Click a cell or a thing to inspect it.")
	default:
		lines = append(lines, inspect.Cell(v.pack, l, *v.selected)...)
		if t := l.Thing(v.thing); t != nil {
			lines = append(lines, "")
			lines = append(lines, inspect.Thing(v.pack, t)...)
		}
	}
	lines = append(lines, "",
		"U use  T toggle  1-9 run hook  E follow exit",
		"O drop object  X take object  P place thing",
		"Shift+click move thing  Space pause  N step",
		"[ ] level  M magnify  Home recenter")
	for i, s := range lines {
		lines[i] = clip(s, width)
	}
	return lines
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n && n > 3 {
		return string(r[:n-3]) + "..."
	}
	return s
}
