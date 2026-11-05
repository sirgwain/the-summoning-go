package content

import (
	"errors"
	"fmt"
	"slices"
)

// Verify checks that every reference in the pack resolves and every record is consistent.
func (p *Pack) Verify() error {
	v := &verifier{p: p}
	v.manifest()
	v.combat()
	v.skin()
	for _, id := range sortedKeys(p.Palettes) {
		v.errs = append(v.errs, p.Palettes[id].verify()...)
	}
	for _, id := range sortedKeys(p.Sprites) {
		v.sprite(p.Sprites[id])
	}
	for _, id := range sortedKeys(p.Floors) {
		f := p.Floors[id]
		if n := v.frames("floor set "+id, f.Sprite); n >= 0 && f.Variants > n {
			v.fail("floor set %s: %d variants, sprite bank has %d frames", id, f.Variants, n)
		}
		switch f.Shape {
		case "", FloorBox:
		case FloorDiamond:
			if b, ok := p.Sprites[f.Sprite]; ok && b.FrameSize != f.FrameSize() {
				v.fail("floor set %s: %v frames, a diamond set wants %v", id, b.FrameSize, f.FrameSize())
			}
		default:
			v.fail("floor set %s: unknown shape %q", id, f.Shape)
		}
	}
	for _, id := range sortedKeys(p.Walls) {
		w := p.Walls[id]
		n := v.frames("wall set "+id, w.Sprite)
		for s, shape := range w.Shapes {
			if shape != nil && (shape.Frame < 0 || (n >= 0 && shape.Frame >= n)) {
				v.fail("wall set %s: shape %d uses frame %d of %d", id, s, shape.Frame, n)
			}
			if shape != nil && shape.Sight != nil && (len(shape.Sight) != 2 || !direction(shape.Sight[0]) || !direction(shape.Sight[1])) {
				v.fail("wall set %s: shape %d has sight %v", id, s, shape.Sight)
			}
			for _, u := range shapeUnseen(shape) {
				if u[0] < 0 || u[0] >= 8 || w.Shape(u[1]) == nil {
					v.fail("wall set %s: shape %d takes %v when unseen", id, s, u)
				}
			}
			if shape == nil || shape.Diagonal == "" {
				continue
			}
			if _, _, ok := CornerSigns(shape.Diagonal); !ok {
				v.fail("wall set %s: shape %d fills unknown corner %q", id, s, shape.Diagonal)
			}
		}
	}
	for _, id := range sortedKeys(p.Looks) {
		v.look(p.Looks[id])
	}
	for _, id := range sortedKeys(p.Items) {
		v.itemRecord(p.Items[id])
	}
	for _, id := range sortedKeys(p.Things) {
		v.thing(p.Things[id])
	}
	for _, id := range sortedKeys(p.Portraits) {
		v.portrait(p.Portraits[id])
	}
	if m := p.Messages; m != nil {
		for _, key := range sortedKeys(m.Messages) {
			if !messageKeys[key] {
				v.fail("messages.json: unknown message %q", key)
			}
			v.text("messages.json", m.Messages[key])
		}
	}
	for _, id := range sortedKeys(p.Levels) {
		v.level(p.Levels[id])
	}
	v.magic()
	v.keywords()
	v.pointers()
	v.inventory()
	v.creation()
	for _, id := range sortedKeys(p.Scenes) {
		v.scene(p.Scenes[id])
	}
	for _, id := range sortedKeys(p.Flipbooks) {
		v.flipbook(p.Flipbooks[id])
	}
	for _, id := range sortedKeys(p.Sounds) {
		v.sound(p.Sounds[id])
	}
	v.cues()
	v.effectCues()
	v.transitions()
	for _, id := range sortedKeys(p.Spells) {
		v.spell(p.Spells[id])
	}
	for _, id := range sortedKeys(p.Conversations) {
		v.conversation(p.Conversations[id])
	}
	return errors.Join(v.errs...)
}

type verifier struct {
	p    *Pack
	errs []error
}

