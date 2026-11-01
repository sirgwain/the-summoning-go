# The Summoning: specification

This directory describes *The Summoning* (Event Horizon Software, 1992): what the game does and how its data files are laid out. It is the source of truth for the engine's Summoning rules module and for the importer. Someone with these files and a copy of the game's data must be able to build both.

| File | Covers |
| --- | --- |
| [formats.md](formats.md) | Every data file: containers, sprites, fonts, music, text, levels, scripts, records, palettes |
| [world.md](world.md) | Levels, cells and terrain, the projection, drawing a level, visibility, liquids |
| [entities.md](entities.md) | Entity kinds, updates and timing, movement and collision |
| [rules.md](rules.md) | Cell features, triggers, teleporters, doors, level procedures |
| [player.md](player.md) | Attributes, hit and magic points, levels, skills, status effects, speed and load |
| [items.md](items.md) | Object templates, inventory slots, containers, powers, throwing |
| [creatures.md](creatures.md) | NPC records, creature behavior, experience |
| [combat.md](combat.md) | To-hit, damage, damage kinds, resistances, wear |
| [spells.md](spells.md) | Schools, casting, gestures, every spell's effect |
| [dialogue.md](dialogue.md) | Keywords, conversation scripts, interaction variables |
| [scenes.md](scenes.md) | View effects, story screens, endings, palette animation |
| [audio.md](audio.md) | The music data and how it plays |
| [saves.md](saves.md) | The game's save files, and importing them as engine saves |

Each file gives the engine model, the mapping from the game's data, the special cases, and an **Unknowns** list. A section is written before the milestone that implements it, and corrected whenever testing against the game shows it wrong.

## Conventions

- Numbers ending in `h` are hexadecimal (`1Bh` = 27); others are decimal.
- Multi-byte integers in files are little-endian. `u8`, `u16`, `u32` are unsigned; `i8`, `i16` signed.
- `NdM` is the sum of N rolls of a die with faces 1..M. `rand(n)` is a uniform integer in 0..n−1. "A 1 in n chance" is `rand(n) == 0`.
- Division of integers truncates toward zero unless a section says otherwise.
- **Cells.** A level is a grid of cells addressed `(x, y)`: `x` is the row (first coordinate in the files), `y` the column. Cell `(x, y)` covers world units `x*32 .. x*32+31` by `y*32 .. y*32+31`. A position's cell is `(wx >> 5, wy >> 5)`.
- **Ticks.** The game's clock runs at 18.2 Hz ("ticks"). Every duration given in ticks means these. The engine runs its simulation at 60 Hz and converts: a duration of `t` ticks is `t * 60 / 18.2` steps, rounded to the nearest step.
- **Directions** are 0..7. The world delta of one step in each, as `(dx, dy)` in cells:

  | Dir | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | dx | −1 | −1 | 0 | 1 | 1 | 1 | 0 | −1 |
  | dy | 0 | 1 | 1 | 1 | 0 | −1 | −1 | −1 |

  On screen, the odd directions are the screen axes: 1 right, 3 down, 5 left, 7 up. The even directions are the screen diagonals. The table is also stored in INIT (formats.md).
- **Resource ids** are indexes into RESOURCE. Several groups are addressed as a base plus an offset; the bases are in INIT and named there (`terrain base`, `npc base` and so on).
- **Text ids** are indexes into TEXT. A "message n" is TEXT record n unless a section says it comes from the level's text pool.
