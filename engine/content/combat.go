package content

import "fmt"

// Combat is combat.json: the pack's damage kinds and the statuses that shield the player from
// them (docs/pack.md, Combat). Without it a pack has the Summoning's.
type Combat struct {
	ID string `json:"id"`
	// Kinds are the damage kinds, the first what blows, falls and boulders do.
	Kinds []DamageKind `json:"kinds"`
	// Shields name, by timed status, the kind of damage each cuts to 1 point.
	Shields map[string]string `json:"shields,omitempty"`
}

// DamageKind is a kind of damage: its id and a text key naming it, for the book.
type DamageKind struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// The Summoning's damage kinds, the engine's default.
const (
	KindPhysical  = "physical"
	KindFire      = "fire"
	KindLightning = "lightning"
	KindArrow     = "arrow"
	KindPoison    = "poison"
)

// DefaultCombat is the combat record of a pack without one: the Summoning's five kinds, and its
// fire and lightning shields.
var DefaultCombat = Combat{
	ID:    combatID,
	Kinds: []DamageKind{{ID: KindPhysical}, {ID: KindFire}, {ID: KindLightning}, {ID: KindArrow}, {ID: KindPoison}},
	Shields: map[string]string{
		TimedFireShield: KindFire, TimedLightningShield: KindLightning,
	},
}

const combatID = "combat"

// CombatRules returns the pack's combat record, or the default.
func (p *Pack) CombatRules() *Combat {
	if p.Combat != nil {
		return p.Combat
	}
	return &DefaultCombat
}

// Kind returns the place of damage kind id in the pack's list, and whether it has one.
func (p *Pack) Kind(id string) (int, bool) {
	for i, k := range p.CombatRules().Kinds {
		if k.ID == id {
			return i, true
		}
	}
	return 0, false
}

// KindOr returns the place of damage kind id, or of the first kind (blows') when the pack has no
// such kind or id is empty.
func (p *Pack) KindOr(id string) int {
	k, _ := p.Kind(id)
	return k
}

// Shot is what a fire action shoots, each part falling back to the ammunition's missile (docs/
// pack.md, Items): the item that flies (and so its look), its damage and kind, how far and how
// fast it flies.
type Shot struct {
	Item   string `json:"item,omitempty"`
	Damage Dice   `json:"damage,omitzero"`
	Kind   string `json:"kind,omitempty"`
	Range  *Range `json:"range,omitempty"`
	// Speed is the units it moves each step, 8 without one.
	Speed int `json:"speed,omitempty"`
}

// Range is how many steps a shot flies: base, plus times the attribute, at most max (none
// without one). The Summoning's arrows fly 5 × strength, at most 127.
type Range struct {
	Base      int    `json:"base,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Times     int    `json:"times,omitempty"`
	Max       int    `json:"max,omitempty"`
}

// DefaultRange is the range of a shot that names none.
var DefaultRange = Range{Attribute: "strength", Times: 5, Max: 127}

// Steps returns the range for attributes a.
func (r Range) Steps(a map[string]int) int {
	n := r.Base
	if r.Attribute != "" {
		n += r.Times * a[r.Attribute]
	}
	if r.Max > 0 {
		n = min(n, r.Max)
	}
	return max(n, 1)
}

// Ammo is where a fire action's shots come from when they are objects: the first object with
// the tag inside what the worn slot holds.
type Ammo struct {
	Tag  string `json:"tag"`
	Slot int    `json:"slot"`
}

// DefaultAmmo is the ammunition of a fire action that names no source: arrows from the quiver.
var DefaultAmmo = Ammo{Tag: "arrows", Slot: 25}

// Shoots is a creature's ranged attack: it shoots item's look at the player in reach, every
// interval, without going through a spell.
type Shoots struct {
	Item       string `json:"item"`
	Damage     Dice   `json:"damage"`
	Kind       string `json:"kind,omitempty"`
	Range      int    `json:"range"`
	IntervalMS int    `json:"interval_ms"`
	Speed      int    `json:"speed,omitempty"`
}

// verifyCombat checks the combat record and the kinds the records name.
func (v *verifier) combat() {
	p := v.p
	if c := p.Combat; c != nil {
		if len(c.Kinds) == 0 {
			v.fail("combat.json: no damage kinds")
		}
		seen := map[string]bool{}
		for _, k := range c.Kinds {
			if k.ID == "" || seen[k.ID] {
				v.fail("combat.json: damage kind %q empty or twice", k.ID)
			}
			seen[k.ID] = true
			if k.Name != "" {
				v.text("combat.json", k.Name)
			}
		}
		if len(c.Kinds) > 64 {
			v.fail("combat.json: %d damage kinds, at most 64", len(c.Kinds))
		}
		for _, s := range sortedKeys(c.Shields) {
			if !timedEffects[s] {
				v.fail("combat.json: shield %q is no timed effect", s)
			}
			v.kind("combat.json shield "+s, c.Shields[s])
		}
	}
	for _, id := range sortedKeys(p.Items) {
		it := p.Items[id]
		if m := it.Missile; m != nil && m.Kind != "" {
			v.kind("item "+id, m.Kind)
		}
		for i, a := range it.Actions {
			owner := fmt.Sprintf("item %s action %d", id, i)
			if a.Kind != ActionFire {
				if a.Ammo != nil || a.Charges != 0 || a.Power != 0 || a.Shot != nil || a.NeedsRoom {
					v.fail("%s: shot fields on a %s action", owner, a.Kind)
				}
				continue
			}
			sources := 0
			for _, b := range []bool{a.Ammo != nil, a.Charges > 0, a.Power > 0} {
				if b {
					sources++
				}
			}
			if sources > 1 {
				v.fail("%s: more than one of ammo, charges and power", owner)
			}
			if (a.Charges > 0 || a.Power > 0) && (a.Shot == nil || a.Shot.Item == "") {
				v.fail("%s: a charge or power shot names no shot item", owner)
			}
			if s := a.Shot; s != nil {
				if s.Item != "" {
					if _, ok := p.Items[s.Item]; !ok {
						v.fail("%s: unknown shot item %q", owner, s.Item)
					}
				}
				if s.Kind != "" {
					v.kind(owner, s.Kind)
				}
			}
		}
	}
	for _, id := range sortedKeys(p.Things) {
		t := p.Things[id]
		for _, list := range [][]string{t.Resists, t.Ignores, t.WeakTo} {
			for _, k := range list {
				v.kind("thing "+id, k)
			}
		}
		if s := t.Shoots; s != nil {
			if _, ok := p.Items[s.Item]; !ok {
				v.fail("thing %s: unknown shot item %q", id, s.Item)
			}
			if s.Kind != "" {
				v.kind("thing "+id, s.Kind)
			}
			if s.Range <= 0 || s.IntervalMS <= 0 {
				v.fail("thing %s: a ranged attack needs a range and an interval", id)
			}
		}
	}
}

// kind checks owner names a damage kind the pack has.
func (v *verifier) kind(owner, id string) {
	if _, ok := v.p.Kind(id); !ok {
		v.fail("%s: unknown damage kind %q", owner, id)
	}
}
