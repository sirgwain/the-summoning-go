# Level rules

Nothing in a level carries code. Its puzzles come from cell features, small records the level's entity stream creates (formats.md), and the level procedures that triggers and conversations run.

## Cell features

A cell can have:

| Feature | Record | Created by entity type |
| --- | --- | --- |
| Decoration | ornament and decoration (world.md) | 1 |
| Teleporter | a teleporter record | 3 (in-level), 8 (exit) |
| Door | a door record | 4 |
| Trigger | a trigger record | 7 |
| Plate | none | 10 |
| Armed | a bit | 3, 7, 8 |

Exits are always armed. An in-level teleporter is armed at the start when its record's first byte has bit 80h. A trigger is armed when its entry's second byte is 0.

## Triggers

A trigger record: `counter` (starts at 0), `threshold`, `procedure a`, `procedure b`. Procedure FFh means none. The cell's terrain decides what the trigger is.

### Levers (19h, 1Ah)

Using a lever in the player's cell runs procedure a (lever 19h) or b (1Ah), with a sound, and flips the lever to the other tile. A disarmed lever does nothing: message 86h ("will not budge").

### Weight plates (1Fh up, 20h down)

Things on a plate add weight to its counter: 100 for an entity, an object's weight for an object. When an addition takes the counter from below the threshold to the threshold or more, and the plate is armed, procedure a runs and the plate goes down (20h). When a removal takes it below the threshold, and the plate is armed, procedure b runs and the plate goes up (1Fh). The counter never goes below 0.

Arming a plate re-evaluates it: a down plate below its threshold runs b and rises; an up plate at its threshold runs a and goes down.

Things in flight, wall spells, creatures that resist teleporting and a levitating player do not press plates. An entity keeps its weight on the plate until it stands on another cell. Levitation lifts the player off.

### Item slots (17h, 18h)

The player drops an object onto the slot's cell:

- If the object's template is not the trigger's threshold: message 8Fh with the object's name, and the object stays in hand.
- If the slot is disarmed: message 87h, and the object stays in hand.
- Otherwise the object is consumed and procedure a runs. On level 30, at cell (40, 11), a slot whose procedure a is 0 runs procedure 18h instead.

Most slots end their work by disarming themselves, so a second object is refused. A slot that takes several objects in turn has each procedure set the slot's next procedure, and sometimes the next object it takes, and its last procedure disarms it or leaves it as it is. Some slots never disarm: in the game they go on consuming the object they take, running their procedure again each time, so the extra objects are lost. A few of those hand the object back: their procedure creates an object of the template the slot takes (a pass returned past the teleporter it opens, a key put back where the level placed it), so the slot can be used again.

**Engine model.** The engine adds a slot that is full. A slot becomes full when an object it takes is consumed and procedure a, once run, has left the slot armed, with the same procedure a and the same template, and procedure a creates no object of that template. A full slot refuses every object with message 87h, and the object stays in hand; arming or disarming it does not change that. A slot that disarms itself, moves on to its next procedure, or hands its object back, never becomes full, and behaves as in the game.

## Doors

A door record: its cell and a state byte.

| Bits | Meaning |
| --- | --- |
| 0–2 | Stage: 0 closed .. 4 open; 5–7 sealed (closed) |
| 08h | Moving toward open |
| 10h | Moving |
| 20h | Opened by the Kano spell |
| 40h | Vertical |

A closed door's cell is terrain 10h (11h vertical) and blocks movement and sight. An open door's is floor (1Bh). A moving door steps one stage per update toward open or closed, playing a sound between the ends. Opening, its cell leaves terrain 10h/11h at the first step (16h, then floor 1Bh from stage 2); closing, it takes 10h/11h again once it is down to stage 2.

- **Switch on** (open): a closed door starts opening; a door closing reverses.
- **Switch off** (close): an open door starts closing; a door opening reverses.
- **Toggle**: a moving door reverses; a still door starts moving the other way.

**Closing on entities.** When a door closes on its cell, boulders, traps, conjured things and corpses there are removed. Anything else is pushed to the cell's edge, hurt 1 point with a 3 in 10 chance, and the door opens again.

**Sealed doors** (stages 5–7) open only to a thrown sphere of the matching kind flying into them: the Rainbow Sphere (C9h) for stage 5, the Energy Sphere (CBh) for 6, the Shadow Sphere (CDh) for 7. The door opens at once (stage 4) with sound 16h and, when the door is on screen, a sparkle (scenes.md, Sparkles), and the sphere is used up.

**Kano** (spells.md) switches on a door with bit 20h; any other door gives message 34h.

## Teleporters and exits

A teleporter record has five bytes: `kind, b1, b2, b3, b4`.

### In-level teleporters (terrain 24h)

`kind` bit 04h selects which of two jumps the teleporter makes next; `kind` bits 4–6 give the turn applied to things in flight (and half of it to boulders); bit 08h means things in flight pass over it; bit 80h means armed at the start. The jumps are `(b1, b3)` and `(b2, b4)`, signed cell offsets. Each use takes the selected jump and flips bit 04h, so a pair of teleporters can send things back and forth.

