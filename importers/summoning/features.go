package summoning

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// More terrain values (world.md).
const (
	terrainDoorH       = 0x10
	terrainDoorV       = 0x11
	terrainSealedDoor  = 0x15
	terrainSlot        = 0x17 // 17h, 18h
	terrainLever       = 0x19 // 19h off, 1Ah on
	terrainLeverOn     = 0x1A
	terrainPlateDown   = 0x20
	terrainFirstMark   = 0x21 // 21h–23h
	terrainLastMark    = 0x23
	terrainPit         = 0x25
	terrainTrapdoorOpn = 0x26
)

// Placement fields (formats.md, Entity stream; rules.md).
const (
	triggerDisarmed  = 1
	triggerThreshold = 2
	triggerProcA     = 3
	triggerProcB     = 4
	noProcedure      = 0xFF
	exitMarker       = 0
	teleporterNext   = 0x04
	teleporterFly    = 0x08
	doorKano         = 0x20
	doorMoving       = 0x18
	boulderRolling   = 1
	boulderShuttling = 2
)

// The level-30 slot special case (rules.md, Item slots).
const (
	specialSlotLevel     = 30
	specialSlotProcedure = 0x18
)

var specialSlotCell = content.Cell{40, 11}

// Where looks place their art, from the cell's tile box (world.md, Drawing a level).
var (
	plateAt   = content.Offset{-16, 8}
	overlayAt = content.Offset{4, 11}
	socketAt  = content.Offset{-1, 16}
	// standingAt places exit markers and mouths (entities.md, Engine model).
	standingAt = content.Offset{8, -20}
	doorAtH    = [2]content.Offset{{16 - 33, -3}, {16 - 1, -14}}
	doorAtV    = [2]content.Offset{{16 - 30, -14}, {16 + 2, -3}}
)

// Fixed resources (formats.md, Fixed resources).
const (
	exitMarkerResource = 99
	mouthResource      = 56
)

// levelBuilder turns one level's placements, terrain features and procedures into content.
type levelBuilder struct {
	im    *importer
	index int
	l     *Level
	cl    *content.Level
	// plain is the floor type of cells that are not 1Bh–1Eh.
	plain int

	features map[content.Cell]*content.Feature
	// boulders are the things placed as boulders, by the cell they were placed on.
	boulders map[content.Cell]string
	thingIDs map[string]int
	// The records placed at each cell.
	triggers, teleporters, exits, doors, mouths map[content.Cell]*Placement
	// usedProcs are procedures hooks and actions refer to.
	usedProcs map[int]bool
}

func (lb *levelBuilder) what(c content.Cell, format string, args ...any) string {
	return fmt.Sprintf("level %d (%d, %d): ", lb.index, c[0], c[1]) + fmt.Sprintf(format, args...)
}

func (lb *levelBuilder) report(c content.Cell, format string, args ...any) {
	lb.im.report.addf("%s", lb.what(c, format, args...))
}

// floorTypes gives the level its floor types (world.md, Import) and returns the plain type.
func (lb *levelBuilder) floorTypes(variants int) {
	h := &lb.l.Header
	types := make([]content.FloorType, variants)
	for k := range types {
		types[k].Variant = k
	}
	var plain *content.FloorType
	fire := &content.Effect{Damage: content.Dice{Count: 2, Sides: 2}, Type: "fire"}
	burning := &content.Effect{Damage: fire.Damage, Type: fire.Type, Cue: content.CueHurt}
	switch {
	case h.DeepWater():
		for k := range types {
			types[k].Depth, types[k].Sinks = k+1, true
		}
		plain = &content.FloorType{Sinks: true}
	case h.LiquidKind == LiquidWater:
		if variants > 3 {
			types[2].Enter, types[2].Avoided = &content.Effect{Damage: content.Dice{Bonus: 1}, Cue: content.CueHurt}, true
			types[3].Enter = &content.Effect{DrainMagic: 1}
		}
		plain = &content.FloorType{}
	case h.LiquidKind == LiquidFire:
		for k := range types {
			types[k].Enter, types[k].Tick = fire, burning
		}
		plain = &content.FloorType{Tick: burning}
	}
	lb.cl.FloorTypes = types
	if plain != nil {
		lb.plain = len(types)
		lb.cl.FloorTypes = append(lb.cl.FloorTypes, *plain)
	}
}

