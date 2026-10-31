# Content pack format

Status: v1, 2026-10-31. Covers what Milestones 1 (Viewer), 2 (World) and 4 (Walk) need: the manifest,
palettes, sprites, floor and wall sets, text, engine messages, item and thing templates, looks, and
levels with their floor types, features, procedures, things and objects. Later milestones extend it.
The Go types are in `engine/content`.

The format is the engine's own. It describes a game in the engine's terms (ground points, named
states, named hooks, ids) and carries none of the original's encodings: an importer translates a
game's data into it, and an author writing a pack by hand never needs to know how *The Summoning*
stored anything.

A pack is a directory. Paths below are relative to its root. All JSON is UTF-8, and unknown fields
are an error. Every record has a stable string `id`; references are by id, never by path or index.
A record's file is named after its id without the kind prefix (`item.dagger` is
`items/dagger.json`), but the loader goes by the `id` inside.

## IDs

- Lowercase, dot-separated: `<kind>.<name>`, for example `palette.0`, `item.dagger`, `level.03`.
- Mods override a record by giving the same id, and add records with new ids.
- Ids are names, not numbers, wherever there is something to name. The Summoning importer names
  items after their names (`item.dagger`; a second Dagger is `item.dagger.2`), levels by number
  (`level.03`), and art and records the original leaves unnamed by kind and number
  (`sprite.res.1045`, `walls.1056`, `thing.npc.12`), so an import is stable across runs. Nothing
  depends on the number in an id; an author can rename any record.
- Inside a level, its features, things and procedures have **local ids**, unique in the level:
  `door.east_gate`, `boulder.1`, `open_gate`. Actions refer to features and things by local id,
  so moving a door in the editor does not break the actions that open it. The importer names them
  by kind and cell (`door.12.9`) and procedures by number (`p5`).

## Conventions

- **World units.** Positions are in world units, 32 to a cell. Cell `(x, y)` is row `x`, column `y`
  and covers units `32x .. 32x + 31` by `32y .. 32y + 31`.
- **Ground point.** A thing's position is the point where it stands on the ground. The cell under it
  is its cell: the one it presses, triggers and is found in.
- **Directions** are 0..7, one step in each being `(dx, dy)` in cells:

  | Dir | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | dx | −1 | −1 | 0 | 1 | 1 | 1 | 0 | −1 |
  | dy | 0 | 1 | 1 | 1 | 0 | −1 | −1 | −1 |

  On screen, 1 is right, 3 down, 5 left and 7 up.
- **Durations** are in milliseconds. **Dice** are strings: `"3d10+8"`, `"2d2"`, `"5"`.
- **Pixel offsets** are `(dx, dy)` on screen. A cell's **tile box** is the 32×32 box its floor art
  fills (`docs/spec/world.md`, Projection); offsets "from the cell" are from its top-left corner.

## pack.json

```json
{
  "id": "summoning",
  "name": "The Summoning",
  "version": "0.1.0",
  "format": 1,
  "depends": [],
  "palette": "palette.0",
  "start": { "level": "level.00", "cell": [16, 15] },
  "player": "thing.player"
}
```

`format` is this document's version; the loader rejects a format it does not know. `palette` is the
palette gameplay draws with. `player` is the thing template of kind `player` the player is made
from; a pack without one runs its levels with no player.

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

## Sprites

`sprites/<name>.png` and `sprites/<name>.json`.

- The PNG is indexed. Its pixel values are palette indexes; the engine ignores its PLTE and draws
  through the palette in use. The importer writes the gameplay palette into PLTE, with the
  transparent index fully transparent, so the sheet looks right in an image editor.
- Frames are laid out in a grid, left to right, top to bottom, in frame order. Every frame of a
  sheet has the same size.

```json
{
  "id": "sprite.res.1092",
  "frame_size": [32, 40],
  "columns": 8,
  "frames": 12,
  "transparent": 254,
  "anchor": [18, 35],
  "animations": {
    "walk.0": { "frames": [0, 4], "ms": 330 },
    "attack.0": { "frames": [8] },
    "idle": { "frames": [0, 1, 2], "ms": [110, 440], "mode": "random" }
  }
}
```

- `transparent` is a palette index or `null` for an opaque sheet. Frame `i` is at
  `(i % columns * w, i / columns * h)`.
- `anchor` is the pixel in each frame that is drawn at the thing's ground point. `anchors` may give
  one per frame instead. Sprites drawn only as tiles and looks need no anchor.
