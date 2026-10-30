# Engine design

Status: draft, 2026-10-30.

A Go engine, over Ebitengine, for real-time, single-character, grid-based fantasy RPGs in the style of
*The Summoning* (1992): an isometric view of a dungeon, palette pixel art, a character panel,
real-time combat and spellcasting, NPC conversation by keyword, and puzzle-heavy levels of
pressure plates, levers, keyholes, teleporters and movable bodies and objects.

The engine is general. *The Summoning* is its first game: a separate import step converts the
original's data (levels, NPCs, objects, text, art, music) into the engine's content format. The
result plays like *The Summoning* without reproducing its code. Its rules start from the original's
as the specification describes them; exact numeric reproduction is not a goal.

The engine is built from its specification in `docs/spec/`, which describes the game in its own
terms and is complete enough to implement from on its own.

## Goals

- **A general engine for this kind of game.** Nothing in the engine is specific to *The Summoning*;
  its content and its rules configuration live in a content pack.
- **Data driven.** Maps, entities, items, spells, dialogue, rules and the numbers behind combat load
  from content. A new game or a mod is content, not engine code.
- **Easy to edit and mod.** Plain JSON and indexed PNGs, stable IDs, override packs, and an editor
  built on the same rule model the engine runs.
- **The Summoning's look and feel.** Palette pixel art with palette effects, the isometric view,
  panel-style UI.
- **Bigger screens.** A wide world view, collapsible panels, a quickbar, UI text at native resolution.
- **A Summoning importer** that brings over the story, levels, NPCs, objects, puzzles, art and music.

## Non-goals

- Fidelity to the original's code, timing, 16-bit arithmetic or random numbers.
- Reading or writing the original's save files.
- A scripting language for now. Rules are data; a scripting layer could later add conditions and
  actions to the same registry.
- Shipping the original's data. See [Distribution](#distribution).

## Overview

```
 Summoning files ──► importer ──► content pack ◄── mods, editor
                                       │
                                       ▼
                 commands ──►  ┌──────────────┐ ──► events (sound, message, effect)
                               │  simulation  │ ──► state for render and UI
                               └──────────────┘
                                       │
                         ┌─────────────┴─────────────┐
                         ▼                           ▼
              render (palette shader)        UI, audio, input
```

- **The simulation never knows about the screen.** It takes commands, advances on a fixed timestep
  and emits events. Rendering and UI read its state.
- **Content is the only input.** The engine reads content packs and nothing else. Only the importer
  reads *The Summoning*'s files.
- **The engine defines mechanisms; content defines the game.** The engine knows what a pressure
  plate, a stat, a spell effect or a damage roll is. Content says which plates exist, which stats a
  character has and how much damage a sword does.

## Repository layout

```
cmd/                   cobra commands: play, import, pack, verify
cmd/editor/            the editor, a separate binary built on engine/render and engine/ui
engine/sim/            the simulation: world, entities, movement, rules, spells, AI, inventory,
                       dialogue, scenes. No Ebitengine import.
engine/combat/         the combat interface and the rule sets that implement it
engine/combat/summoning/  the original's calculations
engine/content/        pack schema (Go structs), loader, validation, mod overlay
engine/render/         palette shader, sprite atlases, world draw list, effects
engine/ui/             panels, quickbar, windows, text, pointer, gestures
engine/audio/          music and effects
engine/input/          commands and bindings
engine/save/           save format
importers/summoning/   reads the original files, writes a content pack
docs/
```

`engine/sim` builds and tests without Ebitengine, so tests, the importer and tools run without a
display.

## Simulation

### Shape

```go
type World struct { /* plain data; records refer to each other by ID */ }

func New(p *content.Pack, seed uint64) *World
func (w *World) Update(cmds []input.Command) []Event // one fixed step
```

- **Fixed timestep** at 60 Hz. Durations in content are in milliseconds or ticks, never frames of
  the renderer.
- **Seeded RNG** owned by the world. That makes replays, tests and bug reports reproducible. Exact
  reproduction across versions is not promised.
- **Plain-data state.** Records refer to each other by ID, so a save is the state encoded, and the
  editor can inspect anything.
- **Events** tell the presentation what to do: play a sound, show a message, start an effect, open a
  conversation, change level, start a scene.
- **Scenes** cover the non-dungeon parts: intro, character creation, story screens, endings. They are
  content (images, text, choices, actions) run by the same engine.

### The world

- A **level** is a grid of cells. A cell has a terrain type: open floor, a solid wall, a diagonal wall
  that fills half the cell, or a special floor (water, pit, slippery, damaging…). It also holds a list
  of objects lying on it.
