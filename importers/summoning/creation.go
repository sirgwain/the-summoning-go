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
	if s := im.storyScene(); s != nil {
		c.Story = s.ID
	}
	// The ending power's choices: the item held, the NPC record stored in the counter, and the
	// ending's name. Their end scenes come after the audio (endscenes.go).
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

// storyTrack plays from Create Character through the story (audio.md, Music in the frontend).
const storyTrack = 1

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

// The story in short (scenes.md, The story, in short): its pictures, as frontend resources
// (INIT's frontend base +), and where each goes on the 320 × 400 screen.
const (
	storySceneID  = "scene.story"
	resStoryTop   = 0x16
	resStoryBelow = 0x17
	resStoryGifts = 0x18
	resStoryLift  = 0x19
	resStoryHands = 0x1A
	resStoryArms  = 0x26
	// storyGoneMS is how long Rowena shows alone once the hands and gifts are gone.
	storyGoneMS = 2000
)

// storyWords are the story's words, in the engine's own telling: the summary's pages, Rowena's
// first page, the gift by weapon class, the conversation (true for Rowena's lines) and the last
// page.
var (
	storySummary = []string{
		"War has spread across the land, and the black mage Shadow Weaver grows stronger with every season. In a fortress hidden on a lonely hill, a Council of mages led by the archmage Rowena searched for one of the bloodline of the hero who long ago overcame the tower of DarkSpyre. Their search found you.",
		"Brought before the Council, you swore to stand against Shadow Weaver. Through the months that followed, the weapon master Khail trained your arm, and the mage Jairus taught you to bend magic to your will.",
		"On the night your training ended, a gateway opened within the fortress and Shadow Weaver's horde poured through. Khail and his warriors fell holding the hall. Jairus held the enemy back with fire and lightning until an arrow found him.",
	}
	storyFear  = "Rowena turns to you, and for the first time you see fear in her face."
	storyGifts = []string{"a falchion", "a war axe", "a spear", "a bow and a quiver of arrows"}
	storyLines = []struct {
		rowena bool
		text   string
	}{
		{true, "@, the gateway will not stay open for long. Take these, and listen."},
		{true, "We are lost here. The Council and I will flee and find shelter where we can, but you must go on alone."},
		{false, "Then I will go, and finish what my ancestor began."},
		{true, "I will send you to the entrance of the labyrinth. A man named Malachi waits for you there, and you can trust him. Be sure to ask him about the coins: they will help you on your way."},
		{true, "The way to destroy Shadow Weaver lies somewhere in the labyrinth. Find it. Now, be ready..."},
	}
	storyAway = "Rowena lifts her hands, the air around you begins to shimmer, and the fortress falls away."
)

// readingMS is how long a page of words holds: 2 seconds and 45 ms a letter.
func readingMS(text string) int { return 2000 + 45*len([]rune(text)) }

// storyScene adds the story in short (scenes.md, The story, in short).
func (im *importer) storyScene() *content.Scene {
	base := int(im.init.Bases.Frontend)
	strip := func(res int) (string, bool) {
		id, err := im.stripSprite(base+res, "sprite.strip")
		return id, err == nil
	}
	top, ok1 := strip(resStoryTop)
	below, ok2 := strip(resStoryBelow)
	gifts, ok3 := strip(resStoryGifts)
	lift, ok4 := strip(resStoryLift)
	hands, ok5 := strip(resStoryHands)
	var arms []string
	for class := range storyGifts {
		if id, ok := strip(resStoryArms + class); ok {
			arms = append(arms, id)
		}
	}
	_, ok6 := im.pack.Portraits[rowenaID]
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || len(arms) != len(storyGifts) {
		im.report.addf("the story scene: its pictures or Rowena's portrait are missing; left out")
		return nil
	}
	text := func(key, words string) string {
		im.strings[key] = words
		return key
	}
	pic := func(id string, x, y int) content.SceneStep {
		return content.SceneStep{Picture: &content.Image{Sprite: id}, At: content.Offset{x, y}}
	}
	summaryBox, columnBox := &[4]int{16, 60, 288, 280}, &[4]int{0, 30, 132, 340}
	column := func(key, words string) content.SceneStep {
		return content.SceneStep{Text: text(key, words), Box: columnBox, FromTop: true, MS: readingMS(words)}
	}
	s := &content.Scene{ID: storySceneID, Size: [2]int{content.SceneWidth, 2 * content.SceneHeight}}
	for i, words := range storySummary {
		st := content.SceneStep{Text: text(fmt.Sprintf("story.%d", i+1), words), Box: summaryBox, MS: readingMS(words)}
		st.Clear = i == 0
		s.Steps = append(s.Steps, st)
	}
	// rowena draws Rowena, her hands lifted or holding out the gifts.
	rowena := func(lifted, keepText bool) {
		first := pic(top, 132, 26)
		first.Clear, first.KeepText = true, keepText
		s.Steps = append(s.Steps, first, pic(below, 132, 226))
		if lifted {
			s.Steps = append(s.Steps, pic(lift, 164, 54))
		} else {
			s.Steps = append(s.Steps, pic(gifts, 164, 68))
		}
	}
	// picture draws the farewell with the hands holding the gifts and the arms by weapon class.
	picture := func(lifted bool) {
		rowena(lifted, false)
		s.Steps = append(s.Steps, pic(hands, 164, 198))
		var byWeapon []content.SceneStep
		for _, id := range arms {
			byWeapon = append(byWeapon, pic(id, 204, 266))
		}
		s.Steps = append(s.Steps, content.SceneStep{ByWeapon: byWeapon})
	}
	picture(false)
	s.Steps = append(s.Steps, column("story.fear", storyFear))
	var gift []content.SceneStep
	for class, g := range storyGifts {
		words := "She presses into your hands " + g + ", an apple, a worn old parchment and a scroll of your school's magic."
		gift = append(gift, column(fmt.Sprintf("story.gift.%d", class), words))
	}
	s.Steps = append(s.Steps, content.SceneStep{ByWeapon: gift})
	for i, l := range storyLines {
		say := &content.SceneSay{Side: content.SayBottom, Text: text(fmt.Sprintf("story.say.%d", i+1), l.text)}
		if l.rowena {
			say.Side, say.Portrait = content.SayTop, rowenaID
		}
		s.Steps = append(s.Steps, content.SceneStep{Say: say, MS: readingMS(l.text)})
	}
	picture(true)
	s.Steps = append(s.Steps, column("story.away", storyAway))
	// The hands and gifts are gone: Rowena alone, the page still showing.
	rowena(true, true)
	s.Steps[len(s.Steps)-1].MS = storyGoneMS
	s.Steps = append(s.Steps, content.SceneStep{Fade: "#000000", MS: 1000})
	im.pack.Scenes[s.ID] = s
	return s
}
