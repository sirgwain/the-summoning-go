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

## Drawn between updates

The game draws each entity where it is, so a walking figure jumps a step, and a thing in flight 8 units, at each of its updates. The engine draws the player, creatures, boulders and things in flight gliding instead, and this changes nothing in play. At its update, step `m`, an entity notes where it is. Until its next update, step `n`, it is drawn at step `s` back from where it is now toward there on screen by the fraction `(n − s) / (n − m)` of the way between their screen points (world.md, Projection), each pixel coordinate rounded toward where it is; from step `n` on, where it is. Working on screen keeps a step along a screen axis on its line: a step screen left or right, 4 units along each world axis, stays on its row throughout. **Letting go.** At a world update in which no walk is asked for, the player's glide, if more than 2 steps of it are left, is made to finish within 2 steps from where it is drawn: the same part of the way remains, over 2 steps, with its whole span rounded to the nearest step. So the player stops where the keys are let go, not up to an update later. A move of more than 16 units along either axis since its update (a teleporter, being placed, a level entered) is drawn as a jump, and so is everything until its first update, and after a saved game is restored. What the player pushes glides with it: it notes where it was as it is pushed, and glides until the player's next update, the push's longer wait included, as does the player; when the keys are let go it finishes with the player. Its frames change as they do in the game, at its updates; only its place glides. The view follows the player where the player is drawn.

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

A trap is a kind-5 entity with the level's trap sprite bank, interval 128. A boulder has the level's boulder bank, speed 8, interval 4. An exit marker uses resource 99, a mouth resource 56; both have interval 100. Both stand on floor cells: a mouth is drawn up against the wall behind it.

Placing an entity where something already stands crushes it: when the player is placed (by a teleporter, a rune mark, changing level), whatever stands on it is removed, a creature, corpse, boulder or trap alike, living creatures dropping what they carry; another entity placed on a blocked spot is removed itself.

## Movement

Movement is free, in world units.

**Feet.** The player and creatures move their feet point, `(u + width/3, v + height − 4)` with `width, height` by kind: player (32, 32), creature (32, 32). A move by `(du, dv)` tries the `u` step first, shrinking it by 1 toward 0 until the feet point's cell accepts it; then the `v` step the same way, with the original `u`. A cell accepts the feet if the terrain is walkable for that entity (world.md, rules.md) and entering it does not change the level.

**Other entities.** Each kind has a collision rectangle in view pixels, `(top, left, height, width)` from where its sprite's top-left is drawn (the reference point of its position, world.md), so it lies around the figure's feet:

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

Rectangles overlap when their open intervals overlap on both axes. After fitting the terrain, a move that would overlap another entity is shortened by 2 units per axis toward zero until it does not, or to nothing. A creature with hit points, and a levitating player, pass over corpses: a corpse does not count for their moves.

**Pushing.** When the player walks into a trap block, a creature corpse (a creature with no hit points) or a conjured thing, it pushes it the same distance, recursively through a line of pushable things. A pushed thing is not blocked by what it already overlaps. A push costs 3 ticks, and with a 3 in 8 chance adds exertion (player.md). A levitating player passes over corpses instead of pushing them.

**Sliding.** When the player moves diagonally into a diagonal wall (shapes 0, 2, 4, 5), the player slides 4 units along it.

**Distance.** The distance between two entities is half the Manhattan distance between their positions. A creature farther than 320 from the player does not move.

**Corner tests.** Boulders, traps and conjured things test four corners instead of feet. For a move to `(u', v') = (u + du − 8, v + dv + 8)`, all of `(u' + 12, v' + 20)`, `(u' + r, v' + 20)`, `(u' + 12, v' + d)`, `(u' + r, v' + d)` must be accepted, with `(r, d)` = boulder (27, 35), trap (23, 31), conjured (19, 23). Things in flight test `(u + 10, v + 20)`.

## Boulders (kind 4)

A boulder rests (state 0) until a level procedure starts it (rules.md). Rolling, it moves 8 units each update along one of the four world axes: direction table entries 0, 2, 4, 6 (boulder direction 0..3). Its frame cycles through 0–4 for boulder directions 0 and 2, 5–9 for 1 and 3, forward for directions 0, 1 and backward for 2, 3.

When blocked, it hurts what blocked it 3d4. In state 1 it then stops; if it hit another boulder, that one starts rolling in the same direction. In state 2 it reverses and keeps rolling (shuttling back and forth), also starting a boulder it hit. A boulder that enters a teleporter is moved by it.

## Simple kinds (5–9)

They do nothing on their own, except:

- A thing on a teleporter is moved by it, to the center of the destination cell; if the player or another thing's box overlaps its own there (a living creature leaving corpses out), it is crushed instead: removed, dropping what it carries on that cell.
- Conjured things with a lifetime count it down each update and vanish at 0, leaving any weight plate.
- Entities using resource 60 cycle frames 0, 1, 2, 1.