// placements reads the entity stream: objects and things, decorations and the records the
// terrain features use.
func (lb *levelBuilder) placements() error {
	for i := range lb.l.Placements {
		p := &lb.l.Placements[i]
		c := content.Cell{int(p.X), int(p.Y)}
		if !lb.cl.InBounds(c) {
			lb.report(c, "placement type %d outside the level; left out", p.Type)
			continue
		}
		switch p.Type {
		case PlaceNPC:
			lb.creature(c, p)
		case PlaceContainer:
			lb.container(c, p.Objects)
		default:
			for _, t := range p.Objects {
				if o, ok := lb.object(c, t); ok {
					lb.cl.Objects = append(lb.cl.Objects, &content.FloorObject{Cell: c, Object: o})
				}
			}
		}
		if p.Trap {
			if err := lb.trap(c); err != nil {
				return err
			}
		}

		var records map[content.Cell]*Placement
		switch p.Type {
		case PlaceDecoration:
			if err := lb.decoration(c, p); err != nil {
				return err
			}
		case PlaceBoulder:
			if err := lb.boulder(c, p); err != nil {
				return err
			}
		case PlaceDoor:
			records = lb.doors
		case PlaceTeleporter:
			records = lb.teleporters
		case PlaceTrigger:
			records = lb.triggers
		case PlaceExit:
			records = lb.exits
		case PlaceMouth:
			records = lb.mouths
		}
		if records != nil {
			if records[c] != nil {
				lb.report(c, "two placements of type %d; the first is left out", p.Type)
			}
			records[c] = p
		}
	}
	return nil
}

// object returns an object of template t; template 0 is a random object.
func (lb *levelBuilder) object(c content.Cell, t byte) (content.Object, bool) {
	if t == templateRandom {
		return content.Object{Item: content.RandomItem}, true
	}
	id, ok := lb.im.item(int(t), lb.what(c, "object"))
	return content.Object{Item: id}, ok
}

// container places a container holding the objects after it; what does not fit lies beside it.
func (lb *levelBuilder) container(c content.Cell, templates []byte) {
	if len(templates) == 0 {
		return
	}
	box, ok := lb.object(c, templates[0])
	tm := (*ObjectTemplate)(nil)
	if ok && box.Item != content.RandomItem {
		tm = &lb.im.objects.Templates[templates[0]]
	}
	if tm == nil || !tm.Container() {
		lb.report(c, "container placement whose first object is not a container; all on the floor")
		tm = nil
	}
	var floor []content.Object
	if ok {
		floor = append(floor, box)
	}
	for _, t := range templates[1:] {
		o, ok := lb.object(c, t)
		if !ok {
			continue
		}
		fits := tm != nil && len(floor[0].Contents) < tm.Capacity() &&
			(t == templateRandom || lb.im.objects.Templates[t].Size() <= tm.Largest())
		if fits {
			floor[0].Contents = append(floor[0].Contents, o)
		} else {
			floor = append(floor, o)
		}
	}
	for _, o := range floor {
		lb.cl.Objects = append(lb.cl.Objects, &content.FloorObject{Cell: c, Object: o})
	}
}

// thingID returns a new local id for a thing of kind at c.
func (lb *levelBuilder) thingID(kind string, c content.Cell) string {
	id := fmt.Sprintf("%s.%d.%d", kind, c[0], c[1])
	if lb.thingIDs[id]++; lb.thingIDs[id] > 1 {
		id = fmt.Sprintf("%s.%d", id, lb.thingIDs[id])
	}
	return id
}

