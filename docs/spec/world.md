# The world

## Levels

The game has 41 levels, numbered 0..40 by their index in LEVELS. A level is a grid of at most 64×64 cells, each holding one terrain byte. Cells outside the grid are solid. A level keeps its changes when the player leaves and returns: doors, triggers, objects on the floor, creatures and their hit points stay as they were left.

The player is not part of any level and moves between them. A new game starts on the level and cell given in INIT.

## Coordinates

World positions are in units, 32 per cell (README.md). The first coordinate `u` is the grid row, the second `v` the column.

### Projection

The view is 2:1 isometric: neighboring cells along a screen row are 64 pixels apart, and each screen row is 16 pixels below the last. In view pixels, relative to a scrolling origin:

- The **tile box** of cell `(a, b)` is a 32×32 rectangle with its top-left corner at `(32·(b − a), 16·(a + b))`. All terrain art below is placed relative to this corner. The boxes of all cells tile the view like bricks: each column of boxes is offset 16 pixels down from its neighbors. Floor art is opaque and fills its box, with the diamond outlines of the grid painted into the art, so the floor looks like diamonds although every cell draws a rectangle.
- The **reference point** of world position `(u, v)` is `(16 + v − u, ⌊u/2⌋ + ⌊v/2⌋)`. An entity's sprite is drawn with its top-left corner at its position's reference point, so an entity's position is the top-left of its sprite, and the point that decides which cell it stands on is offset from it by kind (entities.md).

Both share one origin: the view's origin cell `(R, C)` has its tile box at `(0, 0)`, and the world point `(32R, 32C)` has its reference point at `(16, 0)`. Moving the origin shifts both by the same amount.

Increasing `u` moves down-left on screen; increasing `v` moves down-right.

### Draw order

The view is drawn back to front: by increasing `a + b` (screen row), and within a row floors first, then walls and things standing on the floors in order of their reference row. Things standing in a cell are drawn after the cell's floor and before the walls of cells in front of it.

## Terrain

| Terrain | Meaning | Walkable |
| --- | --- | --- |
| 00h–0Eh | Wall, shape 0..14 | no |
| 0Fh | Unused (drawn black) | no |
| 10h | Closed door, horizontal | no |
| 11h | Closed door, vertical | no |
| 12h–14h | Unused (drawn black) | no |
| 15h | Floor under a door in motion; sealed doors stand on it | no |
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

"Walkable" here is the terrain alone; doors, entities and triggers add their own rules (entities.md, rules.md). Walls 0, 2, 4 and 5 are diagonal shapes; walking diagonally into them slides along them. Wall shapes 1 and 3 are the two straight faces that can hold a decoration or an exit. A wall beside a cell the player cannot see takes another shape for as long as it cannot be seen (**Walls out of sight** below), and everything that reads its terrain meanwhile reads that shape.

Each cell may also have **features**: a decoration, a trigger, a door, a teleporter or exit, a weight plate, and an armed bit (rules.md).

## Drawing a level

All art comes from the banks named in the level header (formats.md). Offsets are `(dx, dy)` from the cell's tile box corner.

### Floors

The floor bank holds floor tiles as half-tiles: frame `2k` is the upper half (32×16) at `(0, 0)` and frame `2k+1` the lower half at `(0, 16)`.

- Every cell that is not off the map first gets a floor, by halves (see **Floor halves** below). Walls get floor 0 under them.
- Terrain 1Bh–1Eh uses floor `k = terrain − 1Bh`; every other walkable or door cell uses floor 0.
- Terrain 17h–1Ah add frame `terrain − 17h` of the floor-overlay bank at `(4, 11)`.
- Terrain 1Fh–28h add frame `terrain − 1Fh` of the plate bank at `(−16, 8)`, except a teleporter (24h) that is not armed and a closed trapdoor (27h), which show only floor.
- Terrain 28h (gem socket) instead shows the socket: if its six gems are all set (mask 3Fh) frame 1 of terrain bank +37h at `(−1, 16)`; otherwise frame 0 there, and for each set bit `p` of the mask, frame `p` of terrain bank +38h at `(−1, 16)` plus the offset for `p`: 0 `(11, 1)`, 1 `(3, 1)`, 2 `(21, 1)`, 3 `(19, 6)`, 4 `(1, 6)`, 5 `(9, 7)`.
- Terrain 29h–35h add frame `terrain − 29h` of resource 67 at `(0, 0)`.

#### Floor halves

Each half of a cell's floor (the upper half at `(0, 0)`, the lower at `(0, 16)`) is either drawn or filled black, by the cell's terrain:

