package summoning

import (
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// The inventory panel's art and words, and each item's worn look and action icons (items.md, The
// inventory panel).

// Resources and texts of the inventory panel.
const (
	resDoll         = 8
	resWeaponIcons  = 0x12
	resSchoolIcons2 = 0x13
	resActionIcons  = 0x30
	resBareHands    = 0x32
	resStatusIcons  = 0x52
	wornQuiverSlot  = 0x35
	wornQuiver      = 0x36
	iconStrike      = 8
	iconThrow       = 0
	skillRatings    = 10
	msgFirstTitle   = 0x15
	levelTitles     = 12
	msgWeight       = 0x81
	categoryQuiver  = 4
)

// inventory adds the inventory record, and gives every item its worn look and action icons.
func (im *importer) inventory() {
	in := &content.Inventory{ID: "inventory"}
	if sid, err := im.stripSprite(resDoll, "sprite.strip"); err == nil {
		in.Doll = &content.Image{Sprite: sid}
	} else {
		im.report.addf("inventory: the doll: %v", err)
	}
	in.QuiverSlot = im.wornLook(wornQuiverSlot)
	frames := func(res int, list ...int) []content.Image {
		sid, b, err := im.rawSprite(res)
		if err != nil {
			im.report.addf("inventory: resource %d: %v", res, err)
			return nil
		}
		var out []content.Image
		for _, f := range list {
			if f < len(b.Frames) {
				out = append(out, content.Image{Sprite: sid, Frame: f})
			}
		}
		return out
	}
	in.BareHands = frames(resBareHands, 1, 0)
	if icons := frames(resActionIcons, iconStrike, iconThrow); len(icons) == 2 {
		in.StrikeIcon, in.ThrowIcon = &icons[0], &icons[1]
	}
	in.WeaponIcons = frames(resWeaponIcons, 0, 1, 2, 3)
	// The status icons, in the game's order from the right (scenes.md, The game screen).
	for f, st := range []string{content.StatusConfused, content.StatusLightningShield, content.StatusBattleRage,
		content.StatusPoisoned, content.StatusFireShield} {
		if icon := frames(resStatusIcons, f); len(icon) == 1 {
			in.StatusIcons = append(in.StatusIcons, content.StatusIcon{Status: st, Image: icon[0]})
		}
	}
	in.SchoolIcons = frames(resSchoolIcons2, 0, 1, 2, 3)
	text := func(n int) (string, bool) {
		id := msgID(n)
		_, ok := im.strings[id]
		return id, ok
	}
	for l := 1; l <= skillRatings; l++ {
		if id, ok := text(l); ok {
			in.WeaponRatings = append(in.WeaponRatings, id)
		}
		if id, ok := text(l + skillRatings); ok {
			in.SchoolRatings = append(in.SchoolRatings, id)
		}
	}
	for l := range levelTitles {
		if id, ok := text(msgFirstTitle + l); ok {
			in.LevelTitles = append(in.LevelTitles, id)
		}
	}
	in.Labels = map[string]string{}
	for i, k := range []string{content.LabelWeight, content.LabelDamage, content.LabelDamageAfter, content.LabelArmor, content.LabelTwoHanded} {
		if id, ok := text(msgWeight + i); ok {
			in.Labels[k] = id
		}
	}
	im.pack.Inventory = in

	for i := range im.objects.Templates {
		tm := &im.objects.Templates[i]
		it := im.pack.Items[im.itemIDs[i]]
		look := tm.Image()
		if tm.Category() == categoryQuiver {
			look = wornQuiver
		}
		it.Doll = im.wornLook(look)
		if i == templateChest {
			it.DollLeft = im.wornLook(look + 1)
		}
		it.DollOver = tm.Category() == categoryOverHands && tm.Armor() > 0
		if it.Fill != nil {
			for k := range it.Fill.Icons {
				if d := im.wornLook(look + k + 1); d != nil {
					it.Fill.Dolls = append(it.Fill.Dolls, *d)
				}
			}
		}
		var icons []int
		for a := range 2 {
			switch icon := tm.ActionIcon(a); {
			case icon >= 100 && icon < 255:
				icons = append(icons, (icon-100)/10+1)
			case icon >= 9 && icon <= 20:
				icons = append(icons, icon)
			}
		}
		for n := range it.Actions {
			if n < len(icons) {
				if f := frames(resActionIcons, icons[n]); len(f) == 1 {
					it.Actions[n].Icon = &f[0]
				}
			}
		}
	}
}

// templateChest shows a look of its own in the left hand; objects of categoryOverHands with
// armor are drawn over both hands' others (items.md, The inventory panel).
const (
	templateChest     = 3
	categoryOverHands = 10
)

// wornLook returns worn image n placed at its offset on the doll, or nil when there is none.
func (im *importer) wornLook(n int) *content.DollImage {
	offsets := im.init.WornOffsets
	if n < 0 || n >= len(offsets) {
		return nil
	}
	sid, err := im.stripSprite(int(im.init.Bases.Worn)+n, "sprite.worn")
	if err != nil {
		im.report.addf("worn image %d: %v", n, err)
		return nil
	}
	// The offset's first byte is the row, its second the column.
	return &content.DollImage{Image: content.Image{Sprite: sid}, At: content.Offset{int(offsets[n][1]), int(offsets[n][0])}}
}
