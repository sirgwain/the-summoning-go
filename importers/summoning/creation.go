package summoning

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// Character creation, the new game and the ending (scenes.md, The frontend, New game, Endings).

// Templates a new character starts with (player.md, Starting a new game), and those the ending
// power needs one of (items.md, Powers).
const (
	templateApple      = 0x7C
	templatePalimpsest = 0x01
	templateTrews      = 0x0E
	templateBoots      = 0x24
	templateQuiver     = 0x02
	quiverArrows       = 4
	npcEnding          = 0x79
	templateSkull      = 0xF7
	templateNecklace   = 0x59
)

// weaponTemplates and schoolScrolls are each weapon class's weapon and each school's scroll.
var (
	weaponTemplates = [4]int{0x61, 0x6A, 0x67, 0x6E}
	schoolScrolls   = [4]int{0xDF, 0xE5, 0xEC, 0xF4}
)

// Resources of the door and the ending (scenes.md).
const (
	resClosedDoor   = 0x3E
	resDoorHalves   = 84
	resDoorFirst    = 85
	resDoorSecond   = 86
	resDungeon      = 3
	resFinalScene   = 0x5A
	resFinalFigure  = 0x5B
	resBanner       = 0x5C
	msgFirstClosing = 0x157
	msgLastClosing  = 0x170
	doorTiles       = 160
	doorColumns     = 40
	doorTileW       = 8
	doorTileH       = 50
)

// creation adds the creation record from F's portrait rows and premade characters.
func (im *importer) creation(rows int, premades []Premade) {
	item := func(t int) string { id, _ := im.item(t, "character creation"); return id }
	// slots are the objects of templates ts in slots ss, leaving out templates OBJECTS lacks.
	slots := func(pairs ...[2]int) []content.InventoryObject {
		var out []content.InventoryObject
		for _, p := range pairs {
			if id := item(p[1]); id != "" {
				out = append(out, content.InventoryObject{Slot: p[0], Object: content.Object{Item: id}})
			}
		}
		return out
	}
	items := func(ts ...int) []string {
		var out []string
		for _, t := range ts {
			if id := item(t); id != "" {
				out = append(out, id)
			}
		}
		return out
	}
	c := &content.Creation{ID: "creation",
		Methods: []content.AttributeMethod{
			{Name: "Rolled", Roll: []content.Dice{{Count: 3, Sides: 4}, {Count: 2, Sides: 5}}, Max: 20},
			{Name: "Spent", Points: []content.Dice{{Count: 28, Sides: 5}}, Max: 20},
			{Name: "Rolled and spent", Roll: []content.Dice{{Count: 2, Sides: 5}}, Points: []content.Dice{{Count: 14, Sides: 5}}, Max: 20},
		},
		// A new game sets no counter and starts no conversation (dialogue.md, variable 34).
		Inventory: slots([2]int{0, templateApple}, [2]int{1, templatePalimpsest}, [2]int{sim24, templateBoots}),
	}
	for i := range rows {
		if _, ok := im.pack.Portraits[portraitID(i)]; ok {
			c.Portraits = append(c.Portraits, portraitID(i))
		}
	}
	c.BodyArmor = items(templateTrews, templateTrews+1, templateTrews+2, templateTrews+3)
	inv := im.pack.Inventory
	for i, w := range weaponTemplates {
		t := content.Training{Inventory: slots([2]int{sim22, w})}
		if inv != nil && i < len(inv.WeaponIcons) {
			icon := inv.WeaponIcons[i]
			t.Icon = &icon
		}
		if quiverID := item(templateQuiver); i == 3 && quiverID != "" && item(templateArrow) != "" {
			quiver := content.Object{Item: quiverID}
			for range quiverArrows {
				quiver.Contents = append(quiver.Contents, content.Object{Item: item(templateArrow)})
			}
			t.Inventory = append(t.Inventory, content.InventoryObject{Slot: sim25, Object: quiver})
		}
		c.WeaponClasses = append(c.WeaponClasses, t)
	}
	for i, s := range schoolScrolls {
		n := i * SpellsPerSchool
		c.Schools = append(c.Schools, content.Training{Name: msgID(msgFirstSchool + i), Icons: icons(im.gesture(n)),
			Inventory: slots([2]int{2, s})})
	}
	names := []string{"strength", "agility", "endurance", "accuracy", "talent", "power"}
	for _, pm := range premades {
		if pm.Portrait >= rows || pm.WeaponClass > 3 || pm.School > 3 {
			im.report.addf("premade character %s: portrait %d, weapon class %d, school %d; left out", pm.Name, pm.Portrait, pm.WeaponClass, pm.School)
			continue
		}
		p := content.Premade{Name: pm.Name, Portrait: portraitID(pm.Portrait), WeaponClass: pm.WeaponClass, School: pm.School,
			Attributes: map[string]int{}}
		for a, n := range names {
			p.Attributes[n] = pm.Attributes[a]
		}
		c.Premade = append(c.Premade, p)
	}
	if s := im.doorScene(); s != nil {
		c.Scene = s.ID
	}
	// The ending power's choices: the item held, the NPC record stored in the counter, and the
	// ending's name. Their end scenes are the game's program's, so none is given.
	c.Ending = &content.Ending{Counter: varID(varEnding)}
	for _, ch := range []struct {
		template, record int
		name             string
	}{{templateSkull, 4, "F"}, {templateWarMonger, 5, "G"}, {templateNecklace, 3, "E"}} {
		if id := item(ch.template); id != "" {
			c.Ending.Choices = append(c.Ending.Choices, content.EndingChoice{Name: ch.name, Holding: id, Value: ch.record})
		}
	}
	if id, err := im.creature(npcEnding); err == nil && im.pack.Things[id].Conversation != "" {
		c.Ending.Conversation = id
	}
	if s := im.endingScene(); s != nil {
		c.Ending.Scene = s.ID
	}
	im.pack.Creation = c
}