func (v *verifier) fail(format string, args ...any) {
	v.errs = append(v.errs, fmt.Errorf(format, args...))
}

func (v *verifier) manifest() {
	m := v.p.Manifest
	if _, ok := v.p.Palettes[m.Palette]; !ok {
		v.fail("pack.json: palette %q not found", m.Palette)
	}
	v.music("pack.json: start", m.Start.Music)
	if l, ok := v.p.Levels[m.Start.Level]; !ok {
		v.fail("pack.json: start level %q not found", m.Start.Level)
	} else if !l.InBounds(m.Start.Cell) {
		v.fail("pack.json: start cell %v outside %s", m.Start.Cell, l.ID)
	}
	if m.Rules != "" && m.Rules != RulesSummoning {
		v.fail("pack.json: unknown rules %q", m.Rules)
	}
	if c := m.Counters; c != nil && c.TransformedLook != "" {
		v.frames("pack.json transformed look", c.TransformedLook)
	}
	if m.Player != "" {
		if t, ok := v.p.Things[m.Player]; !ok {
			v.fail("pack.json: player %q not found", m.Player)
		} else if t.Kind != ThingPlayer {
			v.fail("pack.json: player %q is a %s, not a player", m.Player, t.Kind)
		}
	}
	if m.Menu != "" {
		if t := v.thingRef("pack.json menu", m.Menu); t != nil && t.Conversation == "" {
			v.fail("pack.json: menu %q holds no conversation", m.Menu)
		}
	}
}

// frames returns the number of frames of sprite, or -1 after reporting it missing.
func (v *verifier) frames(owner, sprite string) int {
	b, ok := v.p.Sprites[sprite]
	if !ok {
		v.fail("%s: sprite bank %q not found", owner, sprite)
		return -1
	}
	return b.Frames
}

// figureLook checks that a figure look's sprites exist.
func (v *verifier) figureLook(owner string, l FigureLook) {
	v.frames(owner, l.Sprite)
	if l.Female != "" {
		v.frames(owner+" female", l.Female)
	}
}

// frame checks that sprite has frame i.
func (v *verifier) frame(owner, sprite string, i int) {
	if n := v.frames(owner, sprite); n >= 0 && (i < 0 || i >= n) {
		v.fail("%s: frame %d of %d in %s", owner, i, n, sprite)
	}
}

// text checks that key is a string in every language.
func (v *verifier) text(owner, key string) {
	for _, id := range sortedKeys(v.p.Texts) {
		if _, ok := v.p.Texts[id].Strings[key]; !ok {
			v.fail("%s: text %q not in %s", owner, key, id)
		}
	}
}

func (v *verifier) item(owner, id string) {
	if _, ok := v.p.Items[id]; !ok {
		v.fail("%s: item %q not found", owner, id)
	}
}

func (v *verifier) sprite(b *SpriteBank) {
	id := b.ID
	if b.FrameSize[0] <= 0 || b.FrameSize[1] <= 0 || b.Columns <= 0 || b.Frames < 0 {
		v.fail("sprite bank %s: bad layout", id)
		return
	}
	if b.Transparent != nil && (*b.Transparent < 0 || *b.Transparent > 255) {
		v.fail("sprite bank %s: transparent index %d", id, *b.Transparent)
	}
	if b.Frames > 0 && !b.FrameRect(b.Frames-1).In(b.Sheet.Bounds()) {
		v.fail("sprite bank %s: %d frames do not fit the %v sheet", id, b.Frames, b.Sheet.Bounds().Size())
	}
	if b.Anchors != nil && len(b.Anchors) != b.Frames {
		v.fail("sprite bank %s: %d anchors for %d frames", id, len(b.Anchors), b.Frames)
	}
	v.animations("sprite bank "+id, b.Animations, b.Frames)
}