- **Entities** (the player, NPCs, monsters, projectiles, rolling balls, movable bodies) have a
  position in world units, a facing, a kind and a set of data-driven attributes.
- **Objects** (items) live on cells, in containers or in inventories. Each has a template from
  content, with weight, slots, uses and effects.

### Movement

Movement is free, not cell to cell, as in the original. The grid is the unit of terrain and triggers,
not of position.

- **World units.** A position is in world units, 32 to a cell (the original's world pixels), held as
  integers. An entity moves in eight directions by a step from its kind.
- **Standing cell.** The cell an entity stands on is the cell under its feet point, an offset from its
  position set per kind. That cell decides terrain effects and fires enter and leave events, so a
  character can stand partly over a neighboring cell without triggering it.
- **Terrain.** A move is tested at the feet against the terrain. If blocked, the step shrinks a unit
  at a time until it fits, so a character slides up to a wall rather than stopping short.
- **Entities.** Each kind has a collision rectangle. A move that would overlap another entity's
  rectangle backs off.
- **Diagonal walls.** Walking diagonally into a diagonal wall slides along it.
- **Pushing.** The player pushes pushable things (bodies, boulders, conjured objects) ahead,
  recursively, at an exertion cost.
- Projectiles and rolling things test their corners instead of a feet point.

The step sizes, rectangles and feet offsets are content, per kind.
- **Visibility and activity.** What the player can see (line of sight on the grid) and which entities
  are active (a radius around the player) are simulation concepts, independent of how large the view
  on screen is.

### Rules: triggers and puzzles

Puzzles and story logic are rules: an event, conditions and actions.

```json
{ "id": "l3.plate_gate",
  "on": { "weight_changed": { "cell": [12, 7] } },
  "if": [ { "weight_at_least": { "cell": [12, 7], "value": 50 } } ],
  "do": [ { "open": "door.l3.east" }, { "sound": "sfx.gate" } ],
  "else": [ { "close": "door.l3.east" } ] }
```

- The engine has a **registry** of events (enter cell, leave cell, weight changed, object placed,
  item used on, lever pulled, keyword spoken, entity died, timer, level entered…), conditions (has
  item, flag set, counter compare, weight at least, entity alive, stat at least…) and actions (open,
  close, toggle, teleport, spawn, remove, give, take, set flag, add to counter, message, start
  conversation, play sound, start scene, change level…).
- Rule state (flags, counters, timers) is in the world's state, never in the rule.
- The editor builds rules by choosing from the registry. Validation checks every reference.
- Standard puzzle pieces (pressure plate, lever, keyhole, object hole, teleporter, door, magic mouth,
  rolling ball) are content templates built from rules, so a level author places a pressure plate and
  sets a few fields rather than writing rules by hand.

### Characters, stats and combat

- **Attributes** are declared in content (for the Summoning: strength, dexterity, intelligence and so
  on, hit points, magic points, weapon skills, magic classes). The engine treats them as named
  numbers.
- **Combat is a module.** `engine/combat` defines an interface the simulation calls: attack timing,
  whether an attack hits, damage, armor and resistances, experience on a kill, and the effect of
  exertion and wounds. A rule set implements it, and a content pack names the rule set it uses and
  gives its parameters.
- **The first rule set is the Summoning's** (`engine/combat/summoning`), implementing the formulas
  in `docs/spec/combat.md` with their constants as parameters.
- A game that wants different combat writes another rule set or tunes the parameters.
- **Leveling** and character creation (point allocation, classes, premade characters) are content.

### Spells

- A spell is content: its gesture, cost, requirements (skill class and level), targeting
  (self, direction, cell, entity) and a list of effects drawn from the same action registry as rules,
  plus projectile, area and duration effects.
- Gestures are learned, not just looked up. The player knows a set of gesture icons (from scrolls,
  conversations and the starting school) and enters gestures in the spell-learning UI. A gesture that
  matches any spell, read or not, puts the spell in the player's spellbook. After that the spellbook
  lists it, and memorizing it is one click instead of re-entering the gesture.
- Memorizing and casting keep the game's rules: per-school capacity, random forgetting past it, one
  memorization used per cast, the cost, the conditions and the gesture's cast time
  (`docs/spec/spells.md`).
- The UI sends the commands (`learn gesture`, `memorize spell`, `cast`). Hotkeys and the spell
  panel send the same `cast` command.

### NPCs and dialogue

- Conversation is keyword driven: an NPC has topics keyed by keywords, each with text, conditions and
  actions (give, take, set flag, join, turn hostile). New keywords are learned from text.
- **AI** is a small set of behaviors chosen by content (wander, guard, chase, flee, keep distance and
  cast, follow a path), each with parameters.

## Content pack

```
pack.json                  id, version, dependencies, load order
palettes/                  palettes and remap tables
sprites/<bank>.png         8-bit indexed PNG sheet; <bank>.json frames, offsets, animations
tiles/                     floor and wall sets
fonts/
text/<lang>.json           strings by ID
attributes.json            the stat model
items/<id>.json            item templates
creatures/<id>.json        NPC and monster templates
spells/<id>.json
templates/<id>.json        puzzle-piece templates (plate, lever, teleporter…)
levels/<id>/level.json     grid, placements, rules
scenes/<id>.json
audio/music/<id>.ogg       with loop points in a sidecar
audio/sfx/<id>.ogg
```

- **Stable string IDs** (`creature.guard`, `item.healing_potion`). Mods override and add by ID.
- **Indexed PNGs** keep palette effects working on edited art. A true-color sprite is allowed and
  simply opts out of palette effects.
- **Levels** are the engine's own JSON, edited with the editor.
- A level's `map` setting is `shown` (the default), or `refused` with either a message ID or a
  conversation ID, which runs when the player tries to open the map.
- `verify` checks a pack: schema, every ID resolves, every rule uses registered events, conditions and
  actions, every sprite frame exists.

## Rendering

- **Palette shader.** Indexed sprites upload as single-channel textures, palettes as a 256×N texture,
  and a Kage shader looks up colors. Fades, flashes, remapped creature colors and color cycling change
  the lookup, not the art.
- **World buffer.** The world draws into an offscreen image at the art's pixel scale, then scales to
  the window by a whole number. UI draws at native resolution on top.
- **Projection.** Pixel-art isometric (2:1), as the Summoning draws: the grid is turned 45 degrees,
  each cell a 64×32 diamond, and a world point (x, y) lands on screen at
  ((y − x), (x + y) / 2) from the view's origin. Walls rise from their cell's diamond. The cell size
  is content.
- **Draw order.** Floors first, then walls, objects and entities in one list sorted back to front
  by their depth (x + y of their feet), each frame.
- **Walls** are one sprite per cell, from a **wall set**: one sprite for each wall shape (straight,
  corner, end, diagonal…), with its face, upper part and edge caps already composed. The importer
  composes the original's pieces into wall sets. A wall whose look changes in play (a door, a secret
  wall) has a sprite per state. The level stores each wall cell's shape, so drawing a wall is a
  straight lookup: wall set + shape (+ state) → sprite.
