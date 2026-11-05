# Mods plan

Status: 2026-11-05. Nothing landed. This plan covers mods: packs that load over another pack to
add or change its content, so that people can give The Summoning new levels and new art, high
resolution included, and share them without the game's files. It also changes phase 4 of
[packs-plan.md](packs-plan.md) (higher-resolution art) so that one picture can mix the base
pack's art with a mod's. It is milestone 10 (Modding) in [design.md](design.md).

Each phase changes [design.md](design.md) and [pack.md](pack.md) first, then the code. Update this
file as each phase lands, fold what's settled into those documents, and delete it once the phases
are done.

## The two scenarios

1. **A new game with high-resolution art.** A standalone pack whose art
   is drawn at two or four times the Summoning's pixel scale, in true color. This is packs-plan
   phase 4, with density per sprite instead of per pack (phase 1 below).
2. **A mod of The Summoning.** A small pack that depends on the imported `summoning` pack and
   changes it: new levels linked into the game, existing levels edited, existing levels re-dressed
   with new art (high resolution or not), new items and creatures. It plays over anyone's own
   import of the game.

## Principles

- **A mod holds only its own work.** The imported Summoning pack is made from the game's files on
  the player's machine and is never shared (AGENTS.md, rule 3). A mod must never contain the
  imported pack's art, text or level grids, copied or edited. So a mod changes the base by
  **patching**: it says what to change, never repeats what it leaves. A mod that replaces a whole
  imported record would carry the original's data, and `verify` warns of it (phase 2).
- **Records by id.** Everything a mod touches is named by its stable id (`level.03`,
  `thing.npc.116`, `sprite.res.1092`, `floors.1035`). The importer's ids are a contract: an
  importer change that renames one breaks mods, so it needs a format or major version bump
  (phase 2).
- **Mixed art in one picture.** A mod re-dresses one level or one creature at high resolution
  while the rest of the game stays as it is, so resolution and color are a sprite's, not a
  pack's.
- **The simulation is untouched.** 32 world units a cell, 60 steps a second. Art density and
  color are presentation; levels, collision and saves do not change with them.
- **The Summoning is the default.** No mods, no change: the imported pack plays exactly as now.

## Phases

### 1. Art: density and color per sprite (replaces packs-plan phase 4)

Today every sprite is indexed through one 256-color palette at the Summoning's pixel scale
(32-pixel tile boxes, 64 × 32 floor diamonds), and the world buffer is drawn at that scale and
scaled up by a whole number (design.md, Rendering).

- [ ] **`density` on a sprite sheet** (1, 2 or 4): art pixels per projection pixel. A sheet of
      density 2 has frames, anchors and offsets twice the size: a floor diamond 128 × 64, a 32-pixel
      tile box 64 pixels. A floor or wall set, look part, held layer, portrait or icon takes its
      sheet's density. The projection, anchors and offsets the renderer uses are in projection
      pixels, so mixed densities line up.
- [ ] **`"color": "rgba"` on a sprite sheet**: a true-color PNG with alpha instead of indexes. It
      is lit and shadowed as indexed art is, and the scenes' fades and flashes apply to it, but
      palette cycling and creature color remaps do not. A mod that recolors an indexed creature
      keeps it indexed.
- [ ] **The world buffer at the highest density in view**: drawn at that density, indexed and
      density-1 art scaled up nearest-neighbour into it, then the window's whole-number zoom on
      top. A pack or level with no density-2 art draws exactly as now.