- `animations` are named sequences of frames. `ms` is how long each frame shows, or a range
  `[min, max]` to pick from each time. `mode` is `loop` (the default), `once`, `random` (a random
  frame of the list each time) or `flicker` (hold the first frame for one duration, then play the
  rest in a loop for another). Templates name the animations they use (below).

## Floor sets

`tiles/floors/<name>.json`: floor art, one image per variant.

```json
{ "id": "floors.1036", "sprite": "sprite.floors.1036", "variants": 4, "shape": "box" }
```

Variant `k` is frame `k` of the sprite. A floor set is only art; what a floor does is its level's
floor type (below).

`shape` says what each image covers:

- `box` (the default): a 32×32 image filling the cell's tile box. The box holds the middle of the
  cell's diamond and a corner of each of its four neighbors' (top left `(x, y − 1)`, top right
  `(x − 1, y)`, bottom left `(x + 1, y)`, bottom right `(x, y + 1)`), and the boxes of all cells
  tile the view like bricks. The art paints those corners without knowing the neighbors, so it
  suits floors that look alike. A cell with no floor, and each cell just past the level's far
  edges, blanks its box, so the corners of the floors behind it do not show.
- `diamond`: a 64×32 image of the cell's diamond alone, its corners transparent, drawn at
  `(−16, 0)` from the tile box. Diamonds tile without overlap, so a cell looks the same whatever
  floor its neighbors have, and nothing is blanked.

Halves (`floor_halves`, below) are the upper and lower 16 rows of either image.

## Wall sets

`tiles/walls/<name>.json`: one composed image per wall shape, with its upper part, lower part and
caps already in it.

```json
{
  "id": "walls.1056",
  "sprite": "sprite.walls.1056",
  "offset": [0, -8],
  "shapes": [
    { "frame": 0, "diagonal": "bottom" },
    { "frame": 1, "face": true },
    { "frame": 2, "diagonal": "left" },
    { "frame": 3, "face": true },
    null
  ]
}
```

- `shapes[s]` describes shape `s`, or is `null` when the set has no such shape. `offset` places each
  image relative to the cell's tile box (the caps rise 8 pixels above it).
- `diagonal` marks a wall that fills half its cell, and names the corner of the cell's diamond
  it fills: `top`, `right`, `bottom` or `left` (on screen; `top` is toward −u and −v, `bottom`
  toward +u and +v, `right` toward −u and +v, `left` toward +u and −v). Walking along one axis
  into it, toward that corner, slides along it (`docs/spec/entities.md`, Moving).
- `face` marks a straight wall whose face can carry a decoration or a wall feature (an exit, a
  mouth).
- The shapes and their properties are the set's own. The engine reads the properties, never the
  shape numbers, so a set can have any number of shapes.

## Text

`text/<lang>.json`: every string of the pack in one language.

```json
{
  "id": "text.en",
  "strings": {
    "msg.134": "The lever won't budge",
    "item.dagger.name": "Dagger",
    "msg.l03.2": "…"
  }
}
```

Records refer to a string by its key. In a string, `*word*` marks a keyword
(`docs/spec/dialogue.md`) and `@` stands for the player's name. The importer converts the original's
text to UTF-8.

## Messages

`messages.json`: the strings the engine itself shows, by the engine's own keys.

```json
{
  "id": "messages",
  "messages": {
    "lever_stuck": "msg.134",
    "slot_wrong_item": "msg.143",
    "slot_stuck": "msg.135",
    "no_drop_here": "msg.114",
    "no_room": "msg.127"
  }
}
```

The engine defines the keys; the pack gives the text. In `slot_wrong_item` and `slot_stuck`,
`{item}` stands for the item's name. A key the pack leaves out shows nothing. Later milestones add
keys.

## Items

`items/<name>.json`: an item template.

```json
{
  "id": "item.bag_of_lightness",
  "name": "item.bag_of_lightness.name",
  "weight": 2,
  "size": 3,
  "container": { "capacity": 4, "largest": 3, "weightless": true },
  "tags": ["bag"],
  "random": false,
  "icon": { "sprite": "sprite.res.112", "frame": 0 },
  "floor": { "sprite": "sprite.res.578", "frame": 0 }
}
```

- `weight` and `size` are the item's own, without contents.
- `container` is present only for containers: it takes up to `capacity` objects of size at most
  `largest`. With `weightless`, its contents add no weight.