// animations checks animations over a sprite of n frames.
func (v *verifier) animations(owner string, anims map[string]Animation, n int) {
	for _, name := range sortedKeys(anims) {
		a := anims[name]
		if len(a.Frames) == 0 {
			v.fail("%s: animation %s has no frames", owner, name)
		}
		for _, f := range a.Frames {
			if f < 0 || f >= n {
				v.fail("%s: animation %s uses frame %d of %d", owner, name, f, n)
			}
		}
		if a.MS.Min < 0 || a.MS.Max < a.MS.Min || (len(a.Frames) > 1 && a.MS.Max == 0) {
			v.fail("%s: animation %s: bad duration %v", owner, name, a.MS)
		}
		switch a.Mode {
		case "", AnimLoop, AnimOnce, AnimRandom, AnimFlicker:
		default:
			v.fail("%s: animation %s: unknown mode %q", owner, name, a.Mode)
		}
	}
}

func (v *verifier) look(l *Look) {
	switch l.Layer {
	case LayerFloor, LayerStand, LayerWall:
	default:
		v.fail("look %s: unknown layer %q", l.ID, l.Layer)
	}
	for _, state := range sortedKeys(l.States) {
		for _, part := range l.States[state] {
			v.frame(fmt.Sprintf("look %s state %s", l.ID, state), part.Sprite, part.Frame)
		}
	}
}

func (v *verifier) itemRecord(i *Item) {
	owner := "item " + i.ID
	v.text(owner, i.Name)
	for _, key := range []string{i.Effect, i.Lore} {
		if key != "" {
			v.text(owner, key)
		}
	}
	if i.Weight < 0 || i.Size < 0 {
		v.fail("%s: negative weight or size", owner)
	}
	if c := i.Container; c != nil && (c.Capacity <= 0 || c.Largest < 0) {
		v.fail("%s: bad container %+v", owner, *c)
	}
	if i.Wear != "" && !wears[i.Wear] {
		v.fail("%s: unknown wear slot %q", owner, i.Wear)
	}
	if c := i.WeaponClass; c != nil && (*c < 0 || *c > 3) {
		v.fail("%s: weapon class %d", owner, *c)
	}
	for hand := range 2 {
		if l := i.Held.Hand(hand); l != nil {
			for _, s := range []string{l.Behind, l.Front} {
				if s != "" {
					v.frames(owner+" held", s)
				}
			}
		}
	}
	for _, ref := range []string{i.Spent, i.BreaksInto} {
		if ref != "" {
			v.item(owner, ref)
		}
	}
	if len(i.Actions) > 2 {
		v.fail("%s: %d actions, at most 2", owner, len(i.Actions))
	}
	for n, a := range i.Actions {
		if !actionKinds[a.Kind] || a.Delay < 0 || a.Skill < 0 {
			v.fail("%s: action %d: bad kind %q, delay or skill", owner, n, a.Kind)
		}
		v.effect(fmt.Sprintf("%s: action %d sound", owner, n), a.Sound)
	}
	v.frame(owner+" icon", i.Icon.Sprite, i.Icon.Frame)
	v.frame(owner+" floor", i.Floor.Sprite, i.Floor.Frame)
	v.itemFight(owner, i)
	if i.DollLeft != nil {
		v.frame(owner+" left doll", i.DollLeft.Image.Sprite, i.DollLeft.Image.Frame)
	}
	if i.Doll != nil {
		v.frame(owner+" doll", i.Doll.Image.Sprite, i.Doll.Image.Frame)
	}
	for name, l := range i.Looks {
		v.figureLook(owner+" look "+name, l)
	}
	if f := i.Fill; f != nil {
		if f.Per <= 0 || i.Container == nil {
			v.fail("%s: fill needs a container and per above 0", owner)
		}
		for n, im := range f.Icons {
			v.frame(owner+" fill icon "+itoa(n+1), im.Sprite, im.Frame)
		}
		for n, d := range f.Dolls {
			v.frame(owner+" fill doll "+itoa(n+1), d.Image.Sprite, d.Image.Frame)
		}
	}
	for n, a := range i.Actions {
		if a.Icon != nil {
			v.frame(owner+" action "+itoa(n)+" icon", a.Icon.Sprite, a.Icon.Frame)
		}
	}
	if i.Casts != nil {
		v.spellRef(owner, i.Casts.Spell)
	}
	if i.Conversation != "" {
		if t, ok := v.p.Things[i.Conversation]; !ok {
			v.fail("%s: conversation thing %q not found", owner, i.Conversation)
		} else if t.Conversation == "" {
			v.fail("%s: thing %s holds no conversation", owner, i.Conversation)
		}
	}
	if i.Casts != nil && i.Conversation != "" {
		v.fail("%s: both casts and starts a conversation", owner)
	}
	v.icons(owner, i.Teaches)
	if i.Picture != nil {
		v.frame(owner+" picture", i.Picture.Sprite, i.Picture.Frame)
	}
}