- **Wall decorations** (torches, banners, carvings) are separate, possibly animated sprites placed on
  a wall face and drawn right after their wall. They are never baked into wall art.
- **View size is free.** A wider window shows more of the level; it does not change the simulation.
- Optional post shaders (CRT, scanlines) are presentation only.

## UI

- Built from panel art in content, as resizable frames, so it keeps the Summoning's look.
- **Text** uses a custom outline font drawn for the project to match the original's lettering,
  rendered at native resolution with Ebitengine's `text/v2`. The font is content, so a game can
  supply its own.
- Collapsible character, inventory and spell panels; the world view takes the freed space.
- A quickbar for potions and other usable items, bound to number keys.
- Spell gestures with the mouse, plus hotkeys and a spell list.
- Conversations with keyword entry and a list of known keywords.
- A level map, always available, with no item needed: a minimap in a corner of the view, and a
  full-screen map on a key. Both show only the cells the player has seen. A level can refuse the
  map (`docs/spec/world.md`): the minimap then hides, and opening the map shows the level's message
  or starts its conversation instead.
- The UI only sends commands; it never changes game state directly.

## Input

`input.Command` names what the player does (move, turn, use, attack, cast, pick up…). Keys, mouse and
gamepad bind to commands in a user configuration. Replays record commands.

## Audio

- Music and effects play from Ogg files with loop points.
- The Summoning's music and effects are AdLib (OPL) sequences. The importer renders them once: it
  plays each sequence through an OPL emulator and encodes the result to Ogg with loop points. The
  sequence format and how it plays are in `docs/spec/audio.md`.
- How many sounds play at once, and which take priority, is a mixer setting.

## Saves

- A save is the world state, versioned and encoded as compressed JSON.
- It records the content packs and versions it was made with, and warns on a mismatch.
- Quick save, quick load and autosave on level change.

## The Summoning importer

`importers/summoning` reads a user's copy of the game's twelve data files and writes a content pack.

- **Formats.** The importer reads the files as `docs/spec/formats.md` describes them.
- **Meanings.** Each spec maps the original's fields to the engine's model; the importer follows
  those mappings.
- **Mapping, not copying.** Original records become engine content: cells and walls to the level grid,
  OBJECTS to item templates, NPC to creature templates, TEXT and KEYWORDS to strings and dialogue,
  INTERACT and the level procedures to rules and puzzle templates, RESOURCE sprite banks to indexed
  PNG sheets, COLORS to palettes.
