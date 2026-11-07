# Saved games

The game saves to `SAVES\dNNN.sav`, a snapshot of its running state: the player, the story's variables, the level the player is on and every level visited before. It keeps no state of its random numbers: after a load, chances go on from wherever they stand, so loading the same save twice and doing the same thing can turn out differently. **Engine model.** The engine's saves hold the world's random state, so a restored state replays exactly; a game loaded in play (quick load, the disk menu, a file) is reseeded from the viewer's seed and how long it has run, so loads differ in live play and repeat in a script with a fixed seed. The importer reads these files and turns each into an engine save, so a game played in the original can be carried on in the engine.

A save holds the game's own records as they are while it runs, not the LEVELS grammar. The importer reads it alongside the data files (it needs LEVELS, OBJECTS, NPC and TEXT to name what the records refer to) and the pack imported from them.

## The file

All integers are little-endian.

| Offset | Size | What |
| --- | --- | --- |
| 0000h | 256 | A text header: four TEXT messages (4Ch–4Fh) each followed by CR LF, then 1Ah, then zeros |
| 0100h | 50 | The save's name, NUL-padded |
| 0132h | | Level states (below) |
| then | 54F7h | The world block |
| then | 326h | The player block |
| then | 2710h | The object pool |

**Level states.** A `u16` count (the levels that follow plus one; readers ignore it), then for each level visited: `u16` level, `u32` size (49E2h) and the level's state, the world block's bytes from 0B15h to its end (below). An `i16` of −1 ends the list. The world block holds the current level's state in the same layout. The list can hold the current level too, as it was when the player last left it; that copy is stale (its object numbers may have been reused since) and the world block's is the one that counts.

## The world block

| Offset | Size | What |
| --- | --- | --- |
| 0000h | 2 | World flags |
| 0002h | 2 | View rows (unused here) |
| 0004h | 2 | The object on the pointer: an object number, FFFFh for none |
| 0006h | 2 | The hands' action state |
| 0008h | 1 | 1 on levels 21h–23h |
| 0009h | 3 | Unknown |
| 000Ch | 4 | Play time in ticks |
| 0010h | 14 | Timed counters, in status ticks: 0–5 the six attributes' boosts, in attribute order (player.md); 6–13 sight, fire shield, lightning shield, battle rage, haste, invisibility, weapon skills, magic skills |
| 001Eh | 1 | Status ticks until the next magic point |
| 001Fh | 5 | Unknown |
| 0024h | 13 | The gesture playing on the gesture panel |
| 0031h | 99 | Interaction variables 0–98 (dialogue.md) |
| 0094h | 1 | Variable 99, the transcript option |
| 0095h | 2 | NPC record count (127) |
| 0097h | 127 × 16 | The NPC records, as NPC holds them, with the bits the game changes (creatures.md) |
| 0887h | 48 | Unknown |
| 08B7h | 8 | Worn slots 20–27: 1 while the slot's object runs a timed power |
| 08BFh | 3 | A stored level and cell (unused here) |
| 08C2h | 10 × 2 | The shared counts of templates 72h–7Bh (items.md, Object values) |
| 08D6h | 41 | 1 for each level visited |
| 08FFh | 100 × 5 | Objects bound for another level: `u16` object (FFFFh: free), `u8` level, `u8` x, `u8` y |
| 0AF3h | 34 | The player's entity |
| 0B15h | 99 × 34 | The level's other entities; **the level state starts here** |
| 183Bh | 1 | Entities in use, the player included: entries 1 to this count − 1 are the level's; those past it are left over from levels loaded before and mean nothing |
| 183Ch | 647 × 6 | Cell features |
| 2766h | 1 | The level's number |
| 2767h | 4096 | Terrain, 64 × 64: cell (x, y) at `x × 64 + y` (world.md) |
| 3767h | 4096 | Cell view flags, the same layout |
| 4767h | 2 | Rows, columns |
| 4769h | 64 | The level header (formats.md), with its banks made absolute |
| 47A9h | 1 | Trigger count |
| 47AAh | 75 × 5 | Triggers |
| 4921h | 1 | Door count |
| 4922h | 75 × 7 | Doors |
| 4B2Fh | 1 | Teleporter count |
| 4B30h | 100 × 5 | Teleporters and exits |
| 4D24h | 2 | Text pool size |
| 4D26h | 2000 | The level's text pool, then its script pool (formats.md, LEVELS) |
| 54F6h | 1 | Unknown |

**Entity** (34 bytes):