func (lb *levelBuilder) creature(c content.Cell, p *Placement) {
	tid, err := lb.im.creature(int(p.Extra[0]))
	if err != nil {
		lb.report(c, "creature: %v; left out", err)
		return
	}
	offset := creatureOffset
	t := &content.PlacedThing{ID: lb.thingID("creature", c), Thing: tid, Cell: c, Offset: &offset}
	for _, tmpl := range p.Objects {
		if o, ok := lb.object(c, tmpl); ok {
			t.Items = append(t.Items, o)
		}
	}
	lb.cl.Things = append(lb.cl.Things, t)
}

func (lb *levelBuilder) trap(c content.Cell) error {
	tid, err := lb.im.trap(lb.im.terrain(int(lb.l.Header.TrapBank)))
	if err != nil {
		return err
	}
	lb.cl.Things = append(lb.cl.Things, &content.PlacedThing{ID: lb.thingID("trap", c), Thing: tid, Cell: c})
	return nil
}

func (lb *levelBuilder) boulder(c content.Cell, p *Placement) error {
	tid, err := lb.im.boulder(lb.im.terrain(int(lb.l.Header.BoulderBank)))
	if err != nil {
		return err
	}
	t := &content.PlacedThing{ID: lb.thingID("boulder", c), Thing: tid, Cell: c}
	state, dir := p.Extra[0], int(p.Extra[1])
	switch {
	case state == 0:
	case dir > 3:
		lb.report(c, "boulder direction %d; placed resting", dir)
	case state == boulderRolling || state == boulderShuttling:
		t.Rolling = &content.Rolling{Direction: 2 * dir, Shuttle: state == boulderShuttling}
	default:
		lb.report(c, "boulder state %d; placed resting", state)
	}
	lb.boulders[c] = t.ID
	lb.cl.Things = append(lb.cl.Things, t)
	return nil
}

// decoration adds a wall decoration, drawn only on shapes 1 and 3.
func (lb *levelBuilder) decoration(c content.Cell, p *Placement) error {
	h := &lb.l.Header
	t := int(lb.l.TerrainAt(c[0], c[1]))
	if t != 1 && t != 3 {
		return nil
	}
	ornament, decoration := p.Decoration()
	if decoration >= len(h.DecorationBanks) {
		lb.report(c, "decoration %d: no such decoration bank; not drawn", decoration)
		return nil
	}
	id := int(lb.im.init.Bases.WallDecoration) + int(h.DecorationBanks[decoration])
	return lb.im.place(&lb.cl.Decorations, c, id, ornament, content.Offset{8, 8}, "decoration")
}

// add adds feature f, named after its kind and cell.
func (lb *levelBuilder) add(f *content.Feature) {
	f.ID = fmt.Sprintf("%s.%d.%d", f.Kind, f.Cell[0], f.Cell[1])
	if old := lb.features[f.Cell]; old != nil {
		lb.report(f.Cell, "%s and %s on one cell; the %s is left out", old.Kind, f.Kind, f.Kind)
		return
	}
	lb.features[f.Cell] = f
}

// hook returns the name of procedure n, or "" for none.
func (lb *levelBuilder) hook(c content.Cell, n byte) string {
	if n == noProcedure {
		return ""
	}
	if int(n) >= len(lb.l.Procedures) {
		lb.report(c, "procedure %d does not exist; hook left empty", n)
		return ""
	}
	lb.usedProcs[int(n)] = true
	return procName(int(n))
}

func procName(n int) string { return "p" + strconv.Itoa(n) }

