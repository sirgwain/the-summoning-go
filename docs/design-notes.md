# Design notes: progress and what remains

Status: 2026-11-03. Where the milestones of [design.md](design.md#milestones) stand, and the work each
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

5. **Fight.** `engine/combat` with the Summoning's rule set; attributes, skills,
   practice, experience and levels; striking with either hand by an item's actions, with delays,
   skill requirements and bare hands; creatures that wander, chase, strike, hurt on contact, poison,
   die, drop what they carry and leave corpses; worn slots with armor class, body armor and weapon
   wear, weapons breaking, and the speed powers while worn; the player dying; poses for striking,
   walking and corpses. The importer writes item actions, armor, wear slots, values and powers,
   creature attacks, experience, vulnerabilities and behaviors, and the player's strike frames. The
   dev pack's cellar has a brute to fight (`TestCellarFight`), and boots and mail.
   Things in flight: throwing from either hand, firing bows from a worn quiver,
   arrow traps (`shoot_item`), hitting, landing, arrows breaking, bottles breaking, teleporters
   turning them, and thrown spheres unsealing doors (the sphere is used up). Using objects by their
   `use` and `use_up` actions, with charges and spent objects; the powers that need only the player
   (healing, magic, cures, skills, attributes and their boosts, statuses, timed effects, speed,
   teleporting to rune marks, smiting and freezing creatures); powers that start worn only with the
   80h flag. Statuses and timed effects in combat, walking, plates, pits and floors; death protection
   and the death-blow ward; exertion, fatigue and confusion. The starting inventory, whose boots now
   give the speed bonus. The dev pack's hall is solved by playing it, sphere throw included.
   Held objects drawn in the player's hands from the game's held-item table; the
   corpse frame as the bank's last; damage shown over the hurt; the dev pack's arena, an e2e test of
   a whole fight.
   **Finished.** The counted weapons (the Rusted Blade growing into WarMonger, WarMonger's
   levels), the Staff of the Serpent's poison, the mirroring creature (79h), charges absorbing
   damage, the Decanter's breaking (`on_break`), the kill message with the creature's name and the
   kill practice, the thrown looks of the seven templates, the animated patterns (driven by the
   simulation: `animate`), teleported creatures spinning, quiet combat, held objects hidden in
   deep water. Tested in `engine/sim/special_test.go`; a fight with the mercenaries of level 1
   of the imported game seen on screen.

6. **Inventory** (done before Fight's last items). The inventory panel (`engine/viewer/inventory.go`),
   opened with I, Tab or a click on the portrait: the game's paper doll with each worn object's
   worn look (mirrored in the right hand, the quiver slot's own look), a box for each worn slot,
   the pack of a bag in the left hand, the details of the object under the pointer or on it
   (weight in half kilograms, damage, armor, two hands), and the attributes with their maxima,
   points, armor, level title, experience and the skills with their icons and ratings. The
   hotbar's hands have three action boxes each, the action's icon alone, the third throwing. The importer writes `inventory.json` and each item's `doll` look and
   action icons; the Ctrl stand-in keys are gone. Scripts keep the cursor where they last tapped,
   so the pointer's object shows in their screenshots. Tested by `TestInventoryPanel` and
   `TestHandActionBoxes`.

7. **Magic and talk** (done out of order, before Fight's last items).
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

## In progress: 8. Game

**Done (saves).** The world as plain data (`engine/sim/state.go`): everything that changes, with
templates and things by id and the random source's state, so a restored world runs on exactly as
the saved one would (`TestStateRoundTrip`). `engine/save` writes it gzipped as JSON with the save's
name and time, and warns when it loads on another pack or version. In play, F5 saves to the quick
save, F8 loads it, and changing level writes the autosave; saves go under the user's config
directory per pack (`--saves` to choose). A conversation in progress cannot be saved, nor a world
while a transition, view effect or sparkle plays or something waits for one; the autosave of a
change of level waits until they end (`TestNoSaveWhileHolding`, `TestNoStateWhileWaiting`).
`--load` starts play or the viewer from a save file, or a bare name in the saves directory.

**Done (the original's saves).** `import-save` reads the game's save files (spec/saves.md) with its
data files and the imported pack, and writes engine saves: a new world with each visited level
loaded, then the save laid over it (the player, inventory, magic, counters, NPC records,
terrain, feature states, creatures, boulders, traps, floor objects). What the engine has no place
for is reported (`TestReadSave`, `TestImportSave`).

**Done (audio, most of it).** The importer renders the game's songs through `internal/opl2` to
WAV with loop sidecars; `engine/audio` mixes music and effects in layers with the game's fades,
and `speaker` plays them (`TestLayers`, `TestFades`). Combat, object, missile and gesture effects,
footsteps, the new game's track, the theme of NPC record 0Dh's creature, the fades on level
change, restore and leaving play, the door's sound at each step of its opening and the splash's
track 3 are in; F9 cycles music and effects, effects only and silence.
Checked by ear against the game. What is left is in [audio-plan.md](audio-plan.md) phase 6.

**Done (the level map).** Levels have names (TEXT 129h + level) and a map setting; levels 13h and
21h–23h refuse the map with a message or conversation 7Dh. M shows the full map and the minimap
shows the level about the player in the top-left corner, unless the level refuses it (Remaster
UI, below, for their look and the links). Power 22h (the Palimpsest) shows the map too and is never used up.

**Done (character creation, in the simulation).** The pack's creation record (`creation.json`):
portraits, the three attribute methods, the weapon classes and schools with what each adds, the
objects every character starts with, the body armor picked at random, the premade characters, the
new game's counter, opening conversation and scene, and the ending. The importer writes it from F
(portraits and premade characters) and the starting rules of player.md, with the door and the
ending as scenes (`scenes/`). `sim.NewGame` starts a world from a made character
(`TestNewGame`); the player's name replaces @ in texts. The dev pack has a creation record.

**Done (the ending, in the simulation).** Power 5Eh chooses an ending by the item held, sets
variable 32 and runs conversation 79h; when it ends, play ends (`World.Ended`, `EventEnding`)
with the ending's name. The conversations' `ending` action ends play with the closing scene
(`TestEndingPower`, `TestEndingAction`). Not done: ending G's changes to the world and its save.

**Done (the frontend).** `summoning play` starts at the main menu when the pack has character
creation (`engine/viewer/frontend.go`, drawn by `engine/viewer/window`): create a character (the
descendant, the name, the attributes by the pack's methods, rolled or spent within their limits,
the weapon and the school), restore a save, or pick a premade character; then the new game's
scene and play. A dead player shows for three seconds, then the menu; an ending plays its scene,
then the menu. Scenes play pictures, parts, captions over banners and fades by the clock; a key
or click skips. Tested by `TestCreateCharacter`, `TestPremadeAndBack`, `TestDeathAndEndingGoToMenu`
and `TestSceneRun`. The splash fades its art in, fills its title in from the left and plays the
opening track; the menus sit over it in the remake's iron frame, lettered in `engine/font`. Seen
on screen: the splash, the menus and the door.

**Done (the story, in short).** Create Character (or a premade) starts track 1, which plays
through the creation screens, fades on going back to the menu, and fades as the story ends.
Before the door, the story scene plays on the frontend's 320 × 400 screen (shown with half-height
rows): three pages of the engine's own summary, Rowena's farewell picture with the gifts by
weapon class, the conversation in two iron-framed panels over the picture darkened (Rowena's
portrait, F's third header word, above; the character's, mirrored as its row says, below), and
the second picture as she sends you away. Scenes gained `size`, `text`/`box`, `say`, `by_weapon`
and `crop`; a key or click ends a page early, Escape skips the story (`TestStory`,
`TestGameStory`). Scripts can start at the menu with `--menu`. Seen on screen through a script.
The full seven-part story is still to be specified.

**Done (the end scenes).** Each ending plays its end scene after the closing: the god of Magic
(E), Chesschantra's return (F) and Borel's farewell (G), on the 320 × 400 screen in palette 7
with its reds and golds cycling. The importer plays each by spec/scenes.md's rules at a fixed
seed (`importers/summoning/endscenes.go`): what stays is the scene's pictures, and what moves (the
glow, figures called up and growing, sparkles, the blast's trail, the world's end's battles and
flights, Abighael's walk, the two shrinking into the portal) is a frame of pictures over them,
one an update. Scenes gained `over`, `palette`, `mirror`, captions in a box and `no_caption`,
fades in from and over a color and of some palette colors, `cancel` runs, `escape_only`, and
music steps. The captions are the engine's own words. `summoning play --scene` plays a scene
first. Tested by `TestSceneOver`, `TestEndScenes` and `TestGameEndScenes`; seen frame by frame
through the pack.

**Done (the disk menu).** O opens the pack's menu conversation (NPC record 0's; the dev pack's
offers each menu by keyword). Its `menu` actions open frontend screens over play: save under a
typed name or over a save, restore, delete, rename, and the quit and leave questions; then the
conversation goes on (`TestDiskMenu`). Saves are named files under the save directory.

**Done (smaller rules).** Changing sex (power 14h) takes the portrait's other-sex portrait. The
transcript: while variable 99 is set, conversation lines and questions, and the headings of
action 1Ch, go to `transcript.txt` beside the saves, wrapped at 80 columns with keywords
underlined (`TestTranscriptFile`). A debug command sets a counter (`input.SetCounter`). The map
shows the game's kinds (plates, rune marks, gem sockets, trapdoors besides walls, doors, exits,
mouths, teleporters and pits) with a legend. Holding N naps: the world runs 64 times as fast
until something needs the player. Play starts with the debug overlay hidden (`--debug` shows it).

**Done (Fight and Inventory leftovers).** The player's figure wears what it wears: torso, legs,
body and head looks chosen by the worn slots and the portrait's sex, drawn frame for frame. The
Sack shows how full it is, in its boxes and on the doll (items' `fill`). Hand delays count in
points of three player updates; the swing pose holds two updates; action and spell boxes
recharge clockwise. A click near the figure puts down or picks up. A right click on a container's
box opens it as the pack where it lies; resting the pointer on one lists what it holds.

## Left from Walk

- **Play a real level through.** Only a short walk on level 0 of the imported game has been tried.
  Play the first level or two to the next level, and fix what the import or the simulation gets
  wrong on the way (the import report lists procedure ops it dropped).
- **Smooth steps (optional).** The player and boulders move in steps of 8 units, as in the game.
  Drawing them gliding between steps would change only the picture.

## Handed on

### 5–7. Fight, Inventory, Magic and talk, left

Done since: timed idle animations (`drawlist.Animator`), the figure's click regions, the chest's
left-hand look and the hands' armor drawn last (`doll_left`, `doll_over`), attributes in color
and the armor gauge, the gesture timing with the game's holds (`magic.json` `holds`), Shape
Change's frame by facing, and the hostile speed of records 23h and 2Bh (`hostile_speed`).

- Interaction variables the game reads itself beyond 14, 19, 32, 96 (quiet combat), 97 and 98
  (magnify): 34, set when play goes on after ending G, waits for its milestone.
- Gateway's level 30 and the conversations that return from it are imported but not played
  through on the imported game; nor is any quest that a conversation drives across levels.
- Powers are still numbers the engine knows (items.md, Powers); casting and talking powers
  became item fields (`casts`, `conversation`), the rest could become records too.
- See a spell in flight and its burst on the imported game on screen: the dev pack's are drawn
  (`TestStudy`), the imported ones only checked by rule.

### 8. Game

- View effects, sparkles and arrival styles, specified in scenes.md. Done: the importer renders
  the 16 effects and 6 sparkles into flipbooks (`flipbooks/`, pack.md), and `summoning flipbook`
  plays them in a window. The simulation names every effect and sparkle by flipbook (`effect` and
  `sparkle` events; exits', sockets' and sealed doors' fields, spells' actions, and
  `effects.json`'s cues for pits, casters, death blows and powers), and holds what follows an
  effect to its next update. The view plays them with the world paused, placed by scenes.md, and
  its tools play any flipbook (B, K). The dev pack draws a flipbook for each cue and wires one
  to the hall's exit D, socket and sealed door, the Heal spell and the sage's word on fireballs
  (`TestEffectsPlay`, `TestPitEffectInViewer`). Transitions: each change of level or teleport names its cue, and the
  pack gives it a style (iris, slide, squeeze, ripple, fade, cut), drawn from the view as last
  drawn to the new; lost objects shatter or fade in their slot.
- The intro, the new character's full story (it plays in short for now) and the end scenes,
  whose captions are in the game's program: the plan for their spec is
  [scenes-plan.md](scenes-plan.md). Ending G's changes to
  the world and its save, and going on from that save with variable 34 set (dialogue.md).
- Audio's last phase ([audio-plan.md](audio-plan.md)): the view effects' sounds and tracks, and
  the endings' tracks, with the scenes. The story's track plays.
- Play *The Summoning* from start to end on the imported pack.

### 9. Remaster UI

- Play uses the game's key layout (Space and Q for the hands, X and Z to throw, F take, G put
  down, Enter lever, E talk, H recenter, P pause; 1–4 and R wait for spells).
- Done: the remake's own panel frames (iron and sapphire, runic slate for the spell panel) and
  outline font (`engine/font`). The game's panel art is not used, by choice.
- Hotkeys for the hotbar's objects.
- Done: the map in the engine's own style (`docs/spec/world.md`, The engine's map): ink on a
  parchment made from noise (`frame.Parchment`, `frame.Roll`) that unrolls, raised walls,
  glowing symbols, the minimap following the player. The world records the links the
  player travels, an exit's way back when one is beside the arrival, and the order levels are
  visited, what each mouth has said and where each creature that talks was last seen, kept in
  saves (`TestLinksRecorded`, `TestExitTeachesTheWayBack`, `TestMouthsAndTalkersRecorded`); the
  map draws levers, the game's three rune marks by their runes, and those creatures, and quotes a
  heard mouth's words in its note (`TestMapFeatures`); the full
  map shows them on hover and selection, follows them with a right-click, steps back with
  Backspace and shows other visited levels, and opens anew each time (`TestMapLinks`). The
  unrolling is timed by the clock, so scripts' screenshots show the map open.
- Quality of life, designed in [design.md](design.md) (UI; Editor, Links):
  - Done: the spellbook as a scrollable list by school. Each school is headed by its name and its
    memorizations of its limit, each spell a row of icon, name, gesture, cost and memorizations
    with a Memorize button; eight rows show, scrolled by the wheel, Page Up and Page Down or the
    scroll bar. Any gesture that matches a spell adds it (`TestSpellbookList`).
  - Done: the conversation log. The world logs what each speaker says once, by the
    conversation it was first said in (who, the portrait, the level, the step, and each new line
    and gesture taught, with the question that brought it), leaves out the menu's, and keeps it
    in saves, filtering a restored log the same way (`TestConversationLog`); L or the hotbar's Log box opens a window of the people talked
    with, their conversations newest first, and a search that jumps to the line
    (`TestConversationLogWindow`). Saves imported from the game start with an empty log.
  - Done: the book (player.md, The book). The world enters each creature and item template the
    player comes across, in stages (seen, fought or handled, killed or used), with the kills and
    drops, and each defense once it works on one, kept in saves (`TestBookCreatures`,
    `TestBookItems`, `TestBookDefenses`). J or the hotbar's Book box opens it: Creatures and Items
    tabs filtered by a name field, the entries by name, and the chosen one's picture and what its
    stage shows, with a hint of the next; scrolls show their spell from first sight
    (`TestBookWindow`, `TestBookFilter`, `TestBookStages`). Items' and spells' `effect` texts are
    the importer's own words; `lore` is for packs to write.
  - Done: the viewer's link overlay over the pack's own links (`inspect.Links`, `TestLinks`,
    `TestSelectionLinksAndMarks`). Selecting a cell shows what its feature's hooks do, what a
    set_hook would make them do (dashed), where a teleporter sends things and where an exit, pit
    or trapdoor leads on the level, as arrows labeled at their far end; what acts on the
    selection shows the same way and is listed in the panel (but for where things are sent,
    shown only from the feature that sends them). The panel colors the actions it lists as their
    arrows. The selection and the features and
    things its links reach glow in the link's color (`render.Glow`). Left: destinations on other
    levels, and following a link off the screen.
  - Hovering a lever or plate on the map to see what it changed, from what the player has
    observed (design.md, Map of past levels and their links); after the map above.

### 10. Modding

- Override packs, `summoning verify` as a command, and the editor.

## Simplifications to revisit

Recorded with their reasons in [design-questions.md](design-questions.md):

- Floor halves fixed at import; the game's wall shapes that change with what is visible are not
  done (world.md, Visibility).
- Exit markers as floor features rather than entities.
- The player arrives at the center of the destination cell (entities.md, Unknowns).
