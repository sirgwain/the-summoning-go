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
// socket's wall, the sphere behind it unseals door D, and exit D leads to the cellar.
func TestHallSolves(t *testing.T) {
	w, err := sim.New(Build(), 1)
	if err != nil {
		t.Fatal(err)
	}
	do := func(cmds ...input.Command) { w.Update(cmds) }
	l := w.Current()
	if !l.Blocked(SocketWall) || !l.Blocked(SealedDoor) {
		t.Fatal("the socket's wall or door D starts open")
	}
	do(input.Drop{Cell: Socket, Item: ItemRedGem})
	do(input.Drop{Cell: Socket, Item: ItemBlueGem})
	if l.Blocked(SocketWall) {
		t.Fatal("the full socket did not open its wall")
	}
	if got := l.Feature("mouth").Message; got != "msg.mouth.after" {
		t.Errorf("mouth says %q", got)
	}
	do(input.Remove{Cell: content.Cell{15, 22}})
	do(input.Drop{Cell: SealedDoor, Item: ItemSphere})
	for range 120 {
		do()
	}
	if l.Blocked(SealedDoor) {
		t.Fatal("the sphere did not open door D")
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
