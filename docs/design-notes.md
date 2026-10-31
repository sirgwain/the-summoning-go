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

## In progress: 5. Fight

**Done (stage 1, melee).** `engine/combat` with the Summoning's rule set; attributes, skills,
practice, experience and levels; striking with either hand by an item's actions, with delays,
skill requirements and bare hands; creatures that wander, chase, strike, hurt on contact, poison,
die, drop what they carry and leave corpses; worn slots with armor class, body armor and weapon
wear, weapons breaking, and the speed powers while worn; the player dying; poses for striking,
walking and corpses. The importer writes item actions, armor, wear slots, values and powers,
creature attacks, experience, vulnerabilities and behaviors, and the player's strike frames. The
dev pack's cellar has a brute to fight (`TestCellarFight`), and boots and mail.

**Done (stage 2).** Things in flight: throwing from either hand, firing bows from a worn quiver,
arrow traps (`shoot_item`), hitting, landing, arrows breaking, bottles breaking, teleporters
turning them, and thrown spheres unsealing doors (the sphere is used up). Using objects by their
`use` and `use_up` actions, with charges and spent objects; the powers that need only the player
(healing, magic, cures, skills, attributes and their boosts, statuses, timed effects, speed,
teleporting to rune marks, smiting and freezing creatures); powers that start worn only with the
80h flag. Statuses and timed effects in combat, walking, plates, pits and floors; death protection
and the death-blow ward; exertion, fatigue and confusion. The starting inventory, whose boots now
give the speed bonus. The dev pack's hall is solved by playing it, sphere throw included.

**Done since.** Held objects drawn in the player's hands from the game's held-item table; the
corpse frame as the bank's last; damage shown over the hurt; the dev pack's arena, an e2e test of
a whole fight.

**Left.**
- Powers that cast spells, start conversations, show the map or choose the ending (they report
  "not yet"); transmuting and joining staff halves; charges absorbing damage.
- Special weapons: the Rusted Blade and WarMonger, the Staff of the Serpent's poison, creature 79h,
  the Decanter's breaking.
- Practice of 2 for killing a creature of record 50 or above without a conversation, and the dead
  bit on its record (both need records, with dialogue).
- The flying looks of the seven thrown-look templates (the bank is unknown); flights draw the
  item's floor image.
- Animated objects' patterns (drawn still), teleported creatures spinning.
- Messages hidden by "quiet combat"; held objects hidden in water 3 deep or more; worn looks
  (head and body armor). The player's figure and held layers draw with the palette's own
  colors, unanimated (`drawlist.Cmd.Fixed`); character creation sets them from the portrait.
- Damage numbers are drawn with the debug font until the UI has its own (Remaster UI).
- See a fight on the imported game on screen. The dev pack's arena fight runs through the viewer
  in `TestArenaFight`; watch it with
  `summoning view packs/dev --level level.arena --script devpack/scripts/arena.json --out /tmp/arena`.

## Left from Walk

- **Play a real level through.** Only a short walk on level 0 of the imported game has been tried.
  Play the first level or two to the next level, and fix what the import or the simulation gets
  wrong on the way (the import report lists procedure ops it dropped).
- **Smooth steps (optional).** The player and boulders move in steps of 8 units, as in the game.
  Drawing them gliding between steps would change only the picture.

## Handed on

### 5. Fight

- See above. Also confusion in walking, and haste and battle rage in the speed formula.

### 6. Magic and talk

- Spells and gestures, `shoot_spell`, conjured things (pushable, timed), wall spells (Gateway).
- Dialogue: talking to creatures, keywords, conversation actions and the procedures they run.

### 7. Game

- Scenes: the intro, character creation (which replaces the starting attributes and bonus the
  importer writes, and sets the figure's colors E0h–E3h from the portrait: the presenter's fixed
  table), story screens, endings, and the three arrival styles when changing level.
- Saves, audio (music and effects as Ogg with loop points), the level map and its refusals.
- Level names, which the import leaves empty.

### 8. Remaster UI

- Play uses the game's key layout (Space and Q for the hands, X and Z to throw, F take, G put
  down, Enter lever, E talk, H recenter, P pause; 1–4 and R wait for spells). The inventory's
  stand-in keys are on Ctrl until the inventory has its panel and the mouse works it.
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