func (v *verifier) thing(t *Thing) {
	owner := "thing " + t.ID
	v.music(owner+": theme", t.Theme)
	if t.Lore != "" {
		v.text(owner, t.Lore)
	}
	switch t.Kind {
	case ThingCreature, ThingBoulder, ThingTrap, ThingPlayer, ThingConjured, ThingPortal:
	default:
		v.fail("%s: unknown kind %q", owner, t.Kind)
	}
	if n := v.frames(owner, t.Sprite); n >= 0 {
		v.animations(owner, t.Animations, n)
	}
	if t.Facing < 0 || t.Facing > 7 {
		v.fail("%s: facing %d", owner, t.Facing)
	}
	if t.Speed < 0 || t.IntervalMS < 0 || t.Weight < 0 || t.Footprint < 0 {
		v.fail("%s: negative speed, interval, weight or footprint", owner)
	}
	if b := t.Box; b != nil && (b[2] <= 0 || b[3] <= 0) {
		v.fail("%s: box %v has no area", owner, *b)
	}
	for _, l := range t.Figure {
		v.figureLook(owner+" figure "+l.Name, l.FigureLook)
	}
	for _, io := range t.Inventory {
		if t.Kind != ThingPlayer || io.Slot < 0 || io.Slot >= 28 {
			v.fail("%s: inventory slot %d", owner, io.Slot)
		}
		v.item(owner+" inventory", io.Item)
	}
	for _, b := range t.Behaviors {
		if !behaviors[b] {
			v.fail("%s: unknown behavior %q", owner, b)
		}
	}
	for _, pt := range []string{t.Portrait, t.TalkPortrait} {
		if _, ok := v.p.Portraits[pt]; pt != "" && !ok {
			v.fail("%s: portrait %q not found", owner, pt)
		}
	}
	if t.Spell != "" {
		v.spellRef(owner, t.Spell)
	}
	if t.Conversation != "" {
		v.conversationRef(owner, t.Conversation)
	}
	v.icons(owner, t.Icons)
	v.thingFight(owner, t)
}

func (v *verifier) portrait(pt *Portrait) {
	owner := "portrait " + pt.ID
	v.frame(owner, pt.Image.Sprite, pt.Image.Frame)
	for _, ref := range []string{pt.Other, pt.Talk} {
		if _, ok := v.p.Portraits[ref]; ref != "" && !ok {
			v.fail("%s: portrait %q not found", owner, ref)
		}
	}
	if len(pt.FigureColors) > 256-FigureColor {
		v.fail("%s: %d figure colors", owner, len(pt.FigureColors))
	}
	for _, c := range pt.FigureColors {
		if c < 0 || c > 255 {
			v.fail("%s: figure color %d", owner, c)
		}
	}
}

func (v *verifier) level(l *Level) {
	if l.Footsteps != "" && v.p.Sounds[l.Footsteps] == nil {
		v.fail("level %s: footsteps %q is not a sound", l.ID, l.Footsteps)
	}
	if c := l.TeleporterConfusion; c != nil && *c < 0 {
		v.fail("level %s: teleporter confusion %d", l.ID, *c)
	}
	lv := &levelVerifier{verifier: v, l: l}
	lv.verify()
}

type levelVerifier struct {
	*verifier
	l        *Level
	floors   *FloorSet
	walls    *WallSet
	features map[string]*Feature
	things   map[string]*PlacedThing
}