// terrainFeatures makes the features of slots, levers, plates, marks, teleporters, pits,
// trapdoors and the gem socket (rules.md, Import).
func (lb *levelBuilder) terrainFeatures() error {
	h := &lb.l.Header
	plates, overlays := lb.im.terrain(int(h.PlateBank)), lb.im.terrain(int(h.FloorOverlayBank))
	for x := range lb.l.Rows {
		for y := range lb.l.Columns {
			c := content.Cell{x, y}
			t := int(lb.l.TerrainAt(x, y))
			trig := lb.triggers[c]
			if trig != nil && !triggerTerrain(t) {
				lb.report(c, "trigger on terrain %02Xh; left out", t)
				trig = nil
			}
			var f *content.Feature
			var err error
			switch {
			case t == terrainSlot || t == terrainSlot+1:
				if trig == nil {
					lb.report(c, "slot without a trigger; drawn as fixed art")
					err = lb.im.place(&lb.cl.Overlays, c, overlays, t-terrainSlot, overlayAt, "slot")
					break
				}
				accepts, ok := lb.im.item(int(trig.Extra[triggerThreshold]), lb.what(c, "slot"))
				if !ok {
					break
				}
				a := trig.Extra[triggerProcA]
				if lb.index == specialSlotLevel && c == specialSlotCell && a == 0 {
					a = specialSlotProcedure
				}
				f = &content.Feature{Kind: content.FeatureSlot, Accepts: accepts, Armed: trig.Extra[triggerDisarmed] == 0,
					Keyhole: t == terrainSlot, OnInsert: lb.hook(c, a)}
				if trig.Extra[triggerProcB] != noProcedure {
					lb.im.report.addf("slot procedure b: not used (rules.md, Unknowns)")
				}
				f.Look, err = lb.im.look(fmt.Sprintf("look.slot.%d.%d", overlays, t-terrainSlot), content.LayerFloor,
					map[string][]lookArt{content.FeatureSlot: {{overlays, t - terrainSlot, overlayAt}}})
			case t == terrainLever || t == terrainLeverOn:
				f = &content.Feature{Kind: content.FeatureLever, On: t == terrainLeverOn}
				if trig != nil {
					f.Armed = trig.Extra[triggerDisarmed] == 0
					f.OnTurnOn, f.OnTurnOff = lb.hook(c, trig.Extra[triggerProcA]), lb.hook(c, trig.Extra[triggerProcB])
				}
				f.Look, err = lb.im.look(fmt.Sprintf("look.lever.%d", overlays), content.LayerFloor, map[string][]lookArt{
					"off": {{overlays, 2, overlayAt}}, "on": {{overlays, 3, overlayAt}},
				})
			case t == terrainPlate || t == terrainPlateDown:
				f = &content.Feature{Kind: content.FeaturePlate, Down: t == terrainPlateDown}
				if trig != nil {
					f.Armed, f.Threshold = trig.Extra[triggerDisarmed] == 0, int(trig.Extra[triggerThreshold])
					f.OnPress, f.OnRelease = lb.hook(c, trig.Extra[triggerProcA]), lb.hook(c, trig.Extra[triggerProcB])
				}
				f.Look, err = lb.im.look(fmt.Sprintf("look.plate.%d", plates), content.LayerFloor, map[string][]lookArt{
					"up": {{plates, 0, plateAt}}, "down": {{plates, 1, plateAt}},
				})
			case t >= terrainFirstMark && t <= terrainLastMark:
				n := t - terrainFirstMark + 1
				f = &content.Feature{Kind: content.FeatureMark, Rune: strconv.Itoa(n)}
				f.Look, err = lb.im.look(fmt.Sprintf("look.mark.%d.%d", plates, n), content.LayerFloor,
					map[string][]lookArt{content.FeatureMark: {{plates, t - terrainPlate, plateAt}}})
			case t == terrainTeleporter:
				f, err = lb.teleporter(c, plates)
			case t == terrainPit:
				ex := lb.exits[c]
				if ex == nil {
					lb.report(c, "pit without an exit; drawn as fixed art")
					err = lb.im.place(&lb.cl.Overlays, c, plates, t-terrainPlate, plateAt, "pit")
					break
				}
				to, _ := exitDestination(ex)
				f = &content.Feature{Kind: content.FeaturePit, To: to}
				f.Look, err = lb.im.look(fmt.Sprintf("look.pit.%d", plates), content.LayerFloor,
					map[string][]lookArt{content.FeaturePit: {{plates, t - terrainPlate, plateAt}}})
			case t == terrainTrapdoorOpn || t == terrainTrapdoorShut:
				ex := lb.exits[c]
				if ex == nil {
					lb.report(c, "trapdoor without an exit; drawn as fixed art")
					if t == terrainTrapdoorOpn {
						err = lb.im.place(&lb.cl.Overlays, c, plates, t-terrainPlate, plateAt, "trapdoor")
					}
					break
				}
				to, _ := exitDestination(ex)
				f = &content.Feature{Kind: content.FeatureTrapdoor, Open: t == terrainTrapdoorOpn, To: to}
				f.Look, err = lb.im.look(fmt.Sprintf("look.trapdoor.%d", plates), content.LayerFloor, map[string][]lookArt{
					"open": {{plates, terrainTrapdoorOpn - terrainPlate, plateAt}}, "closed": nil,
				})
			case t == terrainGemSocket:
				f, err = lb.socket(c)
			}
			if err != nil {
				return err
			}
			if f != nil {
				f.Cell = c
				lb.add(f)
			}
		}
	}
	return nil
}

