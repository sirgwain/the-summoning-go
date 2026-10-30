# The world

## Levels

The game has 41 levels, numbered 0..40 by their index in LEVELS. A level is a grid of at most 64×64 cells, each holding one terrain byte. Cells outside the grid are solid. A level keeps its changes when the player leaves and returns: doors, triggers, objects on the floor, creatures and their hit points stay as they were left.

The player is not part of any level and moves between them. A new game starts on the level and cell given in INIT.

## Coordinates

World positions are in units, 32 per cell (README.md). The first coordinate `u` is the grid row, the second `v` the column.

### Projection

The view is 2:1 isometric. Each cell is a 64×32 diamond. In view pixels, relative to a scrolling origin:

- The **tile box** of cell `(a, b)`, the 64×32 rectangle around its floor diamond, has its top-left corner at `(32·(b − a), 16·(a + b))`. All terrain art below is placed relative to this corner.
- The **reference point** of world position `(u, v)` is `(16 + v − u, ⌊u/2⌋ + ⌊v/2⌋)`. An entity's sprite is drawn with its top-left corner at its position's reference point, so an entity's position is the top-left of its sprite, and the point that decides which cell it stands on is offset from it by kind (entities.md).

Both share one origin: the view's origin cell `(R, C)` has its tile box at `(0, 0)`, and the world point `(32R, 32C)` has its reference point at `(16, 0)`. Moving the origin shifts both by the same amount.

Increasing `u` moves down-left on screen; increasing `v` moves down-right.

### Draw order

The view is drawn back to front: by increasing `a + b` (screen row), and within a row floors first, then walls and things standing on the floors in order of their reference row. Things standing in a cell are drawn after the cell's floor and before the walls of cells in front of it.

## Terrain

| Terrain | Meaning | Walkable |
| --- | --- | --- |
| 00h–0Eh | Wall, shape 0..14 | no |
| 0Fh | Unused (drawn as floor) | no |
| 10h | Closed door, horizontal | no |
| 11h | Closed door, vertical | no |
| 12h–14h | Unused (drawn as floor) | no |
| 15h | Floor under a door in motion | no |
| 16h | Door opening | no |
| 17h, 18h | Item slot (keyhole, object hole) | yes |
| 19h, 1Ah | Lever, two positions | yes |
| 1Bh | Floor | yes |
| 1Ch–1Eh | Floor variants; liquids on liquid levels | yes |
| 1Fh, 20h | Weight plate up, down | yes |
| 21h–23h | Rune marks: teleport targets for spells and powers | yes |
| 24h | Teleporter | yes |
| 25h | Pit | (falls) |
| 26h, 27h | Trapdoor open, closed | open: falls; closed: yes |
| 28h | Gem socket | yes |
| 29h–35h | Floor hazard tiles (spikes and the like) | yes |
| FFh | Outside the map | no |

"Walkable" here is the terrain alone; doors, entities and triggers add their own rules (entities.md, rules.md). Walls 0, 2, 4 and 5 are diagonal shapes; walking diagonally into them slides along them. Wall shapes 1 and 3 are the two straight faces that can hold a decoration or an exit.

Each cell may also have **features**: a decoration, a trigger, a door, a teleporter or exit, a weight plate, and an armed bit (rules.md).

## Drawing a level

All art comes from the banks named in the level header (formats.md). Offsets are `(dx, dy)` from the cell's tile box corner.

### Floors

The floor bank holds floor tiles as half-tiles: frame `2k` is the upper half (64×16) at `(0, 0)` and frame `2k+1` the lower half at `(0, 16)`.

- Every cell that is not a wall or off the map first gets a floor. Walls also get floor 0 under them.
- Terrain 1Bh–1Eh uses floor `k = terrain − 1Bh`; every other walkable or door cell uses floor 0.
- Terrain 17h–1Ah add frame `terrain − 17h` of the floor-overlay bank at `(4, 11)`.
- Terrain 1Fh–28h add frame `terrain − 1Fh` of the plate bank at `(−16, 8)`, except a teleporter (24h) that is not armed and a closed trapdoor (27h), which show only floor.
- Terrain 28h (gem socket) instead shows the socket: if its six gems are all set (mask 3Fh) frame 1 of terrain bank +37h at `(−1, 16)`; otherwise frame 0 there, and for each set bit `p` of the mask, frame `p` of terrain bank +38h at `(−1, 16)` plus the offset for `p`: 0 `(11, 1)`, 1 `(3, 1)`, 2 `(21, 1)`, 3 `(19, 6)`, 4 `(1, 6)`, 5 `(9, 7)`.
- Terrain 29h–35h add frame `terrain − 29h` of resource 67 at `(0, 0)`.

