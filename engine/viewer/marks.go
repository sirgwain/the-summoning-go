package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/inspect"
)

// The kinds of mark of the selection and of what is under the mouse; other marks are of a link's
// kind.
const (
	MarkSelected inspect.LinkKind = "selected"
	MarkHover    inspect.LinkKind = "hover"
)

// Mark is a draw command of Cmds the shell outlines, for the selection or a link of its kind.
type Mark struct {
	Cmd  int
	Kind inspect.LinkKind
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
	// What to mark, in order: a command's later marks win.
	type target struct {
		thing, feature string
		kind           inspect.LinkKind
	}
	var targets []target
	l := v.World.Current()
	if v.HoverThing != "" {
		targets = append(targets, target{v.HoverThing, "", MarkHover})
	} else if v.Hover != nil {
		if f := l.FeatureAt(*v.Hover); f != nil {
			targets = append(targets, target{"", f.ID, MarkHover})
		}
	}
	if v.Selected != nil {
		for _, lk := range links {
			if lk.Out {
				targets = append(targets, target{lk.Thing, lk.Feature, lk.Kind})
			} else {
				targets = append(targets, target{"", lk.From, lk.Kind})
			}
		}
		if v.Thing != "" {
			targets = append(targets, target{v.Thing, "", MarkSelected})
		} else if f := l.FeatureAt(*v.Selected); f != nil {
			targets = append(targets, target{"", f.ID, MarkSelected})
		}
	}
	if len(targets) == 0 {
		return nil
	}
	var marks []Mark
	for i, c := range v.Cmds {
		if c.Thing == "" && c.Feature == "" {
			continue
		}
		for _, t := range targets {
			if t.thing != "" && c.Thing == t.thing || t.feature != "" && c.Feature == t.feature {
				marks = append(marks, Mark{Cmd: i, Kind: t.kind})
			}
		}
	}
	return marks
}