// triggerTerrain reports whether a trigger record on terrain t makes a slot, lever or plate.
func triggerTerrain(t int) bool {
	return t >= terrainSlot && t <= terrainLeverOn || t == terrainPlate || t == terrainPlateDown
}

func (lb *levelBuilder) teleporter(c content.Cell, plates int) (*content.Feature, error) {
	tp := lb.teleporters[c]
	if tp == nil || tp.Extra[0] == teleporterExit {
		lb.report(c, "teleporter terrain without a teleporter; drawn as floor")
		return nil, nil
	}
	kind := tp.Extra[0]
	jump := func(du, dv byte) content.Cell {
		return content.Cell{c[0] + int(int8(du)), c[1] + int(int8(dv))}
	}
	f := &content.Feature{
		Kind:         content.FeatureTeleporter,
		Armed:        kind&teleporterArmed != 0,
		Destinations: []content.Cell{jump(tp.Extra[1], tp.Extra[3]), jump(tp.Extra[2], tp.Extra[4])},
		Turn:         int(kind>>4) & 7,
		FlyOver:      kind&teleporterFly != 0,
	}
	if kind&teleporterNext != 0 {
		f.Next = 1
	}
	for _, d := range f.Destinations {
		if !lb.cl.InBounds(d) {
			lb.report(c, "teleporter to %v outside the level; left out", d)
			return nil, nil
		}
	}
	var err error
	f.Look, err = lb.im.look(fmt.Sprintf("look.teleporter.%d", plates), content.LayerFloor, map[string][]lookArt{
		"armed": {{plates, terrainTeleporter - terrainPlate, plateAt}}, "disarmed": nil,
	})
	return f, err
}

// socket makes the gem socket, whose completion opens the wall before it (rules.md, Gem socket).
func (lb *levelBuilder) socket(c content.Cell) (*content.Feature, error) {
	f := &content.Feature{Kind: content.FeatureSocket, Effect: effectID(viewSocket)}
	for g := range gemCount {
		id, ok := lb.im.item(templateFirstGem+g, lb.what(c, "socket"))
		if !ok {
			return nil, nil
		}
		f.Places = append(f.Places, id)
	}
	x, y := c[0], c[1]
	var open []content.Action
	for _, d := range []int{0, 1} {
		open = append(open, content.Action{SetFloor: &content.SetFloor{Cell: content.Cell{x - 1, y + d}, Type: 0}})
	}
	for _, d := range []int{-1, 2} {
		open = append(open, content.Action{SetWall: &content.SetWall{Cell: content.Cell{x - 1, y + d}, Shape: 3}})
	}
	for _, a := range open {
		cell := content.Cell{}
		if a.SetFloor != nil {
			cell = a.SetFloor.Cell
		} else {
			cell = a.SetWall.Cell
		}
		if !lb.cl.InBounds(cell) {
			lb.report(c, "socket opens %v, outside the level; no opening", cell)
			open = nil
			break
		}
	}
	if open != nil {
		name := fmt.Sprintf("socket.%d.%d.open", x, y)
		lb.cl.Procedures[name] = open
		f.OnComplete = name
	}

	sockets, gems := lb.im.terrain(gemSocketBank), lb.im.terrain(gemBank)
	states := map[string][]lookArt{"empty": {{sockets, 0, socketAt}}, "full": {{sockets, 1, socketAt}}}
	for g := range gemCount {
		at := content.Offset{socketAt[0] + gemOffsets[g][0], socketAt[1] + gemOffsets[g][1]}
		states[content.SocketSetState(g)] = []lookArt{{gems, g, at}}
	}
	var err error
	f.Look, err = lb.im.look(fmt.Sprintf("look.socket.%d", sockets), content.LayerFloor, states)
	return f, err
}