func (lv *levelVerifier) fail(format string, args ...any) {
	lv.verifier.fail("level %s: "+format, append([]any{lv.l.ID}, args...)...)
}

func (lv *levelVerifier) owner() string { return "level " + lv.l.ID }

func (lv *levelVerifier) verify() {
	l := lv.l
	if l.Name != "" {
		lv.text(lv.owner(), l.Name)
	}
	if m := l.Map; m != nil {
		switch {
		case (m.Message == "") == (m.Conversation == ""):
			lv.fail("map refusal: a message or a conversation")
		case m.Message != "":
			lv.text(lv.owner(), m.Message)
		default:
			if t := lv.thingRef(lv.owner(), m.Conversation); t != nil && t.Conversation == "" {
				lv.fail("map refusal: %s holds no conversation", m.Conversation)
			}
		}
	}
	var ok bool
	if lv.floors, ok = lv.p.Floors[l.Floors]; !ok {
		lv.fail("floor set %q not found", l.Floors)
	}
	if lv.walls, ok = lv.p.Walls[l.Walls]; !ok {
		lv.fail("wall set %q not found", l.Walls)
	}
	for i, ft := range l.FloorTypes {
		if lv.floors != nil && (ft.Variant < 0 || ft.Variant >= lv.floors.Variants) {
			lv.fail("floor type %d: variant %d out of range", i, ft.Variant)
		}
		if ft.Depth < 0 || ft.Depth > 4 {
			lv.fail("floor type %d: depth %d", i, ft.Depth)
		}
		for _, e := range []*Effect{ft.Enter, ft.Tick} {
			if e != nil {
				lv.cue(fmt.Sprintf("level %s: floor type %d", l.ID, i), e.Cue)
			}
		}
	}

	lv.grid("floor", l.Floor, lv.floorType)
	lv.grid("wall", l.Wall, lv.wallShape)
	if l.FloorHalves != nil {
		lv.grid("floor_halves", l.FloorHalves, func(v int) bool { return v <= HalvesBoth })
	}
	for name, g := range map[string]Grid{"upper_sight": l.UpperSight, "lower_sight": l.LowerSight} {
		if g != nil {
			lv.grid(name, g, func(v int) bool { return v < 8 })
		}
	}
	lv.doorSight(l)

	for _, layer := range []struct {
		kind string
		list []Placed
	}{{"overlay", l.Overlays}, {"decoration", l.Decorations}, {"prop", l.Props}} {
		for _, pl := range layer.list {
			if !l.InBounds(pl.Cell) {
				lv.fail("%s at %v outside the level", layer.kind, pl.Cell)
			}
			lv.frame(fmt.Sprintf("%s %s at %v", lv.owner(), layer.kind, pl.Cell), pl.Sprite, pl.Frame)
		}
	}

	lv.features = map[string]*Feature{}
	cells := map[Cell]string{}
	for _, f := range l.Features {
		if f.ID == "" {
			lv.fail("feature at %v has no id", f.Cell)
		} else if _, dup := lv.features[f.ID]; dup {
			lv.fail("duplicate feature %s", f.ID)
		}
		lv.features[f.ID] = f
		if other, dup := cells[f.Cell]; dup {
			lv.fail("features %s and %s share cell %v", other, f.ID, f.Cell)
		}
		cells[f.Cell] = f.ID
	}
	lv.things = map[string]*PlacedThing{}
	for _, t := range l.Things {
		if t.ID == "" {
			lv.fail("thing at %v has no id", t.Cell)
		} else if _, dup := lv.things[t.ID]; dup {
			lv.fail("duplicate thing %s", t.ID)
		}
		lv.things[t.ID] = t
	}

	for _, f := range l.Features {
		lv.feature(f)
	}
	for _, name := range sortedKeys(l.Procedures) {
		for i, a := range l.Procedures[name] {
			lv.action(fmt.Sprintf("procedure %s action %d", name, i), a)
		}
	}
	for _, t := range l.Things {
		lv.placedThing(t)
	}
	for _, o := range l.Objects {
		if !l.InBounds(o.Cell) {
			lv.fail("object at %v outside the level", o.Cell)
		}
		lv.object(fmt.Sprintf("object at %v", o.Cell), o.Object)
	}
}