| Offset | Size | What |
| --- | --- | --- |
| 00h | 2 + 2 | `i16 u, v`: the position, the top-left of the sprite in world units (entities.md) |
| 04h | 2 + 2 | A destination |
| 08h | 2 | The first object it carries, FFFFh for none |
| 0Ah | 1 | Kind, 0 for a free entry (entities.md, Kinds) |
| 0Bh | 3 | Drawing state |
| 0Eh | 4 | Next update, in ticks |
| 12h | 1 | Flags: 20h hostile, for a creature |
| 13h | 1 | Speed |
| 14h | 1 | Interval |
| 15h | 1 | Target |
| 16h | 1 | Frame |
| 17h | 1 | Direction |
| 18h | 1 | State |
| 19h | 1 | Countdown |
| 1Ah | 2 | Sprite bank |
| 1Ch | 2 | Parameter |
| 1Eh | 2 | Data |
| 20h | 2 | The plate cell it weighs on |

By kind:

| Kind | Direction | State | Parameter | Data |
| --- | --- | --- | --- | --- |
| 1 player | the player's facing (player.md) | | | |
| 2 creature | creature direction `d` | its NPC record | | hit points; 0 a corpse |
| 4 boulder | boulder direction 0..3 | 0 resting, 1 rolling, 2 shuttling | the cell it was placed on, `x × 256 + y` | |
| 5 trap | | | | |
| 7 mouth | | its level message | | |

**Cell feature** (6 bytes): `u8 x, u8 y`, `u8 bits`, `u16` the first object on the cell's floor (FFFFh for none), `u8 data`. A feature is in use when bit 01h is set. The other bits: 02h armed, 04h teleporter or exit, 08h trigger, 10h door, 20h weight plate, 40h decoration. `data` is the index of the cell's teleporter, trigger or door record, or for a decoration its ornament and decoration; on the gem socket's cell, one bit per gem set (rules.md, Gem socket).