- **Summoning defaults.** The importer also writes the Summoning's stat model, combat parameters,
  spells and character creation, as the specs give them.
- **Report.** Each import writes a report of anything it could not map, so gaps are visible.

## Editor

A separate program (`cmd/editor`) for building content packs. It shares the engine's renderer, UI
widgets and mouse handling, so a level looks the same in the editor as in the game, and it reads and
writes the same pack format.

- **Levels.** Paint terrain, place walls, objects, creatures and puzzle templates, and see the level
  in the game's projection with its animations running.
- **Walls.** The author paints wall cells; the editor picks each cell's shape from its neighbors
  (straight, corner, end, diagonal) so ends and corners are always capped, and stores the shape in
  the level. The game never works shapes out at run time.
- **Decorations.** Placed on wall faces and shown animating, so the author sees the level's feel.
  The editor allows a decoration only on a face that borders open floor.
- **Rules.** Build rules from the registry's events, conditions and actions, with references picked
  from the pack.
- **Content.** Edit items, creatures, spells, dialogue and text as forms over the pack's JSON.
- **Play-test.** Start the simulation from the level being edited.
- It runs `verify` continuously and shows broken references where they are.

## Specification

`docs/spec/` describes the game the engine runs and the files the importer reads. It is the source
of truth: code implements the spec, and a change in behavior starts as a change to the spec.

- **Self-contained.** An implementer with only the spec and a copy of the game's data files can build
  the engine and the importer. The spec cites no other implementation.
- **In its own words.** Behavior is described as rules, tables and math, not as code.
- **Each topic** gives the engine's model (which becomes the content schema), how the original's
  data maps onto it, any special cases the game applies to particular levels, creatures or objects,
  and what is still unknown.
- **Written ahead of the milestone that needs it.**

| File | Covers |
|---|---|
| `formats.md` | The game's data files, byte by byte, and their compression |
| `world.md` | Levels, cells, terrain, wall shapes, special floors, visibility, level links |
| `entities.md` | Entity kinds, positions, collision, feet points, movement, pushing |
| `player.md` | Attributes, hit and magic points, exertion, experience, skills, character creation |
| `items.md` | Item templates, equipment, containers, potions, keys, uses |
| `creatures.md` | Creature templates, AI behaviors, loot |
| `rules.md` | Events, conditions, actions, and the puzzle templates built from them |
| `combat.md` | The combat interface and the Summoning's formulas |
| `spells.md` | Spells, gestures, costs, targeting, effects |
| `dialogue.md` | Keywords, topics, NPC reactions |
| `scenes.md` | Intro, character creation, story screens, endings |
| `audio.md` | The music sequence format and how it plays |

## Distribution

The original's data is copyrighted and is not in this repository. The importer reads the user's own
copy and writes the pack locally; that pack is not committed or distributed. The engine, the importer
and original content made for the engine can be distributed. Tests use small hand-made fixtures, not
original data.

## Milestones

1. **Viewer.** Import levels, sprites and palettes; draw a level with the palette shader in the
   isometric projection; move a camera.
2. **Walk.** Player entity, movement and collision, objects on cells, pick up and drop.
3. **Puzzles.** Rule registry, puzzle templates, imported rules; a level whose puzzles can be solved.
4. **Fight.** Attributes, combat pipeline, creatures and AI, projectiles.
5. **Magic and talk.** Spells and gestures, dialogue and keywords.
6. **Game.** Scenes, character creation, saves, audio; *The Summoning* playable from start to end.
7. **Remaster UI.** Wide view, panels, quickbar, hotkeys.
8. **Modding.** Override packs and `verify`, then the editor.

## Decisions

- **Movement:** free movement in world units, as the original; the grid holds terrain and triggers.
- **Combat:** a module behind an interface; the first rule set is the original's formulas.
- **Editor:** a separate program sharing the engine's rendering, UI and mouse handling.
- **Fonts:** a custom outline font drawn for the project, at native resolution.
- **Projection:** 2:1 pixel-art isometric, as the original.
- **Walls:** composed into one sprite per wall shape on import; the editor chooses shapes, the
  renderer only looks them up. Decorations stay separate, animated sprites.
- **Wall caps:** each wall draws only its own caps. The engine redraws the whole view back to front
  each frame, so a wall in front always covers the one behind it.
- **Spells:** learned once by entering the gesture, then memorized from a spellbook. The
  memorization and casting rules stay the game's.
- **Map:** built into the UI rather than tied to the Palimpsest. Levels can refuse it with a
  message or a conversation.
