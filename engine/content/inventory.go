package content

// Inventory is what the inventory panel and the hotbar's hands draw (docs/pack.md, Inventory).
// Every field is optional.
type Inventory struct {
	ID string `json:"id"`
	// Doll is the figure worn objects are drawn on; QuiverSlot how the quiver slot's object
	// shows on it.
	Doll       *Image     `json:"doll,omitempty"`
	QuiverSlot *DollImage `json:"quiver_slot,omitempty"`
	// BareHands are the right and left hands' pictures when empty; StrikeIcon and ThrowIcon the
	// icons of striking bare-handed and of throwing.
	BareHands  []Image `json:"bare_hands,omitempty"`
	StrikeIcon *Image  `json:"strike_icon,omitempty"`
	ThrowIcon  *Image  `json:"throw_icon,omitempty"`
	// WeaponIcons and SchoolIcons are the skills' icons; the ratings are text keys for skill
	// levels 1, 2, …; LevelTitles text keys for the player's levels from 0.
	WeaponIcons   []Image  `json:"weapon_icons,omitempty"`
	SchoolIcons   []Image  `json:"school_icons,omitempty"`
	WeaponRatings []string `json:"weapon_ratings,omitempty"`
	SchoolRatings []string `json:"school_ratings,omitempty"`
	LevelTitles   []string `json:"level_titles,omitempty"`
	// Labels are text keys of the details' words, by the Label constants.
	Labels map[string]string `json:"labels,omitempty"`
	// StatusIcons are drawn, in order from the right, for each status that holds.
	StatusIcons []StatusIcon `json:"status_icons,omitempty"`
}

// StatusIcon shows a status of the player while it holds; Name, a text key, names it under the
// pointer in place of the engine's own name.
type StatusIcon struct {
	Status string `json:"status"`
	Image  Image  `json:"image"`
	Name   string `json:"name,omitempty"`
}

// The player's statuses a status icon can show (docs/spec/scenes.md, The game screen).
const (
	StatusConfused         = "confused"
	StatusPoisoned         = "poisoned"
	StatusFireShield       = "fire_shield"
	StatusLightningShield  = "lightning_shield"
	StatusBattleRage       = "battle_rage"
	StatusHaste            = "haste"
	StatusInvisible        = "invisible"
	StatusSight            = "sight"
	StatusRegenerating     = "regenerating"
	StatusManaRegenerating = "mana_regenerating"
	StatusLevitating       = "levitating"
	StatusDeathProtection  = "death_protection"
)

// StatusNames are the engine's own names of the statuses.
var StatusNames = map[string]string{
	StatusConfused: "Confused", StatusPoisoned: "Poisoned", StatusFireShield: "Fire shield",
	StatusLightningShield: "Lightning shield", StatusBattleRage: "Battle rage", StatusHaste: "Haste",
	StatusInvisible: "Invisible", StatusSight: "Sight", StatusRegenerating: "Regenerating",
	StatusManaRegenerating: "Regenerating magic", StatusLevitating: "Levitating",
	StatusDeathProtection: "Protected from dying",
}

// Labels of an object's details.
const (
	LabelWeight      = "weight"
	LabelDamage      = "damage"
	LabelDamageAfter = "damage_after"
	LabelArmor       = "armor"
	LabelTwoHanded   = "two_handed"
)

var labels = map[string]bool{LabelWeight: true, LabelDamage: true, LabelDamageAfter: true, LabelArmor: true, LabelTwoHanded: true}

// DollImage is an image drawn on the inventory's figure, its top-left at At.
type DollImage struct {
	Image Image  `json:"image"`
	At    Offset `json:"at"`
}

func (v *verifier) inventory() {
	in := v.p.Inventory
	if in == nil {
		return
	}
	const owner = "inventory.json"
	img := func(im *Image) {
		if im != nil {
			v.frame(owner, im.Sprite, im.Frame)
		}
	}
	img(in.Doll)
	if in.QuiverSlot != nil {
		img(&in.QuiverSlot.Image)
	}
	img(in.StrikeIcon)
	img(in.ThrowIcon)
	for _, list := range [][]Image{in.BareHands, in.WeaponIcons, in.SchoolIcons} {
		for i := range list {
			img(&list[i])
		}
	}
	for _, list := range [][]string{in.WeaponRatings, in.SchoolRatings, in.LevelTitles} {
		for _, k := range list {
			v.text(owner, k)
		}
	}
	for _, k := range sortedKeys(in.Labels) {
		if !labels[k] {
			v.fail("%s: unknown label %q", owner, k)
		}
		v.text(owner, in.Labels[k])
	}
	seen := map[string]bool{}
	for _, si := range in.StatusIcons {
		switch {
		case StatusNames[si.Status] == "":
			v.fail("%s: unknown status %q", owner, si.Status)
		case seen[si.Status]:
			v.fail("%s: status %q listed twice", owner, si.Status)
		}
		seen[si.Status] = true
		img(&si.Image)
		if si.Name != "" {
			v.text(owner, si.Name)
		}
	}
}
