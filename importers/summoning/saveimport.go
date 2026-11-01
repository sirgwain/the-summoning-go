package summoning

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// SaveImporter turns the game's save files into engine states (saves.md, Import). It holds what
// that needs from the game's data files.
type SaveImporter struct {
	levels  []*Level
	npcs    []NPCRecord
	itemIDs []string
}

// NewSaveImporter reads what importing saves needs from src.
func NewSaveImporter(src *Source) (*SaveImporter, error) {
	levels, err := ReadLevels(src.Levels)
	if err != nil {
		return nil, err
	}
	text, err := ReadText(src.Text)
	if err != nil {
		return nil, err
	}
	objects, err := ReadObjects(src.Objects)
	if err != nil {
		return nil, err
	}
	npcs, err := ReadNPCs(src.NPC)
	if err != nil {
		return nil, err
	}
	strs := make(map[string]string, len(text))
	for i, s := range text {
		strs[msgID(i)] = s
	}
	ids, err := itemIDs(objects, strs)
	if err != nil {
		return nil, err
	}
	return &SaveImporter{levels: levels, npcs: npcs, itemIDs: ids}, nil
}

// Offsets from an entity's position to its ground point (entities.md, Standing cell).
var (
	standingCreature = [2]int{10, 28}
	standingBlock    = [2]int{8, 32}
	standingMouth    = [2]int{16, 24}
)

// Player status bits (saves.md, The player block).
const (
	statusFemale          = 0x0002
	statusResistQuarter   = 0x0004
	statusResistForty     = 0x0008
	statusRegenerating    = 0x0010
	statusCharges         = 0x0020
	statusManaRegen       = 0x0040
	statusDeathProtection = 0x0080
	statusPoisoned        = 0x0100
	statusFireImmune      = 0x0200
	statusInvisible       = 0x0400
	statusDeathBlowWard   = 0x0800
	statusAccurate        = 0x1000
	statusTransformed     = 0x2000
	statusBlowProtection  = 0x4000
	statusLevitating      = 0x8000
)

// NPC behavior bits the game changes (creatures.md, NPC record).
const (
	behaviorTalked = 0x0001
	behaviorDead   = 0x0002
)

const (
	// boostAmount is what a timed attribute boost adds (player.md, Status tick).
	boostAmount = 2
	// firstSharedTemplate is the first of the templates with a shared count (items.md).
	firstSharedTemplate = 0x72
	firstGesture        = 'A'
)

// saveImport is one save being imported.
type saveImport struct {
	si     *SaveImporter
	p      *content.Pack
	sf     *SaveFile
	report *Report
	// muted are the thing templates whose creatures no longer talk.
	muted map[string]bool
	// newThings counts the creatures given new ids, per level.
	newThings map[string]int
}

// Import returns the engine state save sf describes, on pack p, with a report of what it left out.
// The seed makes the new world's random choices, which the save then overrides.
func (si *SaveImporter) Import(p *content.Pack, sf *SaveFile, seed uint64) (*sim.State, *Report, error) {
	im := &saveImport{si: si, p: p, sf: sf, report: &Report{}, muted: map[string]bool{}, newThings: map[string]int{}}
	w, err := sim.New(p, seed)
	if err != nil {
		return nil, nil, err
	}
	held := append([]*SaveLevel{sf.Current}, sf.Levels...)
	for _, l := range held {
		if err := im.checkLevel(l); err != nil {
			return nil, nil, err
		}
		if err := w.LoadLevel(levelID(l.Number)); err != nil {
			return nil, nil, err
		}
	}
	for _, b := range sf.Bound {
		if err := w.LoadLevel(levelID(int(b.Level))); err != nil {
			im.report.addf("an object bound for level %d, which the pack does not have; left out", b.Level)
		}
	}
	st, err := w.State()
	if err != nil {
		return nil, nil, err
	}
	st.Current = levelID(sf.Current.Number)
	im.counters(st)
	im.records(st)
	if err := im.player(st); err != nil {
		return nil, nil, err
	}
	for _, l := range held {
		im.level(st, l, l == sf.Current)
	}
	im.bound(st)
	if _, err := sim.Restore(p, st); err != nil {
		return nil, nil, fmt.Errorf("the imported state does not restore: %w", err)
	}
	return st, im.report, nil
}