func (lv *levelVerifier) floorType(v int) bool { return v < len(lv.l.FloorTypes) }

func (lv *levelVerifier) wallShape(v int) bool { return lv.walls == nil || lv.walls.Shape(v) != nil }

func (lv *levelVerifier) grid(name string, g Grid, valid func(v int) bool) {
	l := lv.l
	if len(g) != l.Size[0] {
		lv.fail("%s has %d rows, want %d", name, len(g), l.Size[0])
		return
	}
	for x, row := range g {
		if len(row) != l.Size[1] {
			lv.fail("%s row %d has %d cells, want %d", name, x, len(row), l.Size[1])
			continue
		}
		for y, v := range row {
			if v != NoFrame && !valid(v) {
				lv.fail("%s (%d, %d): %d out of range", name, x, y, v)
			}
		}
	}
}

// cell checks that c is inside the level.
func (lv *levelVerifier) cell(what string, c Cell) {
	if !lv.l.InBounds(c) {
		lv.fail("%s: cell %v outside the level", what, c)
	}
}

func (lv *levelVerifier) feature(f *Feature) {
	what := "feature " + f.ID
	if _, ok := featureHooks[f.Kind]; !ok {
		lv.fail("%s: unknown kind %q", what, f.Kind)
		return
	}
	lv.cell(what, f.Cell)

	if f.Look != "" {
		look, ok := lv.p.Looks[f.Look]
		if !ok {
			lv.fail("%s: look %q not found", what, f.Look)
		} else {
			for _, s := range f.lookStates() {
				if _, ok := look.States[s]; !ok {
					lv.fail("%s: look %s has no state %q", what, f.Look, s)
				}
			}
		}
	}

	for _, h := range featureHooks[f.Kind] {
		if p := *f.Hook(h); p != "" {
			lv.procedure(what, p)
		}
	}
	for _, h := range []struct {
		name, value string
	}{{HookTurnOn, f.OnTurnOn}, {HookTurnOff, f.OnTurnOff}, {HookPress, f.OnPress}, {HookRelease, f.OnRelease}, {HookInsert, f.OnInsert}, {HookComplete, f.OnComplete}} {
		if h.value != "" && !HasHook(f.Kind, h.name) {
			lv.fail("%s: a %s has no hook %s", what, f.Kind, h.name)
		}
	}

	lv.flipbookRef(lv.owner()+" "+what+" sparkle", f.Sparkle)
	lv.flipbookRef(lv.owner()+" "+what+" effect", f.Effect)
	switch f.Kind {
	case FeatureDoor:
		switch f.State {
		case DoorClosed, DoorOpen, DoorOpening, DoorClosing:
		default:
			lv.fail("%s: unknown door state %q", what, f.State)
		}
		if f.SealedBy != "" {
			lv.item(lv.owner()+" "+what, f.SealedBy)
		}
	case FeatureSlot:
		lv.item(lv.owner()+" "+what, f.Accepts)
	case FeatureTeleporter:
		if len(f.Destinations) != 2 {
			lv.fail("%s: %d destinations, want 2", what, len(f.Destinations))
		}
		for _, c := range f.Destinations {
			lv.cell(what+" destination", c)
		}
		if f.Next != 0 && f.Next != 1 {
			lv.fail("%s: next %d", what, f.Next)
		}
		if f.Turn < 0 || f.Turn > 7 {
			lv.fail("%s: turn %d", what, f.Turn)
		}
	case FeatureExit, FeaturePit, FeatureTrapdoor:
		if f.To == nil {
			lv.fail("%s: no destination", what)
		} else if dest, ok := lv.p.Levels[f.To.Level]; !ok {
			lv.fail("%s: level %q not found", what, f.To.Level)
		} else if !dest.InBounds(f.To.Cell) {
			lv.fail("%s: destination %v outside %s", what, f.To.Cell, dest.ID)
		}
	case FeatureSocket:
		if len(f.Places) == 0 {
			lv.fail("%s: no places", what)
		}
		for _, it := range f.Places {
			lv.item(lv.owner()+" "+what, it)
		}
		for _, i := range f.Set {
			if i < 0 || i >= len(f.Places) {
				lv.fail("%s: set place %d of %d", what, i, len(f.Places))
			}
		}
	case FeatureMouth:
		lv.text(lv.owner()+" "+what, f.Message)
	}
}

