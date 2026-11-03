package inspect

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// LinkKind is what a link does to its target.
type LinkKind string

// Link kinds.
const (
	LinkOn     LinkKind = "on"     // switches a feature on: opens, arms
	LinkOff    LinkKind = "off"    // switches a feature off
	LinkToggle LinkKind = "toggle" // switches a feature over
	LinkHook   LinkKind = "hook"   // points a feature's hook at a procedure
	LinkSet    LinkKind = "set"    // sets a plate's threshold, a slot's item or a mouth's message
	LinkCell   LinkKind = "cell"   // makes an object on a cell, or changes its floor or wall
	LinkShoot  LinkKind = "shoot"  // shoots an object or a spell from a cell
	LinkRoll   LinkKind = "roll"   // starts or stops a boulder
	LinkTravel LinkKind = "travel" // a teleporter's destination, or where an exit, pit or trapdoor leads on the level
)

// Link is one way a feature or a creature reaches a place on its level: an action of a procedure
// one of a feature's hooks runs, or a creature's conversation does, or where a feature sends what
// goes through it.
type Link struct {
	Kind LinkKind
	// From is the feature or the creature (a thing, with Conversation set) acting and its cell; To
	// the cell acted on, and Feature or Thing the feature or thing there acted on, if any.
	From         string
	FromCell, To content.Cell
	Feature      string
	Thing        string
	Direction    int    // the direction shot or rolled; -1 for none
	Item         string // the item it makes, shoots or has a slot take
	// Hooks are the hooks that run it, or for a conversation's link the conditions of the clauses
	// that do, in words.
	Hooks     []string
	Procedure string
	// Conversation is the conversation of the creature From whose clauses run it, if one does:
	// it happens only when the player talks to the creature.
	Conversation string
	Label        string
	// Potential marks a link through a hook a procedure's set_hook action would point at the
	// procedure, rather than the hook as it is now.
	Potential bool
}

// Links returns every link of level l, merging those that differ only in the hook that runs them.
func Links(p *content.Pack, l *sim.Level) []Link {
	var links []Link
	add := func(lk Link) {
		for i := range links {
			o := &links[i]
			if o.Kind == lk.Kind && o.From == lk.From && o.To == lk.To && o.Feature == lk.Feature &&
				o.Thing == lk.Thing && o.Label == lk.Label && o.Potential == lk.Potential && o.Direction == lk.Direction &&
				o.Conversation == lk.Conversation {
				for _, h := range lk.Hooks {
					if !slices.Contains(o.Hooks, h) {
						o.Hooks = append(o.Hooks, h)
					}
				}
				return
			}
		}
		links = append(links, lk)
	}
	run := func(f *sim.Feature, hook, proc string, potential bool) {
		for _, a := range l.Def.Procedures[proc] {
			if lk, ok := actionLink(p, l, a); ok {
				lk.From, lk.FromCell, lk.Hooks, lk.Procedure, lk.Potential = f.ID, f.Cell, []string{hook}, proc, potential
				add(lk)
			}
		}
	}
	for _, f := range l.Features {
		for _, h := range hookNames {
			if hp := f.Hook(h); hp != nil && *hp != "" {
				run(f, h, *hp, false)
			}
		}
	}
	// What set_hook actions would make features do.
	for _, id := range sortedKeys(l.Def.Procedures) {
		for _, a := range l.Def.Procedures[id] {
			sh := a.SetHook
			if sh == nil || sh.Procedure == nil || *sh.Procedure == "" {
				continue
			}
			f := l.Feature(sh.Feature)
			if f == nil {
				continue
			}
			if hp := f.Hook(sh.Hook); hp != nil && *hp != *sh.Procedure {
				run(f, sh.Hook, *sh.Procedure, true)
			}
		}
	}
	// What the level's creatures' conversations do to it.
	for _, t := range l.Things {
		conv := p.Conversations[t.Template.Conversation]
		if conv == nil {
			continue
		}
		ConversationActions(l, conv, func(_ int, c content.Clause, a content.Action, proc string) {
			if lk, ok := actionLink(p, l, a); ok {
				lk.From, lk.FromCell, lk.Hooks, lk.Procedure, lk.Conversation = t.ID, t.Cell(), []string{Condition(p, c.If)}, proc, conv.ID
				add(lk)
			}
		})
	}
	for _, f := range l.Features {
		switch f.Kind {
		case content.FeatureTeleporter:
			for i, d := range f.Destinations {
				label := "then"
				switch {
				case len(f.Destinations) == 2 && f.Destinations[0] == f.Destinations[1]:
					if i > 0 {
						continue
					}
					label = "teleport"
				case i == f.Next:
					label = "next"
				}
				add(Link{Kind: LinkTravel, From: f.ID, FromCell: f.Cell, To: d, Direction: -1, Label: label})
			}
		case content.FeatureExit, content.FeaturePit, content.FeatureTrapdoor:
			if f.To != nil && f.To.Level == l.ID {
				add(Link{Kind: LinkTravel, From: f.ID, FromCell: f.Cell, To: f.To.Cell, Direction: -1, Label: f.Kind})
			}
		}
	}
	return links
}