// checkLevel makes sure the pack and LEVELS both have level l, at its size.
func (im *saveImport) checkLevel(l *SaveLevel) error {
	def := im.p.Levels[levelID(l.Number)]
	if def == nil || l.Number >= len(im.si.levels) {
		return fmt.Errorf("the save holds level %d, which the pack or LEVELS does not have", l.Number)
	}
	if orig := im.si.levels[l.Number]; orig.Rows != l.Rows || orig.Columns != l.Columns || def.Size != [2]int{l.Rows, l.Columns} {
		return fmt.Errorf("level %d: the save has it %d×%d, the data files %d×%d", l.Number, l.Rows, l.Columns, orig.Rows, orig.Columns)
	}
	return nil
}

func (im *saveImport) levelState(st *sim.State, n int) *sim.LevelState {
	for _, ls := range st.Levels {
		if ls.ID == levelID(n) {
			return ls
		}
	}
	return nil
}

// counters makes the interaction variables counters.
func (im *saveImport) counters(st *sim.State) {
	st.Counters = map[string]int{}
	for n, v := range im.sf.Vars {
		if v != 0 {
			st.Counters[varID(n)] = int(v)
		}
	}
}

// records makes the world's records of the NPC records the game has changed.
func (im *saveImport) records(st *sim.State) {
	st.Records = map[string]*sim.Record{}
	for n := range im.sf.NPCs {
		id := fmt.Sprintf("thing.npc.%d", n)
		if im.p.Things[id] == nil || n >= len(im.si.npcs) {
			continue
		}
		rec, orig := &im.sf.NPCs[n], &im.si.npcs[n]
		behavior, was := int(rec[14])|int(rec[15])<<8, int(orig[14])|int(orig[15])<<8
		r := &sim.Record{Talked: behavior&behaviorTalked != 0, Dead: behavior&behaviorDead != 0}
		if rec[1] != orig[1] {
			if pid := portraitID(int(rec[1])); im.p.Portraits[pid] != nil {
				r.Portrait = pid
			} else {
				im.report.addf("NPC record %d: portrait %d, which the pack does not have; left as it was", n, rec[1])
			}
		}
		if was&behaviorTalks != 0 && behavior&behaviorTalks == 0 {
			im.muted[id] = true
		}
		if *r != (sim.Record{}) {
			st.Records[id] = r
		}
	}
}

// object returns the object of number n, with what it holds; seen guards against loops.
func (im *saveImport) object(n uint16, seen map[uint16]bool) *sim.Object {
	if int(n) >= len(im.sf.Objects) || seen[n] {
		im.report.addf("object %d: not in the pool, or in two places; left out", n)
		return nil
	}
	seen[n] = true
	so := im.sf.Objects[n]
	if so.Template == saveFreeTemplate || int(so.Template) >= len(im.si.itemIDs) {
		im.report.addf("object %d: template %02Xh; left out", n, so.Template)
		return nil
	}
	id := im.si.itemIDs[so.Template]
	it := im.p.Items[id]
	if it == nil {
		im.report.addf("object %d: the pack has no %s; left out", n, id)
		return nil
	}
	o := &sim.Object{Item: id}
	switch {
	case it.Container != nil:
		o.Contents = im.list(so.Value, seen)
	case int(so.Template) >= firstSharedTemplate && int(so.Template) < firstSharedTemplate+shareCount:
		o.Value = int(im.sf.Shared[int(so.Template)-firstSharedTemplate])
	default:
		o.Value = int(so.Value)
	}
	return o
}

// list returns the objects of the list starting at first, oldest first.
func (im *saveImport) list(first uint16, seen map[uint16]bool) []*sim.Object {
	var objs []*sim.Object
	for n := first; n != saveNoObject; {
		if int(n) >= len(im.sf.Objects) || seen[n] {
			im.report.addf("object %d: not in the pool, or in two places; left out", n)
			break
		}
		next := im.sf.Objects[n].Next
		if o := im.object(n, seen); o != nil {
			objs = append(objs, o)
		}
		n = next
	}
	slices.Reverse(objs)
	return objs
}