Using a mouth (kind 7) shows its level message (formats.md). Using a creature talks to it (dialogue.md).

## Engine model

The engine places a thing by its **ground point**, the point that decides its cell, and draws its sprite with an **anchor**, the pixel of the sprite drawn at the ground point.

- A thing's ground point is its position plus its kind's standing offset (above).
- Its sprite's anchor is where the ground point falls relative to the sprite's top-left. For a standing offset `(su, sv)` (both even) that is `(sv − su, su/2 + sv/2)` pixels.
- A thing placed in a cell has its ground point at the cell's center, `(32a + 16, 32b + 16)`, moved by an offset where the placement above puts it elsewhere.

| Kind | Anchor | Offset from the cell's center |
| --- | --- | --- |
| Creature | (18, 19) | (−6, 12) |
| Boulder, trap | (24, 20) | none |

Exit markers and mouths do not move; the engine models them as cell features drawn at `(8, −20)` from the cell's tile box (rules.md, Import).

The player stands like a creature: its standing offset and feet are both `(10, 28)`, so its ground point is its feet and its sprite's anchor is `(18, 19)`.

### Directions and facing

The engine keeps every facing, the player's included, as a direction of the README's table. The player's facing `p` (player.md) is table direction `d = (p + 7) mod 8`. The even directions move along one world axis (the screen diagonals); the odd ones move along both (the screen axes).

### Collision boxes

Each thing template has a **box**: the game's rectangle for its kind (Movement), measured in view pixels from the reference point of its ground point rather than of its position, so a thing collides the same wherever its sprite is drawn. A box `(x, y, w, h)` has its left edge `x` and its top `y` pixels from that point. For a kind with standing offset `(su, sv)` the ground point is drawn `(sv − su, su/2 + sv/2)` from the position's point, so the rectangle `(top, left, height, width)` becomes `(left − (sv − su), top − (su/2 + sv/2), width, height)`:

| Kind | Box (x, y, w, h) |
| --- | --- |
| Player | (−10, 9, 16, 6) |
| Creature, conjured | (−10, 9, 16, 12) |
| In flight | (−6, 5, 8, 8) |
| Boulder | (−16, 4, 28, 24) |
| Trap | (−24, 8, 44, 20) |
| Wall spell (the portal) | (−8, 10, 16, 10) |

Two boxes overlap when their open intervals overlap on both axes. The boxes are thin and lie below the ground point, at the feet: the player can stand just above a creature on screen with their ground points 6 units apart on each axis, often in the same cell, and a thing a step to its side, 8 units off on each axis.

Each thing template also has a **footprint**, a square of world units centered on its ground point, `footprint` units on a side: boulders and traps test its corners against the terrain (Moving), and it sizes the shadow (world.md). The importer sets 16 for the player, creatures and boulders and 20 for traps. A template with no box collides by one derived from its footprint `n`: `(−n/2, −n/4, n, n/2)`.

### Moving

A move by `(du, dv)` is made in this order:

1. **Terrain.** The player and creatures test their ground point. The `u` step shrinks by 1 toward 0 until the ground point moved by it alone is accepted; then the `v` step the same way, from the original `u`. Boulders and traps test the four corners of their footprint, `footprint/2 − 1` from the ground point on each axis, all moved by the whole step; if any is refused the move is refused. A point is accepted when its cell is inside the level, not blocked (a wall, no floor, or a door that is not open) and not refused by its feature (below).
2. **Things.** While the moved box overlaps another thing's, both steps shrink by 2 toward 0, to nothing at worst. The player and creatures ignore things in flight, and a living creature or a levitating player ignores corpses; what they overlap first is the **blocker**.
3. **Push.** If the player's step shrank to nothing against a blocker that can be pushed, the player pushes it (below).
4. **Slide.** If the player moved nothing, met no thing, and walked along one axis (an even direction) into a diagonal wall whose filled corner (world.md, Import) lies ahead, it slides 4 units along the other axis, away from that corner.

Features refuse the player's ground point at their cell, with an effect:

- a wall cell holding an **exit** is refused, and when the player walks into it along an axis (an even direction) the player takes the exit (rules.md);
- a **pit**, or an open **trapdoor**, is refused, and the player falls through it (rules.md);
- a closed **door** is blocked like a wall.

Creatures are refused by pits and open trapdoors unless they resist teleporting, and by floors their template avoids. Boulders and traps are not refused by features.

A move that changes the thing's cell runs the effects of the cell it leaves and the one it enters: plates, teleporters, floor effects, exit markers (rules.md, world.md).

### Pushing

The player pushes a trap, a corpse (a creature with no hit points) or a conjured thing by moving it the player's whole step, recursively through a line of pushable things; what a pushed thing already overlaps where it stands does not block it, so things lying on each other (two corpses fallen in one place) come apart. If the line moves, the player then moves as far as the first thing did. A push makes the player wait 3 ticks more before its next update. A levitating player passes over corpses rather than pushing them.