func (lv *levelVerifier) procedure(what, name string) {
	if _, ok := lv.l.Procedures[name]; !ok {
		lv.fail("%s: procedure %q not found", what, name)
	}
}

// target returns feature id, reporting it if missing or not one of kinds.
func (lv *levelVerifier) target(what, id string, kinds ...string) *Feature {
	f, ok := lv.features[id]
	switch {
	case !ok:
		lv.fail("%s: feature %q not found", what, id)
		return nil
	case len(kinds) > 0 && !slices.Contains(kinds, f.Kind):
		lv.fail("%s: feature %s is a %s, want %v", what, id, f.Kind, kinds)
		return nil
	}
	return f
}

func (lv *levelVerifier) direction(what string, d int) {
	if d < 0 || d > 7 {
		lv.fail("%s: direction %d", what, d)
	}
}

func (lv *levelVerifier) action(what string, a Action) {
	if !lv.verifier.actionKind(lv.owner()+" "+what, a, levelOps) {
		return
	}
	owner := lv.owner() + " " + what
	switch {
	case a.On != nil:
		lv.target(what, *a.On)
	case a.Off != nil:
		lv.target(what, *a.Off)
	case a.Toggle != nil:
		lv.target(what, *a.Toggle)
	case a.SetHook != nil:
		if f := lv.target(what, a.SetHook.Feature); f != nil && !HasHook(f.Kind, a.SetHook.Hook) {
			lv.fail("%s: a %s has no hook %s", what, f.Kind, a.SetHook.Hook)
		}
		if a.SetHook.Procedure != nil {
			lv.procedure(what, *a.SetHook.Procedure)
		}
	case a.SetThreshold != nil:
		lv.target(what, a.SetThreshold.Feature, FeaturePlate)
	case a.SetAccepts != nil:
		lv.target(what, a.SetAccepts.Feature, FeatureSlot)
		lv.item(owner, a.SetAccepts.Item)
	case a.SetMessage != nil:
		lv.target(what, a.SetMessage.Feature, FeatureMouth)
		lv.text(owner, a.SetMessage.Message)
	case a.Roll != nil:
		lv.boulder(what, a.Roll.Thing)
		lv.direction(what, a.Roll.Direction)
	case a.Stop != nil:
		lv.boulder(what, *a.Stop)
	case a.ShootItem != nil:
		lv.cell(what, a.ShootItem.Cell)
		lv.item(owner, a.ShootItem.Item)
		lv.direction(what, a.ShootItem.Direction)
	case a.ShootSpell != nil:
		lv.cell(what, a.ShootSpell.Cell)
		lv.direction(what, a.ShootSpell.Direction)
		lv.spellRef(owner, a.ShootSpell.Spell)
	case a.Create != nil:
		lv.cell(what, a.Create.Cell)
		lv.item(owner, a.Create.Item)
	case a.SetFloor != nil:
		lv.cell(what, a.SetFloor.Cell)
		if a.SetFloor.Type < 0 || !lv.floorType(a.SetFloor.Type) {
			lv.fail("%s: floor type %d out of range", what, a.SetFloor.Type)
		}
	case a.SetWall != nil:
		lv.cell(what, a.SetWall.Cell)
		if !lv.wallShape(a.SetWall.Shape) {
			lv.fail("%s: wall shape %d not in the wall set", what, a.SetWall.Shape)
		}
	}
}