// player lays the player block and entity over the new world's player.
func (im *saveImport) player(st *sim.State) error {
	ps := st.Player
	if ps == nil {
		return fmt.Errorf("the pack has no player")
	}
	sp, body := &im.sf.Player, &im.sf.Body
	pl := &ps.Player
	pl.U, pl.V = int(body.U)+standingCreature[0], int(body.V)+standingCreature[1]
	pl.Facing = int(body.Direction) & 7
	pl.Name = sp.Name
	pl.HitPoints, pl.MaxHitPoints = int(sp.HP), int(sp.MaxHP)
	pl.MagicPoints, pl.MaxMagicPoints = int(sp.MP), int(sp.MaxMP)
	pl.ArmorClass = int(sp.AC)
	pl.Attributes, pl.MaxAttributes = map[string]int{}, map[string]int{}
	for i, name := range []string{sim.AttrStrength, sim.AttrAgility, sim.AttrEndurance, sim.AttrAccuracy, sim.AttrTalent, sim.AttrPower} {
		pl.Attributes[name], pl.MaxAttributes[name] = int(sp.Attributes[i]), int(sp.MaxAttributes[i])
	}
	pl.IntervalBonus = int(sp.IntervalBonus)
	pl.Level, pl.Experience, pl.NextLevel = int(sp.Level), int(sp.XP), int(sp.NextXP)
	for c := range 4 {
		pl.Skills[sim.SkillWeapon+c] = sim.Skill{Level: int(sp.Weapons[c]), Practice: int(sp.WeaponXP[c]), Threshold: int(sp.WeaponNext[c])}
		pl.Skills[sim.SkillSchool+c] = sim.Skill{Level: int(sp.Schools[c]), Practice: int(sp.SchoolXP[c]), Threshold: int(sp.SchoolNext[c])}
	}
	pl.Fatigue, pl.Exertion, pl.Confusion = int(sp.Fatigue), 0, int(sp.Confusion)
	pl.WeaponCount = int(sp.Counted)
	if pid := portraitID(int(sp.Portrait)); im.p.Portraits[pid] != nil {
		pl.Portrait = pid
	} else {
		im.report.addf("the player's portrait %d, which the pack does not have; left as the new game's", sp.Portrait)
	}

	s := sp.Status
	pl.Poisoned = s&statusPoisoned != 0
	pl.Status = sim.Status{
		Regenerating: s&statusRegenerating != 0, ManaRegenerating: s&statusManaRegen != 0,
		ResistQuarter: s&statusResistQuarter != 0, ResistForty: s&statusResistForty != 0, Charges: s&statusCharges != 0,
		FireImmune: s&statusFireImmune != 0, DeathBlowWard: s&statusDeathBlowWard != 0, Accurate: s&statusAccurate != 0,
		BlowProtection: s&statusBlowProtection != 0, Levitating: s&statusLevitating != 0, Invisible: s&statusInvisible != 0,
		Transformed: s&statusTransformed != 0, DeathProtection: s&statusDeathProtection != 0, Female: s&statusFemale != 0,
	}
	for i := range pl.Timed {
		pl.Timed[i] = int(im.sf.Timed[len(pl.Boosts)+i])
	}
	for i := range pl.Boosts {
		pl.Boosts[i] = sim.Boost{}
		if t := int(im.sf.Timed[i]); t > 0 {
			pl.Boosts[i] = sim.Boost{Amount: boostAmount, Ticks: t}
		}
	}
	if sp.Look != 0 {
		im.report.addf("the player's Shape Change look; left out")
		pl.Status.Transformed = false
	}
	pl.Look = ""

	seen := map[uint16]bool{}
	pl.Hand = nil
	if im.sf.Held != saveNoObject {
		pl.Hand = im.object(im.sf.Held, seen)
	}
	for i := range pl.Slots {
		pl.Slots[i] = nil
		if n := sp.Slots[i]; n != saveNoObject {
			pl.Slots[i] = im.object(n, seen)
		}
	}
	if sp.PackOpen {
		var pack []*sim.Object
		for i := sim.SlotPack; i < slotCount; i++ {
			if n := sp.Slots[i]; n != saveNoObject {
				if o := im.object(n, seen); o != nil {
					pack = append(pack, o)
				}
			}
		}
		if bag := pl.Slots[sim.SlotLeftHand]; bag != nil && im.p.Items[bag.Item].Container != nil {
			bag.Contents = append(bag.Contents, pack...)
		} else if len(pack) > 0 {
			im.report.addf("pack slots with no bag in the left hand; left out")
		}
	}

	im.magic(pl)
	ps.Vision = content.Cell{int(sp.VisionX), int(sp.VisionY)}
	return nil
}