- 15h–28h: both halves drawn.
- 29h–35h (hazard tiles): the upper half black; the lower half drawn only for 2Bh, 2Eh, 30h and 33h. The hazard sprite is drawn over it.
- 0Fh, 12h–14h: both black.
- Walls (00h–0Eh) and closed doors (10h, 11h): each has two check directions (README.md directions), a wall's being those of the shape it has taken (**Walls out of sight**). The upper half is drawn if the neighbor one step in the first direction **counts**, the lower half if the neighbor in the second does. A neighbor counts when it is inside the level, visible, and not a hazard tile (29h–35h). "–" never counts.

  | Terrain | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | Ah | Bh | Ch | Dh | Eh | 10h | 11h |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | Upper | 7 | 0 | 1 | 6 | 5 | 7 | 0 | 6 | – | – | – | 0 | 7 | 6 | 7 | 6 | 0 |
  | Lower | 3 | 4 | 1 | 2 | 4 | 3 | 3 | 3 | 2 | 4 | 3 | 3 | 4 | 3 | 2 | 2 | 4 |

  A wall next to a closed door takes its check directions from another shape. If the cell at `(x + 1, y)` is 10h, shape `s` checks as `below[s]`. Otherwise, if `(x, y − 1)` is 11h, `s` becomes `left[s]`; then, if `(x, y + 1)` is 11h, the result becomes `right[result]`.

  | `s` | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | Ah | Bh | Ch | Dh | Eh |
  | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
  | below | 0 | 6 | 2 | 3 | 7 | 8 | 6 | 7 | 8 | Ah | Ah | 2 | 0 | Dh | Eh |
  | left | 6 | 1 | 2 | 8 | 9 | 5 | 6 | Ah | 8 | 9 | Ah | Bh | Ch | 5 | 2 |
  | right | 0 | 1 | 6 | 7 | 4 | 9 | 6 | 7 | Ah | 9 | Ah | Bh | Ch | 4 | 0 |

A wall beside a door checks as the table above says for the doors beside it that are closed now, not as the level starts: when a door opens (its terrain no longer 10h or 11h), the walls beside it go back to their own shape's check directions, and when it closes again they take the closed door's. A door stops counting as closed at its first step opening, and counts again once closing reaches stage 2 (rules.md). The wall beside a door that has been opened therefore shows the floor half that faces the doorway, rather than a black half over the open door's floor.

So the floor under a wall shows only on the sides that face into the room the player sees, and the black halves keep it from sticking out below the wall's art.

### Walls

The wall bank holds, for shape `s` (0..14), the upper part as frame `s` at `(0, 0)` and the lower part as frame `s + 15` at `(0, 16)`. The bank after it (wall bank + 1) holds the caps: frame 0, the left cap, at `(0, −8)`, and frame 1, the right cap, at `(16, −8)`. A shape's caps come from the INIT wall-top table: bit 1 draws the left cap, bit 0 the right.

The engine combines a shape's upper part, lower part and caps into one image per shape on import.

**Exits in walls.** A wall cell with a teleporter feature is an exit (rules.md). It is drawn from the exit-wall bank, as shape 1 if its terrain is 1, or shape 4 if its terrain is 3: frame `s − 1` at `(0, −16)`, frame `s` at `(0, 0)` and frame `s + 1` at `(0, 16)`, in that order. An exit wall has no caps; its top frame covers the space where they would be. A decoration on an exit wall is not shown.

**Decorations.** A wall cell with a decoration feature shows it only on shape 1 or 3: frame `ornament` of the decoration's bank at `(8, 8)`. The level's decoration byte `b` gives `ornament = b % 10` and `decoration = b / 10`, which selects one of the five decoration banks in the header. Decorations animate only by palette (scenes.md).

### Doors

A door is drawn as two sprites, placed from the reference point `(rx, ry)` of the door cell's corner `(32a, 32b)`. Stage `s` (0 closed .. 4 open, rules.md):

- Horizontal: frames `2s` and `2s + 1` of the door bank, at `(rx − 33, ry − 3)` and `(rx − 1, ry − 14)`.
- Vertical: the same frames of the door bank + 1, at `(rx − 30, ry − 14)` and `(rx + 2, ry − 3)`.
- Sealed stages 5..7: frames `4(s − 5)` and `4(s − 5) + 1` of terrain bank +39h, placed as horizontal.

The door banks have eight frames, stages 0..3. An open door (stage 4) is not drawn.

The two sprites are drawn in the order of their rows among the other things standing in the view.