func (lv *levelVerifier) boulder(what, id string) {
	t, ok := lv.things[id]
	if !ok {
		lv.fail("%s: thing %q not found", what, id)
		return
	}
	if tmpl, ok := lv.p.Things[t.Thing]; ok && tmpl.Kind != ThingBoulder {
		lv.fail("%s: thing %s is not a boulder", what, id)
	}
}

func (lv *levelVerifier) placedThing(t *PlacedThing) {
	what := "thing " + t.ID
	lv.cell(what, t.Cell)
	tmpl, ok := lv.p.Things[t.Thing]
	if !ok {
		lv.fail("%s: template %q not found", what, t.Thing)
		return
	}
	if t.Facing != nil {
		lv.direction(what, *t.Facing)
	}
	if t.Rolling != nil {
		if tmpl.Kind != ThingBoulder {
			lv.fail("%s: only boulders roll", what)
		}
		lv.direction(what, t.Rolling.Direction)
	}
	if tmpl.Kind == ThingPlayer {
		lv.fail("%s: a level does not place the player", what)
	}
	if len(t.Items) > 0 && tmpl.Kind != ThingCreature {
		lv.fail("%s: only creatures carry items", what)
	}
	for _, o := range t.Items {
		lv.object(what, o)
	}
}

func (lv *levelVerifier) object(what string, o Object) {
	if o.Item == RandomItem {
		if len(o.Contents) > 0 {
			lv.fail("%s: a random object has contents", what)
		}
		return
	}
	it, ok := lv.p.Items[o.Item]
	if !ok {
		lv.fail("%s: item %q not found", what, o.Item)
		return
	}
	if len(o.Contents) > 0 {
		if it.Container == nil {
			lv.fail("%s: %s is not a container", what, o.Item)
		} else if len(o.Contents) > it.Container.Capacity {
			lv.fail("%s: %d objects in %s, capacity %d", what, len(o.Contents), o.Item, it.Container.Capacity)
		}
	}
	for _, c := range o.Contents {
		lv.object(what+" in "+o.Item, c)
	}
}

// doorSight checks a level's door_sight and no_sight: each cell in the level, its doors door
// features, and a table of shapes for every combination of them closed.
func (lv *levelVerifier) doorSight(l *Level) {
	doors := map[Cell]bool{}
	for _, f := range l.Features {
		if f.Kind == FeatureDoor {
			doors[f.Cell] = true
		}
	}
	for _, ds := range l.DoorSight {
		if !l.InBounds(ds.Cell) {
			lv.fail("door_sight at %v outside the level", ds.Cell)
		}
		if len(ds.Doors) == 0 || len(ds.Doors) > 4 {
			lv.fail("door_sight at %v: %d doors", ds.Cell, len(ds.Doors))
			continue
		}
		for _, d := range ds.Doors {
			if !doors[d] {
				lv.fail("door_sight at %v: no door at %v", ds.Cell, d)
			}
		}
		if len(ds.As) != 1<<len(ds.Doors) {
			lv.fail("door_sight at %v: %d tables for %d doors", ds.Cell, len(ds.As), len(ds.Doors))
		}
		for _, as := range ds.As {
			if lv.walls != nil && len(as) != len(lv.walls.Shapes) {
				lv.fail("door_sight at %v: %d shapes for a set of %d", ds.Cell, len(as), len(lv.walls.Shapes))
				continue
			}
			for _, s := range as {
				if !lv.wallShape(s) {
					lv.fail("door_sight at %v: no shape %d", ds.Cell, s)
				}
			}
		}
	}
	for _, c := range l.NoSight {
		if !l.InBounds(c) {
			lv.fail("no_sight at %v outside the level", c)
		}
	}
}

// direction reports whether d is a direction or NoFrame.
func direction(d int) bool { return d >= NoFrame && d < 8 }

// shapeUnseen returns shape s's unseen pairs, none for no shape.
func shapeUnseen(s *WallShape) [][2]int {
	if s == nil {
		return nil
	}
	return s.Unseen
}
