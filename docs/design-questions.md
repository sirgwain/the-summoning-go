# Design questions

Choices made while drafting that may need revisiting. Each gives the choice, why, and what would
change it.

## Gem socket opening is content, not engine

**Choice.** The engine has no special rule for the gem socket opening its wall. A socket feature
counts its gems and, when all are set, runs its `a` procedure. The Summoning importer writes that
procedure as `set_floor` and `set_wall` actions on the cells the spec names (`docs/spec/rules.md`,
Gem socket).

**Why.** The opening is level geometry, not a mechanism. As actions, an author can make a socket
open anything, and the engine stays general.

**Revisit if** the opening needs something actions cannot express (the view effect it plays, for
example), in which case the effect becomes an action too rather than a socket rule.

## Plates at level load

**Choice.** Objects and creatures that a level places on a plate add their weight to its counter
when the level is first entered, without running any procedure. The plate starts in the state its
terrain shows.

**Why.** The spec does not say what the original does (`docs/spec/rules.md`, Unknowns). This keeps
a level as its author drew it: no procedure fires before the player arrives, and the first removal
behaves as expected.

**Revisit when** testing against the game shows plates starting differently.

## Level-local ids

**Choice.** A level's features, things and procedures have ids local to the level (`door.12.9`,
`boulder.8.2`, `p5`), and actions reach features and things by those ids. Only actions that act on
a place rather than a thing (`create`, `set_floor`, shooting) take a cell.

**Why.** The original's procedures address cells, but an editor moving a door should not break the
actions that open it. Pack-wide ids would be noise for things that exist only inside one level.

**Revisit if** something outside a level (a conversation, a spell) needs to reach a level's
feature. It would then name the level and the local id together.

## One sprite anchor per sheet

**Choice.** A sprite sheet has one anchor for all its frames, or one per frame, and things are drawn
with it at their ground point. The importer derives anchors from where the original draws each kind
of thing.

**Why.** It replaces the original's per-kind placement and standing offsets with one idea: where a
sprite touches the ground.

**Revisit if** one sheet is used by two kinds of thing that the original places differently. The
importer would then write the sheet twice, or anchors would move to the template.

## Exit markers as floor features

**Choice.** An exit marker is an `exit` feature on a floor cell, taken by walking onto the cell. In
the original it is an entity with its own collision box, taken by walking into it.

**Why.** It is a fixed thing tied to one cell, like the other features, and needs no update or
movement.

**Revisit if** play shows the difference: walking into the marker's box can happen before the player
reaches the cell.

## Readable ids where the data has names

**Choice.** The importer names records after the data's own names where there are any (items) and by
kind and number otherwise (`sprite.res.1045`, `thing.npc.12`).

**Why.** NPC records, sprite banks and tile banks have no names in the original files. Inventing
names would be guesswork; numbered ids are stable and an author can rename them.

**Revisit if** names can be found for them, for example from a creature's conversation script name.

## Footprints in world units

**Choice.** Things collide by a square footprint in world units around their ground point, one
size per template. The game measures each kind's rectangle in view pixels around its sprite.

**Why.** Ground points are the engine's way of placing things; a footprint around one collides
the same however the thing is drawn, and an author sets one number.

**Revisit if** play shows things meeting at visibly different distances than in the game, for
example boulders against the player.

## Putting down where the player stands

**Choice.** The player puts objects down on its own cell only. A sealed door, which a cell cannot be
stood on while closed, is opened by a thrown sphere, so the dev pack's sealed door waits for
throwing (Fight).

**Why.** That is the game's rule for dropping; reaching another cell is what throwing is for.

**Revisit if** a puzzle needs placing an object on a neighboring cell without throwing it.

## Worn slots after Walk

**Choice.** Walk gives the player a hand, twenty shelves, the two hands and the pack. The worn
slots refuse objects until Fight adds what each takes and what wearing does.

**Why.** What a worn slot takes depends on item categories and powers the pack does not describe
yet.

## Floor halves stay fixed

**Choice.** A level's floor halves are computed once, as if every cell were visible, and play draws
only the visible cells. The game recomputes each wall's halves from which neighbors are visible.

**Why.** The visible room's walls already show the halves that face it; the difference is a wall
between two rooms, which shows both halves while only one room is visible. Recomputing would bring
the game's check-direction tables into content.

**Revisit if** the leftover halves are noticeable in play.

## Moving the player between levels from tools

**Choice.** With a player, the viewer's level change and exit following move the player: to the
exit's cell, or to the start cell on the start level and the first open cell elsewhere.

**Why.** The current level is where the player is; a tool that changed level without it would leave
the player standing at its old position on another map.

## Creatures walk unless marked

**Choice.** A creature template walks unless it is `animated` (it only animates in place and cannot
be struck) or `stationary` (it stands and casts). The game tells these apart by its frame count.

**Why.** Walking is what most creatures do; the exceptions are what an author marks.

## Striking reach

**Choice.** The player strikes what its footprint would overlap one full step ahead, as the game
tests a step. With footprints of 16 that means walking right up to a creature.

**Revisit if** fights feel cramped next to the game, where the rectangles in view pixels may reach
a little differently along each axis.
