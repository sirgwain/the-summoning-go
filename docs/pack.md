# Content pack format

Status: draft v0, 2026-10-30. Covers what Milestone 1 (Viewer) needs: the manifest, palettes,
sprite banks, floor and wall sets, and the drawable part of a level. Later milestones extend it.

A pack is a directory (or a zip of one). Paths below are relative to its root. All JSON is UTF-8.
Every record has a stable string `id`; references are by id, never by path or index.

## IDs

- Lowercase, dot-separated: `<kind>.<name>`, for example `palette.game`, `sprite.terrain.44`,
  `level.00`.
- Mods override a record by giving the same id, and add records with new ids.
- The importer names records after the original's groups and offsets (`sprite.terrain.44` is
  terrain base + 44), so an import is stable across runs.

## pack.json

```json
{
  "id": "summoning",
  "name": "The Summoning",
  "version": "0.1.0",
  "format": 0,
  "depends": [],
  "start": { "level": "level.00", "cell": [16, 15] }
}
```

`format` is this document's version; the loader rejects a format it does not know.

## Palettes

`palettes/<name>.json`:

```json
{ "id": "palette.game", "colors": ["#000000", "#0c080c", "…256 entries…"] }
```

Exactly 256 colors. Palette effects (cycling, fades, remaps) are added in a later milestone.

## Sprite banks

`sprites/<name>.png` and `sprites/<name>.json`.

- The PNG is 8-bit indexed. Its pixel values are palette indexes; the engine ignores its PLTE and
  draws through the palette named at draw time. The importer writes `palette.game` into PLTE so the
  sheet looks right in an image editor. The transparent index, if any, is also marked in tRNS.
- Frames are laid out on the sheet in a grid, left to right, top to bottom, in frame order. Each
  frame of a bank has the same size, as in the original's banks.

```json
{
  "id": "sprite.terrain.44",
  "frame_size": [64, 16],
  "columns": 8,
  "frames": 30,
  "transparent": 0
}
```

`transparent` is a palette index or `null` for an opaque bank. A frame's sheet position is
`(i % columns * w, i / columns * h)`. Per-frame origins and animations are added when entities need
them.

## Floor sets

`tiles/floors/<name>.json`: the floor variants of a level, each one 64×32 image covering a cell's
tile box (world.md), composed by the importer from the original's half-tiles.

```json
{ "id": "floors.terrain.24", "sprite": "sprite.floors.terrain.24", "variants": 4 }
```

Variant `k` is frame `k` of the sprite bank.

## Wall sets

`tiles/walls/<name>.json`: one composed image per wall shape, with its upper part, lower part and
caps already in it.

```json
{
  "id": "walls.terrain.44",
  "sprite": "sprite.walls.terrain.44",
  "offset": [0, -8],
  "shapes": 15
}
```

Shape `s` is frame `s`. `offset` places the image's top-left relative to the cell's tile box
(the caps rise 8 pixels above it).

## Levels

`levels/<name>/level.json`:

```json
{
  "id": "level.00",
  "name": "",
  "size": [18, 26],
  "floors": "floors.terrain.24",
  "walls": "walls.terrain.44",
  "floor": [[0, 0, -1, "…"], "…"],
  "wall": [[-1, 1, 1, "…"], "…"],
  "overlays": [
    { "cell": [3, 4], "sprite": "sprite.terrain.33", "frame": 0, "offset": [-16, 8] }
  ],
  "decorations": [
    { "cell": [0, 5], "sprite": "sprite.decoration.1", "frame": 3, "offset": [8, 8] }
  ]
}
```

- `floor` and `wall` are `size[0]` rows of `size[1]` integers. A floor entry is a variant of the
  floor set, a wall entry a shape of the wall set; −1 is none. Cell `(x, y)` is row `x`, column `y`,
  as in the spec. The writer puts one row per line so a level diffs well.
- `overlays` are static pieces drawn on a floor after it (plates, levers, keyholes, hazard tiles).
  `decorations` are drawn on a wall right after it. Offsets are from the cell's tile box.
- Gameplay meaning (walkability, doors, triggers, exits) is not in v0. It arrives with Milestones 2
  and 3, and will replace the hard-coded overlay sprites with typed cell features that the renderer
  draws by state.

## Open questions

- Whether a level's grids should be one grid of cell records (`{"floor":0,"wall":-1,…}`) instead of
  parallel layers. Layers are smaller and diff better; records are easier to extend.
- Whether the level-specific art references (floor set, wall set, overlays) belong in the level or
  in a reusable "tileset" record that several levels share. The original shares banks across levels,
  so the importer would produce shared sets either way.
- Exits in walls and doors need state-dependent sprites; they come with Milestone 2.
