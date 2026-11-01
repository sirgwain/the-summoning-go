package summoning

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// saveBuilder builds a save of the original over importFixture's level 0 (saves.md).
type saveBuilder struct {
	world, player []byte
	objects       []byte
	next          int
	// stale is a level state written into the level list, as the game writes the current level
	// as it was last left.
	stale []byte
}

func newSaveBuilder() *saveBuilder {
	b := &saveBuilder{world: make([]byte, saveWorldSize), player: make([]byte, savePlayerSize),
		objects: make([]byte, saveObjectCount*saveObjectSize)}
	for i := range saveObjectCount {
		b.objects[i*saveObjectSize] = saveFreeTemplate
	}
	w := b.world
	binary.LittleEndian.PutUint16(w[swHeld:], saveNoObject)
	for i := range boundCount {
		binary.LittleEndian.PutUint16(w[swBound+5*i:], saveNoObject)
	}
	// Level 0 as LEVELS has it: 3×4 cells, the terrain of importFixture.
	w[slLevel], w[slRows], w[slRows+1] = 0, 3, 4
	for i := range saveGridSide * saveGridSide {
		w[slTerrain+i] = 0xFF
	}
	for x, row := range [][]byte{{0x01, 0x01, 0x03, 0x01}, {0x1B, 0x1C, 0x1F, 0x24}, {0x17, 0x10, 0x24, 0xFF}} {
		copy(w[slTerrain+x*saveGridSide:], row)
	}
	w[slEntityCount] = 1
	for i := range slotCount {
		binary.LittleEndian.PutUint16(b.player[spSlots+2*i:], saveNoObject)
	}
	return b
}

// object adds an object of template t with value v at the head of list *head and returns its
// number.
func (b *saveBuilder) object(t uint8, v uint16, head *uint16) uint16 {
	n := uint16(b.next)
	b.next++
	o := b.objects[int(n)*saveObjectSize:]
	o[0] = t
	binary.LittleEndian.PutUint16(o[1:], v)
	next := uint16(saveNoObject)
	if head != nil {
		next, *head = *head, n
	}
	binary.LittleEndian.PutUint16(o[3:], next)
	return n
}

func (b *saveBuilder) terrain(x, y int, t byte)  { b.world[slTerrain+x*saveGridSide+y] = t }
func (b *saveBuilder) view(x, y int, flags byte) { b.world[slView+x*saveGridSide+y] = flags }

// cell sets cell feature i.
func (b *saveBuilder) cell(i, x, y int, bits byte, objects uint16, data byte) {
	c := b.world[slCells+i*saveCellSize:]
	c[0], c[1], c[2], c[5] = byte(x), byte(y), bits|cellUsed, data
	binary.LittleEndian.PutUint16(c[3:], objects)
}

// entity adds an entity to the level.
func (b *saveBuilder) entity(kind byte, u, v int16, dir, state byte, param, data, objects uint16, flags byte) {
	n := int(b.world[slEntityCount])
	e := b.world[slEntities+(n-1)*saveEntitySize:]
	binary.LittleEndian.PutUint16(e[0:], uint16(u))
	binary.LittleEndian.PutUint16(e[2:], uint16(v))
	binary.LittleEndian.PutUint16(e[8:], objects)
	e[0x0A], e[0x12], e[0x17], e[0x18] = kind, flags, dir, state
	binary.LittleEndian.PutUint16(e[0x1C:], param)
	binary.LittleEndian.PutUint16(e[0x1E:], data)
	b.world[slEntityCount]++
}

func (b *saveBuilder) bytes() []byte {
	f := fixture{}.str("header").zeros(saveHeaderSize - len("header"))
	f = f.str("Before the door").zeros(saveNameSize - len("Before the door"))
	f = f.u16(2)
	if b.stale != nil {
		f = f.u16(0).u32(saveLevelSize)
		f = append(f, b.stale...)
	}
	f = f.u16(0xFFFF)
	f = append(f, b.world...)
	f = append(f, b.player...)
	return append(f, b.objects...)
}

