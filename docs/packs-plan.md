# Packs plan

Status: 2026-11-05. Nothing landed. This plan covers what the engine needs so that a pack other than
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

- [ ] Specify engine labels in pack.md (Text): each label a text key with an English default kept
      by the engine, overridden by the pack's text. They are named by their role, not their words:
      `ui.hit_points`, `ui.magic_points`, `ui.spell`, `ui.school`, `ui.memorize`, `ui.armor`, and so on.
- [ ] Move every fixed label in `engine/viewer`, `engine/viewer/window` and `engine/inspect` to them.
      A test walks the panels with a pack that renames every label and finds no default left.

### 2. Skins: frames, colors and font from the pack

Panel frames are drawn by code in `engine/frame`, with two styles (`Iron`, `Runic`) assigned by
the viewer. The text color and gauge colors are constants, and the font is the engine's own
(design.md, UI).

- [ ] Specify a `ui.json` record in pack.md:
  - for each panel (hotbar, inventory, spell panel, menus and buttons, conversation, map, book,
    log), either a built-in frame style by name or a pack image cut in nine (corners kept, edges
    and middle stretched or tiled), with its frame width;
  - the inside color, text colors and the button states (chosen, under the pointer, disabled);
  - the gauge colors (hit points, magic points, load, armor);
  - optionally a pack font: an atlas sprite with an advance and line height, at a small and a
    large size, letters it lacks shown as `?` as now.
  The record is optional: without it the panels keep today's frames.
- [ ] Give `engine/frame` a style table instead of a fixed switch, and add a third built-in
      style suited to science fiction (riveted hull plate with amber indicator lights), so a
      pack can be skinned without art of its own.
- [ ] Have the viewer take every frame, color and the font from the skin.

### 3. Ranged combat beyond bows

The `fire` action draws an arrow from a quiver worn in slot 25, flies `5 × STR` steps, does
damage kind `Arrow`, and refuses with "Not enough room to draw bow" within a step of a thing
(spec/items.md, Throwing; spec/combat.md). Damage kinds are a fixed list in `engine/combat`
(`Physical`, `Fire`, `Lightning`, `Arrow`, `Poison`), and creature defenses are fixed flags
matched to them.

- [ ] **Ammunition by tag.** A `fire` action names where its shots come from: `ammo` (an item tag
      and the worn slot that holds it, the quiver and `arrows` for the Summoning), the weapon's
      own `charges` (a magazine), or the player's magic points (an energy weapon), with the
      cost of a shot. The importer writes the Summoning's quiver rule as this data.
- [ ] **The shot on the weapon.** `range` (a formula of attributes, `5 × STR` capped at 127 for
      the Summoning), flight `speed`, `damage`, `kind` and the flying look can be given on the
      `fire` action. Each falls back to the ammunition's `missile` as now.
- [ ] **Room to fire** becomes a per-action flag (`needs_room`), set by the importer on bows, so
      a pistol fires point-blank.
- [ ] **Damage kinds as data.** The pack lists its kinds (ballistic, plasma, EMP…). Creature
      templates resist, ignore or are weak to kinds by name, and the shields and statuses that
      change damage name kinds as well. The Summoning's five kinds, and how its flags map to
      them, become the importer's pack. `combat.Kind` becomes an index into the pack's list.
- [ ] **Creature ranged attacks.** A creature template can name a missile attack (damage, kind,
      range, look, interval) without going through a spell, so a turret or a gunner drone
      shoots.
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