// magic makes the player's icons, spellbook, memorizations and selections.
func (im *saveImport) magic(pl *sim.Player) {
	sp := &im.sf.Player
	m := sim.Magic{Memorized: map[string]int{}}
	var icons strings.Builder
	for i, known := range sp.Icons {
		if known {
			icons.WriteByte(byte(firstGesture + i))
		}
	}
	m.Icons = icons.String()
	for n, count := range sp.Memorized {
		if count == 0 {
			continue
		}
		id, ok := spellID(n)
		if !ok || im.p.Spells[id] == nil {
			im.report.addf("spell %d memorized, which the pack does not have; left out", n)
			continue
		}
		m.Spellbook = append(m.Spellbook, id)
		m.Memorized[id] = int(count)
	}
	for school := range content.Schools {
		n := school*10 + int(sp.Selected[school])
		if id, ok := spellID(n); ok && m.Memorized[id] > 0 {
			m.Selected[school] = id
		}
	}
	pl.Magic = m
}

// level lays saved level l over its state in st.
func (im *saveImport) level(st *sim.State, l *SaveLevel, current bool) {
	ls := im.levelState(st, l.Number)
	def := im.p.Levels[ls.ID]
	orig := im.si.levels[l.Number]
	im.terrain(ls, def, orig, l)

	n := def.Size[0] * def.Size[1]
	ls.Seen, ls.Visible = make([]bool, n), make([]bool, n)
	for x := range def.Size[0] {
		for y := range def.Size[1] {
			i := x*def.Size[1] + y
			ls.Seen[i] = l.ViewAt(x, y)&viewSeen != 0
			ls.Visible[i] = current && l.ViewAt(x, y)&viewVisible != 0
		}
	}

	im.features(ls, def, orig, l)
	im.things(ls, def, l)

	ls.Objects = nil
	seen := map[uint16]bool{}
	for _, c := range l.Cells {
		if c.Objects == saveNoObject {
			continue
		}
		cell := content.Cell{int(c.X), int(c.Y)}
		if !def.InBounds(cell) {
			im.report.addf("level %d: objects on %v, outside the level; left out", l.Number, cell)
			continue
		}
		if objs := im.list(c.Objects, seen); len(objs) > 0 {
			ls.Objects = append(ls.Objects, sim.CellObjects{Cell: cell, Objects: objs})
		}
	}
	sortCellObjects(ls.Objects)
	ls.Flights = nil
}

// terrain applies the walls opened and closed since the level started.
func (im *saveImport) terrain(ls *sim.LevelState, def *content.Level, orig *Level, l *SaveLevel) {
	features := map[content.Cell]bool{}
	for _, f := range ls.Features {
		features[f.Cell] = true
	}
	plain := 0
	if fs := im.p.Floors[def.Floors]; fs != nil && len(def.FloorTypes) > fs.Variants {
		plain = fs.Variants
	}
	floor, wall := ls.Grids[0], ls.Grids[1]
	for x := range l.Rows {
		for y := range l.Columns {
			was, now := int(orig.TerrainAt(x, y)), int(l.TerrainAt(x, y))
			c := content.Cell{x, y}
			switch {
			case was == now || features[c]:
			case now <= terrainLastWall && (was <= terrainLastWall || isPlainFloor(was)):
				wall[x][y] = now
				if isPlainFloor(was) {
					floor[x][y] = plain
				}
			case isPlainFloor(now) && (was <= terrainLastWall || isPlainFloor(was)):
				wall[x][y] = content.NoFrame
				floor[x][y] = now - terrainFloor
			default:
				im.report.addf("level %d (%d, %d): terrain %02Xh became %02Xh; left as it was", l.Number, x, y, was, now)
			}
		}
	}
}

