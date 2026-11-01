package sim

import (
	"slices"
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// CellUnits is the size of a cell in world units.
const CellUnits = 32

// Level is one level's state. Its content definition is never changed; everything that can change
// is copied here when the level is first entered.
type Level struct {
	ID  string
	Def *content.Level
	// Floor and Wall are the grids as they are now.
	Floor, Wall content.Grid
	Features    []*Feature
	Things      []*Thing
	// Flights are the things in flight over the level.
	Flights []*Flight
	// Objects are the objects on each cell's floor, bottom first.
	Objects map[content.Cell][]*Object

	byID   map[string]*Feature
	byCell map[content.Cell]*Feature
	// visible and seen are the player's visibility, by cell index; nil while the player has not
	// been on the level.
	visible, seen []bool
}

// Feature is a feature's state: its definition's fields as they are now, plus what the
// simulation keeps.
type Feature struct {
	content.Feature
	// Stage is a door's stage, 0 closed .. 4 open.
	Stage int
	// Counter is the weight on a plate.
	Counter int
	// next is the step a moving door takes its next stage.
	next int64
}

// Door stages (docs/spec/rules.md, Doors).
const (
	stageClosed = 0
	stageOpen   = 4
)

// LookState returns the state of the feature's look that shows it as it is now.
func (f *Feature) LookState() string {
	if f.Kind == content.FeatureDoor && f.SealedBy == "" && f.Stage > stageClosed && f.Stage < stageOpen {
		return strconv.Itoa(f.Stage)
	}
	return f.Feature.LookState()
}

// Moving reports whether a door is moving.
func (f *Feature) Moving() bool {
	return f.State == content.DoorOpening || f.State == content.DoorClosing
}

// Thing is a thing standing in a level.
type Thing struct {
	ID       string
	Template *content.Thing `json:"-"`
	// U and V are its ground point in world units.
	U, V      int
	Facing    int
	HitPoints int
	Items     []*Object
	Rolling   *content.Rolling
	// Phase counts the moves the thing has made, for its walk and roll animations.
	Phase int
	// Hostile, Poisoned and Moving are a creature's state; Pose an action's pose for its next
	// update ("attack").
	Hostile, Poisoned, Moving bool
	Pose                      string
	// Shown is the damage shown over the thing, until the step ShownUntil.
	Shown      int
	ShownUntil int64
	// Lasts is how many more updates a conjured thing lasts; Frame shows a frame of its sprite
	// in place of its animations; Portal is where a portal leads; Mute is a creature that no
	// longer talks.
	Lasts  int
	Frame  *int
	Portal *content.Destination
	Mute   bool
	// casting counts a stationary caster's updates to its next cast; animCount and animStep an
	// animated creature's frame changes; spin the ticks a teleported creature still spins.
	casting             int
	animCount, animStep int
	spin                int
	spinNext            int64
	// walkDir (0..3, 4 resting) and walkCount are a walker's current walk.
	walkDir, walkCount int
	// next is the step of its next update.
	next int64
}

// Cell returns the cell the thing stands in.
func (t *Thing) Cell() content.Cell {
	return content.Cell{floorDiv(t.U, CellUnits), floorDiv(t.V, CellUnits)}
}

// weight is what the thing adds to a plate it stands on.
func (t *Thing) weight() int {
	if t.Template.Has(content.BehaviorResistsTeleport) {
		return 0
	}
	return t.Template.Weight
}

// Object is an object: its item, its value (durability, charges or count), and what it holds if
// it is a container.
type Object struct {
	Item     string
	Value    int
	Contents []*Object
}

// newObject makes an object of item, with the item's starting value.
func (w *World) newObject(item string) *Object {
	o := &Object{Item: item}
	if it := w.pack.Items[item]; it != nil {
		o.Value = it.Durability
		if o.Value == 0 {
			o.Value = it.Charges
		}
	}
	return o
}

// Weight returns the object's weight with what it holds (docs/spec/items.md, Weight and
// containers).
func (o *Object) Weight(p *content.Pack) int {
	it := p.Items[o.Item]
	w := it.Weight
	if it.Container != nil && it.Container.Weightless {
		return w
	}
	for _, c := range o.Contents {
		w += c.Weight(p)
	}
	return w
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// newLevel copies def's state and places what it holds: things with rolled hit points, objects
// with random ones chosen. Plates start with the weight placed on them, running nothing.
func newLevel(w *World, def *content.Level) *Level {
	l := &Level{
		ID: def.ID, Def: def,
		Floor: copyGrid(def.Floor), Wall: copyGrid(def.Wall),
		Objects: map[content.Cell][]*Object{},
		byID:    map[string]*Feature{}, byCell: map[content.Cell]*Feature{},
	}
	for _, fd := range def.Features {
		f := &Feature{Feature: *fd}
		f.Destinations = slices.Clone(fd.Destinations)
		f.Places, f.Set = slices.Clone(fd.Places), slices.Clone(fd.Set)
		if f.Kind == content.FeatureDoor {
			f.Stage = stageClosed
			if f.State == content.DoorOpen || f.State == content.DoorClosing {
				f.Stage = stageOpen
			}
		}
		l.Features = append(l.Features, f)
		l.byID[f.ID] = f
		l.byCell[f.Cell] = f
	}
	for _, pt := range def.Things {
		tmpl := w.pack.Things[pt.Thing]
		t := w.newThing(pt.ID, tmpl)
		t.U, t.V = cellCenter(pt.Cell)
		if pt.Offset != nil {
			t.U, t.V = t.U+pt.Offset[0], t.V+pt.Offset[1]
		}
		if pt.Facing != nil {
			t.Facing = *pt.Facing
		}
		for _, o := range pt.Items {
			if obj := w.object(o); obj != nil {
				t.Items = append(t.Items, obj)
			}
		}
		if pt.Rolling != nil {
			r := *pt.Rolling
			t.Rolling = &r
		}
		l.Things = append(l.Things, t)
		l.addWeight(t.Cell(), t.weight())
	}
	for _, o := range def.Objects {
		obj := w.object(o.Object)
		if obj == nil {
			continue
		}
		l.Objects[o.Cell] = append(l.Objects[o.Cell], obj)
		l.addWeight(o.Cell, obj.Weight(w.pack))
	}
	return l
}

// newThing makes a thing of template tmpl, rolling its hit points.
func (w *World) newThing(id string, tmpl *content.Thing) *Thing {
	return &Thing{ID: id, Template: tmpl, Facing: tmpl.Facing, HitPoints: w.roll(tmpl.HitPoints),
		Hostile: tmpl.Has(content.BehaviorHostile), walkDir: restDir}
}

// object makes an object from its definition, choosing random ones; it returns nil for a random
// object when the pack has no item to choose.
func (w *World) object(o content.Object) *Object {
	item := o.Item
	if item == content.RandomItem {
		if len(w.randomItems) == 0 {
			return nil
		}
		item = w.randomItems[w.rng.IntN(len(w.randomItems))]
	}
	obj := w.newObject(item)
	for _, c := range o.Contents {
		if co := w.object(c); co != nil {
			obj.Contents = append(obj.Contents, co)
		}
	}
	return obj
}

func copyGrid(g content.Grid) content.Grid {
	out := make(content.Grid, len(g))
	for i, row := range g {
		out[i] = slices.Clone(row)
	}
	return out
}

func cellCenter(c content.Cell) (u, v int) {
	return CellUnits*c[0] + CellUnits/2, CellUnits*c[1] + CellUnits/2
}

// Feature returns the feature id, or nil.
func (l *Level) Feature(id string) *Feature { return l.byID[id] }

// FeatureAt returns the feature on cell c, or nil.
func (l *Level) FeatureAt(c content.Cell) *Feature { return l.byCell[c] }

// Thing returns the thing id, or nil.
func (l *Level) Thing(id string) *Thing {
	for _, t := range l.Things {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// ThingsAt returns the things standing in cell c.
func (l *Level) ThingsAt(c content.Cell) []*Thing {
	var out []*Thing
	for _, t := range l.Things {
		if t.Cell() == c {
			out = append(out, t)
		}
	}
	return out
}

// InBounds reports whether c is inside the level.
func (l *Level) InBounds(c content.Cell) bool { return l.Def.InBounds(c) }

// FloorType returns the floor type of cell c, or nil when it has none.
func (l *Level) FloorType(c content.Cell) *content.FloorType {
	if !l.InBounds(c) {
		return nil
	}
	t := l.Floor[c[0]][c[1]]
	if t == content.NoFrame {
		return nil
	}
	return &l.Def.FloorTypes[t]
}

// Blocked reports whether cell c stops things standing in it: a wall, no floor, or a door that
// is not open (docs/spec/world.md, Terrain).
func (l *Level) Blocked(c content.Cell) bool {
	if !l.InBounds(c) || l.Wall[c[0]][c[1]] != content.NoFrame || l.Floor[c[0]][c[1]] == content.NoFrame {
		return true
	}
	f := l.byCell[c]
	return f != nil && f.Kind == content.FeatureDoor && f.Stage != stageOpen
}

// addWeight adds weight to the plate on cell c, if there is one, running nothing.
func (l *Level) addWeight(c content.Cell, weight int) {
	if f := l.byCell[c]; f != nil && f.Kind == content.FeaturePlate {
		f.Counter = max(0, f.Counter+weight)
	}
}

// removeThing takes thing t out of the level.
func (l *Level) removeThing(t *Thing) {
	l.Things = slices.DeleteFunc(l.Things, func(x *Thing) bool { return x == t })
}