func TestReadSave(t *testing.T) {
	b := newSaveBuilder()
	// The level list holds level 0 as it was last left, with an entity past the count.
	b.stale = slices.Clone(b.world[saveLevelStateAt:])
	b.entity(entityCreature, 0, 0, 0, 0, 0, 5, saveNoObject, 0)
	b.world[slEntityCount] = 1
	sf, err := ReadSave(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if sf.Name != "Before the door" || sf.Current.Number != 0 || sf.Current.Rows != 3 || sf.Current.Columns != 4 {
		t.Errorf("save %q, level %d, %d×%d", sf.Name, sf.Current.Number, sf.Current.Rows, sf.Current.Columns)
	}
	if len(sf.Levels) != 0 {
		t.Errorf("the current level's stale copy was kept: %d levels", len(sf.Levels))
	}
	if len(sf.Current.Entities) != 0 {
		t.Errorf("entities past the count were read: %+v", sf.Current.Entities)
	}
	if _, err := ReadSave(b.bytes()[:len(b.bytes())-1]); err == nil {
		t.Error("a short save was read")
	}
}

func TestImportSave(t *testing.T) {
	src := importFixture()
	p, _, err := Import(src, Options{NoAudio: true})
	if err != nil {
		t.Fatal(err)
	}
	si, err := NewSaveImporter(src)
	if err != nil {
		t.Fatal(err)
	}

	b := newSaveBuilder()
	w, pl := b.world, b.player
	// The wall at (0, 1) has been opened; the plate is down, holding 60 with its threshold
	// lowered to 40; the door is open; the armed teleporter is disarmed and makes jump 1 next.
	b.terrain(0, 1, 0x1C)
	b.terrain(1, 2, terrainPlateDown)
	b.terrain(2, 1, terrainFloor)
	w[slTriggers] = 2
	copy(w[slTriggers+1:], []byte{0, 0, 3, 0, noProcedure})   // the slot
	copy(w[slTriggers+6:], []byte{60, 0, 40, 1, noProcedure}) // the plate
	b.cell(0, 2, 0, cellTrigger|cellArmed, saveNoObject, 0)
	b.cell(1, 1, 2, cellTrigger|cellArmed, saveNoObject, 1)
	w[slDoors] = 1
	w[slDoors+1+4] = doorOpen | doorVertical
	b.cell(2, 2, 1, cellDoor, saveNoObject, 0)
	w[slTeleporters] = 1
	w[slTeleporters+1] = teleporterArmed | teleporterNext
	b.cell(3, 2, 2, cellTeleporter, saveNoObject, 0)

	// On (1, 1): a Sack holding the Key, under a Dagger.
	var floor, sack uint16 = saveNoObject, saveNoObject
	b.object(3, 0, &sack)
	b.object(2, sack, &floor)
	b.object(1, 1500, &floor)
	b.cell(4, 1, 1, 0, floor, 0)

	// The creature has walked toward (1, 3), hurt and carrying a Dagger; the boulder is gone; the
	// trap stands where it was placed.
	var carried uint16 = saveNoObject
	b.object(1, 900, &carried)
	b.entity(entityCreature, 40, 70, 1, 0, 0, 7, carried, entityHostile)
	b.entity(entityTrap, 40, 16, 0, 0, 0, 0, saveNoObject, 0)
	b.view(1, 1, viewSeen|viewVisible)
	b.view(1, 2, viewSeen)

	// The player stands at (0, 1)'s ground point, facing 3.
	binary.LittleEndian.PutUint16(w[swPlayer:], 6)
	binary.LittleEndian.PutUint16(w[swPlayer+2:], 4)
	w[swPlayer+0x0A], w[swPlayer+0x17] = entityPlayer, 3
	w[swVars+3], w[swVars+99] = 7, 1
	w[swTimed+0], w[swTimed+6] = 3, 5 // a strength boost; sight
	// NPC record 0: talked to, with portrait 1.
	copy(w[swNPCs:], src.NPC[2:2+npcRecordSize])
	w[swNPCs+1] = 1
	w[swNPCs+14] |= behaviorTalked
	// A Key bound for (1, 0).
	key := b.object(3, 0, nil)
	binary.LittleEndian.PutUint16(w[swBound:], key)
	w[swBound+3], w[swBound+4] = 1, 0

	copy(pl[spName:], "Tester")
	pl[spWeapons], pl[spWeaponXP], pl[spWeaponNext] = 2, 4, 20
	pl[spLevel] = 1
	binary.LittleEndian.PutUint32(pl[spXP:], 700)
	binary.LittleEndian.PutUint32(pl[spNextXP:], 1000)
	for i, v := range []uint16{30, 40, 5, 25, 2} {
		binary.LittleEndian.PutUint16(pl[spPoints+2*i:], v)
	}
	copy(pl[spAttributes:], []byte{18, 17, 16, 15, 14, 13})
	copy(pl[spMaxAttrs:], []byte{18, 17, 17, 15, 14, 13})
	binary.LittleEndian.PutUint16(pl[spPortrait:], 1)
	binary.LittleEndian.PutUint16(pl[spStatus:], statusPoisoned|statusLevitating|statusFemale)
	binary.LittleEndian.PutUint16(pl[spBonus:], 0xFFFE)
	pl[spIcons], pl[spIcons+3] = 1, 1
	pl[spSpells+5*spellSize] = 2 // spell 5, which the fixture's pack does not have
	binary.LittleEndian.PutUint16(pl[spSlots:], b.object(3, 0, nil))
	binary.LittleEndian.PutUint16(pl[spSlots+2*sim.SlotRightHand:], b.object(1, 1200, nil))
	// The pack is open: the Bag of Lightness in the left hand holds the Dagger in slot 28.
	binary.LittleEndian.PutUint16(pl[spSlots+2*sim.SlotLeftHand:], b.object(5, saveNoObject, nil))
	binary.LittleEndian.PutUint16(pl[spSlots+2*sim.SlotPack:], b.object(1, 800, nil))
	pl[spPackOpen] = 1
	binary.LittleEndian.PutUint16(w[swHeld:], b.object(4, 1300, nil))

	sf, err := ReadSave(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	st, report, err := si.Import(p, sf, 1)
	if err != nil {
		t.Fatal(err)
	}
	if lines := report.Lines(); len(lines) != 1 || !strings.HasPrefix(lines[0], "spell 5 memorized") {
		t.Errorf("report %q", lines)
	}

	if st.Current != "level.00" || st.Counters["var.3"] != 7 || st.Counters["var.99"] != 1 || len(st.Counters) != 2 {
		t.Errorf("current %s, counters %v", st.Current, st.Counters)
	}
	if r := st.Records["thing.npc.0"]; r == nil || !r.Talked || r.Dead || r.Portrait != "portrait.1" {
		t.Errorf("record %+v", r)
	}

	ps := &st.Player.Player
	if ps.U != 16 || ps.V != 32 || ps.Facing != 3 || ps.Name != "Tester" {
		t.Errorf("player at (%d, %d) facing %d, %q", ps.U, ps.V, ps.Facing, ps.Name)
	}
	if ps.HitPoints != 30 || ps.MaxHitPoints != 40 || ps.MagicPoints != 5 || ps.MaxMagicPoints != 25 || ps.ArmorClass != 2 {
		t.Errorf("points %d/%d %d/%d armor %d", ps.HitPoints, ps.MaxHitPoints, ps.MagicPoints, ps.MaxMagicPoints, ps.ArmorClass)
	}
	if ps.Attributes[sim.AttrEndurance] != 16 || ps.MaxAttributes[sim.AttrEndurance] != 17 || ps.Attributes[sim.AttrPower] != 13 {
		t.Errorf("attributes %v / %v", ps.Attributes, ps.MaxAttributes)
	}
	if ps.Level != 1 || ps.Experience != 700 || ps.NextLevel != 1000 || ps.Skills[0] != (sim.Skill{Level: 2, Practice: 4, Threshold: 20}) {
		t.Errorf("level %d, %d/%d, skill %+v", ps.Level, ps.Experience, ps.NextLevel, ps.Skills[0])
	}
	if !ps.Poisoned || !ps.Status.Levitating || !ps.Status.Female || ps.Status.Invisible || ps.IntervalBonus != -2 || ps.Portrait != "portrait.1" {
		t.Errorf("poisoned %v, status %+v, bonus %d, portrait %s", ps.Poisoned, ps.Status, ps.IntervalBonus, ps.Portrait)
	}
	if ps.Timed[sim.TimedSight] != 5 || ps.Boosts[0] != (sim.Boost{Amount: 2, Ticks: 3}) || ps.Boosts[1] != (sim.Boost{}) {
		t.Errorf("timed %v, boosts %v", ps.Timed, ps.Boosts)
	}
	if ps.Magic.Icons != "AD" || len(ps.Magic.Spellbook) != 0 {
		t.Errorf("magic %+v", ps.Magic)
	}
	item := func(o *sim.Object) string {
		if o == nil {
			return ""
		}
		return o.Item
	}
	if item(ps.Hand) != "item.dagger.2" || ps.Hand.Value != 1300 || item(ps.Slots[0]) != "item.key" || ps.Slots[sim.SlotRightHand].Value != 1200 {
		t.Errorf("hand %+v, slot 0 %+v, right hand %+v", ps.Hand, ps.Slots[0], ps.Slots[sim.SlotRightHand])
	}
	if bag := ps.Slots[sim.SlotLeftHand]; item(bag) != "item.bag_of_lightness" || len(bag.Contents) != 1 || bag.Contents[0].Value != 800 {
		t.Errorf("bag %+v", bag)
	}

	ls := st.Levels[slices.IndexFunc(st.Levels, func(l *sim.LevelState) bool { return l.ID == "level.00" })]
	if ls.Grids[1][0][1] != content.NoFrame || ls.Grids[0][0][1] != 1 {
		t.Errorf("the opened wall: floor %d, wall %d", ls.Grids[0][0][1], ls.Grids[1][0][1])
	}
	feature := func(id string) *sim.FeatureState {
		i := slices.IndexFunc(ls.Features, func(f sim.FeatureState) bool { return f.ID == id })
		if i < 0 {
			t.Fatalf("no feature %s", id)
		}
		return &ls.Features[i]
	}
	if f := feature("plate.1.2"); !f.Down || !f.Armed || f.Counter != 60 || f.Threshold != 40 || f.OnPress != "p1" {
		t.Errorf("plate %+v", f)
	}
	if f := feature("door.2.1"); f.State != content.DoorOpen || f.Stage != 4 {
		t.Errorf("door %+v", f)
	}
	if f := feature("teleporter.2.2"); f.Armed || f.Feature.Feature.Next != 1 {
		t.Errorf("teleporter %+v", f)
	}
	if f := feature("slot.2.0"); !f.Armed || f.Accepts != "item.key" || f.OnInsert != "p0" {
		t.Errorf("slot %+v", f)
	}

	if len(ls.Things) != 2 {
		t.Fatalf("things %+v", ls.Things)
	}
	if c := ls.Things[0]; c.ID != "creature.1.1" || c.U != 50 || c.V != 98 || c.Facing != 2 || c.HitPoints != 7 || !c.Hostile ||
		len(c.Items) != 1 || c.Items[0].Value != 900 {
		t.Errorf("creature %+v", c)
	}
	if tr := ls.Things[1]; tr.ID != "trap.1.1" || tr.U != 48 || tr.V != 48 {
		t.Errorf("trap %+v", tr)
	}

	objs := map[content.Cell][]string{}
	for _, co := range ls.Objects {
		for _, o := range co.Objects {
			objs[co.Cell] = append(objs[co.Cell], o.Item)
			for _, in := range o.Contents {
				objs[co.Cell] = append(objs[co.Cell], ">"+in.Item)
			}
		}
	}
	if got := objs[content.Cell{1, 1}]; !slices.Equal(got, []string{"item.sack", ">item.key", "item.dagger"}) {
		t.Errorf("objects on (1, 1): %v", got)
	}
	if got := objs[content.Cell{1, 0}]; !slices.Equal(got, []string{"item.key"}) {
		t.Errorf("objects on (1, 0): %v", got)
	}
	if len(objs) != 2 {
		t.Errorf("objects %v", objs)
	}
	i := func(x, y int) int { return x*4 + y }
	if !ls.Seen[i(1, 1)] || !ls.Seen[i(1, 2)] || ls.Seen[i(0, 0)] || !ls.Visible[i(1, 1)] || ls.Visible[i(1, 2)] {
		t.Errorf("seen %v, visible %v", ls.Seen, ls.Visible)
	}
}

// TestImportSharedCounts makes the save's shared counts of templates 72h–7Bh the world's shared
// charges of their items, and leaves out templates whose items have none.
func TestImportSharedCounts(t *testing.T) {
	ids := make([]string, firstSharedTemplate+2)
	ids[firstSharedTemplate], ids[firstSharedTemplate+1] = "item.shield", "item.blade"
	p := &content.Pack{Items: map[string]*content.Item{
		"item.shield": {ID: "item.shield", Charges: 100, SharedCharges: true},
		"item.blade":  {ID: "item.blade", Durability: 2500},
	}}
	im := &saveImport{si: &SaveImporter{itemIDs: ids}, p: p, sf: &SaveFile{}}
	im.sf.Shared[0], im.sf.Shared[1], im.sf.Shared[5] = 40, 3, 9
	st := &sim.State{}
	im.shared(st)
	if len(st.Shared) != 1 || st.Shared["item.shield"] != 40 {
		t.Errorf("shared %v", st.Shared)
	}
}
