package sim

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// A new game from a character made in the frontend (docs/spec/scenes.md, The frontend and New
// game; docs/spec/player.md, Starting a new game).

// Character is a character made in the frontend.
type Character struct {
	Name     string
	Portrait string
	// Attributes are the six attributes by name.
	Attributes  map[string]int
	WeaponClass int
	School      int
}

// Derived points (docs/spec/player.md, Derived values).
const derivedBase = 20

// NewGame returns a world for pack p started by character c, as its creation record says: the
// attributes and points, the portrait and sex, the trained weapon class and school, the starting
// objects, the icons the school knows, and the new game's counter and opening conversation.
func NewGame(p *content.Pack, seed uint64, c Character) (*World, error) {
	cr := p.Creation
	if cr == nil {
		return nil, fmt.Errorf("pack %s has no creation record", p.Manifest.ID)
	}
	w, err := New(p, seed)
	if err != nil {
		return nil, err
	}
	pl := w.player
	if pl == nil {
		return nil, fmt.Errorf("pack %s has no player", p.Manifest.ID)
	}
	w.music = p.Manifest.Start.Music
	pl.Name = c.Name
	if c.Portrait != "" {
		pl.Portrait = c.Portrait
		if pt := p.Portraits[c.Portrait]; pt != nil {
			pl.Status.Female = pt.Female
		}
	}
	for _, a := range attributeNames {
		if v, ok := c.Attributes[a]; ok {
			pl.Attributes[a], pl.MaxAttributes[a] = v, v
		}
	}
	pl.MaxHitPoints = pl.Attributes[AttrStrength] + pl.Attributes[AttrEndurance] + derivedBase
	pl.MaxMagicPoints = pl.Attributes[AttrTalent] + pl.Attributes[AttrPower] + derivedBase
	pl.HitPoints, pl.MagicPoints = pl.MaxHitPoints, pl.MaxMagicPoints
	for i := range pl.Skills {
		pl.Skills[i] = Skill{Threshold: skillThreshold}
	}
	pl.Skills[SkillWeapon+c.WeaponClass].Level = 1
	pl.Skills[SkillSchool+c.School].Level = 1

	// The starting objects replace the template's.
	for i, o := range pl.Slots {
		if o != nil {
			pl.Slots[i] = nil
			if i >= Shelves {
				w.takeOff(o)
			}
		}
	}
	objects := append([]content.InventoryObject(nil), cr.Inventory...)
	if n := len(cr.BodyArmor); n > 0 {
		objects = append(objects, content.InventoryObject{Slot: SlotBody, Object: content.Object{Item: cr.BodyArmor[w.rng.IntN(n)]}})
	}
	pl.Magic = Magic{}
	if c.WeaponClass >= 0 && c.WeaponClass < len(cr.WeaponClasses) {
		objects = append(objects, cr.WeaponClasses[c.WeaponClass].Inventory...)
	}
	if c.School >= 0 && c.School < len(cr.Schools) {
		s := cr.Schools[c.School]
		objects = append(objects, s.Inventory...)
		pl.Magic.teach(s.Icons)
	}
	for _, io := range objects {
		o := w.object(io.Object)
		if o == nil || io.Slot < 0 || io.Slot >= SlotCount {
			continue
		}
		pl.Slots[io.Slot] = o
		if io.Slot >= Shelves {
			w.putOn(o)
		}
	}
	if cr.Counter != "" {
		w.setCounter(cr.Counter, 1)
	}
	if t := p.Things[cr.Opening]; t != nil {
		w.opening = t
	}
	w.events = nil
	return w, nil
}