### What is drawn

Only visible cells are drawn; the rest of the view is black. Each visible cell is marked **seen**, which the level map shows.

A cell that is not drawn (off the map, or not visible) has its tile box filled black in its place among the floors. This matters: the floor under a wall sticks out below the wall's art, and the black fill of the cells in front covers it. Walls and things standing are drawn after all floors, so the fill never covers them.

## Visibility

Visibility is recomputed whenever the map changes (a door moves, a wall is removed) or the player's vision cell changes. The vision cell is the player's cell, except that it stays where it was while the player stands in a door's cell.

From the vision cell, a scanline flood fill marks every cell reachable through cells whose terrain is above 11h (anything but walls and closed doors), moving along rows and to the rows on either side. The first blocking cell met on each side of every run is marked visible too, and the fill does not pass it. The result is the room the player is in, with its walls.

The fill reaches diagonal neighbors as well: each run marks the blocking cells at its ends, and the rows on either side are searched from the column of one end to the other, those ends included. So the visible cells are every open cell joined to the vision cell through open cells, each step to any of the eight neighbors, together with every blocking cell next to one of them, diagonally included. The vision cell itself always counts as open.

### Walls out of sight

Once visibility is known, every visible wall (00h–0Eh) whose shape joins a wall the player cannot see takes another shape, so that it ends cleanly instead of showing the joint (the caps of a junction, say, pointing at a wall in the dark). Each shape has a list of up to four `(direction, shape)` pairs. They are tried in order: at the first whose neighbor, one step in that direction, lies inside the level and is not visible, the wall becomes that shape, and the new shape's list is tried from its start. When no pair applies, the wall keeps the shape it has come to.

| Shape | Pairs, in order |
| --- | --- |
| 0 | (2, Eh), (4, Ch) |
| 1 | (2, Bh), (6, Ch) |
| 2 | (6, Eh), (4, Bh) |
| 3 | (0, Eh), (4, Dh) |
| 4 | (0, Ch), (2, Dh) |
| 5 | (0, Bh), (6, Dh) |
| 6 | (2, 2), (4, 1), (6, 0) |
| 7 | (0, 0), (2, 3), (4, 4) |
| 8 | (0, 2), (4, 5), (6, 3) |
| 9 | (0, 1), (2, 5), (6, 4) |
| Ah | (0, 6), (2, 8), (4, 9), (6, 7) |
| Bh–Eh | none |

Off-map cells (FFh) count as not visible. The new shape holds only until visibility is next recomputed: every change is undone first, and before the level is saved or left, so the level keeps its own shapes. While it holds, the wall is the new shape for everything that reads its terrain: it is drawn as that shape (with its caps), its floor halves check as that shape (and then as Floor halves says for a door beside it), a decoration shows only if the new shape is 1 or 3, an exit wall is drawn as shape 4 only if the new shape is 3, and a diagonal shape that became a straight one no longer slides the player.

**Engine model.** The simulation keeps, for the current level, the cells visible from the player's vision cell, computed as above with "open" meaning not blocked: a cell with floor, no wall, and no door that is not open. Each cell that has been visible is **seen**, kept with the level. With the visible cells it keeps the shape each visible wall has taken (Walls out of sight), using the shapes' pairs from the wall set; the level's own shapes never change, and saves keep those. The walls are drawn, and slide the player, by the shapes taken; a wall drawn as a shape not its own shows no decoration. A wall that a feature's look draws (an exit) keeps its look. Only visible cells are drawn: their floors, walls, feature looks, objects and things standing in them. A level's floor halves (pack.md) say which halves each cell may ever show, as if every cell were visible; under a closed door that is not sealed, each half also shows only while the neighbor in its check direction (Floor halves) is visible, the check directions being kept with the level. Under a wall, the halves come from the check directions of the shape taken, as the wall set gives them: each half shows while its neighbor is inside the level, has floor, is not a hazard tile, and is visible. A door that is open or moving shows its floor whole, as the game draws terrain 15h and 1Bh. The walls beside a door follow it as Floor halves says, except that the engine counts a door closed only while it is fully closed, so a closing door's walls change two updates later than the game's.

**Import.** Each wall shape gets its check directions from Floor halves and its pairs from Walls out of sight. Hazard tiles are listed as cells that never show a wall's half. For each wall that a door's position can change (a horizontal door at `(x + 1, y)`, a vertical door at `(x, y − 1)` or `(x, y + 1)`), the importer gives, for every combination of those doors closed, the shape each shape checks as, so the engine can follow the doors as they move. The floor halves of walls are still given, as if every cell were visible, for tools that show the level whole. A door cell checks as a closed door of its orientation (10h horizontal, 11h vertical), whatever its starting state; a sealed door's floor shows whole.