// Inventory slots the creation record fills.
const (
	sim22 = 22
	sim24 = 24
	sim25 = 25
)

// doorPicture adds a door picture of 160 tiles 8 × 50, four rows of forty, as a whole picture and
// as its left and right halves, and returns their ids.
func (im *importer) doorPicture(res int) (whole, left, right string, ok bool) {
	b, err := im.bank(res)
	if err != nil || len(b.Frames) < doorTiles || b.Width != doorTileW || b.Height != doorTileH {
		im.report.addf("door picture %d: not 160 tiles of 8 × 50; scene left out", res)
		return "", "", "", false
	}
	compose := func(from, cols int) []byte {
		w := cols * doorTileW
		img := make([]byte, w*4*doorTileH)
		for t := range doorTiles {
			row, col := t/doorColumns, t%doorColumns-from
			if col < 0 || col >= cols {
				continue
			}
			for y := range doorTileH {
				copy(img[(row*doorTileH+y)*w+col*doorTileW:], b.Frames[t][y*doorTileW:(y+1)*doorTileW])
			}
		}
		return img
	}
	whole = fmt.Sprintf("sprite.picture.%d", res)
	im.addSprite(whole, [][]byte{compose(0, doorColumns)}, doorColumns*doorTileW, 4*doorTileH, b.Transparent)
	left, right = whole+".left", whole+".right"
	im.addSprite(left, [][]byte{compose(0, doorColumns/2)}, doorColumns/2*doorTileW, 4*doorTileH, b.Transparent)
	im.addSprite(right, [][]byte{compose(doorColumns/2, doorColumns/2)}, doorColumns/2*doorTileW, 4*doorTileH, b.Transparent)
	return whole, left, right, true
}

// doorSceneID is the new game's door; doorStepEffect sounds at each step of its opening
// (audio.md, Sound effects).
const (
	doorSceneID    = "scene.door"
	doorStepEffect = 0x17
)

// openingTrack plays under the opening picture, which the engine's splash stands in for
// (audio.md, Music in the frontend).
const openingTrack = 3

// doorEndTicks is how long the open door holds; the door's first doorLeadSteps steps of opening
// wait 11 − 2s ticks more than the rest (scenes.md, New game).
const (
	doorEndTicks  = 18
	doorLeadSteps = 6
)