- `tags` are words that rules and other records test (`body_armor`, `quiver`, `arrows`, `bottle`,
  `weapon`, `two_handed`). The engine attaches no meaning to a tag by itself.
- `random` items can be picked for a random object.
- `icon` is the inventory image; `floor` the image drawn on the floor. With magnifying on, the floor
  uses `icon` instead. Both are drawn centered in the cell's tile box.
- Durability, charges and counts, armor, actions, damage and powers are added with the milestones
  that use them, each as its own field.

## Things

`things/<name>.json`: a template for things that stand in a level: creatures, boulders and traps.

```json
{
  "id": "thing.npc.12",
  "kind": "creature",
  "sprite": "sprite.res.1092",
  "facing": 2,
  "speed": 8,
  "interval_ms": 330,
  "weight": 100,
  "hit_points": "3d10+8",
  "behaviors": ["hostile", "leaves_corpse"],
  "stats": { "armor": 1, "accuracy": 10 }
}
```

- `kind` is `creature`, `boulder`, `trap` or `player`. The kind decides what the engine does with
  the thing; later milestones add kinds (things in flight, conjured things). A `player` template is
  named by `pack.json` and never placed by a level.
- `sprite` is drawn with its anchor at the thing's ground point. `animations`, as on a sprite, are
  the thing's own; a name the thing does not have comes from its sprite. Animations are found by
  name: `idle.d`, `walk.d`, `attack.d` and `roll.d` for direction `d`, and `idle` and `dead`. A
  missing animation falls back to `idle.d`, then `idle`, then frame 0. `walk.d` and `roll.d` step
  one frame per move the thing makes, not by time.
- `facing` is the starting direction, `speed` the step in world units, `interval_ms` the time
  between updates.
- `footprint` is the side, in world units, of the square around the ground point the thing takes
  up: two things whose footprints overlap collide. Boulders and traps also test the footprint's
  corners against the terrain. 0 means 16.
- `weight` is what the thing adds to a plate it stands on; 0 presses nothing.
- `hit_points` are rolled when the thing is placed. A creature with none is placed as a corpse.
  `magic_points` (players) are rolled the same way.
- `behaviors` (creatures): `hostile` from the start, `never_hostile`, `leaves_corpse` when it dies,
  `resists_teleport` (teleporters, pits and plates leave it alone), `invulnerable`. Combat behaviors
  and resistances are added with Fight. What the game sets in play (talked to, dead) is world state.
- `stats` are named numbers the pack's rule set reads; the engine does not interpret them, except a
  `player`'s attributes `strength`, `agility`, `endurance`, `accuracy`, `talent` and `power`,
  from which its speed is worked out (`docs/spec/player.md`, Speed), and its starting
  `interval_bonus` in ticks; a player without attributes walks at its base interval.
- A conversation, portrait and loot are added with dialogue and combat.

## Looks

`looks/<name>.json`: how a feature is drawn in each of its states.

```json
{
  "id": "look.plate.1045",
  "layer": "floor",
  "states": {
    "up":   [{ "sprite": "sprite.res.1045", "frame": 0, "offset": [-16, 8] }],
    "down": [{ "sprite": "sprite.res.1045", "frame": 1, "offset": [-16, 8] }]
  }
}
```

- `layer` is `floor`, drawn after all floors and before anything standing; `stand`, drawn back to
  front with walls and things; or `wall`, drawn in place of the cell's wall image.
- Each state is a list of sprite frames, offset from the cell's tile box. An empty list draws
  nothing.
- The states a look must have depend on the kind of feature that uses it (below).

## Levels

`levels/<name>/level.json`:

