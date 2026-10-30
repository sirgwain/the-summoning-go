# Entities

Everything in a level that moves, animates or can be used is an entity: the player, creatures, things in flight, boulders and the rest. Objects lying on the floor are not entities (items.md).

## Kinds

| Kind | What |
| --- | --- |
| 1 | The player |
| 2 | Creatures and animated objects, made from NPC records (creatures.md) |
| 3 | Things in flight: thrown and fired objects, spell missiles (items.md, spells.md) |
| 4 | Boulders |
| 5 | Traps: pushable blocks in the trap sprite bank |
| 6 | Level exit markers |
| 7 | Mouths: wall faces that speak a level message when used |
| 8 | Conjured things: Magic Wall and similar timed entities |
| 9 | Wall spells: the Gateway portal |

A level holds at most 100 entities, the player included. The player belongs to no level and goes with it from level to level.

An entity has a position (world units, the top-left of its sprite; world.md), a kind, a sprite bank and frame, a direction, an update interval in ticks, its next update time, and kind-specific state.

## Updates

Each entity is updated when its next update time arrives, and then rescheduled `interval` ticks later. Each kind's update is described with the kind. The player's interval comes from its speed (player.md); a creature's from its NPC record.

Entities are **active** only while near the player: while their reference point lies within a window around the view, 64 pixels beyond its top and left and the original view's 256×544 extent (rows −64..255, columns −64..543, relative to the view origin). An inactive creature waits an extra 9–17 ticks before its next update. The engine keeps this window fixed in world size, whatever the display size, so creatures wake at the same distance.

## Standing cell

The cell an entity stands on is the cell of its position plus an offset by kind:

| Kind | Offset (u, v) |
| --- | --- |
| 3 in flight | (10, 20) |
| 4 boulder, 5 trap | (8, 32) |
| 9 wall spell | (24, 40) |
| 6 exit marker, 7 mouth | (16, 24) |
| 1 player, 2 creature, 8 conjured | (10, 28) |

Entering a new standing cell runs the cell's effects: weight plates, teleporters, pits (rules.md).

## Placement

Where each kind is placed in cell `(a, b)`:

| Kind | Position |
| --- | --- |
| Creature | (32a, 32b) |
| Boulder, trap | (32a + 8, 32b − 16) |
| Exit marker, mouth | (32a − 16, 32b − 24) |

A trap is a kind-5 entity with the level's trap sprite bank, interval 128. A boulder has the level's boulder bank, speed 8, interval 4. An exit marker uses sprite bank 99, a mouth bank 56; both have interval 100.

Placing an entity where something already stands crushes it: when the player is placed (teleporting, changing level), whatever stands on it is removed, creatures dropping what they carry; another entity placed on a blocked spot is removed itself.

## Movement

Movement is free, in world units.

**Feet.** The player and creatures move their feet point, `(u + width/3, v + height − 4)` with `width, height` by kind: player (32, 32), creature (32, 32). A move by `(du, dv)` tries the `u` step first, shrinking it by 1 toward 0 until the feet point's cell accepts it; then the `v` step the same way, with the original `u`. A cell accepts the feet if the terrain is walkable for that entity (world.md, rules.md) and entering it does not change the level.

**Other entities.** Each kind has a collision rectangle relative to its position, `(top, left, height, width)` along `(u, v)`:

| Kind | Rectangle |
| --- | --- |
| 1 player | (28, 8, 6, 16) |
| 2 creature | (28, 8, 12, 16) |
| 3 in flight | (20, 4, 8, 8) |
| 4 boulder | (24, 8, 24, 28) |
| 5 trap | (28, 0, 20, 44) |
| 6 exit marker | (42, 0, 10, 16) |
| 7 mouth | (42, 4, 10, 8) |
| 8 conjured | (28, 8, 12, 16) |
| 9 wall spell | (42, 8, 10, 16) |

Rectangles overlap when their open intervals overlap on both axes. After fitting the terrain, a move that would overlap another entity is shortened by 2 units per axis toward zero until it does not, or to nothing.

**Pushing.** When the player walks into a trap block, a creature corpse (a creature with no hit points) or a conjured thing, it pushes it the same distance, recursively through a line of pushable things. A push costs 3 ticks, and with a 3 in 8 chance adds exertion (player.md). A levitating player cannot push corpses.

**Sliding.** When the player moves diagonally into a diagonal wall (shapes 0, 2, 4, 5), the player slides 4 units along it.

**Distance.** The distance between two entities is half the Manhattan distance between their positions. A creature farther than 320 from the player does not move.

**Corner tests.** Boulders, traps and conjured things test four corners instead of feet. For a move to `(u', v') = (u + du − 8, v + dv + 8)`, all of `(u' + 12, v' + 20)`, `(u' + r, v' + 20)`, `(u' + 12, v' + d)`, `(u' + r, v' + d)` must be accepted, with `(r, d)` = boulder (27, 35), trap (23, 31), conjured (19, 23). Things in flight test `(u + 10, v + 20)`.

## Boulders (kind 4)

A boulder rests (state 0) until a level procedure starts it (rules.md). Rolling, it moves 8 units each update along one of the four world axes: direction table entries 0, 2, 4, 6 (boulder direction 0..3). Its frame cycles through 0–4 for boulder directions 0 and 2, 5–9 for 1 and 3, forward for directions 0, 1 and backward for 2, 3.

When blocked, it hurts what blocked it 3d4. In state 1 it then stops; if it hit another boulder, that one starts rolling in the same direction. In state 2 it reverses and keeps rolling (shuttling back and forth), also starting a boulder it hit. A boulder that enters a teleporter is moved by it.

## Simple kinds (5–9)

They do nothing on their own, except:

- A thing on a teleporter is moved by it.
- Conjured things with a lifetime count it down each update and vanish at 0, leaving any weight plate.
- Sprite bank 60 entities cycle frames 0, 1, 2, 1.

Using a mouth (kind 7) shows its level message (formats.md). Using a creature talks to it (dialogue.md).

## Unknowns

- The player's spawn offset within the arrival cell when changing level, and the scroll the view does on arrival (three styles: drop, rise and slide), are specified with scenes.