An entity entering an armed teleporter cell is moved by it, unless it resists (creatures with behavior 0200h, a levitating player, wall spells). The player arrives with a teleport effect and sound 16h; everything else goes silently, with no effect, and objects too. A creature or a corpse is first set down at its spot in the teleporter's own cell, and is crushed there if something already stands in the way. A living creature then spins on the teleporter for 30 ticks, turning to the next of its four directions each tick, and at the end is moved by the teleporter it is on, if it is still armed. Anything else is moved at once. Any entity but a thing in flight is set down at its kind's spot in the destination cell, as when placed (entities.md, Placement): the player crushes what stands there, and any other entity that arrives where something already stands is crushed itself, removed with what it carries left on the floor there. A living creature is not stopped by a corpse in this way; boulders and traps are. A teleporter takes its jump, and flips, as the entity is moved, not as it enters.

**Arming** a teleporter (procedure op 0 or 2) sends the objects on its cell along the chain of armed teleporters it leads into, each flipping as it is passed, to the chain's end, and moves the entities on the cell. A disarmed teleporter is drawn as plain floor.

### Exits (records with kind FFh)

`b1` is the destination level and `(b2, b3)` the destination cell. An exit can be:

- **In a wall** of shape 1 or 3: the player walking into it, facing 1 or 5 for shape 1, or 3 or 7 for shape 3, changes level (sliding).
- **A pit (25h)**: the player falls to the destination, with 4d5 damage, unless facing 2–4 (stepping down carefully), which takes no damage. A levitating player passes over. Creatures that do not resist fall and are removed.
- **A trapdoor (26h open, 27h closed)**: open, it acts as a pit. Opening one sends the objects on it to the destination level and cell, and makes the entities on it fall.
- **An exit marker** (entity kind 6, placed when the entry's marker byte is 0): the player walking into it climbs through it (view effect 1) and goes through the cell's exit.

## Level procedures

Procedure `n` is the n-th procedure in the level's script pool (formats.md). Its instructions run in order. Each is an op and four argument bytes; `(x, y)` is a cell.

| Op | Arguments | Effect |
| --- | --- | --- |
| 0 | x, y | Switch on: open a door; open a trapdoor; arm a teleporter; arm a trigger |
| 1 | x, y | Switch off: close a door; close a trapdoor; disarm a teleporter; disarm a trigger |
| 2 | x, y | Toggle any of the above |
| 3 | x, y, which, p | Set the trigger's procedure a (`which` 0) or b (1) to `p` |
| 4 | x, y, state | Set the state (its message) of the mouth at the cell |
| 5 | x, y, dir | Start the boulder placed at the cell rolling along direction `dir`, an even direction table entry (0, 2, 4 or 6); it reverses when blocked and keeps rolling |
| 6 | x, y, dir | As 5, but the boulder stops when blocked |
| 7 | x, y | Stop the boulder placed at the cell |
| 8 | x, y, template, dir | Shoot a new object of the template from the cell along `dir` (arrow traps) |
| 9 | x, y, spell, dir | Shoot a spell from the cell along `dir`, power 6d10 |
| 10 | x, y, threshold | Set the trigger's threshold |
| 11 | x, y, template | Create an object of the template on the cell's floor |

Boulders are found by the cell they were placed on, not where they are now. Ops 5 and 6 on a boulder that is already rolling only turn it to `dir`; it keeps stopping or reversing as it did. Conversations run procedures too (dialogue.md).

## Deep water

On a deep-water level (formats.md), objects cannot be put down except on an up plate: dropping one gives message 72h, and anything else that would land sinks and is lost.

## Gem socket (28h)

A socket takes six gems, templates C0h–C5h (gems 1–6). The socket cell's decoration data keeps one bit per gem set: gem `g` sets bit `g − 1` (world.md draws them).

When the player drops a gem on the socket's cell, the gem is consumed and its bit set. A gem whose bit is already set is consumed too and changes nothing. When all six bits are set (3Fh), a view effect plays and the wall in the row before the socket, at `(x − 1, ·)`, opens:

- `(x − 1, y)` and `(x − 1, y + 1)` become floor (1Bh);
- `(x − 1, y − 1)` and `(x − 1, y + 2)` become wall shape 3, closing the sides of the opening.

Visibility is recomputed afterwards.

## Dropping objects

An object put down on a cell is handled in this order:

1. On deep water it sinks (above).
2. On an item slot or the gem socket, the slot or socket handles it (above).
3. On a pit or an open trapdoor it falls to the cell's exit destination and lands on that level.
4. On an armed teleporter it follows the chain of teleporters (each flipping as it is passed) and lands at the end.
5. It lands on the floor; on a weight plate its weight is added (triggers).

## Changing level

The player changes level by an exit, a pit, a conversation or a spell. The current level keeps its state; the new one is loaded, or restored as it was left. The player arrives at the destination cell. The arrival has three styles (scenes.md, Level transitions): rising from below (pits and open trapdoors), descending from above (exit markers), and scrolling (walls, conversations). A pit plays view effect 0 or 2 first and an exit marker view effect 1 (scenes.md).

## The player and the features

What the player does to a level's features, in the engine's terms:

- **Using.** The player uses the lever in its own cell (Levers). On a keyhole (terrain 17h) instead, the engine adds a convenience the game does not have: using puts the key the keyhole takes into it, from what the player carries (the hand, then the shelves in order, the right and left hands, then the bag in the left hand), as if the player had put it down there (Item slots); a refusal leaves the key where it was, and without the key nothing happens. Object holes (18h) get no such help: what goes in them is the player's to work out. With no lever or keyhole, the player uses the nearest mouth whose cell is the player's or one of its eight neighbors, which says its message. With none, nothing happens.
- **Putting down.** The player puts the object in its hand down on its own cell, handled as Dropping objects says; a refusal keeps it in hand.
- **Plates.** The player weighs 100 on the plate of its cell, unless levitating.
- **Teleporters.** The player entering an armed teleporter's cell is moved to its destination, with a teleport sound, and arrives as below: at the center of the destination cell, crushing what stands there.
- **Exits.** Walking onto a floor exit, or into a wall exit along an axis (entities.md, Moving), takes the player to the exit's level and cell.
- **Pits and open trapdoors.** The player stepping onto one falls to its destination, hurt 4d5, unless it walks along directions 1–3 (stepping down carefully; the game's facings 2–4), which hurts nothing. A levitating player passes over.
- **Arriving.** The player arrives at the center of the destination cell. Things whose boxes overlap the player's there are crushed: removed, creatures dropping what they carry. Corpses, boulders and traps are crushed as creatures are; objects on the floor are not. The new level's visibility is computed from that cell.

## Import

Each cell's feature is named `<kind>.<x>.<y>` and made from its terrain and placement:

| Terrain | Placement | Feature | State and fields |
| --- | --- | --- | --- |
| 10h, 11h, 15h, 1Bh | door (4) | `door` | stage 0 `closed`, 4 `open`; 5–7 `closed`, sealed by the Rainbow, Energy or Shadow Sphere; bit 20h opened by a spell; look by bit 40h |
| 17h, 18h | trigger (7) | `slot` | accepts the threshold's template; procedure a is `on_insert`; `keyhole` at 17h |
| 19h, 1Ah | trigger (7) | `lever` | on at 1Ah; procedure a is `on_turn_on`, b `on_turn_off` |
| 1Fh, 20h | trigger (7) | `plate` | down at 20h; the threshold; a is `on_press`, b `on_release` |
| 21h–23h | any | `mark` | rune `1`–`3` |
| 24h | teleporter (3) | `teleporter` | destinations `(x + b1, y + b3)` and `(x + b2, y + b4)`; next 1 when bit 04h is set; turn, fly over and armed from the kind byte |
| 25h | exit (8) | `pit` | the exit's destination |
| 26h, 27h | exit (8) | `trapdoor` | open at 26h; the exit's destination |
| 28h | any | `socket` | places for gems C0h–C5h; `on_complete` runs the opening below |
| wall 1, 3 | exit (8) or teleporter (3) of kind FFh | `exit` | the destination; drawn from the exit-wall bank |
| 1Bh | exit (8), marker 0 | `exit` | the destination; drawn with resource 99 |
| 1Bh | mouth (11) | `mouth` | its message; drawn with resource 56 |

Triggers are armed as their placement says, and teleporters by bit 80h. A trigger's procedures FFh are none. The slot special case on level 30 becomes the slot's procedure. The gem socket's opening becomes a procedure of `set_floor` (floor type 0) on `(x − 1, y)` and `(x − 1, y + 1)` and `set_wall` (shape 3) on `(x − 1, y − 1)` and `(x − 1, y + 2)`.

Procedure `n` is named `p<n>`. Its ops become actions on the feature or thing at the cell they name:

| Op | Action |
| --- | --- |
| 0, 1, 2 | `on`, `off`, `toggle` |
| 3 | `set_hook`, procedure a or b as the hook of the feature's kind |
| 4 | `set_message`, the level message `state` |
| 5, 6, 7 | `roll` (5 shuttling) and `stop` of the boulder placed at the cell, in direction `dir` |
| 8 | `shoot_item` |
| 9 | `shoot_spell`, power 6d10 |
| 10 | `set_threshold` on a plate, `set_accepts` (the template) on a slot |
| 11 | `create` |

An op naming a cell with nothing it can act on is dropped and reported.

## Unknowns

- The exact way a teleported creature spins and how long the player's teleport effect lasts.
- How often a moving door steps. The engine steps it once per tick, so a door opens or closes in four ticks.
- Where in its destination cell a teleported thing lands. The engine keeps its place within the cell.
- A slot's procedure b. Many slots set one, but the game is not known to run it; the importer leaves it out.
- Whether the objects and creatures a level places on a plate count toward its counter when the level first loads. Until known, the engine adds their weight at load without running any procedure, so the plate starts in the state its terrain shows.