## Level map

The game shows a map of the current level while the player holds the Palimpsest and uses it (power 22h, items.md). The map shows the cells the player has **seen** (drawn while visible; see above), the player's position and the level's name, TEXT `129h + level`. Cells of level exit markers and mouths are marked as exits.

Each seen cell is drawn with a small icon (RESOURCE 4Ch, 8×4 frames) by its terrain; floors, item slots, levers and hazard tiles draw nothing:

| Terrain | Frame |
| --- | --- |
| A wall or door (below 11h) holding a teleporter or exit (feature bits 2 and 4) | 22h for wall shape 1, else 23h |
| 00h–0Eh walls, 10h and 11h doors | terrain + 4 |
| A level exit marker's cell, a mouth's cell | 16h, 17h |
| 1Fh, 20h weight plate up, down | terrain − 7 |
| 21h–23h rune marks | terrain − 7 |
| 24h teleporter, only while armed (feature bits 2 and 4) | terrain − 7 |
| 25h pit, 26h and 27h trapdoor open and closed, 28h gem socket | terrain − 7 |

The player is frames 0–3 side by side. The screen is framed by RESOURCE 4Bh (the backdrop, a line per row) and the strips 4Dh–51h, and "Press F1 to print" (TEXT 94h) shows in its corner.

Some levels refuse the map:

| Level | Instead |
| --- | --- |
| 13h | Message 90h |
| 21h, 22h, 23h | Conversation 7Dh |

### The engine's map

The map belongs to the interface, not to an item: a minimap and a full map are always available, with no Palimpsest needed. Both show the same thing, the level as the player has seen it, and differ in size and in what the mouse can do. The engine draws them in its own style, not with the game's icons.

**Refusing.** Each level has a map setting: shown, or refused with a message or a conversation. While the current level refuses its map the minimap is hidden, and trying to open the full map shows the message or runs the conversation. The importer sets levels 13h, 21h, 22h and 23h from the table above and every other level to shown. Power 22h imports as "open the map", following the same rule. The full map can still show another visited level that does not refuse its map (Other levels, below).

**What a cell shows.** Only seen cells are drawn; the rest is the map's background. A seen cell is a tile in the view's projection, its rows and columns turned as the view turns them, so the map has the view's shape and north. Each cell is drawn by what it is, in this order, the first that applies:

| Cell | Drawn as |
| --- | --- |
| A wall (any shape), or a closed door's frame | A raised block: the cell's diamond in the wall color, lit on its top-left edges and shaded on its bottom-right ones, standing a quarter of the cell's height above the floor |
| A door | Floor, with a bar in the door color along the line of its wall, drawn open (a gap in the middle) while the door is open |
| Water (a floor of depth above 0) | The cell in the water color, a lighter line where it meets a floor that is not water |
| Any other floor | The cell in the floor color, its edges a little darker, so the tiles read as a grid |

Over its floor, a feature draws a symbol of its own, about half the cell's width, centered on the cell:

| Feature | Symbol |
| --- | --- |
| Exit marker, exit or teleporter in a wall | An arrowhead pointing out of the level, in the exit color |
| Armed teleporter | A ring in the teleporter color |
| Pit, open trapdoor | A dark hole with a lighter rim |
| Closed trapdoor | A square outline in the trapdoor color |
| Weight plate | A small square in the plate color, drawn sunk while pressed |
| Rune mark | A dot in the mark color with a ring round it |
| Gem socket | A small diamond in the socket color, filled with each gem set in it |
| Mouth | A speech dot in the mouth color |

A teleporter not armed draws no symbol. Each kind has one color, the same in the legend.

**The player** is a marker on its cell: a filled circle with a pointer showing its facing, in the player's color with a dark outline, its brightness pulsing once a second so it can be found at a glance.

**The minimap** sits in the top-left corner of the view, in a frame of the panels' style (the inventory's iron), 25% of the screen's width and half as tall, within the margin. Unlike the full map it does not fit the whole level: each cell is 12 screen pixels wide at the hotbar's scale (6 panel pixels at its scale), and the map is centered on the player and follows it as the view does. Cells cut by the frame are clipped by it. Under the frame a plaque names the level. The minimap is hidden while the full map, a conversation, the inventory panel, the spell panel or the book is open, and the player can hide or show it with Shift+M; that choice is a setting of the presentation, not of the world. The minimap takes no clicks: they go to the view under it.

