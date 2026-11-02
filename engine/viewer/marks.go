package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The kinds of mark of the selection and of what is under the mouse; other marks are of a link's
// kind.
const (
	MarkSelected = "selected"
	MarkHover    = "hover"
)

// Mark is a draw command of Cmds the shell outlines, for the selection or a link of its kind.
type Mark struct {
	Cmd  int
	Kind string
}

// SelectionLink is a link of the selection: Out when the selected cell's feature acts, else the
// selection is acted on.
type SelectionLink struct {
	inspect.Link
	Out bool
}

// Links are the selection's links: what the selected cell's feature does, and what acts on that
// feature, on the selected thing or on the selected cell, but for where something sends things.
// None while the debug overlay is hidden.
func (v *Viewer) Links() []SelectionLink {
	if v.HideDebug || v.Selected == nil {
		return nil
	}
	l := v.World.Current()
	var feature string
	if f := l.FeatureAt(*v.Selected); f != nil {
		feature = f.ID
	}
	var out []SelectionLink
	for _, lk := range inspect.Links(v.Pack, l) {
		switch {
		case feature != "" && lk.From == feature:
			out = append(out, SelectionLink{lk, true})
		case lk.Kind == inspect.LinkTravel:
			// Where a teleporter, exit, pit or trapdoor sends things shows only from it: nothing
			// on its destination is sent anywhere.
		case feature != "" && lk.Feature == feature,
			v.Thing != "" && lk.Thing == v.Thing,
			lk.Feature == "" && lk.Thing == "" && lk.To == *v.Selected:
			out = append(out, SelectionLink{lk, false})
		}
	}
	return out
}

// Marks are the draw commands to outline: what a click would select (the thing under the mouse,
// else the hovered cell's feature), what the selection's links act on or come from, in their
// kinds, and the selected thing, else the selected cell's feature. Later marks of a command win.
// None while the debug overlay is hidden.
func (v *Viewer) Marks(links []SelectionLink) []Mark {
	if v.HideDebug {
		return nil
	}
	var marks []Mark
	mark := func(thing, feature, kind string) {
		for i, c := range v.Cmds {
			if thing != "" && c.Thing == thing || feature != "" && c.Feature == feature {
				marks = append(marks, Mark{Cmd: i, Kind: kind})
			}
		}
	}
	l := v.World.Current()
	if v.HoverThing != "" {
		mark(v.HoverThing, "", MarkHover)
	} else if v.Hover != nil {
		if f := l.FeatureAt(*v.Hover); f != nil {
			mark("", f.ID, MarkHover)
		}
	}
	if v.Selected == nil {
		return marks
	}
	for _, lk := range links {
		if lk.Out {
			mark(lk.Thing, lk.Feature, string(lk.Kind))
		} else {
			mark("", lk.From, string(lk.Kind))
		}
	}
	if v.Thing != "" {
		mark(v.Thing, "", MarkSelected)
	} else if f := l.FeatureAt(*v.Selected); f != nil {
		mark("", f.ID, MarkSelected)
	}
	return marks
}