// doorLeadMS returns the times of the door's first steps of opening.
func doorLeadMS() []int {
	lead := make([]int, doorLeadSteps)
	for s := range lead {
		lead[s] = ms(3 + 11 - 2*s)
	}
	return lead
}

// doorScene adds the new game's door (scenes.md, New game).
func (im *importer) doorScene() *content.Scene {
	closed, _, _, ok := im.doorPicture(resClosedDoor)
	_, left, right, ok2 := im.doorPicture(resDoorHalves)
	first, _, err := im.rawSprite(resDoorFirst)
	second, _, err2 := im.rawSprite(resDoorSecond)
	dungeon, err3 := im.stripSprite(resDungeon, "sprite.strip")
	if !ok || !ok2 || err != nil || err2 != nil || err3 != nil {
		im.report.addf("the door scene: its pictures are missing; left out")
		return nil
	}
	s := &content.Scene{ID: doorSceneID}
	s.Steps = append(s.Steps, content.SceneStep{Picture: &content.Image{Sprite: closed}, Clear: true, MS: ms(18)})
	at := content.Offset{135, 0}
	for f := range 4 {
		s.Steps = append(s.Steps, content.SceneStep{Picture: &content.Image{Sprite: first, Frame: f}, At: at, MS: ms(5)})
	}
	for f := range 10 {
		s.Steps = append(s.Steps,
			content.SceneStep{Picture: &content.Image{Sprite: second, Frame: f}, At: at},
			content.SceneStep{Picture: &content.Image{Sprite: second, Frame: f + 10}, At: content.Offset{135, 100}, MS: ms(5)})
	}
	s.Steps = append(s.Steps, content.SceneStep{Part: &content.Part{Left: content.Image{Sprite: left}, Right: content.Image{Sprite: right},
		Background: &content.Image{Sprite: dungeon}, From: 0, To: 19, StepMS: ms(3), LeadMS: doorLeadMS()}, MS: ms(doorEndTicks)})
	im.pack.Scenes[s.ID] = s
	return s
}

// endingScene adds the ending (scenes.md, Endings).
func (im *importer) endingScene() *content.Scene {
	closed, _, _, ok := im.doorPicture(resClosedDoor)
	_, left, right, ok2 := im.doorPicture(resDoorHalves)
	final, err := im.stripSprite(resFinalScene, "sprite.strip")
	figure, _, err2 := im.rawSprite(resFinalFigure)
	banner, err3 := im.stripSprite(resBanner, "sprite.strip")
	if !ok || !ok2 || err != nil || err2 != nil || err3 != nil {
		im.report.addf("the ending scene: its pictures are missing; left out")
		return nil
	}
	s := &content.Scene{ID: "scene.ending"}
	s.Steps = append(s.Steps,
		content.SceneStep{Part: &content.Part{Left: content.Image{Sprite: left}, Right: content.Image{Sprite: right}, From: 20, To: 0, StepMS: ms(3)}},
		content.SceneStep{Picture: &content.Image{Sprite: closed}, Clear: true, MS: ms(18)},
		content.SceneStep{Fade: "#ffffff", MS: ms(18)},
		content.SceneStep{Picture: &content.Image{Sprite: final}, At: content.Offset{44, 10}, Clear: true, MS: ms(90)},
	)
	for f := range 5 {
		at, hold := content.Offset{144, 124}, 3
		if f >= 3 {
			at, hold = content.Offset{159, 114}, 18
		}
		s.Steps = append(s.Steps, content.SceneStep{Picture: &content.Image{Sprite: figure, Frame: f}, At: at, MS: ms(hold)})
	}
	s.Steps = append(s.Steps, content.SceneStep{MS: ms(180)})
	for m := msgFirstClosing; m <= msgLastClosing; m++ {
		if _, ok := im.strings[msgID(m)]; ok {
			s.Steps = append(s.Steps, content.SceneStep{Caption: msgID(m), Banner: &content.Image{Sprite: banner}, MS: ms(45)})
		}
	}
	im.pack.Scenes[s.ID] = s
	return s
}
