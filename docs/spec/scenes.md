# Scenes

Everything shown around the play of a level: the menus and character creation, the opening, view effects, level transitions, death and the endings, and the palette's animation.

This file is the least complete part of the specification. It is written ahead of the milestone that implements scenes, and its Unknowns list is long on purpose.

## Palette animation

Every 8 ticks, in the active palette:

- Colors E0h–EFh rotate in four groups of four (E0h–E3h, E4h–E7h, E8h–EBh, ECh–EFh), and F0h–FBh in three (F0h–F3h, F4h–F7h, F8h–FBh). In each group, each color takes the value of the next, and the last takes the first's.
- Color FCh steps through four colors, in 6-bit components: (15, 0, 27), (24, 6, 38), (34, 14, 47), (45, 26, 57).

Anything drawn in E0h–FCh animates without being redrawn: fire, glows, water.

The player's figure does not animate. Its colors E0h–E3h are the character's own (hair, skin), the figure colors of the portrait chosen in character creation (player.md, Portrait), and the figure and the objects drawn in its hands (player.md, Held objects) are drawn with them rather than with the animated palette. Until character creation the importer's starting portrait (portrait 0) gives them.

## View effects

Short animations played in the view, numbered 0–15, from RESOURCE `view-effect base + …` (INIT). Each has a fixed area:

| Effect | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Width | 96 | 96 | 56 | 80 | 48 | 72 | 120 | 104 | 72 | 152 | 96 | 72 | 88 | 72 | 72 | 72 |
| Height | 64 | 64 | 96 | 80 | 112 | 64 | 88 | 64 | 80 | 32 | 80 | 64 | 64 | 80 | 88 | 80 |

Known uses: 2 a fall; 5 a stationary caster's death; 0 and 1 play their frames forward or backward with a mirror image below; 6 is played by conversations and ends in a level change (dialogue.md). Effects can be cut short with Esc or the right mouse button.

Other effects: the pit fall, stepping down into a pit, falling through a trapdoor, the death blow, the teleport ripple, and effects centered on the player for healing and cures.

## Level transitions

When the player changes level the new view is drawn hidden and then shown by one of:

- **Scroll**: the view scrolls in from the side (exits in walls, conversations).
- **Fall**: the old view slides out and the new one in, in steps of 1, 1, 2, 6, 12, 24 rows (pits).
- **Fall-through**: as fall, sliding in from above (trapdoors).
- **Teleport**: a circle closes, the views slide, the player arrives, the circle opens.
- **Ripple**: for ripple teleports within a level.

## The game screen

The view sits in a frame of panels: the character's portrait and gauges, the hands, the spell panel, the message line. Panels are drawn over a stone texture made from the INIT texture shades. The engine draws its own panels (design.md); the original layout is reference only.

## The frontend

The game starts at a main menu: create a character, restore a game, or select a premade character.

**Creating a character** goes through:

1. **The descendant**: one of F's portrait rows (formats.md, F), shown as its portrait; the arrows step through them and R picks one at random. The row gives the character's sex, other-sex portrait and talk portrait.
2. **The name**, at most 25 characters. An empty name takes a stock name by sex.
3. **The attributes**, by one of three methods:
   - **Rolled**: each attribute 3d4 + 2d5, at most 20; rolled again until kept.
   - **Spent**: every attribute starts at 0, with 28d5 points to spend.
   - **Rolled and spent**: each attribute 2d5, with 14d5 points to spend.

   Spent points raise an attribute up to 20, and can be taken back down to its starting value; the character is kept only when every point is spent. Hit points `STR + END + 20` and magic points `TAL + POW + 20` show as the attributes change.
4. **The training**: a weapon class (0 sword, 1 axe, 2 spear, 3 bow) and a magic school.

**A premade character** comes from F's premade records: its name, portrait, attributes, school and weapon class.

## New game

The new character starts as player.md (Starting a new game) says. A new game shows the door: the closed door (RESOURCE 3Eh), the door pictures drawn over it at column 135, each held 5 ticks (RESOURCE 55h frames 0–3 at row 0, then 56h frames 0–9 at row 0 with frame `n + 10` below it at row 100), and the door opening (RESOURCE 54h's halves moving apart over the dungeon behind, RESOURCE 3, in 20 steps). Then the player arrives on the starting level; interaction variable 34 is set to 1, which starts the opening conversation (NPC record 7Dh) on arrival (dialogue.md).

The door pictures 3Eh and 54h are 320 × 200 pictures stored as 160 tiles of 8 × 50 pixels, four rows of forty, row by row. Drawn with a spread `s`, the columns within `s` of either edge are left out and the rest move `s` columns outward from the middle, so the halves part.

**Engine model.** The pack's creation record (`creation.json`) gives what character creation offers: the portraits to choose from, the attribute methods, the weapon classes and schools with their names and what each adds to the starting inventory, the objects every character starts with, the body armor picked among at random, the premade characters, and what a new game does (the counter it sets, the conversation that opens it, the scene it shows). The engine's frontend runs these rules with its own menus; the game's own captions and stock names are in its program, not its data files, so a pack gives its own or does without.

## Death

When the player's hit points reach 0 at a status tick, or a death blow lands, play ends and the game returns to the main menu. The engine shows the player dead for a moment, then the main menu.

## Endings

Power 5Eh (items.md) and the conversations choose an ending:

- holding the Chesschantra Skull (F7h): ending F, with NPC record 4;
- holding WarMonger (64h): ending G, with NPC record 5;
- holding the Necklace of Runes (59h): ending E, with NPC record 3, after conversation 79h.

The choice is stored in interaction variable 32. The ending shows the closing picture spread in (the door closing: RESOURCE 54h's halves coming together over the view, then 3Eh), a fade to white, the final scene (RESOURCE 5Ah, then frames 0–4 of 5Bh), and the closing texts (TEXT records 157h–170h), each shown for 2Dh ticks on the banner RESOURCE 5Ch. Then the frontend plays the chosen end scene. End-scene art starts at RESOURCE `end-scene base` (INIT).

**Scenes, engine model.** A scene is a record of steps the presentation plays in order: show a picture, part or close a picture's halves over a background in a number of steps, show a caption, fade to a color, wait. A key or a click skips to the end. The importer writes the new-game door and the ending as scenes; the end scenes and the intro, whose captions are in the game's program, are left out.

## Unknowns

- The frontend's captions, its stock names and the end scenes' captions, which are in the game's program rather than its data files; the save and load screens.
- The frame lists and placement of each view effect.
- Which track plays in which scene and level.
- The full ending sequences and which closing texts belong to which ending.
- The teleport and ripple timings.
