# Content pack format

Status: v0, 2026-10-30. Covers what Milestone 1 (Viewer) needs: the manifest, palettes, sprite
banks, floor and wall sets, and the drawable part of a level. Later milestones extend it. The Go
types are in `engine/content`.

A pack is a directory. Paths below are relative to its root. All JSON is UTF-8, and unknown fields
are an error. Every record has a stable string `id`; references are by id, never by path or index.
A record's file is named after its id without the kind prefix (`sprite.res.1056` is
`sprites/res.1056.json`), but the loader goes by the `id` inside.

## IDs

- Lowercase, dot-separated: `<kind>.<name>`, for example `palette.0`, `walls.1056`, `level.00`.
- Mods override a record by giving the same id, and add records with new ids.
- The Summoning importer names records after the original's resource ids (`sprite.res.1045` is
  resource 1045 as it is; `walls.1056` is composed from wall bank 1056), so an import is stable
  across runs.

## pack.json

```json
{
  "id": "summoning",
  "name": "The Summoning",
  "version": "0.1.0",
  "format": 0,
  "depends": [],
  "palette": "palette.0",
  "start": { "level": "level.00", "cell": [16, 15] }
}
```

`format` is this document's version; the loader rejects a format it does not know. `palette` is the
palette gameplay draws with.

## Palettes

`palettes/<name>.json`:

```json
{ "id": "palette.0", "colors": ["#000000", "#0c080c", "…256 entries…"] }
```

Exactly 256 colors, plus optional `effects` that animate them:

```json
"effects": [
  { "kind": "rotate", "period_ms": 440, "first": 224, "size": 4, "groups": 4 },
  { "kind": "cycle", "period_ms": 440, "index": 252, "colors": ["#3d006d", "#610f9a"] }
]
```

- `rotate`: `groups` runs of `size` colors starting at `first`. Every `period_ms`, each color in a
  run takes the value of the next, and the last takes the first's.
- `cycle`: every `period_ms`, color `index` becomes the next of `colors`, starting with the first.

Effects run on presentation time from when the view starts, so anything drawn in those colors
animates without being redrawn. Fades, flashes and remaps are added with scenes.

## Sprite banks

`sprites/<name>.png` and `sprites/<name>.json`.

- The PNG is indexed. Its pixel values are palette indexes; the engine ignores its PLTE and draws
  through the palette in use. The importer writes the gameplay palette into PLTE, with the
  transparent index fully transparent, so the sheet looks right in an image editor.
- Frames are laid out in a grid, left to right, top to bottom, in frame order. Each frame of a bank
  has the same size, as in the original's banks.

```json
{
  "id": "sprite.res.1045",
  "frame_size": [64, 32],
  "columns": 8,
  "frames": 8,
  "transparent": 254
}
```

`transparent` is a palette index or `null` for an opaque bank. Frame `i` is at
`(i % columns * w, i / columns * h)`. Per-frame origins and animations are added when entities
need them.

## Floor sets

`tiles/floors/<name>.json`: a level's floor variants. Each is one image covering a cell's tile box
(32×32 in the Summoning; `docs/spec/world.md`), composed by the importer from the original's
half-tiles.

```json
{ "id": "floors.1036", "sprite": "sprite.floors.1036", "variants": 4 }
```

Variant `k` is frame `k` of the sprite bank.

## Wall sets

`tiles/walls/<name>.json`: one composed image per wall shape, with its upper part, lower part and
caps already in it.

```json
{
  "id": "walls.1056",
  "sprite": "sprite.walls.1056",
  "offset": [0, -8],
  "shapes": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14]
}
```

`shapes[s]` is the frame for shape `s`, or −1 when the set has no such shape. `offset` places each
image's top-left relative to the cell's tile box (the caps rise 8 pixels above it). Exits in walls
use a second wall set with only the shapes that can hold an exit.

## Levels

`levels/<name>/level.json`:

```json
{
  "id": "level.00",
  "name": "",
  "size": [18, 26],
  "floors": "floors.1036",
  "walls": "walls.1056",
  "exit_walls": "exitwalls.1014",
  "floor": [
    "0 0 0 0 0 0 0 0 . . .",
    "…one string per row…"
  ],
  "wall": [
    "1 1 1 1 1 1 1 5 . . .",
    "…"
  ],
  "floor_halves": [
    "2 2 2 2 2 2 2 3 . . .",
    "…"
  ],
  "exits": [[0, 9]],
  "overlays": [
    { "cell": [3, 4], "sprite": "sprite.res.1045", "frame": 0, "offset": [-16, 8] }
  ],
  "decorations": [
    { "cell": [0, 5], "sprite": "sprite.res.1187", "frame": 3, "offset": [8, 8] }
  ],
  "props": [
    { "cell": [7, 2], "sprite": "sprite.res.1030", "frame": 0, "offset": [-17, -3] }
  ]
}
```

- `size` is rows, columns. Cell `(x, y)` is row `x`, column `y`, as in the spec.
- `floor` and `wall` hold one string per row of space-separated values, `.` for none, so a level
  diffs well. A floor value is a variant of the floor set, a wall value a shape of the wall set.
- `floor_halves` says which halves of each floor are drawn: 1 the upper, 2 the lower, 3 both, 0
  neither. A half not drawn is blanked. Without it, every floor is drawn whole. In v0 the importer
  computes it as if the whole level were visible (`docs/spec/world.md`, Floor halves); once
  visibility exists (Milestone 2) the rule moves into content and the renderer applies it to what
  the player sees.
- `exits` are wall cells drawn from `exit_walls` instead of `walls`, by the same shape.
- `overlays` are drawn after all floors and before anything standing: plates, levers, keyholes,
  hazard tiles. `decorations` are drawn on their cell's wall right after it. `props` stand in their
  cell and are drawn back to front with the walls (doors). Offsets are from the cell's tile box.
- Gameplay meaning (walkability, doors that move, triggers, exits that lead somewhere) is not in
  v0. It arrives with Milestones 2 and 3, and will replace the fixed overlay and prop sprites with
  typed cell features that the renderer draws by state.
- Level names are empty for now; the spec does not yet say where the game keeps them.

## Decisions

- **Layers, not cell records.** A level's grids are parallel layers. They are smaller and diff
  better; new per-cell data becomes a new layer.
- **Shared tile sets.** Floor and wall sets are records of their own that levels reference, since
  the original shares banks between levels.