A boulder is not pushed. Walking into a rolling boulder hurts the player 2d3, unless it rolls the way the player walks. Walking into a boulder along an axis (an even direction) then starts it rolling the way the player walks: a resting boulder stops when blocked, and a rolling one only turns.

### Boulders

A rolling boulder moves its `speed` along its direction each update (`interval_ms`). When the move is refused, or meets a thing, it hurts the thing it met, if any, 3d4. A boulder that stops when blocked then stops, and starts a boulder it met rolling in its own direction. A shuttling boulder starts a boulder it met the same way, and reverses. Rolling and shuttling are the `roll` action's (rules.md); `stop` stops it.

Boulder animations are named `roll.d` for direction `d`. The importer makes `roll.0` and `roll.4` frames 0–4, `roll.2` and `roll.6` frames 5–9, forward for directions 0 and 2 and backward for 4 and 6, one frame per update.

### Things in flight

A thing in flight is an object thrown or fired, by the player, a creature or a cell (arrow traps). It starts ahead of its thrower's ground point along the thrower's facing, by the **launch distance** of its direction, times the direction's step (README, Directions): 12 units, but 14 along direction 3 and 8 along direction 5; a creature's spell starts 2 units further still. A cell throws from its center the same way. It moves along that facing and updates every tick: the first update it waits; then it moves 8 units along its direction (half that along 1 and 5), its range falling by 1. Its box is the one for things in flight (Collision boxes).

At each update, in this order:

- **Its range runs out:** it lands where it is.
- **Its box overlaps a thing,** where it is or where it would move, other than its thrower and corpses (the player, if a creature or a cell threw it): it hurts it (arrows as damage kind 3, everything else physical, through the target's defenses), the player's throw turns a creature hostile, and it lands.
- **Its ground point would enter a blocked cell:** if that is a sealed door its object opens, the object is used up; otherwise it lands where it is.
- **It enters an armed teleporter** that things in flight do not pass over: the teleporter moves it, and turns its direction by the teleporter's `turn` eighths.
- **Landing:** only the player's throws land; a creature's or a cell's vanish. An arrow breaks 1 time in 25. A bottle becomes its broken item. The object is put down on its cell as rules.md, Dropping objects, says.

A thrower already touching something when it throws (its own box, moved 4 units along its facing, overlaps a thing; for a cell, the flight's box moved so) hits it at once, and the object lands where its flight starts.

**Too close to a wall.** Then, if the flight would start in a blocked cell, it does not fly. The player's throw or shot is refused with message 28h (`too_close_to_wall`): the object stays in the hand or goes back into the quiver, and the hand does not practice or pose, but it waits: a throw 2 points, a shot its action's delay, as every refused shot does; a blow already struck by touching stands. A creature's object is lost. Spells say the same in spells.md, Missiles.

**Drawing it.** It glides between its moves, as Drawn between updates says; spells in flight do the same (spells.md, Missiles).

### The player's update

The player's update comes every `interval_ms` of its template (player.md, Speed). In it, if a walk was asked for since the last update, the player turns to that direction and moves its template's `speed` along it, half that along directions 1 and 5 (screen left and right). A walk that moves shows the walk animation's next frame; after 8 updates without walking the player shows its idle frame. A blocked step that meets a creature touches it (combat, later).

**Asking to walk.** The game reads the keys at the player's update, so a key let go before it walks no further. The engine is told of walks as they are asked, every world update while a walk key is held, and keeps the latest direction for the player's next update, with one exception: when a world update asks for no walk and a walk was asked for in the world update that ran the player's last update, the walk held through that update has been let go, and the direction kept is dropped. A walk asked for afresh after the player's update, a tap, is kept though it is let go before the next.

**Starting from a stand.** The game's player waits for its next update to start walking, up to a whole interval. In the engine, while a walk is asked for, the player's last update did not walk, and it holds no pose, its next update comes once half its interval (rounded up) has passed since its last, or at once if that has already passed, instead of when due. Its updates go on from there a whole interval apart. Walking is no faster for it: a step still follows a step by a whole interval.

**Keys held together.** Two walk keys held together walk the diagonal between them. A single straight key may be the first of a pair pressed a moment apart, or the last of a pair let go a moment apart, so a straight walk that starts from a stand or is left over from a diagonal it is half of is not taken before it has been held 3 world updates: it is asked for at once when the player's next update comes after that, since a second key pressed meanwhile asks for the diagonal in its place, and is held back otherwise.

## Unknowns

- The player's spawn offset within the arrival cell when changing level, and the scroll the view does on arrival (three styles: drop, rise and slide), are specified with scenes. The engine places the player at the arrival cell's center.