```json
{
  "id": "level.03",
  "name": "",
  "size": [18, 26],
  "floors": "floors.1036",
  "walls": "walls.1056",
  "floor_types": [
    { "variant": 0 },
    { "variant": 1 },
    { "variant": 3, "enter": { "damage": "1" }, "avoided": true }
  ],
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
  "overlays": [
    { "cell": [3, 4], "sprite": "sprite.res.67", "frame": 2, "offset": [0, 0] }
  ],
  "decorations": [
    { "cell": [0, 5], "sprite": "sprite.res.1187", "frame": 3, "offset": [8, 8] }
  ],
  "features": [
    { "id": "plate.12.7", "kind": "plate", "cell": [12, 7], "look": "look.plate.1045",
      "armed": true, "threshold": 50, "on_press": "p0", "on_release": "p1" },
    { "id": "door.12.9", "kind": "door", "cell": [12, 9], "look": "look.door.1030.h",
      "state": "closed" },
    { "id": "exit.0.9", "kind": "exit", "cell": [0, 9], "look": "look.exit.1014.1",
      "to": { "level": "level.04", "cell": [16, 3] } }
  ],
  "procedures": {
    "p0": [{ "on": "door.12.9" }],
    "p1": [{ "off": "door.12.9" }]
  },
  "things": [
    { "id": "creature.5.6", "thing": "thing.npc.12", "cell": [5, 6], "offset": [-6, 12],
      "items": [{ "item": "item.dagger" }] },
    { "id": "boulder.8.2", "thing": "thing.boulder.1033", "cell": [8, 2] }
  ],
  "objects": [
    { "cell": [4, 4], "item": "item.dagger" },
    { "cell": [4, 5], "item": "item.bag_of_lightness", "contents": [{ "item": "random" }] }
  ]
}
```

- `size` is rows, columns.
- `floor_types` are the level's kinds of floor. The `floor` grid holds an index into this list, `.`
  for none. `wall` holds a shape of the wall set, `.` for none. A cell with a wall blocks; a cell
  with neither is outside the map. Both grids are one string per row of space-separated values, so
  a level diffs well.
- `floor_halves` says which halves of each floor are drawn: 1 the upper, 2 the lower, 3 both, 0
  neither. A half not drawn is blanked in a box set and left out in a diamond set. Without it,
  every floor is drawn whole. The importer computes it as if the whole level were visible
  (`docs/spec/world.md`, Floor halves). In play only visible cells are drawn at all.
- `overlays` are fixed floor art (hazard tiles), drawn after all floors and before anything
  standing. `decorations` are drawn on their cell's wall face right after it. `props` (not shown)
  are fixed art standing in a cell, drawn back to front with the walls. Offsets are from the cell.
- Level names are empty for now; the spec does not yet say where the game keeps them.

### Floor types

A floor type is a floor set variant and what the floor does. All fields but `variant` are optional.

| Field | Meaning |
| --- | --- |
| `variant` | The floor set's variant drawn |
| `enter` | An effect on whoever enters: `{ "damage": dice, "type": "fire" }` or `{ "drain_magic": n }` |
| `tick` | An effect on whoever stands there, every status tick |
| `depth` | Water depth 1..4 (`docs/spec/world.md`, Liquids) |
| `sinks` | Objects put down here are lost, except on a plate that is up |
| `avoided` | Creatures do not walk onto it |

`enter` and `depth` act on the player as it enters, `tick` and depth 4 on every status tick
(`docs/spec/world.md`, Liquids); `sinks` acts on dropped objects. `avoided` applies to creatures
with Fight.

### Features

A cell has at most one feature. Every feature has an `id`, a `kind` and a `cell`; those drawn have a
`look`. A **hook** names the procedure a feature runs when something happens to it; a hook left out
runs nothing.

| Kind | Fields | Hooks | Look states |
| --- | --- | --- | --- |
| `door` | `state` (`closed`, `open`, `opening`, `closing`), `sealed_by` (an item), `opened_by_spell` | | `closed`, `1` .. `n`, `open`, `sealed` |
| `lever` | `on`, `armed` | `on_turn_on`, `on_turn_off` | `off`, `on` |
| `plate` | `down`, `armed`, `threshold` | `on_press`, `on_release` | `up`, `down` |
| `slot` | `accepts` (an item), `armed` | `on_insert` | `slot` |
| `teleporter` | `armed`, `destinations` (two cells), `next` 0 or 1, `turn` (eighths of a turn for things in flight), `fly_over` | | `armed`, `disarmed` |
| `exit` | `to` (a level and a cell) | | `exit` |
| `pit` | `to` | | `pit` |
| `trapdoor` | `open`, `to` | | `open`, `closed` |
| `socket` | `places` (the item each place takes), `set` (the indexes of the places filled) | `on_complete` | `empty`, `full`, `set.0` .. `set.n` |
| `mouth` | `message` | | `mouth` |
| `mark` | `rune` (a name) | | `mark` |

- A door's look has the stages it moves through between `closed` and `open`, numbered from 1. A
  sealed door shows `sealed` and opens only to its `sealed_by` item landing on it.
- An `exit` on a wall cell is in the wall's face, drawn with a `wall` layer look and taken by
  walking into the face. An `exit` on a floor cell is a marker, taken by walking onto it. A `mouth`
  stands on a floor cell, drawn up against the wall behind it.
