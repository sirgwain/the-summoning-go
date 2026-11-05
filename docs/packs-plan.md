# Packs plan

Status: 2026-11-05. Phases 1 (labels), 2 (skins) and 3 (ranged combat) have landed. This plan covers what the engine needs so that a pack other than
The Summoning can look and play like its own game. A second, original pack built
without engine changes shows the gaps the engine has: fantasy labels, the
iron and slate frames, combat tied to bows and quivers, and art capped at the Summoning's pixel scale.

Each phase changes [design.md](design.md) and [pack.md](pack.md) first, and [spec/](spec/) where
a Summoning rule becomes the default of a general one, then the code. Update this file as each
phase lands, fold what's settled into those documents, and delete it once the phases are done.

## Principles

- **The Summoning is the default.** Every new field is optional, and a pack that leaves it out
  plays exactly as now. The importer's output does not change, except where a phase moves a
  Summoning rule out of the engine and into the importer's pack as data.
- **Data, not code.** A new game is a pack, not a fork of the engine. Where a rule set has to
  differ, it goes behind `combat.Rules` or a like interface. Packs never get hooks into the engine.
- **The simulation keeps its units.** 32 world units a cell and 60 steps a second stay fixed.
  Resolution, skins and words are presentation. `engine/sim` still does not import Ebitengine.
- **Format bumps.** A phase that adds anything an older pack would read wrongly raises
  `format` in `pack.json` (pack.md, pack.json).

## Phases

### 1. Words: engine labels come from the pack

The viewer and the book print fixed English: "Hit points" and "Magic points"
(`engine/viewer/frontend.go`, `engine/inspect/book.go`), "Choose your school of magic", "Memorize"
(`engine/viewer/window/spells.go`), the help screen, the inventory's stats, the hotbar's tips,
the options menu.

- [x] Specify engine labels in pack.md (Labels): each label a text key with an English default kept
      by the engine (`inspect.Labels`), overridden by the pack's text, named by role.
- [x] Move every fixed label in `engine/viewer`, `engine/viewer/window` and `engine/inspect` to them,
      the splash's lettering and the help's play keys included. `TestLabelsRenamed` walks the panels
      with every label renamed and finds no default left; `TestLabelsInSource` finds no default
      among the panels' strings; `TestLabelsDocumented` keeps pack.md's table whole. The debugging
      tools (inspector, tools' help, full book) keep their own words.

### 2. Skins: frames, colors and font from the pack

Panel frames are drawn by code in `engine/frame`, with two styles (`Iron`, `Runic`) assigned by
the viewer. The text color and gauge colors are constants, and the font is the engine's own
(design.md, UI).

- [x] Specify a `ui.json` record in pack.md (Skin): a frame style for each framed panel (`hotbar`,
      `inventory`, `spells`, `menu`, `notice`), built-in or a pack image cut in nine; named colors
      (text, slots, buttons, gems, gauges, scene words); a pack font; the splash artwork. The
      conversation, book, log and map have no frame today and keep their own colors; skinning
      them is left open.
- [x] Give `engine/frame` a style table (`frame.Names`) and a third built-in style, `hull`:
      riveted hull plate with amber indicator lights.
- [x] Have the viewer take every frame, the named colors, the letters and the splash from the
      skin (`window/skin.go`). A pack font keeps the engine's letter cells and spacing, which the
      layout is made to; letters of other sizes wait for phase 4's UI at density.
      splash artwork waits for phase 4's true-color sprites.

### 3. Ranged combat beyond bows

The `fire` action draws an arrow from a quiver worn in slot 25, flies `5 × STR` steps, does
damage kind `Arrow`, and refuses with "Not enough room to draw bow" within a step of a thing
(spec/items.md, Throwing; spec/combat.md). Damage kinds are a fixed list in `engine/combat`
(`Physical`, `Fire`, `Lightning`, `Arrow`, `Poison`), and creature defenses are fixed flags
matched to them.

- [x] **Ammunition by tag.** A `fire` action takes `ammo` (`{tag, slot}`), `charges` or `power`
      (pack.md, Items); without any, the quiver's arrows. The importer writes the bow's quiver
      rule as `ammo`. Charge and power shots fly an object of `shot.item` that vanishes.
- [x] **The shot on the weapon.** `shot: {item, damage, kind, range, speed}`, `range` as
      `{base, attribute, times, max}`, each falling back to the ammunition's `missile`.
- [x] **Room to fire** is `needs_room`, set by the importer on bows.
- [x] **Damage kinds as data.** `combat.json` lists the kinds and the shields (pack.md, Combat);
      `combat.Kind` is an index into it and the rule set gets `combat.Kinds` sets. Creatures take
      `resists`, `ignores`, `weak_to`; the old `resists_*` and `physical_only` behaviors stand for
      them, so saves and the Summoning's book are unchanged. The importer writes the five kinds,
      `kind: arrow` on arrows' missiles and `weak_to: [arrow]` on bow-vulnerable creatures.
- [x] **Creature ranged attacks.** `shoots: {item, damage, kind, range, interval_ms, speed}`: a
      hostile creature shoots the player in reach along one of the eight directions.
- [ ] Later, if the slice needs them: bursts (shots per action and spread), a `reload` action,
      and heat as a hand delay that builds with firing.
- [ ] A second rule set behind `combat.Rules` only if the formulas themselves (to-hit, levels,
      skills) need to differ. Weigh it once the data above is in.

### 4. Higher-resolution art

The 32-pixel tile size is fixed twice (`drawlist.TileSize = 32`, `sim.CellUnits = 32`). All art
is indexed and drawn through one 256-color palette shader. The world buffer is at the art's pixel
scale, scaled to the window by a whole number (design.md, Rendering).

- [ ] **Density.** A `density` in `pack.json` (1, 2 or 4) gives the art pixels per projection
      pixel. The renderer scales the projection, tile boxes, anchors and pixel offsets by it, and the
      simulation's units stay as they are, so levels, collision and saves do not change. The world buffer is drawn at
      the pack's density, and the window's whole-number zoom applies on top.
      `drawlist.TileSize` becomes the pack's tile size times its density, and the projection
      constants become content, as design.md already states.
- [ ] **True-color sprites.** A sheet can be `"color": "rgba"` instead of indexed. The palette
      shader passes its colors through, applying light and shadow but not palette effects or
      remaps. Indexed and true-color sheets mix in one frame. Fades and flashes (scenes) apply
      to both.
- [ ] **UI at density.** Frames, icons, portraits, the doll and a pack font are drawn at the
      pack's density, so the panels match the world.

### 5. Packs without the dev pack

- [ ] A `pack new` command writes the starter set to a directory.

## Order

1 and 3 fix the plainest gaps, and 1 is the cheapest. 2 builds on
1's labels. 4 is the largest change and benefits from 2's skin, since UI at density needs it.

## Open questions

- Should a skin be able to change the layout too (where the hotbar sits, how many shelves), or
  only the look?
- Is a pack font bitmap only, or are scaled outline fonts allowed at higher densities?
- Should damage kinds stay a closed list per pack, or may packs add kinds that only their own
  rule set understands?
- Should a pack choose its frame rate of animation, or its cell size in world units? Both are
  left fixed here.