// placedFeatures makes the features placements create by themselves: doors, exits and mouths.
func (lb *levelBuilder) placedFeatures() error {
	for _, c := range sortedCells(lb.doors) {
		if err := lb.door(c, lb.doors[c].Extra[0]); err != nil {
			return err
		}
	}
	exits := map[content.Cell]*Placement{}
	for c, p := range lb.exits {
		exits[c] = p
	}
	for c, p := range lb.teleporters {
		if p.Extra[0] == teleporterExit {
			exits[c] = p
		}
	}
	for _, c := range sortedCells(exits) {
		if err := lb.exit(c, exits[c]); err != nil {
			return err
		}
	}
	for _, c := range sortedCells(lb.mouths) {
		n := int(lb.mouths[c].Extra[0])
		msg, ok := lb.message(c, n)
		if !ok {
			continue
		}
		f := &content.Feature{Kind: content.FeatureMouth, Cell: c, Message: msg}
		var err error
		if f.Look, err = lb.im.look("look.mouth", content.LayerStand, map[string][]lookArt{
			content.FeatureMouth: {{mouthResource, 0, standingAt}},
		}); err != nil {
			return err
		}
		lb.add(f)
	}
	return nil
}

// message returns the text key of level message n.
func (lb *levelBuilder) message(c content.Cell, n int) (string, bool) {
	if n >= len(lb.l.Messages) {
		lb.report(c, "level message %d does not exist", n)
		return "", false
	}
	return levelMsgID(lb.index, n), true
}

func (lb *levelBuilder) door(c content.Cell, state byte) error {
	t := lb.l.TerrainAt(c[0], c[1])
	if t != terrainDoorH && t != terrainDoorV && t != terrainSealedDoor && t != terrainFloor {
		lb.report(c, "door on terrain %02Xh; left out", t)
		return nil
	}
	stage := int(state & doorStageMask)
	f := &content.Feature{Kind: content.FeatureDoor, Cell: c, State: content.DoorClosed, OpenedBySpell: state&doorKano != 0}
	switch {
	case stage == doorOpen:
		f.State = content.DoorOpen
	case stage >= doorFirstSealed:
		id, ok := lb.im.item(sealingSpheres[stage-doorFirstSealed], lb.what(c, "sealed door"))
		if !ok {
			return nil
		}
		f.SealedBy = id
		f.Sparkle = sparkleID(fmt.Sprintf("seal.%d", stage))
	case stage != 0:
		lb.report(c, "door starting at stage %d; placed closed", stage)
	}
	if state&doorMoving != 0 {
		lb.report(c, "door starting in motion; placed still")
	}

	res, at, orient := lb.im.terrain(int(lb.l.Header.DoorBank)), doorAtH, "h"
	if state&doorVertical != 0 {
		res, at, orient = res+1, doorAtV, "v"
	}
	states := map[string][]lookArt{content.DoorClosed: {{res, 0, at[0]}, {res, 1, at[1]}}, content.DoorOpen: nil}
	for s := 1; s < doorOpen; s++ {
		states[strconv.Itoa(s)] = []lookArt{{res, 2 * s, at[0]}, {res, 2*s + 1, at[1]}}
	}
	id := fmt.Sprintf("look.door.%d.%s", res, orient)
	if f.SealedBy != "" {
		sealed := lb.im.terrain(sealedDoorBank)
		first := 4 * (stage - doorFirstSealed)
		states["sealed"] = []lookArt{{sealed, first, doorAtH[0]}, {sealed, first + 1, doorAtH[1]}}
		id = fmt.Sprintf("%s.sealed%d", id, stage)
	}
	var err error
	if f.Look, err = lb.im.look(id, content.LayerStand, states); err != nil {
		return err
	}
	lb.add(f)
	return nil
}

