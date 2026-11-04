package sim

import (
	"github.com/sirgwain/the-summoning-go/engine/combat"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// Missile constants (docs/spec/spells.md, Missiles).
const (
	missileRange = 255
	// missileEndFrames are the frames an ended missile shows, 1 to missileEndFrames.
	missileEndFrames = 4
	// bigHit is the damage from which a hit practices 2 rather than 1.
	bigHit = 50
	// Practice of school 1 for opening a door or starting a boulder, a door resisting, and
	// scattering; of school 2 for a bane.
	openPractice    = 5
	resistPractice  = 1
	scatterPractice = 3
	banePractice    = 20
	scatterTries    = 100
)

// SpellFlight is what a missile spell in flight carries.
type SpellFlight struct {
	Missile *content.MissileSpell
	Power   int
	// Sprite and Frame are what it shows now.
	Sprite string
	Frame  int
	// ending counts the frames shown since it ended, 0 while it flies.
	ending int
}

// castMissile flies missile m from caster c with power: four of them around for m.Around.
func (w *World) castMissile(l *Level, c caster, m *content.MissileSpell, power int) {
	if !m.Around {
		w.launchSpell(l, c, c.dir, m, power)
		return
	}
	first := c.dir % 2
	if !c.player {
		first = 0
	}
	for i := range 4 {
		w.launchSpell(l, c, first+2*i, m, power)
	}
}

// flightFrame is the frame of a directional look for flying along direction d: the frames are
// numbered by the facing that launches along d, one eighth of a turn on from it
// (docs/spec/spells.md, Missiles).
func flightFrame(d int) int { return (d + 1) % 8 }

// creatureLaunchExtra is how much further than a thing in flight a creature's spell starts.
const creatureLaunchExtra = 2

// launchSpell sends missile m flying from c along direction d, starting its launch distance ahead
// (docs/spec/spells.md, Missiles). A caster touching something ahead hits it at once. One that
// would start in a blocked cell goes off at once: the player's at the player, a creature's
// vanishing.
func (w *World) launchSpell(l *Level, c caster, d int, m *content.MissileSpell, power int) {
	extra := 0
	if c.thing != nil {
		extra = creatureLaunchExtra
	}
	su, sv := launchStart(c.u, c.v, d, extra)
	f := &Flight{U: su, V: sv, Direction: d, Range: missileRange, Damage: power, ByPlayer: c.player, Thrower: c.thing,
		Spell: &SpellFlight{Missile: m, Power: power, Sprite: m.Sprite}, next: w.step + Steps(flightTicks)}
	if m.Directional {
		f.Spell.Frame = flightFrame(d)
	}
	touched := w.flightBlocker(l, f, c.u+Dirs[d][0]*touchReach, c.v+Dirs[d][1]*touchReach, w.throwerBox(f))
	inWall := touched == nil && l.Blocked(f.Cell())
	if inWall {
		if !c.player || w.player == nil {
			return
		}
		if !m.Opens {
			f.U, f.V = c.u, c.v
			touched = &body{u: c.u, v: c.v, box: w.player.Template.Bounds()}
		}
	}
	f.glide.start(f.U, f.V, w.step, f.next)
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: f.Cell(), Detail: "spell flies"})
	l.Flights = append(l.Flights, f)
	if inWall && touched == nil {
		// An opening spell started in a wall or door meets it there.
		w.spellMeetsWall(l, f, f.Cell())
		w.endSpell(l, f)
		return
	}
	if touched != nil {
		w.spellHits(l, f, touched)
		w.endSpell(l, f)
	}
}

// updateSpell moves spell flight f a step: an ended one shows its next frame and then vanishes;
// one out of range ends; one meeting something hits it.
func (w *World) updateSpell(l *Level, f *Flight) {
	s := f.Spell
	if s.ending > 0 {
		if s.ending++; s.ending > missileEndFrames {
			w.removeFlight(l, f)
			return
		}
		s.Frame = s.ending
		return
	}
	if f.Range--; f.Range <= 0 {
		w.endSpell(l, f)
		return
	}
	speed := flightSpeed
	if f.Direction == 1 || f.Direction == 5 {
		speed /= 2
	}
	du, dv := Dirs[f.Direction][0]*speed, Dirs[f.Direction][1]*speed
	u, v := f.U+du, f.V+dv
	if b := w.flightMeets(l, f, u, v); b != nil {
		w.spellHits(l, f, b)
		w.endSpell(l, f)
		return
	}
	c := cellOf(u, v)
	if l.Blocked(c) {
		w.spellMeetsWall(l, f, c)
		if s.Missile.Bounces {
			w.bounce(l, f, du, dv)
			return
		}
		w.endSpell(l, f)
		return
	}
	from := f.Cell()
	f.U, f.V = u, v
	if to := f.Cell(); to != from {
		if t := l.byCell[to]; t != nil && t.Kind == content.FeatureTeleporter && t.Armed && !t.FlyOver {
			dest := w.jump(l, t)
			f.U, f.V = CellUnits*dest[0]+f.U-CellUnits*to[0], CellUnits*dest[1]+f.V-CellUnits*to[1]
			f.Direction = (f.Direction + t.Turn + 8) % 8
			f.still()
			if s.Missile.Directional {
				s.Frame = flightFrame(f.Direction)
			}
		}
	}
}