- [ ] **UI at density.** Frames, icons, portraits, the doll and a skin's font drawn at their own
      density, so panels match the world (packs-plan phase 2's skin carries a `density`).
- [ ] **Tooling.** `summoning sprite` cuts a folder of frame PNGs into a sheet and writes its JSON
      (frame size, columns, anchors from a marker colour or a flag), at any density, indexed or
      rgba. Artists draw frames; the tool lays out the sheet.
- [ ] Tests: a level with a density-2 rgba floor set and density-1 indexed things draws them
      aligned; the same level with all density-1 art matches today's picture pixel for pixel.

### 2. Mods: loading a pack over another

`pack.json` has a `depends` list and design.md promises override packs, but the loader loads one
pack only.

- [ ] **A mod's manifest.** `"kind": "mod"`, `"depends": [{ "id": "summoning", "version": "^1" }]`,
      and no `start`, `player` or `palette` of its own unless it means to change them. A mod is a
      pack directory or a zip of one.
- [ ] **Load order.** The base pack, then each mod in order: `summoning play packs/summoning
      --mod mods/new-crypt --mod mods/hires-caves`, or the mods listed in a `mods.json` beside the
      saves. A later mod's change wins; the loader reports each record two mods both change.
- [ ] **Adding and replacing.** A record with a new id is added. A record the mod writes whole
      under an existing id replaces it; that is for the mod's own records and for sprites (below),
      never for imported levels, text or items.
- [ ] **Patching records.** `patches/<kind>/<id>.json` changes a base record by JSON Merge Patch
      (RFC 7386): the fields given replace the base's, `null` removes one, anything left out stays.
      So `{ "floors": "floors.mod.caves", "walls": "walls.mod.caves" }` re-dresses a level without
      a byte of its grid. Arrays are replaced whole, so lists of records with ids are patched by
      id instead (next item).
- [ ] **Patching levels.** A level patch can also change cells and its placed records without
      repeating the grid:
      - `cells`: `[{ "cell": [12, 30], "floor": 4, "wall": null, "floor_type": 2 }, …]` and
        rectangles `{ "from": [10, 28], "to": [14, 32], "floor": 4 }`;
      - `features`, `things`, `objects`, `props`, `overlays`: entries by id, added, merged or
        removed (`{ "id": "exit.4.9", "remove": true }`);
      - `procedures`: by name, replaced whole;
      - `size` can only grow, the new cells empty wall.
      Wall shapes, floor halves and sight data are recomputed for the cells a patch changes, as
      the importer computes them.
- [ ] **Text and sprites.** A mod's `text/en.json` adds and overrides strings by key. A mod's
      sprite with a base sprite's id replaces it (a high-resolution re-dress of a creature or an
      item): it must have the same frames and named animations, or `verify` refuses it, so every
      record using it still works.
- [ ] **Linking new levels in.** A new level is an ordinary mod record; a patch on a base level's
      exit or teleporter (`features` by id, a new `to`) leads into it, and the new level's exits
      lead back. Story counters and flags a mod needs get names of its own (`var.mod.crypt.1`).
- [ ] **Verify.** `summoning verify packs/summoning --mod mods/x` loads and checks the merged
      pack and reports what each mod changes. It warns when a mod contains a record that replaces
      an imported one whole (it would carry the game's data) and refuses a mod built for a format
      or base version it does not match.
- [ ] **Id stability.** The importer records its version in the pack it writes, and the ids it
      makes are listed in a test with hand-made fixtures, so an importer change that renames an id
      fails a test. Renaming is a major version bump of the imported pack.
- [ ] **Saves.** A save records the base pack and each mod with its version. Loading it with a
      different set warns, and a save naming a record no longer there is refused with a message,
      not a crash. A visited level keeps its saved state; a patch changes a level only before it
      is first visited, or on a new game.
- [ ] Tests, on the dev pack as the base: a mod adding a level linked from a patched exit; a patch
      re-dressing a level; a cell patch; a sprite replacement with matching and with missing
      animations; two mods changing one record; a save made with a mod loaded without it.

### 3. A sample mod

- [ ] `mods/example-crypt`: a small Summoning mod with only original work, a new level of a few
      rooms reached from a patched exit on an early level, drawn with density-2 rgba floors and
      walls, one re-dressed creature sprite, and a few lines of text. It loads over any import of
      the game, and its tests skip without `SUMMONING_DATA` (AGENTS.md, rule 3).
- [ ] `docs/modding.md`: a guide for mod authors, from `summoning import` to a shared zip: finding
      ids (`summoning view` names what is under the pointer), patching a level, drawing at
      density 2, `summoning sprite`, `verify`.

### 4. Later

- [ ] The editor (design.md, Editor) writes patches, not whole records, when editing an imported
      level.
- [ ] Removing records outright (`"remove": ["thing.x"]` in the manifest), if a mod needs it.
- [ ] Script hooks stay out: mods are data (packs-plan, Principles).

## Order

Phase 1 first: scenario 1 needs only it, and it settles the art format mods use. Phase 2 can be
built alongside it, on the dev pack, with density-1 art. Phase 3 needs both.

## Open questions

- Are mods per install (a `mods.json`) or per save (the save names its mods and loading it turns
  them on)?
- Should a mod patch the imported level grids at all, or only add levels and re-dress existing
  ones? Cell patches are the most useful and the most fragile across importer versions.
- Density 4: worth supporting in the first cut, or 1 and 2 only?
- Do true-color sprites need their own light and shadow tint rules, or is the indexed art's
  darkening (multiply) enough?
