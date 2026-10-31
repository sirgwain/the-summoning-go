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

## New game

The frontend makes the character (player.md) and hands it to play. A new game shows the opening door pictures and the door opening, then the player arrives on the starting level, and the opening conversation (7Dh) starts (dialogue.md).

## Death

When the player's hit points reach 0 at a status tick, or a death blow lands, play ends and the game returns to the main menu.

## Endings

Power 5Eh (items.md) and the conversations choose an ending:

- holding the Chesschantra Skull (F7h): ending F, with NPC record 4;
- holding WarMonger (64h): ending G, with NPC record 5;
- holding the Necklace of Runes (59h): ending E, with NPC record 3, after conversation 79h.

The choice is stored in interaction variable 32. The ending shows the closing picture spread in, a fade to white, the final scene, and the closing texts (TEXT records 157h–170h). End-scene art starts at RESOURCE `end-scene base` (INIT).

## Unknowns

- The frontend: menus, character creation (attribute rolls, portraits), the save and load screens.
- The frame lists and placement of each view effect.
- Which track plays in which scene and level.
- The full ending sequences and which closing texts belong to which ending.
- The teleport and ripple timings.