func (lb *levelBuilder) exit(c content.Cell, p *Placement) error {
	t := int(lb.l.TerrainAt(c[0], c[1]))
	to, ok := exitDestination(p)
	switch {
	case !ok:
		return nil
	case t == terrainPit || t == terrainTrapdoorOpn || t == terrainTrapdoorShut:
		return nil // the pit or trapdoor carries it
	case !lb.im.levelExists(to.Level):
		lb.report(c, "exit to %s, which does not exist; left out", to.Level)
		return nil
	}
	f := &content.Feature{Kind: content.FeatureExit, Cell: c, To: to}
	var err error
	switch {
	case t <= terrainLastWall:
		if t != 1 && t != 3 {
			lb.report(c, "exit in a wall of shape %d; drawn as a wall", t)
			return nil
		}
		f.Look, err = lb.im.exitLook(lb.im.terrain(int(lb.l.Header.ExitWallBank)), t)
	case p.Type == PlaceExit && p.Extra[0] == exitMarker:
		f.Effect = effectID(viewClimbThrough)
		f.Look, err = lb.im.look("look.exit_marker", content.LayerStand, map[string][]lookArt{
			content.FeatureExit: {{exitMarkerResource, 0, standingAt}},
		})
	}
	if err != nil {
		return err
	}
	lb.add(f)
	return nil
}

// procedures turns the level's procedures into actions, keeping the empty ones something refers to.
func (lb *levelBuilder) procedures() {
	for n, proc := range lb.l.Procedures {
		if len(proc) == 0 {
			continue
		}
		var actions []content.Action
		for _, in := range proc {
			if a, ok := lb.action(n, in); ok {
				actions = append(actions, a)
			}
		}
		lb.cl.Procedures[procName(n)] = actions
	}
	for n := range lb.usedProcs {
		if _, ok := lb.cl.Procedures[procName(n)]; !ok {
			lb.cl.Procedures[procName(n)] = []content.Action{}
		}
	}
}

