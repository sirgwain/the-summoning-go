package sim

import (
	"slices"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The book (docs/spec/player.md, The book): what the player has come across, by template, and
// how much of it the player has learned.

// Book stages, each reaching the ones before it.
const (
	// StageSeen: a creature or an object of the template was in sight.
	StageSeen = 1 + iota
	// StageFought (creatures) and StageHandled (items): fought with, or held.
	StageFought
	// StageKilled (creatures) and StageUsed (items): one died, or one's power was started.
	StageKilled
)

// StageHandled and StageUsed are the item stages.
const (
	StageHandled = StageFought
	StageUsed    = StageKilled
)

// Entry is what the book keeps about a template: its stage, the level and step it was first
// reached on, and for a creature the kills, the items its dead dropped, in the order first
// dropped, and the defenses learned, in the order learned.
type Entry struct {
	Stage    int      `json:"stage"`
	Level    string   `json:"level"`
	Step     int64    `json:"step"`
	Kills    int      `json:"kills,omitempty"`
	Drops    []string `json:"drops,omitempty"`
	Defenses []string `json:"defenses,omitempty"`
}

// DefenseVulnerable is the defense learned when a creature's vulnerability works; the others are
// named by their behaviors.
const DefenseVulnerable = "vulnerable"

// Knows reports whether the entry has learned defense d.
func (e *Entry) Knows(d string) bool { return e != nil && slices.Contains(e.Defenses, d) }

// Book is the book's entries: creatures by thing template, items by item template; and the
// recipes the player has made, in the order first made.
type Book struct {
	Creatures map[string]*Entry `json:"creatures,omitempty"`
	Items     map[string]*Entry `json:"items,omitempty"`
	Recipes   []Recipe          `json:"recipes,omitempty"`
}

// Recipe is a spell filling the item Holder from the item From in the other hand, none when it
// held nothing the spell turns, making the item Into.
type Recipe struct {
	Spell  string `json:"spell"`
	Holder string `json:"holder"`
	From   string `json:"from,omitempty"`
	Into   string `json:"into"`
}

// learnRecipe records recipe r, once, and enters its items in the book.
func (w *World) learnRecipe(r Recipe) {
	if w.player == nil {
		return
	}
	if !slices.Contains(w.book.Recipes, r) {
		w.book.Recipes = append(w.book.Recipes, r)
	}
	for _, id := range []string{r.Holder, r.From, r.Into} {
		if id != "" {
			w.learnItem(id, StageSeen)
		}
	}
}

// Book returns the book.
func (w *World) Book() *Book { return &w.book }

// reach raises template id's entry in entries to stage, making the entry the first time.
func (w *World) reach(entries *map[string]*Entry, id string, stage int) *Entry {
	if *entries == nil {
		*entries = map[string]*Entry{}
	}
	e := (*entries)[id]
	if e == nil {
		e = &Entry{Level: w.current, Step: w.step}
		(*entries)[id] = e
	}
	e.Stage = max(e.Stage, stage)
	return e
}

// booked reports whether thing t's template goes in the book: a creature, not the player.
func booked(t *Thing) bool {
	return t != nil && t.Template != nil && t.Template.Kind == content.ThingCreature && t.ID != PlayerID
}

// learnCreature raises creature t's entry to stage.
func (w *World) learnCreature(t *Thing, stage int) *Entry {
	if !booked(t) || w.player == nil {
		return nil
	}
	return w.reach(&w.book.Creatures, t.Template.ID, stage)
}

// learnItem raises item id's entry to stage.
func (w *World) learnItem(id string, stage int) {
	if w.player == nil || w.pack.Items[id] == nil {
		return
	}
	w.reach(&w.book.Items, id, stage)
}

// foughtBy raises the entry of the creature on the current level named by, when there is one.
func (w *World) foughtBy(by string) {
	for _, t := range w.Current().Things {
		if t.ID == by {
			w.learnCreature(t, StageFought)
			return
		}
	}
}

// meetDefense learns defense d of creature t, on the current level, when it has it.
func (w *World) meetDefense(l *Level, t *Thing, d string) {
	tm := t.Template
	has := tm.Has(d)
	if d == DefenseVulnerable {
		has = tm.Vulnerability != nil
	}
	if !has || l != w.Current() {
		return
	}
	if e := w.learnCreature(t, StageFought); e != nil && !e.Knows(d) {
		e.Defenses = append(e.Defenses, d)
	}
}

// killed counts creature t's death and the objects it drops.
func (w *World) killed(t *Thing, drops []*Object) {
	e := w.learnCreature(t, StageKilled)
	if e == nil {
		return
	}
	e.Kills++
	for _, o := range drops {
		if !slices.Contains(e.Drops, o.Item) {
			e.Drops = append(e.Drops, o.Item)
		}
	}
}

// look brings the book up to date with what the player sees and holds at the end of an update.
func (w *World) look() {
	p := w.player
	if p == nil {
		return
	}
	l := w.Current()
	if l.Sighted() {
		for _, t := range l.Things {
			if booked(t) && l.Visible(t.Cell()) {
				w.learnCreature(t, StageSeen)
			}
		}
		for c, objs := range l.Objects {
			if n := len(objs); n > 0 && l.Visible(c) {
				w.learnItem(objs[n-1].Item, StageSeen)
			}
		}
	}
	var held func(o *Object)
	held = func(o *Object) {
		if o == nil {
			return
		}
		w.learnItem(o.Item, StageHandled)
		for _, c := range o.Contents {
			held(c)
		}
	}
	held(p.Hand)
	for i, o := range p.Slots {
		held(o)
		// What is worn with a power that starts worn is in use.
		if it := w.objectItem(o); i >= Shelves && it != nil && it.PowerWorn && it.Power != 0 {
			w.learnItem(o.Item, StageUsed)
		}
	}
}

// objectItem returns object o's item, nil for none.
func (w *World) objectItem(o *Object) *content.Item {
	if o == nil {
		return nil
	}
	return w.pack.Items[o.Item]
}

// restoreBook returns a saved book with entries of templates the pack no longer has left out.
func (w *World) restoreBook(b *Book) Book {
	if b == nil {
		return Book{}
	}
	out := Book{}
	for id, e := range b.Creatures {
		if t := w.pack.Things[id]; t != nil && e != nil {
			if out.Creatures == nil {
				out.Creatures = map[string]*Entry{}
			}
			out.Creatures[id] = e
		}
	}
	for id, e := range b.Items {
		if w.pack.Items[id] != nil && e != nil {
			if out.Items == nil {
				out.Items = map[string]*Entry{}
			}
			out.Items[id] = e
		}
	}
	known := func(id string) bool { return id == "" || w.pack.Items[id] != nil }
	for _, r := range b.Recipes {
		if w.pack.Spells[r.Spell] != nil && known(r.Holder) && known(r.From) && known(r.Into) {
			out.Recipes = append(out.Recipes, r)
		}
	}
	return out
}

// bookState returns the book for a save, nil when empty.
func (w *World) bookState() *Book {
	if len(w.book.Creatures) == 0 && len(w.book.Items) == 0 && len(w.book.Recipes) == 0 {
		return nil
	}
	b := w.book
	return &b
}
