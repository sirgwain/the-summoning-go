package devpack

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestVerifies(t *testing.T) {
	if err := Build().Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteLoad(t *testing.T) {
	p := Build()
	dir := t.TempDir()
	if err := content.Write(p, dir); err != nil {
		t.Fatal(err)
	}
	q, err := content.Load(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Verify(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Levels, q.Levels) || !reflect.DeepEqual(p.Items, q.Items) || !reflect.DeepEqual(p.Looks, q.Looks) {
		t.Error("the pack changed through writing and loading")
	}
	for id, b := range p.Sprites {
		if !reflect.DeepEqual(b.Sheet.Pix, q.Sprites[id].Sheet.Pix) {
			t.Errorf("sprite bank %s changed through writing and loading", id)
		}
	}
}

// TestHallSolves works the hall's chain of puzzles with debug commands: the gems open the
// socket's wall, the sphere behind it unseals door E, and exit D leads to the cellar.
func TestHallSolves(t *testing.T) {
	w, err := sim.New(Build(), 1)
	if err != nil {
		t.Fatal(err)
	}
	do := func(cmds ...input.Command) { w.Update(cmds) }
	l := w.Current()
	if !l.Blocked(SocketWall) || !l.Blocked(SealedDoor) {
		t.Fatal("the socket's wall or door E starts open")
	}
	do(input.Drop{Cell: Socket, Item: ItemRedGem})
	do(input.Drop{Cell: Socket, Item: ItemBlueGem})
	if l.Blocked(SocketWall) {
		t.Fatal("the full socket did not open its wall")
	}
	if got := l.Feature("mouth").Message; got != "msg.mouth.after" {
		t.Errorf("mouth says %q", got)
	}
	do(input.Remove{Cell: Sphere})
	do(input.Drop{Cell: SealedDoor, Item: ItemSphere})
	for range 120 {
		do()
	}
	if l.Blocked(SealedDoor) {
		t.Fatal("the sphere did not open door E")
	}
	do(input.FollowExit{Feature: "exit.d"})
	if w.Current().ID != Cellar {
		t.Errorf("exit D led to %s", w.Current().ID)
	}
}

func TestHallDoors(t *testing.T) {
	w, err := sim.New(Build(), 1)
	if err != nil {
		t.Fatal(err)
	}
	l := w.Current()
	steps := func(cmds ...input.Command) {
		w.Update(cmds)
		for range 120 {
			w.Update(nil)
		}
	}
	steps(input.Use{Cell: LeverA})
	steps(input.Move{Thing: "boulder", Cell: PlateB})
	steps(input.Remove{Cell: content.Cell{8, 17}}, input.Drop{Cell: SlotC, Item: ItemKey})
	for _, d := range []content.Cell{DoorA, DoorB, DoorC} {
		if l.Blocked(d) {
			t.Errorf("door at %v closed", d)
		}
	}
	steps(input.Use{Cell: LeverTrapdoor})
	if !l.Feature("trapdoor").Open {
		t.Error("the trapdoor lever did not open it")
	}
	// A creature that falls through a pit is removed (docs/spec/rules.md, Pits).
	steps(input.Place{Cell: Pit, Thing: ThingGuard})
	if len(l.ThingsAt(Pit)) != 0 {
		t.Error("a guard stands on the pit")
	}
	steps(input.Drop{Cell: Pit, Item: ItemRock})
	if c := w.Level(Cellar); c == nil || len(c.Objects[content.Cell{3, 3}]) != 1 {
		t.Error("the rock did not fall to the cellar")
	}
}

// TestCheckedIn checks that the pack in the repository is what Build writes. After changing the
// pack, run "summoning devpack --force".
func TestCheckedIn(t *testing.T) {
	dir := t.TempDir()
	if err := content.Write(Build(), dir); err != nil {
		t.Fatal(err)
	}
	files := func(root string) map[string]string {
		out := map[string]string{}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			rel, _ := filepath.Rel(root, path)
			out[filepath.ToSlash(rel)] = string(data)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	want, got := files(dir), files(filepath.Join("..", "packs", "dev"))
	for name, data := range want {
		if g, ok := got[name]; !ok {
			t.Errorf("packs/dev has no %s", name)
		} else if g != data {
			t.Errorf("packs/dev/%s is out of date", name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("packs/dev/%s is not part of the pack", name)
		}
	}
	if t.Failed() {
		t.Log(`regenerate it with "go run . devpack --force"`)
	}
}

// player drives the dev pack's player through a test.
type player struct {
	t *testing.T
	w *sim.World
}

// do runs one update with cmds.
func (p player) do(cmds ...input.Command) []sim.Event { return p.w.Update(cmds) }

// wait runs updates for about a second, for doors to move.
func (p player) wait() {
	for range sim.StepsPerSecond {
		p.w.Update(nil)
	}
}

// to walks the player to the middle of each cell in turn, along one axis at a time, and fails
// the test if it is stopped on the way.
func (p player) to(cells ...content.Cell) {
	p.t.Helper()
	for _, c := range cells {
		for range 4000 {
			pl := p.w.Player()
			cu, cv := 32*c[0]+16, 32*c[1]+16
			if pl.U == cu && pl.V == cv {
				break
			}
			d := 2
			switch {
			case pl.U > cu:
				d = 0
			case pl.U < cu:
				d = 4
			case pl.V > cv:
				d = 6
			}
			p.w.Update([]input.Command{input.Walk{Direction: d}})
		}
		if pl := p.w.Player(); pl.Cell() != c {
			p.t.Fatalf("stopped at %v (%d, %d) on the way to %v", pl.Cell(), pl.U, pl.V, c)
		}
	}
}

// TestHallPlays plays the hall from the start to the cellar with nothing but the player's
// commands: the lever opens closet A, two rocks hold plate B down, the key from the sack opens
// slot C, the gems from closets B and C fill the socket, lever D in the nook it opens opens door
// D, the sphere from the nook thrown at door E opens it, and exit D leads to the cellar.
func TestHallPlays(t *testing.T) {
	w, err := sim.New(Build(), 1)
	if err != nil {
		t.Fatal(err)
	}
	p := player{t, w}
	l := w.Current()
	if w.Player().Cell() != HallStart {
		t.Fatalf("the player starts at %v", w.Player().Cell())
	}

	p.to(content.Cell{LeverA[0], HallStart[1]}, LeverA)
	p.do(input.Activate{})
	p.wait()
	if l.Blocked(DoorA) {
		t.Fatal("lever A did not open door A")
	}

	for _, rock := range RocksB {
		p.to(content.Cell{rock[0], PlateB[1]}, rock)
		p.do(input.PickUp{Hand: true})
		p.to(content.Cell{rock[0], PlateB[1]}, PlateB)
		p.do(input.PutDown{})
	}
	p.to(content.Cell{6, PlateB[1]})
	p.wait()
	if l.Blocked(DoorB) {
		t.Fatal("the rocks on plate B did not hold door B open")
	}
	p.to(GemB)
	p.do(input.PickUp{Hand: true})
	p.do(input.Swap{Slot: 0})

	p.to(content.Cell{6, GemB[1]}, content.Cell{6, Sack[1]}, Sack)
	p.do(input.PickUp{Hand: true})
	p.do(input.Swap{Slot: sim.SlotLeftHand})
	p.do(input.Swap{Slot: sim.SlotPack})
	if h := w.Player().Hand; h == nil || h.Item != ItemKey {
		t.Fatalf("took %v out of the sack", h)
	}
	p.to(content.Cell{SlotC[0], Sack[1]}, SlotC)
	p.do(input.PutDown{})
	p.wait()
	if l.Blocked(DoorC) {
		t.Fatal("the key did not open door C")
	}
	p.to(GemC)
	p.do(input.PickUp{Hand: true})

	p.to(content.Cell{6, GemC[1]}, content.Cell{6, Socket[1]}, Socket)
	p.do(input.PutDown{})
	p.do(input.Swap{Slot: 0})
	p.do(input.PutDown{})
	if l.Blocked(SocketWall) {
		t.Fatal("the gems did not open the socket's wall")
	}
	p.to(LeverD)
	p.do(input.Activate{})
	p.wait()
	if l.Blocked(DoorD) {
		t.Fatal("lever D did not open door D")
	}
	// The sphere thrown at door E unseals it.
	p.to(Sphere)
	p.do(input.PickUp{Hand: true})
	p.do(input.Swap{Slot: sim.SlotRightHand})
	p.to(content.Cell{6, Sphere[1]}, content.Cell{6, SealedDoor[1]})
	p.w.Player().Facing = 0
	p.do(input.Throw{Hand: 0})
	p.wait()
	if l.Blocked(SealedDoor) {
		t.Fatal("the thrown sphere did not open door E")
	}
	p.to(content.Cell{6, DoorD[1]}, DoorD, content.Cell{3, DoorD[1]})
	for range 60 {
		p.do(input.Walk{Direction: 0})
	}
	if w.Current().ID != Cellar {
		t.Errorf("exit D led to %s", w.Current().ID)
	}
}

// TestCellarFight fights the cellar's brute with the sword, wearing the mail: the brute chases
// the player, the player strikes back until it is a corpse, and it drops its key.
func TestCellarFight(t *testing.T) {
	w, err := sim.New(Build(), 1)
	if err != nil {
		t.Fatal(err)
	}
	w.Update([]input.Command{input.GoTo{Level: Cellar}})
	p, l := w.Player(), w.Current()
	p.Hand = &sim.Object{Item: ItemSword, Value: 2500}
	w.Update([]input.Command{input.PutOn{}})
	p.Hand = &sim.Object{Item: ItemMail, Value: 7500}
	w.Update([]input.Command{input.PutOn{}})
	if p.Slots[sim.SlotRightHand] == nil || p.ArmorClass != 5 {
		t.Fatalf("not armed: right hand %v, armor %d", p.Slots[sim.SlotRightHand], p.ArmorClass)
	}
	brute := l.Thing("brute")
	for range 5000 {
		if brute.HitPoints == 0 || p.Dead {
			break
		}
		du, dv := brute.U-p.U, brute.V-p.V
		d := 2
		switch {
		case abs(du) >= abs(dv) && du < 0:
			d = 0
		case abs(du) >= abs(dv):
			d = 4
		case dv < 0:
			d = 6
		}
		cmds := []input.Command{input.Strike{}}
		if abs(du)+abs(dv) > 24 {
			cmds = []input.Command{input.Walk{Direction: d}}
		} else {
			p.Facing = d
		}
		w.Update(cmds)
	}
	if p.Dead || brute.HitPoints != 0 {
		t.Fatalf("the fight did not end with the brute dead: player %d, brute %d hit points", p.HitPoints, brute.HitPoints)
	}
	if l.Thing("brute") == nil || len(l.Objects[brute.Cell()]) == 0 {
		t.Error("the brute left no corpse or no key")
	}
	if p.Experience == 0 || p.Skills[sim.SkillWeapon].Practice+p.Skills[sim.SkillWeapon].Level < 2 {
		t.Errorf("experience %d, sword skill %+v", p.Experience, p.Skills[sim.SkillWeapon])
	}
}

// TestEnterUsesKey stands the player on keyhole C with the key it takes: Enter puts the key in
// from a shelf, or from the sack in the left hand. As an object hole the slot is left a mystery,
// and a disarmed keyhole keeps the key where it was.
func TestEnterUsesKey(t *testing.T) {
	setup := func(t *testing.T, keyhole, armed bool) (*sim.World, *sim.Level) {
		t.Helper()
		w, err := sim.New(Build(), 1)
		if err != nil {
			t.Fatal(err)
		}
		w.Update([]input.Command{input.GoTo{Level: "level.hall"}, input.PlacePlayer{Cell: SlotC}})
		l := w.Current()
		f := l.FeatureAt(SlotC)
		f.Keyhole, f.Armed = keyhole, armed
		return w, l
	}
	enter := func(w *sim.World) {
		w.Update([]input.Command{input.Activate{}})
		for range 120 {
			w.Update(nil)
		}
	}

	w, l := setup(t, true, true)
	w.Player().Slots[7] = &sim.Object{Item: ItemKey}
	enter(w)
	if w.Player().Slots[7] != nil || l.Blocked(DoorC) {
		t.Errorf("Enter on the keyhole: shelf 7 %v, door C blocked %v", w.Player().Slots[7], l.Blocked(DoorC))
	}

	w, l = setup(t, true, true)
	sack := &sim.Object{Item: ItemSack, Contents: []*sim.Object{{Item: ItemRock}, {Item: ItemKey}}}
	w.Player().Slots[sim.SlotLeftHand] = sack
	enter(w)
	if len(sack.Contents) != 1 || sack.Contents[0].Item != ItemRock || l.Blocked(DoorC) {
		t.Errorf("Enter with the key in the sack: %v, door C blocked %v", sack.Contents, l.Blocked(DoorC))
	}

	w, l = setup(t, false, true)
	w.Player().Slots[0] = &sim.Object{Item: ItemKey}
	enter(w)
	if w.Player().Slots[0] == nil || !l.Blocked(DoorC) {
		t.Error("Enter put the key in an object hole")
	}

	w, l = setup(t, true, false)
	w.Player().Slots[0] = &sim.Object{Item: ItemKey}
	enter(w)
	if w.Player().Slots[0] == nil || !l.Blocked(DoorC) {
		t.Error("a disarmed keyhole took the key")
	}
}