- A socket draws `full` when every place is set; otherwise `empty` and `set.i` for each set place.
- A plate's counter starts at the weight of what the level places on it.
- How each kind behaves is in `docs/spec/rules.md`.

### Procedures

`procedures` maps a local id to a list of actions, run in order.

| Action | Effect |
| --- | --- |
| `{ "on": feature }` | Switch on: open a door or trapdoor, arm a teleporter or trigger |
| `{ "off": feature }` | Switch off |
| `{ "toggle": feature }` | Toggle |
| `{ "set_hook": { "feature", "hook", "procedure" } }` | Point a hook at another procedure, or at none with `null` |
| `{ "set_threshold": { "feature", "value" } }` | Set a plate's threshold |
| `{ "set_accepts": { "feature", "item" } }` | Set the item a slot takes |
| `{ "set_message": { "feature", "message" } }` | Set what a mouth says |
| `{ "roll": { "thing", "direction", "shuttle" } }` | Start a boulder rolling; with `shuttle` it reverses when blocked instead of stopping |
| `{ "stop": thing }` | Stop a boulder |
| `{ "shoot_item": { "cell", "item", "direction" } }` | Fire a new object from a cell |
| `{ "shoot_spell": { "cell", "spell", "direction", "power" } }` | Fire a spell from a cell; `power` is dice |
| `{ "create": { "cell", "item" } }` | Put a new object on a cell's floor |
| `{ "set_floor": { "cell", "type" } }` | Make a cell floor of a type, removing any wall |
| `{ "set_wall": { "cell", "shape" } }` | Make a cell a wall of a shape |

`roll`, `stop` and the shooting actions run in later milestones; until spells exist, `spell` is the
spell's number.

### Things and objects

- `things` are placed when the level is first entered. Each has a local `id`, a `thing` template and
  a `cell`. Its ground point is the cell's center, moved by `offset` in world units if given. It may
  override the template's `facing`. A creature may carry `items`; a boulder may start rolling, with
  `"rolling": { "direction": 2, "shuttle": false }`.
- `objects` lie on a cell's floor in list order. An object names its `item` and gives its `contents`
  if it is a container. The item `random` is a random object, chosen when the level is first
  entered from the templates marked `random`.

## The Summoning import

The importer translates the original's encodings into the format above:

- Each sprite's anchor comes from where the original draws that kind of thing relative to the
  cell it stands on (`docs/spec/entities.md`, Placement and Standing cell), so things land on
  screen where the original puts them. Things placed off center get an `offset`.
- Animations come from the original's frame layouts and patterns (`docs/spec/creatures.md`), with
  update intervals converted from ticks to milliseconds.
- Doors, levers, plates and the rest become features with named states and hooks; procedure a and b
  become the hooks of each kind (lever a `on_turn_on`, b `on_turn_off`; plate a `on_press`, b
  `on_release`; slot a `on_insert`). Teleporter jumps become `destinations`. Exit markers and
  mouths become features. The gem socket's opening becomes its `on_complete` procedure, of
  `set_floor` and `set_wall` actions.
- Procedure ops become actions on the feature or thing at the cell they name; op 10 becomes
  `set_threshold` or `set_accepts` by the feature there.
- Wall shapes get their properties from `docs/spec/world.md` (diagonal shapes 0, 2, 4, 5; faces 1
  and 3). Each level's floor types come from its floor variants and liquid kind.
- Item categories become tags. The original's message numbers become `messages.json` entries.

## Decisions

- **The engine's terms, not the original's.** The format is designed as if no game existed before
  it. The importer does the translating, so a pack never stores an original encoding (a code, a bit
  mask, a byte layout) that an author would need the spec to read.
- **Layers, not cell records.** A level's grids are parallel layers. They are smaller and diff
  better; new per-cell data becomes a new layer.
- **Shared tile sets.** Floor and wall sets are records of their own that levels reference.
- **Features, not terrain codes.** A level stores floor types and walls, plus a typed feature per
  cell with its own state. The simulation changes feature state and the renderer draws the
  feature's look for that state.
- **Procedures before rules.** Hooks run named action lists. General event and condition rules will
  reuse the same actions.
- **Ground points and anchors.** A thing's position is where it stands; its sprite's anchor says how
  to draw it there. No per-kind offsets.
- **Local ids for what a level holds.** Actions reach features and things by id, not by cell.
