package summoning

import (
	"fmt"
	"image"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The player's figure (player.md, Engine model): layers from the character animation banks.
const (
	playerID = "thing.player"
	// playerWalkGroups are the groups of eight frames the walk uses: group 1 stands, and the walk
	// steps through groups 1..5 and 0.
	playerWalkGroups = 6
	// legsBank, bodyLook and headLook are offsets from the character animation base: the legs, the
	// unarmored body and the bare head.
	legsBank = 1
	bodyLook = 0x25
	headLook = 2
	// playerInterval is the player's base interval in ticks.
	playerInterval = 5
)

// starterAttributes are the player's attributes until character creation (player.md, Engine
// model) and starting skills, and its hit and magic points: STR + END + 20 and TAL + POW + 20.
var starterAttributes = map[string]int{
	"strength": 18, "agility": 20, "endurance": 18, "accuracy": 16, "talent": 16, "power": 16,
	"weapon.0": 1, "school.0": 1,
}

// starterInventory are the objects the player starts with, by slot and template, for weapon class
// 0 and school 0 (player.md, Starting inventory).
var starterInventory = []struct{ slot, template int }{
	{0, 0x7C}, {1, 0x01}, {2, 0xDF}, {26, 0x0E}, {24, 0x24}, {22, 0x61},
}

const (
	starterHitPoints   = 18 + 18 + 20
	starterMagicPoints = 16 + 16 + 20
)

// playerSwingGroups are the groups after the walk's that show strikes: frame 30h + 8 × swing +
// facing, swing 0 the right hand, 1 the left, 2 both (combat.md, The player attacks).
const playerSwingGroups = 3

// headFrames are the first frame of the body look's frames for each group of eight.
var headFrames = [playerWalkGroups + playerSwingGroups]int{0, 8, 16, 24, 16, 8, 32, 40, 48}

// player adds the player's template, its figure composed from the character animation banks:
// the body, the legs, the body look and the head look, as the figure stands out of water.
func (im *importer) player() error {
	base := int(im.init.Bases.CharacterAnimation)
	var banks [4]*SpriteBank
	for i, off := range []int{0, legsBank, bodyLook, headLook} {
		b, err := im.bank(base + off)
		if err != nil {
			im.report.addf("player: %v; no player", err)
			return nil
		}
		banks[i] = b
	}
	body, legs, look, head := banks[0], banks[1], banks[2], banks[3]
	at := [4]content.Offset{{0, 13}, {0, 18}, {0, 5}, {8, 0}}
	var box image.Rectangle
	for i, b := range banks {
		box = box.Union(image.Rect(at[i][0], at[i][1], at[i][0]+b.Width, at[i][1]+b.Height))
	}
	n := 8 * playerWalkGroups
	if len(body.Frames) < n || len(legs.Frames) < n || len(look.Frames) < 32 || len(head.Frames) < 8 {
		im.report.addf("player: character animation banks too short; no player")
		return nil
	}
	// The strike groups take the body and legs of the step, and need the body look's later frames.
	if len(look.Frames) >= headFrames[len(headFrames)-1]+8 {
		n += 8 * playerSwingGroups
	} else {
		im.report.addf("player: body look bank has %d frames; no strike frames", len(look.Frames))
	}
	frames := make([][]byte, n)
	for f := range n {
		group, step := f/8, f%8
		body0 := f
		if group >= playerWalkGroups {
			body0 = step
		}
		frames[f] = compose(box, body.Transparent, []part{
			{body, body0, at[0]}, {legs, body0, at[1]}, {look, headFrames[group] + step, at[2]}, {head, step, at[3]},
		})
	}
	sid := "sprite.player"
	im.addSprite(sid, frames, box.Dx(), box.Dy(), body.Transparent)
	anchor := content.Offset{creatureAnchor[0] - box.Min.X, creatureAnchor[1] - box.Min.Y}
	im.pack.Sprites[sid].Anchor = &anchor

	every := content.Fixed(ms(playerInterval))
	anims := map[string]content.Animation{}
	for d := range 8 {
		p := (d + 1) % 8 // the facing that walks direction d
		anims[fmt.Sprintf("idle.%d", d)] = content.Animation{Frames: []int{p + 8}}
		anims[fmt.Sprintf("walk.%d", d)] = content.Animation{Frames: []int{p + 8, p + 16, p + 24, p + 32, p + 40, p}, MS: every}
		if n > 8*playerWalkGroups {
			for swing, name := range []string{"attack", "attack.left", "attack.both"} {
				anims[fmt.Sprintf("%s.%d", name, d)] = content.Animation{Frames: []int{8*(playerWalkGroups+swing) + p}}
			}
		}
	}
	im.pack.Things[playerID] = &content.Thing{
		ID: playerID, Kind: content.ThingPlayer, Sprite: sid, Facing: 3, Speed: creatureSpeed,
		IntervalMS: ms(playerInterval), Weight: creatureWeight, Footprint: content.DefaultFootprint,
		HitPoints: content.Dice{Bonus: starterHitPoints}, MagicPoints: content.Dice{Bonus: starterMagicPoints},
		Stats: starterAttributes, Animations: anims,
	}
	for _, s := range starterInventory {
		if id, ok := im.item(s.template, "starting inventory"); ok {
			im.pack.Things[playerID].Inventory = append(im.pack.Things[playerID].Inventory,
				content.InventoryObject{Slot: s.slot, Object: content.Object{Item: id}})
		}
	}
	im.pack.Manifest.Player = playerID
	return nil
}