func isPlainFloor(t int) bool { return t >= terrainFloor && t < terrainPlate }

// features lays the saved state of doors, triggers, teleporters, trapdoors, the socket and
// mouths over the level's features.
func (im *saveImport) features(ls *sim.LevelState, def *content.Level, orig *Level, l *SaveLevel) {
	placed := map[content.Cell]*Placement{}
	for i := range orig.Placements {
		if p := &orig.Placements[i]; p.Type == PlaceTrigger {
			placed[content.Cell{int(p.X), int(p.Y)}] = p
		}
	}
	mouths := map[content.Cell]int{}
	for _, e := range l.Entities {
		if e.Kind == entityMouth {
			c := content.Cell{floorDiv(int(e.U)+standingMouth[0], cellUnits), floorDiv(int(e.V)+standingMouth[1], cellUnits)}
			mouths[c] = int(e.State)
		}
	}
	for i := range ls.Features {
		f := &ls.Features[i].Feature.Feature
		x, y := f.Cell[0], f.Cell[1]
		cell := l.Cell(x, y)
		t := l.TerrainAt(x, y)
		var trig *SaveTrigger
		if cell != nil && cell.Bits&cellTrigger != 0 && int(cell.Data) < len(l.Triggers) {
			trig = &l.Triggers[cell.Data]
		}
		switch f.Kind {
		case content.FeatureDoor:
			if cell == nil || cell.Bits&cellDoor == 0 || int(cell.Data) >= len(l.Doors) {
				im.report.addf("level %d (%d, %d): a door with no saved state; left as it starts", l.Number, x, y)
				continue
			}
			im.door(&ls.Features[i], l.Doors[cell.Data])
		case content.FeatureLever:
			f.On = t == terrainLeverOn
			im.trigger(f, cell, trig, placed[f.Cell], def, l.Number)
		case content.FeaturePlate:
			f.Down = t == terrainPlateDown
			im.trigger(f, cell, trig, placed[f.Cell], def, l.Number)
			if trig != nil {
				ls.Features[i].Counter = max(int(trig.Counter), 0)
			}
		case content.FeatureSlot:
			im.trigger(f, cell, trig, placed[f.Cell], def, l.Number)
		case content.FeatureTeleporter:
			f.Armed = cell != nil && cell.Bits&cellArmed != 0
			if cell != nil && cell.Bits&cellTeleporter != 0 && int(cell.Data) < len(l.Teleporters) {
				f.Next = 0
				if l.Teleporters[cell.Data][0]&teleporterNext != 0 {
					f.Next = 1
				}
			}
		case content.FeatureTrapdoor:
			f.Open = t == terrainTrapdoorOpn
		case content.FeatureSocket:
			f.Set = nil
			if cell != nil {
				for g := range len(f.Places) {
					if cell.Data&(1<<g) != 0 {
						f.Set = append(f.Set, g)
					}
				}
			}
		case content.FeatureMouth:
			if msg, ok := mouths[f.Cell]; ok {
				if msg < len(orig.Messages) {
					f.Message = levelMsgID(l.Number, msg)
				} else {
					im.report.addf("level %d (%d, %d): mouth message %d does not exist; left as it was", l.Number, x, y, msg)
				}
			}
		}
	}
}

// door sets a door from its saved state byte (rules.md, Doors).
func (im *saveImport) door(fs *sim.FeatureState, state uint8) {
	f := &fs.Feature
	stage := int(state & doorStageMask)
	if stage >= doorFirstSealed {
		f.State, fs.Stage = content.DoorClosed, 0
		return
	}
	f.SealedBy = ""
	fs.Stage = stage
	switch {
	case state&doorInMotion != 0 && state&doorTowardOpen != 0:
		f.State = content.DoorOpening
	case state&doorInMotion != 0:
		f.State = content.DoorClosing
	case stage == doorOpen:
		f.State = content.DoorOpen
	default:
		f.State, fs.Stage = content.DoorClosed, 0
	}
}

