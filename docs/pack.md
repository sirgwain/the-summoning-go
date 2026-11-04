# Content pack format

Status: v1, 2026-11-01. Covers what Milestones 1 (Viewer), 2 (World), 4 (Walk), 5 (Fight) and 7
(Magic and talk) need: the manifest, palettes, sprites, floor and wall sets, text, engine messages,
item and thing templates, portraits, looks, spells, magic, keywords, conversations, levels with
their floor types, features, procedures, things and objects, and audio. Later milestones extend it.
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
  "player": "thing.player",
  "rules": "summoning",
  "counters": { "after_conversation": "var.97", "use_result": "var.19", "transformed": "var.14",
                "quiet_combat": "var.96", "transcript": "var.99", "magnify": "var.98" },
  "menu": "thing.npc.0"
}
```

`format` is this document's version; the loader rejects a format it does not know. `palette` is the
palette gameplay draws with. `rules` names the combat rule set (`docs/design.md`, Characters, stats
and combat); `summoning` is the only one, and the default. `player` is the thing template of kind `player` the player is made
from; a pack without one runs its levels with no player. `counters` names the counters the engine
itself uses (`docs/spec/dialogue.md`, Engine model): `after_conversation` is cleared when every
conversation ends, `use_result` decides whether an object whose conversation has `use_result` is
used, `transformed` is set to 1 by a Shape Change into the look named by `transformed_look`,
and cleared when it ends, and while `quiet_combat` is not 0 the combat messages are not shown.
While `transcript` is not 0, what conversations say goes to the transcript. While `magnify` is not 0, objects on the floor are drawn magnified
(`docs/spec/items.md`, Floor). `menu` is the thing
template whose conversation the menu key opens. `start` may give a `music` track, which a new
game starts.

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
animates without being redrawn. The player's figure, and the items drawn in its hands, are the exception: they
are drawn with the palette's own colors, without its effects, and with its portrait's figure colors. Fades, flashes and remaps are added with scenes.

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
    { "frame": 0, "diagonal": "bottom", "sight": [7, 3], "unseen": [[2, 14], [4, 12]] },
    { "frame": 1, "face": true, "sight": [0, 4], "unseen": [[2, 11], [6, 12]] },
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
- `sight` gives the directions of the two neighbors whose sight shows the upper and lower halves of
  the floor under a wall of the shape, `-1` for none (`docs/spec/world.md`, Floor halves). A set
  whose shapes have no `sight` leaves the halves under walls to the level's `upper_sight` and
  `lower_sight`.
- `unseen` lists `[direction, shape]` pairs, tried in order whenever visibility is computed: at the
  first whose neighbor lies inside the level and is not visible, a visible wall of the shape is
  taken as that shape instead, and the new shape's pairs are tried in turn (`docs/spec/world.md`,
  Walls out of sight). The wall is drawn, slides the player and checks its floor halves as the
  shape taken; it shows no decoration while that is not its own shape. The level's shapes never
  change.
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

Records refer to a string by its key. In a string, an asterisk before a word (`*word`) marks a keyword
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
    "no_room": "msg.127",
    "miss": "msg.91"
  }
}
```

The engine defines the keys; the pack gives the text. In `slot_wrong_item`, `slot_stuck`,
`weapon_breaks`, `vanishes` and `filled`, `{item}` stands for the item's name; in `not_memorized` and
`only_spell`, `{school}` for the school's name; in `killed`, `{name}` for the creature's; in
`absorbed`, `{damage}` for the damage absorbed; in `miss`, `hit`, `creature_hits` and
`creature_misses`, `{score}` and `{damage}` stand for the numbers. The keys: `lever_stuck`,
`slot_wrong_item`, `slot_stuck`, `no_drop_here`, `no_room`, `not_a_creature`, `cannot_hurt`,
`miss`, `hit`, `creature_hits`, `creature_misses`, `weapon_breaks`, `vanishes`, `skill_up`, `level_up`,
`full_health`, `full_magic`, `not_poisoned`, `not_poisoned_or_confused`, `not_fatigued`,
`no_destination`, `no_creatures`, `already_have`, `healed` (`{damage}` the amount), `protected`,
`boost_extended`, `poisoned`, `slowed`, `lost_weapon_skills`, `lost_magic_skills`, `no_quiver`,
`no_arrows`, `no_room_to_shoot`, `cannot_throw`, `cannot_act`; for magic `cannot_cast`,
`not_memorized`, `only_spell`, `no_magic`, `forgot`, `forgot_more`, `no_room_for_wall`, `wall_failed`,
`no_room_for_portal`, `no_spell_near`, `nothing_to_mend`, `no_creature_near`, `no_bottle`,
`filled`, `door_resists`, `missile_wall`, `battle_rage`, `battle_rage_extended`, `haste`,
`fire_shield`, `lightning_shield`, `frozen`, `sight`, `weapon_skills_up`,
`weapon_skills_extended`, `magic_skills_up`, `magic_skills_extended`, `cured`; for fights `killed`
and `absorbed`; for the music key `music_all`, `music_effects` and `music_silent`. A key the pack
leaves out shows nothing. Later milestones add keys.

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
- `armor` is the armor it adds worn. `durability` is a weapon's or armor's starting value, which
  wear takes down; `charges` the starting value of anything else. With `shared_charges` the
  charges are one count for the whole item, kept by the world, which every object of it draws on
  and which making an object of it refills; such an item may have durability besides.
