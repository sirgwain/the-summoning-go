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
// D, and exit D leads to the cellar.
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
		p.do(input.PickUp{})
		p.to(content.Cell{rock[0], PlateB[1]}, PlateB)
		p.do(input.PutDown{})
	}
	p.to(content.Cell{6, PlateB[1]})
	p.wait()
	if l.Blocked(DoorB) {
		t.Fatal("the rocks on plate B did not hold door B open")
	}
	p.to(GemB)
	p.do(input.PickUp{})
	p.do(input.Swap{Slot: 0})

	p.to(content.Cell{6, GemB[1]}, content.Cell{6, Sack[1]}, Sack)
	p.do(input.PickUp{})
	p.do(input.Swap{Slot: sim.SlotRightHand})
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
	p.do(input.PickUp{})

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
	p.to(content.Cell{6, DoorD[1]}, DoorD, content.Cell{3, DoorD[1]})
	for range 60 {
		p.do(input.Walk{Direction: 0})
	}
	if w.Current().ID != Cellar {
		t.Errorf("exit D led to %s", w.Current().ID)
	}
}