// trigger sets a lever's, plate's or slot's arming, and the threshold and procedures that differ
// from its placement.
func (im *saveImport) trigger(f *content.Feature, cell *SaveCell, trig *SaveTrigger, p *Placement, def *content.Level, level int) {
	if trig == nil {
		return
	}
	f.Armed = cell.Bits&cellArmed != 0
	if p == nil {
		im.report.addf("level %d (%d, %d): a trigger the level does not place; arming only", level, f.Cell[0], f.Cell[1])
		return
	}
	hook := func(n uint8) string {
		if name := procName(int(n)); n != noProcedure && def.Procedures[name] != nil {
			return name
		}
		return ""
	}
	if trig.Threshold != p.Extra[triggerThreshold] {
		switch f.Kind {
		case content.FeaturePlate:
			f.Threshold = int(trig.Threshold)
		case content.FeatureSlot:
			if int(trig.Threshold) < len(im.si.itemIDs) {
				f.Accepts = im.si.itemIDs[trig.Threshold]
			}
		}
	}
	if trig.ProcA != p.Extra[triggerProcA] {
		switch f.Kind {
		case content.FeatureLever:
			f.OnTurnOn = hook(trig.ProcA)
		case content.FeaturePlate:
			f.OnPress = hook(trig.ProcA)
		case content.FeatureSlot:
			f.OnInsert = hook(trig.ProcA)
		}
	}
	if trig.ProcB != p.Extra[triggerProcB] {
		switch f.Kind {
		case content.FeatureLever:
			f.OnTurnOff = hook(trig.ProcB)
		case content.FeaturePlate:
			f.OnRelease = hook(trig.ProcB)
		}
	}
}