// action converts one instruction of procedure n (rules.md, Import).
func (lb *levelBuilder) action(n int, in Instruction) (content.Action, bool) {
	c := content.Cell{int(in.Args[0]), int(in.Args[1])}
	arg, arg2 := in.Args[2], in.Args[3]
	drop := func(format string, args ...any) (content.Action, bool) {
		lb.report(c, "procedure %d op %d: %s; left out", n, in.Op, fmt.Sprintf(format, args...))
		return content.Action{}, false
	}
	if !lb.cl.InBounds(c) {
		return drop("cell outside the level")
	}
	f := lb.features[c]
	feature := func(kinds ...string) (string, bool) {
		if f == nil {
			return "", false
		}
		if len(kinds) == 0 {
			return f.ID, true
		}
		for _, k := range kinds {
			if f.Kind == k {
				return f.ID, true
			}
		}
		return "", false
	}
	switch in.Op {
	case 0, 1, 2:
		id, ok := feature(content.FeatureDoor, content.FeatureTrapdoor, content.FeatureTeleporter,
			content.FeatureLever, content.FeaturePlate, content.FeatureSlot)
		if !ok {
			return drop("nothing to switch")
		}
		return [3]content.Action{{On: &id}, {Off: &id}, {Toggle: &id}}[in.Op], true
	case 3:
		id, ok := feature(content.FeatureLever, content.FeaturePlate, content.FeatureSlot)
		if !ok {
			return drop("no trigger")
		}
		hooks := map[string][2]string{
			content.FeatureLever: {content.HookTurnOn, content.HookTurnOff},
			content.FeaturePlate: {content.HookPress, content.HookRelease},
			content.FeatureSlot:  {content.HookInsert, ""},
		}[f.Kind]
		if arg > 1 || hooks[arg] == "" {
			return drop("a %s has no procedure %d", f.Kind, arg)
		}
		var proc *string
		if name := lb.hook(c, arg2); name != "" {
			proc = &name
		}
		return content.Action{SetHook: &content.SetHook{Feature: id, Hook: hooks[arg], Procedure: proc}}, true
	case 4:
		id, ok := feature(content.FeatureMouth)
		if !ok {
			return drop("no mouth")
		}
		msg, ok := lb.message(c, int(arg))
		if !ok {
			return content.Action{}, false
		}
		return content.Action{SetMessage: &content.SetMessage{Feature: id, Message: msg}}, true
	case 5, 6, 7:
		b, ok := lb.boulders[c]
		if !ok {
			return drop("no boulder placed there")
		}
		if in.Op == 7 {
			return content.Action{Stop: &b}, true
		}
		if arg > 7 || arg%2 != 0 {
			return drop("boulder direction %d", arg)
		}
		return content.Action{Roll: &content.Roll{Thing: b, Direction: int(arg), Shuttle: in.Op == 5}}, true
	case 8:
		item, ok := lb.im.item(int(arg), lb.what(c, "procedure %d", n))
		if !ok || arg2 > 7 {
			return drop("item %d, direction %d", arg, arg2)
		}
		return content.Action{ShootItem: &content.ShootItem{Cell: c, Item: item, Direction: int(arg2)}}, true
	case 9:
		spell, ok := spellID(int(arg))
		if arg2 > 7 || !ok {
			return drop("spell %d, direction %d", arg, arg2)
		}
		return content.Action{ShootSpell: &content.ShootSpell{Cell: c, Spell: spell, Direction: int(arg2),
			Power: content.Dice{Count: 6, Sides: 10}}}, true
	case 10:
		if id, ok := feature(content.FeaturePlate); ok {
			return content.Action{SetThreshold: &content.SetThreshold{Feature: id, Value: int(arg)}}, true
		}
		if id, ok := feature(content.FeatureSlot); ok {
			item, ok := lb.im.item(int(arg), lb.what(c, "procedure %d", n))
			if !ok {
				return content.Action{}, false
			}
			return content.Action{SetAccepts: &content.SetAccepts{Feature: id, Item: item}}, true
		}
		return drop("no plate or slot")
	case 11:
		item, ok := lb.im.item(int(arg), lb.what(c, "procedure %d", n))
		if !ok {
			return content.Action{}, false
		}
		return content.Action{Create: &content.Create{Cell: c, Item: item}}, true
	}
	return drop("unknown op")
}

// finish writes the features into the level in cell order.
func (lb *levelBuilder) finish() {
	for _, c := range sortedCells(lb.features) {
		lb.cl.Features = append(lb.cl.Features, lb.features[c])
	}
	if len(lb.cl.Procedures) == 0 {
		lb.cl.Procedures = nil
	}
}

func sortedCells[T any](m map[content.Cell]T) []content.Cell {
	cells := make([]content.Cell, 0, len(m))
	for c := range m {
		cells = append(cells, c)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i][0] != cells[j][0] {
			return cells[i][0] < cells[j][0]
		}
		return cells[i][1] < cells[j][1]
	})
	return cells
}

// lookArt is a frame of a resource, offset from the cell's tile box.
type lookArt struct {
	res, frame int
	at         content.Offset
}

// look adds a look with the given states the first time its id is asked for. A frame the
// resource does not have is reported and left out.
func (im *importer) look(id, layer string, states map[string][]lookArt) (string, error) {
	if _, ok := im.pack.Looks[id]; ok {
		return id, nil
	}
	l := &content.Look{ID: id, Layer: layer, States: map[string][]content.LookPart{}}
	for state, arts := range states {
		parts := []content.LookPart{}
		for _, a := range arts {
			sid, b, err := im.rawSprite(a.res)
			if err != nil {
				return "", fmt.Errorf("%s: %w", id, err)
			}
			if a.frame >= len(b.Frames) {
				im.report.addf("%s %s: frame %d not in resource %d (%d frames); not drawn", id, state, a.frame, a.res, len(b.Frames))
				continue
			}
			parts = append(parts, content.LookPart{Sprite: sid, Frame: a.frame, Offset: a.at})
		}
		l.States[state] = parts
	}
	im.pack.Looks[id] = l
	return id, nil
}