- `wear` is the worn slot it goes in: `head`, `body`, `feet`, `arms`, `quiver`, `neck`, or `hand`.
  `weapon_class` (0–3) is a weapon's class, whose skill it practices.
- `actions` are up to two things the item does in hand: `{ "kind", "damage", "delay", "skill" }`.
  `kind` is `attack`, `attack_both`, `fire`, `use_up` or `use`; `damage` dice; `delay` the player
  updates the hand then waits; `skill` the class skill an attack needs (`docs/spec/items.md`);
  `sound` the sound a `use_up` action makes when it uses its object up; `swings` makes a `use` or
  `use_up` action also show the swing pose, of both hands for a `weapon`, without a blow.
- `power` is the number of the power the item starts when used (`docs/spec/items.md`, Powers), and
  with `power_worn` also when worn. `spent` is the item a used-up object leaves
  in its hand; without one the hand is left empty. A charged object is never replaced.
- `casts` makes using the item cast a spell: `{ "spell": "spell.fireball", "power": "1d6" }`, the
  power optional (the spell's own roll without it), with `message` an engine message shown as it
  casts. `conversation` makes using it start the
  conversation of a thing template; with `use_result` it is used only when that conversation
  leaves the `use_result` counter nonzero, and otherwise it is never used up. Either replaces
  `power`.
- `teaches` are the gesture icons reading a scroll teaches (`"EIB"`), `picture` the image shown
  when it is read.
- `effect` is a text key saying what the item's power does, shown in the book once the player has
  used one (`docs/spec/player.md`, The book). `lore` is a text key the book shows from first sight.
  Both are optional, and only words for the player: the engine never reads them as rules.
- `doll` is how the item shows on the inventory's figure when worn: `{ "image", "at" }`, `at` the
  pixel of the figure its image's top-left goes on (mirrored for the right hand); `doll_left`, if
  given, is how it shows in the left hand instead. `doll_over` draws it in a hand after both
  hands' other objects. An action's `icon` is the image its box shows.
- `held` is how the item looks in the player's hands:
  `{ "right": { "behind": sprite, "front": sprite }, "left": { … } }`. Each sprite has a frame for
  every frame of the player's sprite, drawn with it (an empty frame for none), and its own anchor
  at the player's ground point.
- `fill` gives a container a look for how full it is: `{ "per", "icons", "dolls" }`, step `k`
  (from 1) shown while it holds more than `(k − 1) × per` objects, its own icon and doll empty;
  the figure shows the fill's dolls only in the left hand.
- `looks` are the looks the item puts on the player's figure layers while worn (not in a hand),
  by layer name: `{ "body": { "sprite", "female" }, … }`, `female` the sprite for a female player
  when it differs.
- `missile` is how the item hurts in flight: `{ "damage": dice, "fired_only": bool }`; without it a
  thrown object does `weight / 10 + 1`. `breaks_into` is the item a landing bottle becomes, and
  `on_break` actions run when it breaks (`set_counter`, `converse`, `remove_all`, `experience`).
  `flying` is its look in flight, `{ "sprite", "fired_only" }`: a frame for each direction,
  `(direction + 1) mod 8`, shown only when fired with `fired_only`; without it the floor image.
- `poisons` makes its blows poison what they hit. `grows` makes it a counted weapon:
  `{ "into", "at", "first", "grown", "counter" }`, and `levels` the grown one:
  `{ "every", "max", "counter", "talk" }` (`docs/spec/combat.md`, Special weapons).
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
  between updates. `hostile_speed`, if set, is a creature's speed once a conversation turns it
  hostile.
- `footprint` is the side, in world units, of the square around the ground point the thing takes
  up: two things whose footprints overlap collide. Boulders and traps also test the footprint's
  corners against the terrain. 0 means 16.
- `weight` is what the thing adds to a plate it stands on; 0 presses nothing.
- `hit_points` are rolled when the thing is placed. A creature with none is placed as a corpse.
  `magic_points` (players) are rolled the same way. A player's `stats` may also give its starting
  skills, `weapon.0`–`weapon.3` and `school.0`–`school.3`, and its `inventory` the objects it starts
  with: `[{ "slot": 26, "item": "item.leather_trews" }, …]`, an object with an inventory slot
  number (`docs/spec/items.md`, Inventory). Worn ones start as if put on.
- `behaviors` (creatures): `hostile` from the start, `never_hostile`, `leaves_corpse` when it dies,
  `resists_teleport` (teleporters, pits and plates leave it alone), `invulnerable`, `poisonous`,
  `hurts_on_contact`, `death_blow`, `physical_only`, `resists_fire`, `resists_lightning`,
  `resists_poison`, `animated` (only animates in place, and cannot be struck), `stationary` (stands
  and casts). Other creatures walk.
- `attack` (creatures) is the damage dice of its blow, `experience` the dice of what killing it is
  worth, `vulnerability` the weapon class that does it extra damage (arrows too, for class 3), `spell` its magic
  attack. What the game sets in play (talked to, dead) is world state.
- `stats` are named numbers the pack's rule set reads; the engine does not interpret them, except a
  `player`'s attributes `strength`, `agility`, `endurance`, `accuracy`, `talent` and `power`,
  from which its speed is worked out (`docs/spec/player.md`, Speed), and its starting
  `interval_bonus` in ticks; a player without attributes walks at its base interval.
- `figure` (players) draws the player in layers, bottom up, in place of its sprite:
  `[{ "name", "sprite", "female" }]`. Each layer shows the look a worn item's `looks` give it, or
  its own; each sprite has a frame for every frame of the player's sprite and its own anchor.
- `portrait` (players) is the player's portrait, which also gives its figure's colors. `icons`
  (players) are the gesture icons it starts knowing.
- `conversation` (creatures) is the conversation held with it, and `talk_portrait` the portrait
  shown; the behavior `talks` lets the player talk to it, until it turns hostile. Powers and
  events start conversations with templates that no level places.
- `theme` (creatures) is a music track. It starts when the player first sees a creature of the
  template, and fades out once the player sees none.
- `spell` (creatures) is the spell it casts, and `spell_power` the dice of its power.
- `name` (creatures) is a text key, shown when the player's blow kills it unless it has a
  conversation; `kill_practice` is the practice that blow gives. The behavior `mirrors` passes
  the damage it takes to the player.
- `lore` (creatures) is a text key the book shows from first sight (`docs/spec/player.md`, The
  book).
- `animate` changes an animated creature's frame on each of its updates:
  `{ "kind": "flicker" | "toggle" | "random" | "cycle", "frames": [...], "random": bool }`
  (`docs/spec/creatures.md`, Animated objects); for `flicker` the first frame is the rest frame.
- Kinds `conjured` (a thing a spell puts up: it blocks, can be pushed, and lasts a number of its
  updates) and `portal` (walking into it takes the player where the spell that opened it says)
  are only made by spells.

## Portraits

`portraits/<name>.json`: a face, shown in panels and conversations.

```json
{
  "id": "portrait.0",
  "image": { "sprite": "sprite.portrait.799", "frame": 0 },
  "mirrored": true,
  "figure_colors": [101, 103, 106, 107],
  "other": "portrait.39",
  "talk": "portrait.39"
}
```

- `image` is the picture; `mirrored` draws it flipped left to right in the player's panel.
- `figure_colors` are palette colors: a player with this portrait has its figure's colors 224,
  225, … drawn in them, in order (its hair and skin), with the palette's own colors rather than
  its effects.
- `female` gives the sex; `other` is the portrait of the other sex, and `talk` the one a
  conversation shows for the player when the creature talked to has this portrait too. Both
  default to this portrait.

## Spells

`spells/<name>.json`: a spell (`docs/spec/spells.md`).

```json
{
  "id": "spell.flame_arrow",
  "name": "spell.flame_arrow.name",
  "school": 0,
  "cost": 5,
  "power": "1d8",
  "power_bonus": 4,
  "gesture": "DJ",
  "icon": { "sprite": "sprite.res.69", "frame": 0 },
  "effects": [
    { "missile": { "hurts": "fire", "sprite": "sprite.res.1009", "directional": true,
                   "cloud": "sprite.res.57" } }
  ]
}
```

- `school` is 0–3. `power` is the dice of its power, `power_bonus` the sides of each extra die:
  the caster gets `max(0, (power attribute − 10) / 2 + school skill)` of them.
- `gesture` is its icons, letters `A`–`L` naming the magic record's icons; a spell without one
  cannot be learned, only cast by creatures, objects and procedures.
- `needs` is a condition checked before casting, one of `{ "floor_ahead": distance }`,
  `{ "mark": rune }`, `{ "spell_near": distance }`, `{ "holding": [item, item] }`,
  `{ "creature_near": distance }`, `{ "poisoned": true }`, `{ "fatigued": true }`,
  `{ "hurt": true }`, with `"else"` an engine message shown in place of the condition's own when it
  does not hold.
- `practice` is how much more a cast practices its school than the 1 every cast does.
- `effects` are actions (Procedures, below) run when the spell takes effect, with the cast's power.
- `effect` is a text key saying what the spell does, for the book (`docs/spec/player.md`, The book);
  optional, and only words for the player.

## Magic

`magic.json`: what the spell UI draws, and the schools' names.

```json
{
  "id": "magic",
  "icons": [{ "sprite": "sprite.res.54", "frame": 1 }, "…twelve, for A–L…"],
  "panel": { "sprite": "sprite.res.54", "frame": 0 },
  "schools": ["msg.42", "msg.43", "msg.44", "msg.45"],
  "holds": { "A": "BDEIJK", "C": "" }
}
```

`icons[i]` is the icon of letter `A + i`; `panel` the empty gesture panel; `schools` the text keys of
the schools' names. `holds` gives, for an icon, the icons after which a gesture playing shows the
empty panel before it (`docs/spec/spells.md`, The gesture playing); without it each icon follows
the last at once.

## Pointers

`pointers.json`: the pointer's images, each drawn with its top-left pixel at the mouse.

```json
{ "id": "pointers", "arrow": { "sprite": "sprite.res.2", "frame": 0 },
  "talk": { "sprite": "sprite.res.2", "frame": 9 }, "wait": { "sprite": "sprite.res.2", "frame": 10 } }
```

`arrow` is the pointer over the view; `talk` over a creature that talks or a mouth, which a click
talks to; `wait` while the game waits. Without the file the system's pointer is used.

## Inventory

`inventory.json`: what the inventory panel and the hotbar's hands draw.

```json
{
  "id": "inventory",
  "doll": { "sprite": "sprite.strip.8", "frame": 0 },
  "quiver_slot": { "image": { "sprite": "sprite.strip.386", "frame": 0 }, "at": [12, 3] },
  "bare_hands": [{ "sprite": "sprite.res.50", "frame": 1 }, { "sprite": "sprite.res.50", "frame": 0 }],
  "strike_icon": { "sprite": "sprite.res.48", "frame": 8 },
  "throw_icon": { "sprite": "sprite.res.48", "frame": 0 },
  "weapon_icons": ["…four images…"], "school_icons": ["…four…"],
  "weapon_ratings": ["msg.1", "…ten, for levels 1–10…"], "school_ratings": ["msg.11", "…"],
  "level_titles": ["msg.21", "…one for each level from 0…"],
  "labels": { "weight": "msg.129", "damage": "msg.130", "damage_after": "msg.131",
              "armor": "msg.132", "two_handed": "msg.133" }
}
```

- `doll` is the figure the worn objects are drawn on, `quiver_slot` how an object in the quiver
  slot shows on it (in place of its own `doll` look).
- `bare_hands` are the right and left hands' pictures when empty; `strike_icon` and `throw_icon`
  the icons of striking bare-handed and of throwing.
- The icons, ratings and titles name the skills and levels; `labels` are the details' words.
  Every field is optional.

## Creation

`creation.json`: what character creation offers and how a new game starts
(`docs/spec/scenes.md`, The frontend).

```json
{
  "id": "creation",
  "portraits": ["portrait.0", "…"],
  "methods": [
    { "name": "Rolled", "roll": ["3d4", "2d5"], "max": 20 },
    { "name": "Spent", "points": ["28d5"], "max": 20 },
    { "name": "Rolled and spent", "roll": ["2d5"], "points": ["14d5"], "max": 20 }
  ],
  "weapon_classes": [{ "icon": { "sprite": "…", "frame": 0 }, "inventory": [{ "slot": 22, "item": "item.falchion" }] }],
  "schools": [{ "name": "msg.42", "icons": "DJ", "inventory": [{ "slot": 2, "item": "item.flame_arrow_parchment" }] }],
  "inventory": [{ "slot": 0, "item": "item.apple_of_vigor" }],
  "body_armor": ["item.leather_trews", "…"],
  "premade": [{ "name": "Kordan", "portrait": "portrait.3", "weapon_class": 0, "school": 1,
                "attributes": { "strength": 18, "…": 0 } }],
  "scene": "scene.door",
  "music": "music.3",
  "story": "scene.story",
  "story_music": "music.1",
  "ending": { "scene": "scene.ending", "counter": "var.32", "conversation": "thing.npc.121",
              "choices": [{ "name": "F", "holding": "item.chesschantra_skull", "value": 4 }] }
}
```

- `methods` roll each attribute as the sum of `roll`'s dice (0 without), at most `max`; `points`
  are then spent by hand, each attribute between its roll and `max`.
- A weapon class (by its place, 0–3) and a school have a `name` (a text key) or an `icon`, the
  skill they start at level 1, and the objects they add, by inventory slot. `icons` are the gesture
  icons a school starts knowing.
- `inventory` is what every character starts with; one of `body_armor` goes on the body.
- `music` is the track the frontend starts with its splash; `story_music` the track Create
  Character or a premade character starts, which fades out as the `story` scene ends. `story` plays
  when a character is made or chosen, before `scene` (`docs/spec/scenes.md`, The story, in
  short).
- A new game sets `counter` to 1, plays `scene`, and starts `opening`'s conversation on arrival;
  each is optional. The Summoning's import sets neither `counter` nor `opening`.
- The ending power takes the first of `ending.choices` whose `holding` item is in a hand, sets
  `ending.counter` to its `value`, runs `ending.conversation`'s conversation, and then ends play
  with the choice's `scene`; the conversations' `ending` action ends play with `ending.scene`.

## Scenes

`scenes/<name>.json`: steps the presentation plays in order on a 320 × 200 picture
(`docs/spec/scenes.md`, Scenes).

```json
{ "id": "scene.door", "steps": [
  { "picture": { "sprite": "sprite.picture.62", "frame": 0 }, "ms": 1000 },
  { "picture": { "sprite": "sprite.res.85", "frame": 0 }, "at": [135, 0], "ms": 275 },
  { "part": { "left": {…}, "right": {…}, "background": {…}, "from": 0, "to": 20, "step_ms": 110,
              "step_sounds": ["sfx.23"] } },
  { "caption": "msg.343", "banner": { "sprite": "…" }, "ms": 2500 },
  { "fade": "#ffffff", "ms": 500 }
] }
```

A `picture` draws over what is there (`clear` first to start afresh); `part` draws the background
and the two halves moved `8 × spread` pixels outward, spread going from `from` to `to`, a step
each `step_ms` (the first steps each `lead_ms`'s, if given); `caption`
shows a text at the bottom, over `banner` if given; `fade` fades to a color. `ms` holds the step.
`sounds` start as a step shows, and a part's `step_sounds` at each of its steps.

A scene is 320 × 200 unless its `size` says otherwise; one of 320 × 400 (the frontend's screen)
shows in the same place, each row half as tall. More steps, for the story:

```json
{ "id": "scene.story", "size": [320, 400], "steps": [
  { "text": "story.1", "box": [16, 60, 288, 280], "ms": 9000 },
  { "picture": { "sprite": "sprite.strip.1254" }, "at": [132, 26], "clear": true },
  { "by_weapon": [{ "text": "story.gift.0", "box": [0, 30, 132, 340], "from_top": true,
                   "ms": 6000 }, "…"] },
  { "say": { "portrait": "portrait.rowena", "side": "top", "text": "story.say.1" }, "ms": 5000 }
] }
```

- `text` shows a page of words, wrapped and centered in `box` (`[x, y, w, h]`), or from the box's
  top with `from_top`, in place of the page before; `@` is the character's name. A key or click
  ends its step early.
- `by_weapon` plays the step of its list at the character's weapon class.
- `say` shows a line in a speaker's panel, `top` or `bottom`, with `portrait` (the character's
  own when empty, mirrored as its portrait says) and clears the other panel's line; the picture
  under the panels shows darkened. A key or click ends its step early.
- A `clear` also clears the page of words and the panels; `keep_text` keeps the page.

More steps, for the end scenes:

```json
{ "id": "scene.end.e", "size": [320, 400], "escape_only": true, "steps": [
  { "palette": "palette.7" },
  { "picture": { "sprite": "sprite.end.frame.15x40" }, "at": [0, 80], "clear": true },
  { "fade": "#000000", "fade_in": true, "ms": 133, "music_fade": true, "music": "music.6" },
  { "over": [{ "sprite": "sprite.end.glow", "frame": 3, "at": [28, 108] }], "ms": 16 },
  { "caption": "end.E.magic", "box": [0, 340, 320, 50] },
  { "over": [{ "sprite": "sprite.end.figure.1463", "frame": 4, "at": [178, 72], "mirror": true }],
    "ms": 16, "cancel": true },
  { "fade": "#000000", "fade_colors": [0, 191], "ms": 133 },
  { "music_fade": true }
] }
```

- `over` draws its pictures (`sprite`, `frame`, `at`, `mirror`) over the scene's, in place of the
  last `over`'s; `[]` draws none. A `clear` clears them too. `mirror` on a `picture` flips it
  left to right as well.
- `palette` switches the palette the scene draws in, its effects running from that step (the
  pack's palette before any).
- A `caption` with a `box` shows its words centered in the box, in place of the caption before;
  `no_caption` clears it.
- `fade_in` fades in from the color rather than to it; a fade over a scene already covered in a
  color goes from that color to the new one. `fade_colors` (`[first, last]`) fades only those
  palette colors, which stay faded until a `clear`. A fade of no `ms` is done at once.
- `music_stop` stops the track playing at once, `music_fade` fades it out, and `music` starts a
  track, in that order.
- A step with no `ms` (and no page of words, line or part) goes on at once to the next, so a frame
  built of several steps shows whole.
- A key or click during a `cancel` step skips it and the `cancel` steps after it. With
  `escape_only`, only Escape skips the scene; another key or a click does nothing outside pages,
  lines and `cancel` steps.

The importer writes the end scenes as `scene.end.e`, `scene.end.f` and `scene.end.g`, named by the
ending's choices (`docs/spec/scenes.md`, End scenes), and their palettes as `palette.end` (palette
7 with colors 224–239 and 240–247 rotating) and `palette.end.green`.

## Flipbooks

`flipbooks/<name>.json`: an animation rendered ahead of time, played as a list of steps over
whatever is behind it (`docs/spec/scenes.md`, View effects and Sparkles). The importer writes the
view effects as `flipbook.effect.<n>` and the sparkles as `flipbook.sparkle.<kind>`.

```json
{ "id": "flipbook.effect.2", "sprite": "sprite.flipbook.effect.2", "framed": true, "steps": [
  { "frame": 0, "ms": 275, "cancel": true, "sounds": ["sfx.21"] },
  { "frame": 1, "ms": 275, "cancel": true },
  { "frame": 0, "ms": 17, "music": "music.15" },
  { "frame": 2, "ms": 989, "music_fade": true }
] }
```

- `sprite` holds the frames, all of the flipbook's size. The size is the sprite's frame size.
  Transparent pixels show what is behind: the view, for a view effect.
- `framed` draws a border around the flipbook.
- Each step shows `frame` for `ms`. With `cancel`, Esc or the right mouse button during the step
  ends the flipbook at once.
- As a step starts, `music_fade` fades out the music track playing, or `music_stop` stops it at once.
  Then `music` starts a track and `sounds` start effects.

A flipbook plays as a **view effect** (framed, in a box placed by a creature, `effect` events) or
a **sparkle** (unframed, over an area, `sparkle` events); the world waits while either plays.
Levels, spells and conversations name the flipbooks they play (Features, Procedures). What the
engine plays on its own it names by cue, and `effects.json` gives each cue's flipbook:

```json
{ "id": "effects", "cues": { "fall": "flipbook.effect.2", "heal": "flipbook.sparkle.heal" } }
```

The cues: `climb_down` (the player stepping down into a pit or open trapdoor, along directions
1–3), `fall` (falling into one any other way), `caster_dies` (a `stationary` creature dying, by it),
`death_blow` (a death blow landing, by the player, before the player dies), and the sparkles of the
powers that show one, `cure` (power 15h) and `heal` (powers 1Dh and 45h). A cue the pack leaves out
plays nothing, and a level change it would have held happens at once.

### Transitions and lost objects

`effects.json` also says how the presentation shows the player moving by a teleport or a change of
level, and an object breaking or running out in one of the player's slots
(`docs/spec/scenes.md`, Level transitions; `docs/spec/items.md`, Timed objects). Both are styles
the engine draws, so a pack chooses its look without art:

```json
{ "id": "effects", "cues": { "fall": "flipbook.effect.2" },
  "transitions": {
    "teleporter": { "style": "iris", "ms": 700, "hold_ms": 150, "color": "#000000" },
    "ripple":     { "style": "ripple", "ms": 750, "size": 8, "rows": 16 },
    "exit_wall":  { "style": "slide", "from": "right", "ms": 350 },
    "fall":       { "style": "squeeze", "from": "bottom", "ms": 300, "steps": 5 }
  },
  "item_lost": { "style": "shatter", "ms": 1200, "inventory": true } }
```

- The transition cues are what moved the player: `teleporter`, `ripple` (to a rune mark),
  `exit_wall`, `exit_floor` (an exit marker), `fall` (a pit or an open trapdoor) and `travel` (a
  conversation's travel or return, a portal), and `load` (a saved game loaded during play). A cue left out, a `cut`, or one of 0 `ms` shows the
  new view at once.
- `style` is `iris`, `slide`, `squeeze`, `ripple`, `fade`, `curtains` or `cut` (scenes.md says what each does),
  and `ms` how long the whole of it takes.
- `hold_ms` is how long an iris or curtains stay closed, within `ms`; `color` what an iris or
  curtains close to and a fade passes through, `#rrggbb`, black by default.
- `from` is the side a slide or squeeze comes in from: `left`, `right` (a slide's default), `top`
  or `bottom` (a squeeze's). `steps`, above 0, moves it in that many even steps.
- `size` is a ripple's largest shift in view pixels (8 by default), and `rows` how many rows each
  zag takes (16).
- `item_lost` is how an object lost from a slot goes: `shatter` (its box flashes and its image
  breaks into pieces that fly apart, fall and fade), `fade` or `none`, over `ms`. With
  `"inventory": true` the inventory panel opens for it and the world waits until it has gone;
  without, it goes where its slot shows (the panel, the hotbar, or over the player's head) while
  the game goes on. Left out, it goes with nothing shown.

## Keywords

`keywords.json`: the words the player can ask about in conversations.

```json
{ "id": "keywords",
  "keywords": [ { "id": "keyword.wizard", "word": "keyword.wizard.word", "answer": "msg.1700" } ] }
```

`word` and `answer` are text keys: the word, and what asking it says (optional). A typed word
matches the first keyword whose first four letters it shares, ignoring case.

## Conversations

`conversations/<name>.json`: a conversation, run as `docs/spec/dialogue.md` says.

```json
{
  "id": "conversation.12",
  "name": "Wizard",
  "clauses": [
    { "if": { "first_meeting": true },
      "do": [{ "say": "msg.400" }, { "offer": "keyword.wizard" }] },
    { "if": { "asked": ["keyword.wizard"] },
      "do": [{ "say": "msg.401" }, { "set_counter": { "counter": "var.12", "value": 1 } }] },
    { "if": { "counter": { "counter": "var.12", "equals": 1 } }, "do": [{ "jump": 0 }] }
  ]
}
```

A clause's `if` is one condition: `first_meeting`, `greeting`, `asked` (keywords), `never` (stops
the pass), `carries` (items), `counter` (`above`, `equals` or `below` a value), `dead` (a thing
template), `full_health`, `carries_count` (`{ "item", "count" }`), `slot_empty` (a slot),
`slot_holds` (`{ "slot", "item" }`), `always`. Its `do` are actions (Procedures, below).

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
- `upper_sight` and `lower_sight` give, for the floor under a closed door (and under a wall, when
  the wall set's shapes have no `sight`), the direction of the neighbor that must be visible for
  that half to show in play, `.` for none (the half shows whenever `floor_halves` lets it). An open
  or moving door's floor shows whole. Under a wall whose shape has `sight`, the engine works the
  halves out in play from the shape the wall has taken: each half shows while its neighbor is
  inside the level, has floor, is not listed in `no_sight`, and is visible.
- `no_sight` lists the cells whose sight never shows the floor half of a wall beside them (hazard
  tiles).
- `door_sight` follows the doors for the walls beside them whose floor halves depend on whether
  those doors are closed (`docs/spec/world.md`, Visibility): each entry has the wall's `cell`, its
  `doors` (cells of door features) and `as`, one list per combination of those doors closed, giving
  for each shape of the wall set the shape whose `sight` the wall checks by. `as[m]` applies while
  door `i` is closed (fully, not sealed) exactly for the bits `i` set in `m`.
- `overlays` are fixed floor art (hazard tiles), drawn after all floors and before anything
  standing. `decorations` are drawn on their cell's wall face right after it. `props` (not shown)
  are fixed art standing in a cell, drawn back to front with the walls. Offsets are from the cell.
- `name` is a text key, the level's name. `map` refuses the level map:
  `{ "message": text }` or `{ "conversation": thing }` (`docs/spec/world.md`, Level map).
  `footsteps` is the sound of the player's steps on the level, in place of the `footstep` cue.

### Floor types

A floor type is a floor set variant and what the floor does. All fields but `variant` are optional.

| Field | Meaning |
| --- | --- |
| `variant` | The floor set's variant drawn |
| `enter` | An effect on whoever enters: `{ "damage": dice, "type": "fire" }` or `{ "drain_magic": n }`; `"cue"` names a cue its damage to the player plays |
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
| `door` | `state` (`closed`, `open`, `opening`, `closing`), `sealed_by` (an item), `opened_by_spell`, `sparkle` (a flipbook) | | `closed`, `1` .. `n`, `open`, `sealed` |
| `lever` | `on`, `armed` | `on_turn_on`, `on_turn_off` | `off`, `on` |
| `plate` | `down`, `armed`, `threshold` | `on_press`, `on_release` | `up`, `down` |
| `slot` | `accepts` (an item), `armed`, `keyhole` | `on_insert` | `slot` |
| `teleporter` | `armed`, `destinations` (two cells), `next` 0 or 1, `turn` (eighths of a turn for things in flight), `fly_over` | | `armed`, `disarmed` |
| `exit` | `to` (a level and a cell), `effect` (a flipbook) | | `exit` |
| `pit` | `to` | | `pit` |
| `trapdoor` | `open`, `to` | | `open`, `closed` |
| `socket` | `places` (the item each place takes), `set` (the indexes of the places filled), `effect` (a flipbook) | `on_complete` | `empty`, `full`, `set.0` .. `set.n` |
| `mouth` | `message` | | `mouth` |
| `mark` | `rune` (a name) | | `mark` |

- A door's look has the stages it moves through between `closed` and `open`, numbered from 1. A
  sealed door shows `sealed` and opens only to its `sealed_by` item landing on it, playing its
  `sparkle` by the door.
- An exit's `effect` is the view effect the player plays going through it; a socket's plays, by
  the player, when its last place is filled. What follows waits for the effect: the level change,
  or the `on_complete` hook.
- An `exit` on a wall cell is in the wall's face, drawn with a `wall` layer look and taken by
  walking into the face. An `exit` on a floor cell is a marker, taken by walking onto it. A `mouth`
  stands on a floor cell, drawn up against the wall behind it.
- A `keyhole` slot takes a key: using it puts that key in from what the player carries. A slot
  without it is an object hole, which gives no hint of what it takes.
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

`shoot_spell` casts a spell from a cell along a direction, its power rolled from the dice.

Spells' effects use these actions too, with the cast's power:

| Action | Effect |
| --- | --- |
| `{ "missile": { "hurts", "opens", "scatters", "bane", "bane_damage", "sprite", "directional", "cloud", "bounces", "around" } }` | Fly a missile (`docs/spec/spells.md`, Missiles): `hurts` is `fire`, `lightning` or `poison`; `opens` opens doors and starts boulders; `scatters` moves what it hits to a random seen cell; `bane` is a thing template it hurts by `bane_damage`. `directional` draws the frame of the direction it flies; `cloud` is the sprite it ends as; `around` casts four, along the diagonals; `sound` is the sound it makes on hitting something or meeting a wall where the player sees it |
| `{ "fire_item": item }` | Fire a new object of the item from the caster |
| `{ "timed": effect }` | A timed effect for `power` status ticks |
| `{ "conjure": { "thing", "distance", "look_per_skill" } }` | A new thing half a cell ahead, lasting `power` of its updates, showing frame school skill / `look_per_skill` |
| `{ "portal": { "thing", "distance", "to" } }` | A new portal a distance ahead, replacing any other |
| `{ "teleport_to_mark": rune }` | Ripple the player to a mark |
| `{ "dispel": distance }` | End spells in flight near the player |
| `{ "join": { "items", "into" } }` | Join the two items held in the hands into one |
| `{ "freeze": ticks }` | Delay every other thing by `power × ticks` |
| `{ "shape_change": distance }` | Take the look of the nearest creature |
| `{ "transmute": { "holder", "table", "otherwise" } }` | Fill the `holder` item in a hand from the other hand's object: by `table`, using it up, or else `otherwise`, valued at the power |
| `{ "cure_poison": true }`, `{ "restore": true }`, `{ "heal": dice }` | Cure poison; recover `power` fatigue; heal |
| `{ "set_counter": { "counter", "value" } }`, `{ "converse": thing }` | Set a counter; start the conversation of a thing template |
| `{ "effect": flipbook }`, `{ "sparkle": flipbook }` | Play a view effect by the player, the spell's effects after it waiting until it ends; play a sparkle by the player |

Conversations use these:

| Action | Effect |
| --- | --- |
| `{ "say": text }` | Say a text, beside the conversation's portrait |
| `{ "say_as": { "portrait", "text" } }` | Say a text beside another portrait (none: the player's) |
| `{ "offer": keyword }` | Offer a keyword (at most six) |
| `{ "forget_offers": true }` | Empty the offered keywords |
| `{ "take": item }`, `{ "give": object }` | Take an object of an item; give a new object (with `contents`) |
| `{ "set_counter": { "counter", "value" } }`, `{ "add_counter": counter }` | Set a counter; add 1 |
| `{ "end": true }` | End the conversation |
| `{ "remove_speaker": true }`, `{ "hostile": true }`, `{ "set_frame": n }` | Remove the creature spoken to; turn it and its kind hostile; show a frame of its sprite |
| `{ "heal_fully": true }` | Cure poison and confusion and heal fully |
| `{ "run": procedure }` | Run a procedure of the current level |
| `{ "teach": icons }` | Show gesture icons and teach them |
| `{ "set_portrait": portrait }` | Change the conversation's portrait, and the template's |
| `{ "effect": flipbook }` | Play a view effect: a flipbook, in a framed box placed by the speaker |
| `{ "travel": { "level", "cell" } }`, `{ "return": true }` | After the conversation, go to a level and cell, or back to where the player was when a portal took it |
| `{ "jump": clause }` | Go on evaluating from a clause |
| `{ "ending": true }`, `{ "menu": name }`, `{ "transcript": name }` | Play the ending; open the game's menu (`quit`, `save`, `restore`, `delete`, `rename`, `leave`); print the transcript (scenes and saves) |

### Things and objects

- `things` are placed when the level is first entered. Each has a local `id`, a `thing` template and
  a `cell`. Its ground point is the cell's center, moved by `offset` in world units if given. It may
  override the template's `facing`. A creature may carry `items`; a boulder may start rolling, with
  `"rolling": { "direction": 2, "shuttle": false }`.
- `objects` lie on a cell's floor in list order. An object names its `item` and gives its `contents`
  if it is a container. The item `random` is a random object, chosen when the level is first
  entered from the templates marked `random`.

## Audio

Music and sound effects are audio files, each with a record beside it that says how it loops.

```
audio/music/<name>.json   music.<name>
audio/music/<name>.wav
audio/sfx/<name>.json     sfx.<name>
audio/sfx/<name>.wav
```

```json
{ "id": "music.1", "file": "1.wav", "intro": 529200, "loop": 1058400 }
```

- `file` is the audio file, in the same directory: a 16-bit PCM WAV (mono or stereo, any rate) or
  an Ogg Vorbis file.
- `intro` and `loop` count samples (per channel). The sound plays its first `intro` samples, then
  repeats the `loop` samples after them until it is stopped. With `loop` absent or 0 it plays the
  whole file once. `intro + loop` must not run past the file.
- Music ids start with `music.` and effect ids with `sfx.`. One music track plays at a time;
  effects, and the sounds cues play, sound over it and over each other.

`audio/cues.json` gives the sounds of the engine's cues, the sounds the simulation makes:

```json
{ "id": "cues", "cues": { "lever": "sfx.9", "level_up": "music.10" } }
```

The cues: `footstep` (every third walk step, not while levitating), `lever` (a lever pulled),
`door` (a door moving), `unseal` (a sealed door opening), `teleport`, `level_up`, `weapon_skill_up`,
`magic_skill_up`, `miss` (the player's blow fails or does no damage), `hit` (it hurts), `hurt` (the
player hurt by a creature's blow, a boulder or a closing door, or a floor effect naming it), `bow`
(a bow shot, an update after the arrow leaves), `shatter` (a thrown object breaking where the
player sees it), `take` (an object taken from the floor or exchanged with a slot), `drop` (one put
down), `cast` (a spell's cast starting) and `gesture` (an icon added to or taken off the gesture
being entered, a gesture entered, the spell panel opened). A cue the pack leaves out is silent.

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
  and 3; `sight` from Floor halves; `unseen` from Walls out of sight). Hazard tiles become
  `no_sight`. Each level's floor types come from its floor variants and liquid kind.
- Item categories become tags. The original's message numbers become `messages.json` entries.
- Each song is rendered through an OPL2 emulator (`docs/spec/audio.md`) to a mono 16-bit WAV at
  44100 Hz. Track `n` becomes `music.n` and sound effect `n` becomes `sfx.n`. A track the game
  loops keeps two passes: the first is the intro and the second, which starts with the first's
  notes still ringing, is the loop. A song that plays once keeps its release until it is silent.
  The cues name the tracks and effects the game plays; the start's `music`, the theme of the
  creature the game plays a track for, and the levels with other footsteps are set the same way.
  `import --no-audio` skips rendering.

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