// things matches the saved creatures, boulders and traps to the things the level places.
func (im *saveImport) things(ls *sim.LevelState, def *content.Level, l *SaveLevel) {
	placedAt := map[string]content.Cell{}
	for _, pt := range def.Things {
		placedAt[pt.ID] = pt.Cell
	}
	// The candidates are the things the level places, as the new world has them, or made anew
	// where the new world's player, arriving, crushed them.
	loaded := map[string]sim.ThingState{}
	for _, ts := range ls.Things {
		loaded[ts.ID] = ts
	}
	var fresh []sim.ThingState
	for _, pt := range def.Things {
		ts, ok := loaded[pt.ID]
		if !ok {
			tm := im.p.Things[pt.Thing]
			if tm == nil {
				continue
			}
			d := tm.HitPoints
			ts = sim.ThingState{Template: pt.Thing, WalkDir: 4}
			ts.ID, ts.Facing, ts.HitPoints = pt.ID, tm.Facing, d.Bonus+d.Count*(d.Sides+1)/2
		}
		fresh = append(fresh, ts)
	}
	used := make([]bool, len(fresh))
	// nearest returns the unused fresh thing of template tid nearest to ground point (u, v).
	nearest := func(tid string, u, v int) int {
		best, bestD := -1, 0
		for i, ts := range fresh {
			if used[i] || ts.Template != tid {
				continue
			}
			c := placedAt[ts.ID]
			du, dv := c[0]*cellUnits+cellUnits/2-u, c[1]*cellUnits+cellUnits/2-v
			if d := du*du + dv*dv; best < 0 || d < bestD {
				best, bestD = i, d
			}
		}
		return best
	}
	var things []sim.ThingState
	left := map[int]int{}
	for _, e := range l.Entities {
		switch e.Kind {
		case entityCreature:
			tid := fmt.Sprintf("thing.npc.%d", e.State)
			if im.p.Things[tid] == nil {
				im.report.addf("level %d: a creature of NPC record %d, which the pack does not have; left out", l.Number, e.State)
				continue
			}
			u, v := int(e.U)+standingCreature[0], int(e.V)+standingCreature[1]
			ts := sim.ThingState{Template: tid, WalkDir: 4}
			if i := nearest(tid, u, v); i >= 0 {
				used[i], ts = true, fresh[i]
			} else {
				im.newThings[ls.ID]++
				ts.ID = fmt.Sprintf("creature.saved.%d", im.newThings[ls.ID])
			}
			ts.U, ts.V = u, v
			ts.Facing = 2 * int(e.Direction) & 7
			ts.HitPoints = int(e.Data)
			ts.Hostile = e.Flags&entityHostile != 0
			ts.Mute = im.muted[tid]
			ts.Moving, ts.Pose = false, ""
			ts.Items = im.list(e.Objects, map[uint16]bool{})
			things = append(things, ts)
		case entityBoulder:
			at := content.Cell{int(e.Param >> 8), int(e.Param & 0xFF)}
			i := slices.IndexFunc(fresh, func(ts sim.ThingState) bool {
				return placedAt[ts.ID] == at && im.p.Things[ts.Template] != nil && im.p.Things[ts.Template].Kind == content.ThingBoulder
			})
			if i < 0 || used[i] {
				im.report.addf("level %d: a boulder placed on %v, which the level does not place; left out", l.Number, at)
				continue
			}
			used[i] = true
			ts := fresh[i]
			ts.U, ts.V = int(e.U)+standingBlock[0], int(e.V)+standingBlock[1]
			ts.Rolling = nil
			if (e.State == boulderRolling || e.State == boulderShuttling) && e.Direction < 4 {
				ts.Rolling = &content.Rolling{Direction: 2 * int(e.Direction), Shuttle: e.State == boulderShuttling}
			}
			things = append(things, ts)
		case entityTrap:
			u, v := int(e.U)+standingBlock[0], int(e.V)+standingBlock[1]
			i := -1
			for j, ts := range fresh {
				if !used[j] && im.p.Things[ts.Template] != nil && im.p.Things[ts.Template].Kind == content.ThingTrap {
					if i < 0 || dist(placedAt[ts.ID], u, v) < dist(placedAt[fresh[i].ID], u, v) {
						i = j
					}
				}
			}
			if i < 0 {
				im.report.addf("level %d: a trap the level does not place; left out", l.Number)
				continue
			}
			used[i] = true
			ts := fresh[i]
			ts.U, ts.V = u, v
			things = append(things, ts)
		case entityFlight, entityConjured, entityWallSpell:
			left[int(e.Kind)]++
		}
	}
	for kind, n := range left {
		im.report.addf("level %d: %d entities of kind %d (transient); left out", l.Number, n, kind)
	}
	ls.Things = things
}

// dist returns the squared distance from cell c's center to ground point (u, v).
func dist(c content.Cell, u, v int) int {
	du, dv := c[0]*cellUnits+cellUnits/2-u, c[1]*cellUnits+cellUnits/2-v
	return du*du + dv*dv
}

// bound puts the objects bound for other levels on their cells.
func (im *saveImport) bound(st *sim.State) {
	seen := map[uint16]bool{}
	for _, b := range im.sf.Bound {
		ls := im.levelState(st, int(b.Level))
		if ls == nil {
			continue
		}
		cell := content.Cell{int(b.X), int(b.Y)}
		if !im.p.Levels[ls.ID].InBounds(cell) {
			im.report.addf("an object bound for level %d %v, outside the level; left out", b.Level, cell)
			continue
		}
		o := im.object(b.Object, seen)
		if o == nil {
			continue
		}
		i := slices.IndexFunc(ls.Objects, func(co sim.CellObjects) bool { return co.Cell == cell })
		if i < 0 {
			ls.Objects = append(ls.Objects, sim.CellObjects{Cell: cell})
			i = len(ls.Objects) - 1
		}
		ls.Objects[i].Objects = append(ls.Objects[i].Objects, o)
	}
	for _, ls := range st.Levels {
		sortCellObjects(ls.Objects)
	}
}

func sortCellObjects(objs []sim.CellObjects) {
	slices.SortFunc(objs, func(a, b sim.CellObjects) int {
		if a.Cell[0] != b.Cell[0] {
			return a.Cell[0] - b.Cell[0]
		}
		return a.Cell[1] - b.Cell[1]
	})
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
