package inspect

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

func TestLinks(t *testing.T) {
	p := content.NewPack(content.Manifest{Start: content.Start{Level: "level.00"}})
	p.Texts["text.en"] = &content.Text{Strings: map[string]string{"msg.key": "Key"}}
	p.Items["item.key"] = &content.Item{ID: "item.key", Name: "msg.key", Weight: 1}
	p.Levels["level.00"] = &content.Level{
		ID: "level.00", Size: [2]int{4, 4},
		FloorTypes: []content.FloorType{{}},
		Floor:      content.Grid{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		Wall:       content.Grid{{-1, -1, -1, -1}, {-1, -1, -1, -1}, {-1, -1, -1, -1}, {-1, -1, -1, -1}},
		Features: []*content.Feature{
			{ID: "lever", Kind: content.FeatureLever, Cell: content.Cell{0, 0}, Armed: true, OnTurnOn: "p0", OnTurnOff: "p0"},
			{ID: "plate", Kind: content.FeaturePlate, Cell: content.Cell{1, 0}, Armed: true, Threshold: 1, OnPress: "p1"},
			{ID: "door", Kind: content.FeatureDoor, Cell: content.Cell{2, 0}, State: content.DoorClosed},
			{ID: "tele", Kind: content.FeatureTeleporter, Cell: content.Cell{3, 3}, Armed: true, Destinations: []content.Cell{{0, 3}, {1, 3}}, Next: 1},
		},
		Procedures: map[string][]content.Action{
			// The lever opens the door and rewires the plate to p2.
			"p0": {{On: ptr("door")}, {SetHook: &content.SetHook{Feature: "plate", Hook: content.HookPress, Procedure: ptr("p2")}}},
			"p1": {{Create: &content.Create{Cell: content.Cell{2, 2}, Item: "item.key"}}},
			"p2": {{Toggle: ptr("door")}, {ShootItem: &content.ShootItem{Cell: content.Cell{3, 0}, Item: "item.key", Direction: 2}}},
		},
	}
	w, err := sim.New(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	type got struct {
		Kind          LinkKind
		From, Feature string
		To            content.Cell
		Dir           int
		Text          string
	}
	var links []got
	for _, lk := range Links(p, w.Current()) {
		links = append(links, got{lk.Kind, lk.From, lk.Feature, lk.To, lk.Direction, lk.Text()})
	}
	want := []got{
		// Both of the lever's hooks run p0, so its links merge.
		{LinkOn, "lever", "door", content.Cell{2, 0}, -1, "turn_on/turn_off: on"},
		{LinkHook, "lever", "plate", content.Cell{1, 0}, -1, "turn_on/turn_off: press runs p2"},
		{LinkCell, "plate", "", content.Cell{2, 2}, -1, "press: make Key"},
		// What the plate does once the lever rewires it.
		{LinkToggle, "plate", "door", content.Cell{2, 0}, -1, "press: toggle (if rewired)"},
		{LinkShoot, "plate", "", content.Cell{3, 0}, 2, "press: shoot Key (if rewired)"},
		{LinkTravel, "tele", "", content.Cell{0, 3}, -1, "then"},
		{LinkTravel, "tele", "", content.Cell{1, 3}, -1, "next"},
	}
	if !reflect.DeepEqual(links, want) {
		t.Errorf("got\n%v\nwant\n%v", links, want)
	}
}

func TestCellLineKinds(t *testing.T) {
	w := testWorld(t)
	kinds := map[string]LinkKind{}
	items := map[string]string{}
	for _, l := range CellLines(w.Pack(), w.Current(), content.Cell{0, 0}) {
		kinds[strings.TrimSpace(l.Text)] = l.Kind
		items[strings.TrimSpace(l.Text)] = l.Item
	}
	// Lines naming an item carry it: what an action makes, and an object on the floor.
	if items["make Key at (1, 0)"] != "item.key" || items["Sack (weight 3)"] != "item.sack" ||
		items["Key (weight 1)"] != "item.key" {
		t.Errorf("items %v", items)
	}
	// The plate's p0 opens the door and makes a key.
	if kinds["switch on door"] != LinkOn || kinds["make Key at (1, 0)"] != LinkCell {
		t.Errorf("kinds %v", kinds)
	}
	if kinds["Feature plate (plate)"] != "" {
		t.Errorf("the feature's heading has kind %q", kinds["Feature plate (plate)"])
	}
}