**The full map** opens with M (or power 22h) over the view, which shows darkened behind it, in the same frame, filling the screen within the margin. It fits the level's seen cells, with a margin of two cells, as large as fits, then can be zoomed with the mouse wheel (from that size to 32 screen pixels a cell, about the pointer) and panned by dragging with the left button. The level's name is its title. Down its left side a legend shows a swatch and a name for the player and for each kind the map shows. Esc or M closes it; other keys and clicks do what the map's own controls say, and are otherwise ignored while it is open. The world holds while the full map is open.

**Links the player has been through.** The world records, for the player, each link it has travelled: a teleporter (by its feature), an exit marker or an exit in a wall, a pit or an open trapdoor. A record is the link's level and cell, and the level and cell the player arrived at; travelling the same link to the same place again adds nothing, and a teleporter that sent the player to several places keeps each. Travel by a conversation, a portal, a rune mark or a spell is not a link and is not recorded. The records are world state, kept in saves; a save imported from the game starts with none, since the game keeps none.

On both maps a cell with a recorded link draws a small bright dot at its symbol's corner, so the player can see which ways it knows. On the full map, resting the pointer on such a cell shows where it leads:

- **On the same level**, a curved arrow from the cell to each place it led, in the link's color, ending in a small ring on the arrival cell, with the place's distance in cells beside it when the arrow is long.
- **To another level**, a label by the cell naming the level (its name, or "Level n" without one) and the arrival cell's coordinates, with a short arrow from the cell to the label. A label for a level the full map can show is a button: clicking it switches the map to that level, centered on the arrival cell, with the arrival cell marked.

The same arrows and labels show for the links leading to the cell under the pointer, drawn dashed, so the player can also see where they came from. With the pointer on nothing, the full map shows no arrows.

**Other levels.** The full map can show any level the player has visited whose map setting is shown, with its seen cells as they were last seen (each level keeps its own). Page Up and Page Down step through the visited levels in the order they were first visited, and clicking a link's label goes to its level, as above; Home comes back to the current level. On a level other than the current one there is no player marker, and the title says it is not the current level ("Level name — visited"). The minimap always shows the current level.

## Liquids

The header's liquid kind:

- **2, fire.** Entering a cell of terrain 1Bh–1Eh hurts the player 2d2 fire damage. Every status tick (player.md) on the level hurts 2d2 fire damage wherever the player stands. Fire immunity and levitation prevent both.
- **1, water-like.** Terrain 1Dh hurts the player 1 on entering, and creatures avoid it. Terrain 1Eh drains 1 magic point on entering.
- **Deep water** (formats.md): terrain 1Bh, 1Ch, 1Dh, 1Eh are depths 1, 2, 3, 4. At depth 4, every move has a 1 in 4 chance of costing 1 hit point, and every status tick costs 1, unless the player levitates. At depth 3 or more the player cannot throw or use hand actions.

## Import

**Floor types.** Terrain 1Bh–1Eh becomes floor types 0–3, drawing floor variants 0–3. Every other cell with a floor (walls, doors, plates, levers, slots, marks, teleporters, pits, trapdoors, the socket, hazards) draws variant 0; on a level with liquid effects these get a plain type of their own, so the effects stay on 1Bh–1Eh:

| Level | Types 0–3 (1Bh–1Eh) | Plain type |
| --- | --- | --- |
| No liquid | variants 0–3 | type 0 |
| Water-like (1) | 1Dh enters for 1 damage and is avoided; 1Eh enters for 1 magic point | variant 0 |
| Fire (2) | enters for 2d2 fire damage; every status tick 2d2 fire | variant 0, every status tick 2d2 fire |
| Deep water | depths 1–4; objects sink | variant 0; objects sink |

**Wall shapes.** Shapes 0, 2, 4 and 5 are diagonal; 1 and 3 are faces. A diagonal wall fills the half of its cell at one corner of the cell's diamond, which decides which way the player slides along it (entities.md, Moving):

| Shape | 0 | 2 | 4 | 5 |
| --- | --- | --- | --- | --- |
| Filled corner | bottom (+u, +v) | left (+u, −v) | right (−u, +v) | top (−u, −v) |

**Features.** A cell's terrain and its placement (formats.md) make its feature (rules.md, Import). Hazard tiles (29h–35h) stay fixed art.

## Unknowns

- The engine's map: its exact colors, which belong to the remake's palette of panels.
- The use of terrain 0Fh and 12h–14h, which no level is known to contain.
