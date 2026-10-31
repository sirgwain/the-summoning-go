# Design notes: progress and what remains

Status: 2026-10-31. Where the milestones of [design.md](design.md#milestones) stand, and the work each
has left or handed on. Choices that may need revisiting are in
[design-questions.md](design-questions.md); what the game's behavior leaves open is in each spec's
**Unknowns**. Update this file as work lands: move an item to the milestone that does it, and drop
it once done.

## Done

1. **Viewer.** Levels, sprites and palettes import; levels draw with the palette shader in the
   isometric projection; the camera pans and zooms; palettes animate.
2. **World.** Features, procedures, items, things and placements import; the simulation runs every
   mechanism; the viewer inspects cells and things and works mechanisms by debug commands.
3. **Harness.** Input snapshots, the viewer's controls without Ebitengine, scripts with shots and
   dumps, state dumps, and the dev pack checked in at `packs/dev`.
4. **Walk.** The player with attributes, speed and load; movement, footprints, sliding, pushing;
   rolling boulders; visibility and seen cells; exits, pits, teleporters, plates and floor
   effects; picking up, putting down, the hand, shelves, hands and pack; using levers and mouths;
   `summoning play`, with the view following the player as the game's does. The dev pack's hall
   is solved by playing it (`TestHallPlays`).

## Left from Walk

- **Play a real level through.** Only a short walk on level 0 of the imported game has been tried.
  Play the first level or two to the next level, and fix what the import or the simulation gets
  wrong on the way (the import report lists procedure ops it dropped).
- **Smooth steps (optional).** The player and boulders move in steps of 8 units, as in the game.
  Drawing them gliding between steps would change only the picture.

## Handed on

### 5. Fight

- Worn slots (20, 21, 24–27): what each takes by category, armor, and starting and ending worn
  objects' powers. Then the Leather Boots give the starting player's −2 interval bonus, which the
  template's `interval_bonus` stands in for now.
- The starting inventory (player.md, Starting a new game).
- Touching creatures, creature AI and `avoided` floors, combat, death of creatures and the player.
- Confusion, haste and battle rage in the speed formula; exertion and fatigue.
- Throwing and firing: things in flight, landing, breaking; the sealed doors that thrown spheres
  open (the dev pack's door E waits for this); arrow traps (`shoot_item`).
- Using objects: action icons, charges, using up, and the powers table (items.md, Powers). Many
  powers also need statuses or spells.
- Statuses: levitation (plates, pits, pushing corpses), fire immunity, poison, regeneration.

### 6. Magic and talk

- Spells and gestures, `shoot_spell`, conjured things (pushable, timed), wall spells (Gateway).
- Dialogue: talking to creatures, keywords, conversation actions and the procedures they run.

### 7. Game

- Scenes: the intro, character creation (which replaces the starting attributes and bonus the
  importer writes), story screens, endings, and the three arrival styles when changing level.
- Saves, audio (music and effects as Ogg with loop points), the level map and its refusals.
- Level names, which the import leaves empty.

### 8. Remaster UI

- Panels from the game's art, the quickbar, hotkeys, an inventory the mouse works, magnified floor
  objects as a game setting, and the custom outline font. Play mode's text panel is a stand-in.

### 9. Modding

- Override packs, `summoning verify` as a command, and the editor.

## Simplifications to revisit

Recorded with their reasons in [design-questions.md](design-questions.md):

- Footprints in world units instead of the game's per-kind rectangles in view pixels.
- Floor halves fixed at import; the game's wall shapes that change with what is visible are not
  done (world.md, Visibility).
- Exit markers as floor features rather than entities.
- The player arrives at the center of the destination cell (entities.md, Unknowns).