### Walls

The wall bank holds, for shape `s` (0..14), the upper part as frame `s` at `(0, 0)` and the lower part as frame `s + 15` at `(0, 16)`. The bank after it (wall bank + 1) holds the caps: frame 0, the left cap, at `(0, −8)`, and frame 1, the right cap, at `(16, −8)`. A shape's caps come from the INIT wall-top table: bit 1 draws the left cap, bit 0 the right.

The engine combines a shape's upper part, lower part and caps into one image per shape on import.

**Exits in walls.** A wall cell with a teleporter feature is an exit (rules.md). It is drawn from the exit-wall bank, as shape 1 if its terrain is 1, or shape 4 if its terrain is 3: frame `s − 1` at `(0, −16)`, frame `s` at `(0, 0)` and frame `s + 1` at `(0, 16)`, with caps from the exit-wall bank + 1 as for walls.

**Decorations.** A wall cell with a decoration feature shows it only on shape 1 or 3: frame `ornament` of the decoration's bank at `(8, 8)`. The level's decoration byte `b` gives `ornament = b % 10` and `decoration = b / 10`, which selects one of the five decoration banks in the header. Decorations animate only by palette (scenes.md).

### Doors

A door is drawn as two sprites, placed from the reference point `(rx, ry)` of the door cell's corner `(32a, 32b)`. Stage `s` (0 closed .. 4 open, rules.md):

- Horizontal: frames `2s` and `2s + 1` of the door bank, at `(rx − 33, ry − 3)` and `(rx − 1, ry − 14)`.
- Vertical: the same frames of the door bank + 1, at `(rx − 30, ry − 14)` and `(rx + 2, ry − 3)`.
- Sealed stages 5..7: frames `4(s − 5)` and `4(s − 5) + 1` of terrain bank +39h, placed as horizontal.

The two sprites are drawn in the order of their rows among the other things standing in the view.

### What is drawn

Only visible cells are drawn; the rest of the view is black. Each visible cell is marked **seen**, which the level map shows.

## Visibility

Visibility is recomputed whenever the map changes (a door moves, a wall is removed) or the player's vision cell changes. The vision cell is the player's cell, except that it stays where it was while the player stands in a door's cell.

From the vision cell, a scanline flood fill marks every cell reachable through cells whose terrain is above 11h (anything but walls and closed doors), moving along rows and to the rows on either side. The first blocking cell met on each side of every run is marked visible too, and the fill does not pass it. The result is the room the player is in, with its walls.

The game also swaps the shapes of some visible walls according to which of their neighbors are visible, so that walls seen from behind look closed. This specification does not cover that; walls are drawn with the shapes the level gives them.

## Level map

The game shows a map of the current level while the player holds the Palimpsest and uses it (power 22h, items.md). The map shows the cells the player has **seen** (drawn while visible; see above), the player's position and the level's name. Cells of level exit markers and mouths are marked as exits.

Some levels refuse the map:

| Level | Instead |
| --- | --- |
| 13h | Message 90h |
| 21h, 22h, 23h | Conversation 7Dh |

**Engine model.** The map belongs to the interface, not to an item. A minimap and a full map are always available and show the same seen cells. Each level has a map setting: shown, or refused with a message or a conversation. While the map is refused the minimap is hidden, and trying to open the map shows the message or runs the conversation. The importer sets levels 13h, 21h, 22h and 23h from the table above and every other level to shown. Power 22h imports as "open the map", following the same rule.

## Liquids

The header's liquid kind:

- **2, fire.** Entering a cell of terrain 1Bh–1Eh hurts the player 2d2 fire damage. Every status tick (player.md) on the level hurts 2d2 fire damage wherever the player stands. Fire immunity and levitation prevent both.
- **1, water-like.** Terrain 1Dh hurts the player 1 on entering, and creatures avoid it. Terrain 1Eh drains 1 magic point on entering.
- **Deep water** (formats.md): terrain 1Bh, 1Ch, 1Dh, 1Eh are depths 1, 2, 3, 4. At depth 4, every move has a 1 in 4 chance of costing 1 hit point, and every status tick costs 1, unless the player levitates. At depth 3 or more the player cannot throw or use hand actions.

## Unknowns

- The use of terrain 0Fh and 12h–14h, which no level is known to contain.