// actionLink returns the link action a makes, without its source; false for an action that acts
// on no place of the level.
func actionLink(p *content.Pack, l *sim.Level, a content.Action) (Link, bool) {
	lk, ok := actionTarget(p, l, a)
	lk.Kind, lk.Item = ActionKind(a), ActionItem(a)
	return lk, ok
}

// ActionKind returns the kind of link action a makes; empty for an action that acts on no place
// of the level.
func ActionKind(a content.Action) LinkKind {
	switch {
	case a.On != nil:
		return LinkOn
	case a.Off != nil:
		return LinkOff
	case a.Toggle != nil:
		return LinkToggle
	case a.SetHook != nil:
		return LinkHook
	case a.SetThreshold != nil, a.SetAccepts != nil, a.SetMessage != nil:
		return LinkSet
	case a.Roll != nil, a.Stop != nil:
		return LinkRoll
	case a.ShootItem != nil, a.ShootSpell != nil:
		return LinkShoot
	case a.Create != nil, a.SetFloor != nil, a.SetWall != nil:
		return LinkCell
	}
	return ""
}

// actionTarget returns where action a acts and what it does there, without the link's kind.
func actionTarget(p *content.Pack, l *sim.Level, a content.Action) (Link, bool) {
	feature := func(id, label string) (Link, bool) {
		f := l.Feature(id)
		if f == nil {
			return Link{}, false
		}
		return Link{To: f.Cell, Feature: id, Direction: -1, Label: label}, true
	}
	thing := func(id string, dir int, label string) (Link, bool) {
		t := l.Thing(id)
		if t == nil {
			return Link{}, false
		}
		return Link{To: t.Cell(), Thing: id, Direction: dir, Label: label}, true
	}
	cell := func(c content.Cell, dir int, label string) (Link, bool) {
		return Link{To: c, Direction: dir, Label: label}, true
	}
	switch {
	case a.On != nil:
		return feature(*a.On, "on")
	case a.Off != nil:
		return feature(*a.Off, "off")
	case a.Toggle != nil:
		return feature(*a.Toggle, "toggle")
	case a.SetHook != nil:
		to := "nothing"
		if a.SetHook.Procedure != nil && *a.SetHook.Procedure != "" {
			to = *a.SetHook.Procedure
		}
		return feature(a.SetHook.Feature, shortHook(a.SetHook.Hook)+" runs "+to)
	case a.SetThreshold != nil:
		return feature(a.SetThreshold.Feature, "threshold "+strconv.Itoa(a.SetThreshold.Value))
	case a.SetAccepts != nil:
		return feature(a.SetAccepts.Feature, "takes "+ItemName(p, a.SetAccepts.Item))
	case a.SetMessage != nil:
		return feature(a.SetMessage.Feature, "message")
	case a.Roll != nil:
		label := "roll"
		if a.Roll.Shuttle {
			label = "shuttle"
		}
		return thing(a.Roll.Thing, a.Roll.Direction, label)
	case a.Stop != nil:
		return thing(*a.Stop, -1, "stop")
	case a.ShootItem != nil:
		return cell(a.ShootItem.Cell, a.ShootItem.Direction, "shoot "+ItemName(p, a.ShootItem.Item))
	case a.ShootSpell != nil:
		return cell(a.ShootSpell.Cell, a.ShootSpell.Direction, "cast "+a.ShootSpell.Spell)
	case a.Create != nil:
		return cell(a.Create.Cell, -1, "make "+ItemName(p, a.Create.Item))
	case a.SetFloor != nil:
		return cell(a.SetFloor.Cell, -1, fmt.Sprintf("floor %d", a.SetFloor.Type))
	case a.SetWall != nil:
		return cell(a.SetWall.Cell, -1, fmt.Sprintf("wall %d", a.SetWall.Shape))
	}
	return Link{}, false
}

// shortHook is a hook's name without its "on_".
func shortHook(h string) string { return strings.TrimPrefix(h, "on_") }

// Text is the link in words, for a label: the hooks that run it, then what it does. A
// conversation's link starts with "talk" and the conditions of the clauses that run it.
func (lk Link) Text() string {
	if len(lk.Hooks) == 0 {
		return lk.Label
	}
	if lk.Conversation != "" {
		s := "talk, " + strings.Join(lk.Hooks, " / ")
		if lk.Procedure != "" {
			s += " (" + lk.Procedure + ")"
		}
		return s + ": " + lk.Label
	}
	hooks := make([]string, len(lk.Hooks))
	for i, h := range lk.Hooks {
		hooks[i] = shortHook(h)
	}
	s := strings.Join(hooks, "/") + ": " + lk.Label
	if lk.Potential {
		s += " (if rewired)"
	}
	return s
}
