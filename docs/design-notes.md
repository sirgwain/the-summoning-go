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
   is solved by playing it (`TestHallPlays`). The hotbar (`engine/viewer/hotbar.go`): portrait,
   gauges, both hands with their action buttons and the twenty shelves; the mouse takes and puts
   down by clicking the player, swaps the pointer with a box, and sends the pointer's object to a
   shelf with the right button, through the slot code the inventory panel will share
   (`slots.go`); a key goes into a keyhole by mouse (`TestKeyIntoSlotWithMouse`). Portraits
   import from F and give the figure its colors E0h–E3h.

7. **Magic and talk** (done out of order, before Fight's last items and the Inventory panel).
   Spells are records with effects from the action registry (missiles, timed effects, conjured
   walls, portals, teleports, dispel, join, freeze, shape change, transmute, cures); the player
   knows gesture icons, learns spells into a spellbook by entering gestures, memorizes within
   the school's limit and casts with cost, conditions and the gesture playing one icon a player
   update (`engine/sim/magic.go`, `missile.go`). Creatures cast (walkers 1 step in 10, stationary
   casters on their cycle), procedures `shoot_spell`, and objects cast or start conversations.
   Conversations are records of clauses run in resumable passes, stopping at each line
   (`engine/sim/talk.go`); keywords match on four letters; counters, per-template records
   (talked, dead, portrait) and the engine's counters named in `pack.json`. The importer reads
   INIT's spells, KEYWORDS and INTERACT, scrolls and their pictures, the NPCs' talk portraits
   (people banks past F's), and casting and talking powers. The hotbar has the schools' spell
   boxes and the gesture panel; F10 opens the spell panel; the conversation window, drawn at
   twice its size and centered in the view above the hotbar, shows lines with marked keywords,
   offered keywords and a field. The pointer is the game's (resource 2, `pointers.json`): the
   speech bubble over a creature that talks or a mouth, where a click talks to it; a mouth's
   message shows beside it. An object coming onto the pointer is read, so selecting a scroll
   teaches its icons and shows its picture above the hotbar. The dev pack's study is played by
   `TestStudy` (`devpack/scripts/study.json`); the imported game's first conversation (Malachi,
   level 0) plays through.

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
- Powers that show the map or choose the ending (they report "not yet"); charges absorbing
  damage.
- Special weapons: the Rusted Blade and WarMonger, the Staff of the Serpent's poison, creature 79h,
  the Decanter's breaking.
- Practice of 2 for killing a creature of record 50 or above without a conversation (the world's
  records now keep talked and dead).
- The flying looks of the seven thrown-look templates (the bank is unknown); flights draw the
  item's floor image.
- Animated objects' patterns (drawn still), teleported creatures spinning.
- Messages hidden by "quiet combat"; held objects hidden in water 3 deep or more; worn looks
  (head and body armor). The player's figure and held layers draw with the palette's own
  colors, unanimated (`drawlist.Cmd.Fixed`), and the portrait's figure colors.
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

### 6. Inventory

- The inventory panel: the mannequin with the worn slots (head, arms, hands, feet, quiver, body,
  neck), the pack of the bag in the left hand, the stats (attributes, skills, armor, experience),
  the chosen object's details. Built on the hotbar's `SlotBox` and pointer code; the game's object
  icons, no panel art. It replaces the Ctrl stand-in keys (Ctrl+R/L hands, Ctrl+W put on,
  Ctrl+F1–F8 pack).
- Hotbar action buttons as the game's action icons rather than words.
- The pointer's object drawn over the hotbar's cursor in scripts (scripts park the cursor off
  screen between clicks).

### 7. Magic and talk, left

- Gesture timing: the game holds some icons over two updates by a table of which may follow
  which (spells.md, Unknowns); the engine shows one an update.
- Shape Change draws the creature's first frame, not its walk; the player's held objects hide.
- Records 23h and 2Bh get speed 16 when a conversation turns them hostile (creatures.md,
  Hostility); not done.
- Interaction variables the game reads itself beyond 14, 19, 32 and 97: 96 (quiet combat), 98
  (sight), 34 (the opening conversation on a new game) wait for their milestones.
- Gateway's level 30 and the conversations that return from it are imported but not played
  through on the imported game; nor is any quest that a conversation drives across levels.
- Powers are still numbers the engine knows (items.md, Powers); casting and talking powers
  became item fields (`casts`, `conversation`), the rest could become records too.
- See a spell in flight and its burst on the imported game on screen: the dev pack's are drawn
  (`TestStudy`), the imported ones only checked by rule.

### 8. Game

- From Magic and talk: conversation actions that report "not yet" (the ending, the disk menu's
  quit, save, restore, delete and rename, printing the transcript) and view effects (`effect.n`
  events, and effect 6's level change, which does happen); the player's name in texts (the
  viewer says "Stranger").
- Scenes: the intro, character creation (which replaces the starting attributes and bonus the
  importer writes, and chooses the portrait in place of portrait 0; F's premade characters,
  and the school's first gesture icons, which the importer gives school 0's for now),
  changing sex (the other-sex portrait), story screens, endings, and the three arrival styles
  when changing level.
- Saves, audio (music and effects as Ogg with loop points), the level map and its refusals.
- Level names, which the import leaves empty.

### 9. Remaster UI

- Play uses the game's key layout (Space and Q for the hands, X and Z to throw, F take, G put
  down, Enter lever, E talk, H recenter, P pause; 1–4 and R wait for spells).
- Panels from the game's art (the hotbar, the inventory and spell panels and the conversation
  window draw plain boxes and the debug font until then; the game's spell panel art is resource
  7, its icon strip resources 51 and 52),
  hotkeys for the hotbar's objects, magnified floor objects as a game setting, and the custom
  outline font (the hotbar's labels and tooltips use the debug font). Play mode's text panel is a
  stand-in.

### 10. Modding

- Override packs, `summoning verify` as a command, and the editor.

## Simplifications to revisit

Recorded with their reasons in [design-questions.md](design-questions.md):

- Footprints in world units instead of the game's per-kind rectangles in view pixels.
- Floor halves fixed at import; the game's wall shapes that change with what is visible are not
  done (world.md, Visibility).
- Exit markers as floor features rather than entities.
- The player arrives at the center of the destination cell (entities.md, Unknowns).