**Trigger** (5 bytes): `i16` counter (the weight on a plate), `u8` threshold (a slot's template), `u8` procedure a, `u8` procedure b, as rules.md describes.

**Door** (7 bytes): `i16 u, v` (the cell's corner in world units), `u8` state (rules.md, Doors), two unknown bytes.

**Teleporter** (5 bytes): as the placement record (rules.md, Teleporters and exits). Exits have kind FFh. Bit 04h of an in-level teleporter's kind says which jump it makes next.

**Cell view flags**: bit 80h, the player has seen the cell; 40h, it is visible now.

## The player block

| Offset | Size | What |
| --- | --- | --- |
| 0000h | 26 | Name, NUL-padded |
| 001Ah | 4 | Unknown |
| 001Eh | 4 | The schools' levels, schools 0–3 |
| 0022h | 4 | The weapon classes' levels, classes 0–3 |
| 0026h | 1 | Level |
| 0027h | 4 × 2 | The schools' practice |
| 002Fh | 4 × 2 | The schools' practice thresholds |
| 0037h | 4 × 2 | The weapon classes' practice |
| 003Fh | 4 × 2 | The weapon classes' practice thresholds |
| 0047h | 4 | Experience |
| 004Bh | 4 | Experience for the next level |
| 004Fh | 2 × 5 | Hit points, maximum; magic points, maximum; armor class |
| 0059h | 6 | Attributes, in order (player.md) |
| 005Fh | 6 | Attribute maxima |
| 0065h | 2 | Modifiers (unused here) |
| 0067h | 1 | Fatigue |
| 0068h | 4 | Modifiers (unused here) |
| 006Ch | 40 × 14 | Spells, by spell number (spells.md): `u8` memorizations, `u8` cost, three power bytes, the gesture (9) |
| 029Ch | 4 | Each school's selected spell, as its slot 0–9 in the school |
| 02A0h | 2 | Confusion |
| 02A2h | 2 × 3 | Portrait; the other sex's portrait; the talk portrait (formats.md, F) |
| 02A8h | 2 | Status bits (below) |
| 02AAh | 40 × 2 | Inventory slots 0–39 (items.md): object numbers, FFFFh empty. Slots 28–39 are the pack, while the pack is open |
| 02FAh | 2 | Pack slots |
| 02FCh | 1 | 1 while the pack is open: the bag in the left hand then holds nothing itself, and its contents are in slots 28 on |
| 02FDh | 2 | The player's cell |
| 02FFh | 2 | The vision cell (world.md, Visibility) |
| 0301h | 1 | Liquid depth |
| 0302h | 2 × 3 | Interval base, bonus and load penalty (player.md, Speed) |
| 0308h | 14 | Figure looks |
| 0316h | 2 | The damage counted weapons have dealt (combat.md) |
| 0318h | 2 | The look the player has taken (spells.md, Shape Change), 0 for its own |
| 031Ah | 12 | The gesture icons known: byte `i` is 1 when letter `A + i` is known |

**Status bits**: 0002h female, 0004h resist quarter, 0008h resist forty, 0010h regenerating, 0020h charges, 0040h mana regenerating, 0080h death protection, 0100h poisoned, 0200h fire-immune, 0400h invisible, 0800h death-blow ward, 1000h accurate, 2000h transformed, 4000h blow protection, 8000h levitating (player.md, Status). Bit 0001h is unknown.

## The object pool

2000 objects of 5 bytes, numbered from 0: `u8` template (FFh: free), `u16` value, `u16` next. A container's value is the number of the first object it holds, FFFFh for none; anything else's is its value (items.md, Object values). `next` links an object to the next in its list (a cell's floor, a container, a creature's inventory), FFFFh ending it. A new object goes at the head of its list, so a list runs from the newest to the oldest.

## Import

The importer makes the engine state a save describes by starting from a new world on the pack, loading each level the save holds, and then laying the save over it. Anything the save keeps that the engine has no place for is left out and reported.

**Objects.** An object number becomes an object of its template's item, with its value. The shared counts of templates 72h–7Bh become the world's shared counts of their items. A container's list becomes its contents, oldest first. Objects of template FFh, or numbers past the pool, are reported and left out.

**The world.**

- The current level is the one the world block holds.
- Interaction variable `n` becomes counter `var.n` when not 0, variable 99 included.
- An NPC record `n` the pack has as `thing.npc.n` gets a record when its behavior has bit 0001h (talked to) or 0002h (dead), or when its portrait (byte 1) differs from NPC's.
- The conversation log (dialogue.md, Engine model) starts empty: the game keeps none.
- Objects bound for another level go onto the floor of their cell on that level, which is loaded if the save does not hold it.

**The player.**

- Its ground point is its entity's position plus (10, 28); its facing is the entity's direction.
- Name, hit and magic points with their maxima, armor class, attributes and their maxima, the interval bonus, level, experience and the next level's, fatigue and confusion come as they are; exertion starts at 0. Skills: weapon class `c` is skill `c`, school `s` skill `4 + s`, each with its level, practice and threshold.
- Portrait `p` is `portrait.p`.
- Status bits become statuses, and timed counters 6–13 the timed effects; poisoned is bit 0100h. Boost `i` gets the counter of attribute `i` in ticks, worth +2 while it runs.
- The pointer's object is the hand. Slots 0–27 are the slots. While the pack is open, slots 28 on, in order, become the left hand's container's contents.
- The known gesture letters are the icons, in letter order. Every spell with memorizations goes in the spellbook with that many; each school's selected spell is its slot's spell when that has memorizations.
- A look taken by Shape Change is left out and reported: the player arrives in its own.

**Levels.** Each level the save holds is level `level.NN` of the pack, laid over that level as the engine loads it new. What the save records of the level's start (terrain, placements, records) is compared with LEVELS, and only what changed is applied:

- **Terrain.** A cell whose terrain changed from a wall to a floor (1Bh–1Eh) or back has its wall and floor set as the level's import would set them (world.md, Import). Changes between the terrains of one feature (a lever's two tiles, a plate up and down, a trapdoor open and closed, a door's tiles) are the feature's state, below; any other change is reported.
- **Seen cells** are those with view flag 80h. On the current level, the visible cells are those with 40h, and the player's vision cell is the saved one.
- **Doors.** A door's state byte gives its stage (4 open, 0–3 closed or between); with bit 10h it is moving, toward open with 08h. A door that was sealed and is now at stage 4 or below is no longer sealed.
- **Triggers.** Armed comes from the cell's 02h bit. A lever is on at terrain 1Ah, a plate down at 20h, and the plate's counter is the trigger's. A threshold or procedure that differs from the level's placement replaces the plate's threshold, the slot's accepted item, or the hook (a procedure FFh, or one the level does not have, is none). The game keeps no full slots, so no slot is full (rules.md, Item slots).
- **Teleporters** are armed by the cell's 02h bit, and make jump 1 next when their kind has bit 04h. **Trapdoors** are open at terrain 26h.
- **Socket.** The gems set are the bits of the socket cell's data.
- **Mouths.** A mouth entity's level message is its feature's message; it stands on cell `((u + 16) / 32, (v + 24) / 32)`.
- **Floor objects.** Each cell's floor list replaces the level's objects there, bottom (oldest) first.
- **Creatures** stand at their position plus (10, 28), facing `2d`, with their hit points, hostility (flag 20h) and the objects they carry. Each is matched to a creature the level places with the same NPC record, the one placed nearest first, and takes its id; one with no match left gets a new id. A creature whose record has lost bit 0008h, while its template talks, no longer talks.
- **Boulders** are matched by the cell they were placed on, and stand at their position plus (8, 32), rolling (state 1) or shuttling (2) in direction `2d`. **Traps** stand at their position plus (8, 32), matched to the level's traps nearest first.
- Things the level places that the save does not have (killed, fallen, crushed) are gone. Entities of kinds 3, 8 and 9 (things in flight, conjured things, portals) are transient and left out, and so is the gesture playing.

## Unknowns

- The fields marked unknown above, and the use of the world flags.
- Whether the bound-for counters are ever used for objects other than those that fall or teleport to another level.
