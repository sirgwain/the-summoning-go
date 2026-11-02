package viewer

import (
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// The level map (docs/spec/world.md, Level map): the cells the player has seen, by what they
// are, and where the player stands. The full map opens with M while playing and closes on the
// next key or click; the minimap shows in a corner unless the level refuses the map.

// Map cell kinds: what the game's map tells apart, and floor and water.
const (
	MapFloor = iota
	MapWall
	MapDoor
	MapExit
	MapWater
	MapHole
	MapTeleporter
	MapMouth
	MapPlate
	MapMark
	MapTrapdoor
	MapSocket
	MapKinds
)

// MapKindNames name the kinds in the map's legend.
var MapKindNames = [MapKinds]string{
	MapFloor: "Floor", MapWall: "Wall", MapDoor: "Door", MapExit: "Exit", MapWater: "Water",
	MapHole: "Pit", MapTeleporter: "Teleporter", MapMouth: "Mouth", MapPlate: "Plate",
	MapMark: "Rune", MapTrapdoor: "Trapdoor", MapSocket: "Socket",
}

// MapCell is a seen cell of the level map.
type MapCell struct {
	Cell content.Cell
	Kind int
}

// LevelMap is the current level's map.
type LevelMap struct {
	Name  string
	Size  [2]int
	Cells []MapCell
	// Player is the player's cell.
	Player content.Cell
	// Legend are the kinds the map shows, in kind order.
	Legend []int
}

// Map returns the current level's map as the player has seen it, or nil without a player.
func (v *Viewer) Map() *LevelMap {
	p := v.World.Player()
	if p == nil {
		return nil
	}
	l := v.World.Current()
	m := &LevelMap{Size: l.Def.Size, Player: p.Cell()}
	var shown [MapKinds]bool
	if l.Def.Name != "" {
		m.Name = inspect.Text(v.Pack, l.Def.Name)
	}
	for x := range l.Def.Size[0] {
		for y := range l.Def.Size[1] {
			c := content.Cell{x, y}
			if !l.Seen(c) {
				continue
			}
			kind := MapFloor
			ft := l.FloorType(c)
			switch f := l.FeatureAt(c); {
			case f != nil && f.Kind == content.FeatureDoor:
				kind = MapDoor
			case f != nil && f.Kind == content.FeatureExit:
				kind = MapExit
			case f != nil && f.Kind == content.FeatureMouth:
				kind = MapMouth
			case f != nil && f.Kind == content.FeatureTeleporter && f.Armed:
				kind = MapTeleporter
			case f != nil && (f.Kind == content.FeaturePit || f.Kind == content.FeatureTrapdoor && f.Open):
				kind = MapHole
			case f != nil && f.Kind == content.FeatureTrapdoor:
				kind = MapTrapdoor
			case f != nil && f.Kind == content.FeaturePlate:
				kind = MapPlate
			case f != nil && f.Kind == content.FeatureMark:
				kind = MapMark
			case f != nil && f.Kind == content.FeatureSocket:
				kind = MapSocket
			case l.Wall[x][y] != content.NoFrame:
				kind = MapWall
			case ft == nil:
				continue
			case ft.Depth > 0:
				kind = MapWater
			}
			m.Cells = append(m.Cells, MapCell{Cell: c, Kind: kind})
			shown[kind] = true
		}
	}
	for k, ok := range shown {
		if ok {
			m.Legend = append(m.Legend, k)
		}
	}
	return m
}

// MinimapShown reports whether the minimap shows: while playing, on a level that does not refuse
// its map, with no window open over the view.
func (v *Viewer) MinimapShown() bool {
	return v.Play && v.World.Player() != nil && !v.World.MapRefused() && !v.MapOpen && v.World.Talk() == nil
}

// openMap opens the full map when the events show it.
func (v *Viewer) openMap(events []sim.Event) {
	for _, e := range events {
		if e.Kind == sim.EventShowMap {
			v.MapOpen = true
		}
	}
}