// bounce turns flight f back off the wall a step (du, dv) met: a diagonal step that a wall stops
// along one axis only turns that axis back.
func (w *World) bounce(l *Level, f *Flight, du, dv int) {
	d := f.Direction
	blockU, blockV := l.Blocked(cellOf(f.U+du, f.V)), l.Blocked(cellOf(f.U, f.V+dv))
	switch {
	case d%2 == 1 && blockU && !blockV:
		du = -du
	case d%2 == 1 && blockV && !blockU:
		dv = -dv
	default:
		du, dv = -du, -dv
	}
	for i, dir := range Dirs {
		if sign(dir[0]) == sign(du) && sign(dir[1]) == sign(dv) {
			f.Direction = i
			break
		}
	}
	if f.Spell.Missile.Directional {
		f.Spell.Frame = flightFrame(f.Direction)
	}
}

// spellMeetsWall is spell flight f meeting blocked cell c: one that opens opens a door that a
// spell opens, or else says the door or wall resists.
func (w *World) spellMeetsWall(l *Level, f *Flight, c content.Cell) {
	w.soundInSight(l, f.Cell(), f.Spell.Missile.Sound)
	if !f.Spell.Missile.Opens {
		return
	}
	door := l.byCell[c]
	switch {
	case door != nil && door.Kind == content.FeatureDoor && door.OpenedBySpell:
		w.switchFeature(l, door, input.SwitchOn, 0)
		if f.ByPlayer {
			w.practice(SkillSchool+1, openPractice)
		}
	case door != nil && door.Kind == content.FeatureDoor:
		if f.ByPlayer {
			w.message(l, c, content.MsgDoorResists, "")
			w.practice(SkillSchool+1, resistPractice)
		}
	case f.ByPlayer:
		w.message(l, c, content.MsgMissileWall, "")
	}
}

// spellHits is spell flight f hitting what b is: hurting it, opening or starting a boulder,
// scattering it, or a bane's damage; the player's spells practice and turn creatures hostile
// (docs/spec/spells.md, Missiles).
func (w *World) spellHits(l *Level, f *Flight, b *body) {
	s := f.Spell
	m := s.Missile
	w.soundInSight(l, f.Cell(), m.Sound)
	t := b.thing
	if t != nil && f.ByPlayer {
		w.turnHostile(t)
	}
	practice := func(school, n int) {
		if f.ByPlayer && w.player != nil {
			w.practice(SkillSchool+school, n)
		}
	}
	switch {
	case m.Hurts != "":
		k := map[string]combat.Kind{content.HurtsFire: combat.Fire, content.HurtsLightning: combat.Lightning, content.HurtsPoison: combat.Poison}[m.Hurts]
		if t == nil {
			w.hurtPlayer(s.Power, k, "a spell")
		} else {
			w.hurtThing(l, t, s.Power, k)
		}
		if s.Power < bigHit {
			practice(0, 1)
		} else {
			practice(0, 2)
		}
	case m.Opens:
		if t != nil && t.Template.Kind == content.ThingBoulder {
			w.startBoulder(t, f.Direction&^1, false)
			practice(1, openPractice)
		}
	case m.Scatters:
		if t != nil && w.scatter(l, t) {
			practice(1, scatterPractice)
		}
	case m.Bane != "":
		if t != nil && t.Template.ID == m.Bane && alive(t) {
			t.HitPoints = max(0, t.HitPoints-m.BaneDamage)
			w.emit(Event{Kind: EventThing, Level: l.ID, Cell: t.Cell(), Thing: t.ID, Detail: "hurt", Damage: m.BaneDamage})
			if t.HitPoints == 0 {
				w.creatureDies(l, t)
			}
			practice(2, banePractice)
		}
	}
}

// scatter moves thing t to a random floor cell the player has seen, and reports whether it did.
// Boulders, traps, conjured things and walking creatures scatter; others stay.
func (w *World) scatter(l *Level, t *Thing) bool {
	switch {
	case t.Template.Kind == content.ThingCreature && (t.Template.Has(content.BehaviorAnimated) || t.Template.Has(content.BehaviorStationary)),
		t.Template.Kind == content.ThingPortal:
		return false
	}
	rows, cols := l.Def.Size[0], l.Def.Size[1]
	if rows < 3 || cols < 3 {
		return false
	}
	for range scatterTries {
		c := content.Cell{1 + w.rng.IntN(rows-2), 1 + w.rng.IntN(cols-2)}
		if l.Blocked(c) || !l.Seen(c) {
			continue
		}
		u, v := cellCenter(c)
		if w.blocker(l, t, false, u, v, t.Template.Bounds(), nil) != nil {
			continue
		}
		w.moveThing(l, t, c, 0)
		w.emit(Event{Kind: EventThing, Level: l.ID, Cell: c, Thing: t.ID, Detail: "scattered"})
		return true
	}
	return false
}

// endSpell ends spell flight f where it is: it shows its ending frames, as its cloud if it has
// one, and then vanishes.
func (w *World) endSpell(l *Level, f *Flight) {
	s := f.Spell
	if s.ending > 0 {
		return
	}
	s.ending, s.Frame = 1, 1
	if s.Missile.Cloud != "" {
		s.Sprite = s.Missile.Cloud
	}
	w.emit(Event{Kind: EventThing, Level: l.ID, Cell: f.Cell(), Detail: "spell ends"})
}
