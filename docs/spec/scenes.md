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

A view effect is a short animation played in a framed box over the view while the game waits. There are sixteen, numbered 0–15. Their art is RESOURCE `view-effect base + n` (INIT), written `+n` below.

**What plays them.**

| Effect | Played when | Box (w × h) |
| --- | --- | --- |
| 0 | The player steps into a pit along the game's facings 2–4, climbing down (rules.md) | 96 × 64 |
| 1 | The player walks into a level exit marker, climbing through it (rules.md) | 96 × 64 |
| 2 | The player falls into a pit or an open trapdoor any other way | 56 × 96 |
| 3 | Conversation action 1Dh (Chesschantra's ghost) | 80 × 80 |
| 4 | Conversation action 1Dh (Dunstan) | 48 × 112 |
| 5 | A stationary caster dies (creatures.md) | 72 × 64 |
| 6 | Conversation action 1Dh (the citadel guards); ends in a level change (dialogue.md) | 120 × 88 |
| 7 | Conversation action 1Dh (SteelSmiter) | 104 × 64 |
| 8 | The sixth gem goes into the gem socket, before its wall opens (rules.md) | 72 × 80 |
| 9 | Mending joins the staff halves, before conversation 79h (spells.md) | 152 × 32 |
| 10 | Conversation action 1Dh (Rukn) | 96 × 80 |
| 11 | Conversation action 1Dh (Honorah) | 72 × 64 |
| 12 | A death blow lands, before play ends (combat.md) | 88 × 64 |
| 13 | Conversation action 1Dh (Kern) | 72 × 80 |
| 14 | Conversation action 1Dh (Murchad) | 72 × 88 |
| 15 | Conversation action 1Dh (Padrake) | 72 × 80 |

The names say which conversations play each effect in the game's data; any conversation can play any effect. Effects 16 and up do nothing.

**Placing the box.** The box is placed by a creature: the conversation's speaker, the dying caster, or the player for the rest. Its top-left is 16 rows above and 16 columns left of the creature's point on screen, or row 10, column 100 when the creature is off screen or there is none. It is then moved to lie at least 8 pixels inside the 312 × 120 view, its column rounded down to a multiple of 8. A panel border 8 pixels wide is drawn around it. Positions below are (row, column) from the box's top-left. Art drawn as a whole is drawn at its top-left. A bank of one-pixel frames (formats.md, Strips) is the same picture one line per frame, and some effects move its lines one by one: "line `i`" is frame `i`, drawn at row `i` of the picture.

**Time.** The world stands still while an effect plays. A tick is 1/18.2 s. A frame "with no wait" is shown and the next is drawn at once, so its speed was the machine's. A wait marked *cancellable* ends the effect at once on Esc or the right mouse button. Play then goes on as if it had finished. Sounds are sound effects, and tracks are music tracks (audio.md). "Quick fade" is audio.md's.

**Effects 0 and 1 (climbing).** These 24 frames are 5 ticks apart, each wait cancellable. Frame `k` shows picture `p` of `+0`…`+2`: effect 1 steps forward (0, 1, 2, 0, …) and effect 0 backward (0, 2, 1, 0, …). The player's figure for that picture, `+3 + p`, is drawn at (26h, 0) unless the player is invisible. Each figure picture is mirrored on every other use; effect 1's `+3` starts mirrored. Sound 0Fh plays on frames 2, 5, 8, … for effect 1 and on frames 0, 3, 6, … for effect 0.

**Effect 2 (a fall).** Sound 15h plays, then `+6`, `+7` and `+8` in turn, nine frames, 5 ticks apart, each wait cancellable.

**Effect 3 (the ghost).** Track 0Eh starts. There are 100 frames. Each draws `+0Bh` at (0, 0), then the lines of `+0Ch` at column 9, then `+0Ah` at (32h, 0). The lines show only from an edge that starts at row 4Bh and rises a row a frame to row 0, so the figure appears from the bottom up. Each line sways by `s = (i + phase) & 0Fh`, where `phase` counts up by 1 each frame, mod 16. Lines below 1Eh are shifted −1 for `s` 0–2, 0 for 3–7, +1 for 8–10 and 0 for 11–15. Lines from 1Eh on are shifted `−3 + d`, where `d` is 0 for `s` 0–2, `s − 2` for 3–7, 6 for 8–10 and `16 − s` for 11–15. Whenever `phase` is 0Fh, the base column swaps between 9 and 8. Frames have no wait while the edge rises, and are cancellable after. Then the quick fade.

**Effect 4 (Dunstan).** `+0Dh` is the backdrop, drawn first in every frame. It shows alone for 0Fh ticks. Then `+0Eh` is at (31h, 14h), and `+0Fh` slides in from the left, its lines at (31h, −(17h − k)) for k = 0…17h. These 18h frames have no wait, and the figure shows only within rows 31h–4Ch and columns 0–18h. Then `+0Fh` stands at (31h, 0) for 32h frames with no wait. After each, on a chance of 1 in 4, sound 11h plays and the figure moves between row 31h and 32h. It slides back out the same way. Then one frame of the backdrop alone, then `+10h` at (2Ch, 0) for 5 ticks. Then 96h cancellable frames with no wait of `+10h` and `+11h` at (5, 0Fh). Through frame 32h, `+12h` at (5Ah, 8) blinks, changing every fifth frame from frame 5.

**Effect 5 (a caster dies).** Every wait is cancellable.

1. `+13h` alone, 0Ah ticks.
2. `+13h` with `+14h`, `+15h`, `+16h` in turn at (8, 10h), 4 ticks each.
3. `+13h` with `+17h` at (3, 0) and sound 12h, 4 ticks.
4. `+79h` alone, 4 ticks.
5. Fourteen frames a tick apart, for `f` from 1Eh to 38h by 2. Each draws `+18h`, `+19h` at (1Eh, 10h), and three copies of `+7Ah` falling: at (f, 0Ah), mirrored at (f − 2, 1Eh), and at (f + 3, 33h). Four single pixels fall with them, at (r + f − 1Eh, c) in color k for (r, c, k) = (19h, 0Fh, 38h), (14h, 25h, A4h), (1Ch, 19h, A4h) and (1Eh, 1Dh, 38h). The second pixel stops showing once `f` reaches 2Ch.
6. `+18h` with `+1Ah`, then with `+1Bh`, at (1Eh, 10h), 0Ah ticks each.
7. `+18h` alone, 0Fh ticks.

**Effect 6 (the gates).** There are 19h frames, 5 ticks apart, with sound 17h on each. For frame `k`, let `a = max(0, k − 4)` and `b = min(k, 14h)`. Each frame draws:

- `+1Ch` at (0, 28h);
- the gate halves: the lines of `+1Dh` at (0, 28h − a), and mirrored at (0, 38h + b), shown only within columns 28h–50h;
- `+1Eh` at (0, 0), and mirrored at (0, 50h);
- `+1Fh + (a mod 3)` at (8, 0), and `+1Fh + (b mod 3)` mirrored at (8, 58h).

Then a cancellable 24h ticks.

**Effect 7 (SteelSmiter).** Sound 13h plays. Then come four runs of frames: stage 0 for `t` from 14h to 1Dh, then stage 1, stage 0 and stage 1 again, each for `t` from 0 to 1Dh. That makes 100 frames, 5 ticks apart and not cancellable. A pose counts 1, 2, 0, 1, … from the first frame. Each frame draws:

- `+25h` at (0, 30h), `+28h + stage` at (5, 44h), and `+22h + pose` at (0, 0);
- in stage 0, `+26h` at (23h, 27h) and the lines of `+2Ah + t / 10` at (1Fh, 0Eh), each line shifted −1, 0, +1, 0 for `(i + t) mod 4` = 0, 1, 2, 3;
- in stage 1, `+27h` at (15h, 30h) and `+2Dh + q` at (4, 33h), with `q` = 0 below `t` 10, 1 for 10–14 and 2 from 15. For `t` 14h–16h, `+30h`, `+31h`, `+32h` are drawn at (−1, 3Eh).

At the end the music track playing stops.

**Effect 8 (the gem socket).** Track 0Fh starts. A flash draws `+37h` for one tick at (3, 2), or mirrored at (3, 3Eh), chosen at random, and then takes it away.

1. There are 13h frames, 5 ticks apart. Each draws `+33h` at (0, −4) and mirrored at (0, 24h), `+34h` at (4, 0Ch), a flash, and `+35h` at (3Ch, 8). The `+35h` changes over the frames:
   - frames 0–6 show only its pixels of colors E8h–EBh, drawn in color 0;
   - frames 7–0Eh show it whole;
   - frames 0Fh–12h show only its colors E8h–EAh, then E8h–E9h, then E8h, then nothing, with `+36h` drawn at (3Ch, 8) on frame 12h.
2. For 36h ticks, a flash comes on about one tick in nine.
3. There are 31h frames with no wait. Each draws the two `+33h`, `+36h` at (3Ch, 8), a flash on odd frames, and the wall `+34h` parting: its line `i` at (4 + i, 0Ch + k) for odd `i` and (4 + i, 0Ch − k) for even `i` on frame `k`, shown only within columns 0Ch–3Bh.
4. Then 6Ch ticks of flashes and the quick fade.

None of it is cancellable.

**Effect 9 (Mending).** The background is the lines of `+38h` drawn six times, at columns 0, 1Dh, 3Ah, 57h, 74h and 91h.

1. There are 61h frames with no wait, for `s` from 60h down to 0. Each draws the halves sliding together: the lines of `+39h` at (8, −s), `+44h` at (0Bh, 57h − s), `+3Ah` at (5, 5Ch + s) and `+45h` at (9, 5Ch + s). It also draws the lines of `+43h` at (2, 4Eh + s), mirrored anew on every third frame.
2. Sound 0Eh plays. Then 5Ah frames with no wait. Each draws `+39h` at (8, 0) and `+3Ah` at (5, 5Ch). It draws `+43h` at (2, 4Eh), mirrored anew every third frame, except on the last frame. Through frame 1Dh it also draws `+44h` at (0Bh, 57h) and `+45h` at (9, 5Ch); after that it draws `+3Bh`, `+3Ch`, `+3Dh`, `+3Eh` (0Fh frames each) at (0Ah, 55h).
3. Then 12h ticks.
4. Then 1Eh frames, 5 ticks apart. Each draws `+39h`, `+3Ah` and `+3Eh` at (0Ah, 55h). Over them comes one of `+40h`–`+42h` at random at (0, 0), with sound 0Dh whenever the choice changes; on the last frame, `+3Fh` at (5, 0) instead.
5. Sound 10h plays, then a cancellable 6Ch ticks.

**Effect 10 (Rukn).** `+5Ah` and `+5Bh` (at (18h, 0)) are the backdrop. For the first 36h ticks the box is not yet drawn. Then track 0Fh starts, and 50h frames show a figure growing in four stages. A stage lasts 0Fh frames and the last lasts to the end:

1. `+5Fh` at (8, 0Dh), with phase A;
2. `+60h` at (2, 0Bh), with phase A;
3. `+61h` at (6, 11h) with phase A, and `+62h` at (6, 1Bh) with phase B;
4. `+5Dh` at (6, 12h) with phase A, `+5Eh` at (6, 25h) with phase B, and `+5Ch` at (2, 35h) with phase C.

Each is drawn line by line. Line `i` is shifted right by 0, 0, 1, 2, 3, 3, 2, 1 for `(i + phase) & 7` = 0…7. For `+5Ch` in stage 4, lines 0–16h shift by 1. Lines 17h–27h shift by the entry for `e = ((i + phase) & 7) / 2 + 2`, or `7 − e` when that is 4 or more. Phases A and C count up by 1 a frame and B counts down, all mod 8. `+5Bh` is drawn again over the figure. Frames have no wait until stage 4, and are cancellable then. Then the quick fade.

**Effect 11 (Honorah).** Track 0Fh starts. `+46h` is the backdrop, drawn first in every frame.

1. The backdrop alone, 12h ticks.
2. The lines of `+47h` slide in at row 13h, column 48h down to 28h, one frame each with no wait, shown within the box.
3. `+47h`, `+48h`, `+49h`, `+48h`, `+47h` at (13h, 28h), 12h ticks each.
4. `+47h` slides back out to column 48h.
5. `+4Ah` at (1Eh, 26h) three times, 12h ticks each, the second time with `+4Bh` at (28h, 20h).
6. Six blinks. Each shows the backdrop alone for a cancellable 3 ticks, which ends the blinking, then `+4Ch` at (11h, 1Bh) for 1, 2, 4, 8, 10h and 20h ticks in turn.
7. `+7Bh` at (2, 4), a cancellable 48h ticks.
8. The quick fade.

**Effect 12 (the death blow).** Track 10h starts. `+4Dh` is the backdrop. Every frame has no wait and is cancellable.

Each frame draws three falling columns: the lines of `+7Ch` at (r, c), for (r, c) starting at (31h, 28h), (30h, 2Eh) and (1Eh, 45h). Each `r` grows by 4 a frame and goes back to its start on reaching start + 23h, 1Bh and 3Bh respectively.

1. For 36h ticks: the backdrop, the columns, and `+4Eh` at (4, 0).
2. Five stages of 9 ticks. Stages 2–4 add `+4Fh`, `+50h`, `+51h` at (4, 0). Each stage also shows `+58h` at (23h, 24h) in stage 0 and `+59h` after.
3. Six pieces fall out of the box. Each piece is drawn line by line, from its start to the bottom of the box, and moves down 5 rows a frame once it starts falling. The pieces, with (row, column, art), are:

   | Piece | Start | Art | Starts falling |
   | --- | --- | --- | --- |
   | 0 | (5, 44h) | `+52h` | 36h + random 0..24 ticks from now |
   | 1 | (4, 30h) | `+53h` | when piece 0 is gone |
   | 2 | (16h, 50h) | `+54h` | when piece 1 is gone |
   | 3 | (15h, 4) | `+56h` | when piece 2, or piece 4, is gone |
   | 4 | (23h, 0Dh) | `+55h` | 36h + random 0..24 ticks from now |
   | 5 | (27h, 0) | `+57h` | when piece 3 is gone |

   `+59h` stays at (23h, 24h) until every piece is gone.
4. Then `+59h`, `+58h` at (23h, 24h), and a blank, for 12h ticks each, and 12h more ticks.

**Effect 13 (Kern).** Every frame draws `+63h` at (0, 0) and waits 5 ticks, not cancellable. In stage 0, frames alternate `+64h` and `+65h` at (1, 0), with sound 18h on each `+65h`. After a `+65h` on frame 19h, 1Ah or past 32h, stage 1 runs for four frames: `+66h` at (0Ch, 0), with `+67h` at (0Ah, 18h) on its second to fourth frames. Then stage 0 again. The effect ends after the stage 1 that comes after frame 32h.

**Effect 14 (Murchad).** Track 0Fh starts. Every frame draws `+68h` at (0, 0).

1. `+69h` rises from row 17h to row 0Fh at column 18h, one frame a row with no wait, with `+6Ch` at (23h, 18h).
2. Stage 0: `+6Ah` and `+6Bh` alternate at (21h, 18h), each frame waiting 4–6 ticks at random. On each frame, a chance of 1 in 10 starts stage 1 with `b` = 19h.
3. Stage 1: `+69h` at (b, 18h) and `+6Ch` at (23h, 18h), with no wait. `b` falls by 1 a frame until below 14h. It then alternately falls by 2 and rises by 1. On reaching 0Eh, stage 0 resumes.
4. A change of stage after frame 4Bh ends the frames.
5. Then a cancellable 12h ticks and the quick fade.

**Effect 15 (Padrake).** `+6Dh` is the backdrop, drawn first in every frame. Every wait is cancellable.

1. The backdrop alone, 36h ticks.
2. Three rounds of 12h frames, drawn within the box. For frame `k`:
   - k 0–2: `+6Eh` at (28h, 24h), 3 ticks;
   - k 3–8: `+6Eh`, and `+6Fh` dropping from (33h, 25h) by 3 rows a frame, with no wait;
   - k 9–0Dh: frames 0…min(k − 9, 3) of `+75h` at (3Fh, 0Fh), the lines of `+70h + (k − 9)` at (3Ah, 1Ah), and from k 0Bh `+76h`, `+77h`, `+78h` at (16h, 0Bh), 1 tick each. Sound 14h plays on frame 9;
   - k 0Eh–11h: the same going back. `+78h`, `+77h`, `+76h` and then none at (16h, 0Bh), frames 0…11h − k of `+75h`, and the lines of `+73h` down to `+70h` mirrored, 1 tick each.
3. Each round ends on the backdrop alone for 36h ticks.

**Engine model.** The importer plays each effect once by the rules above and writes the result as a flipbook. A flipbook is an indexed sprite of the box's size, one frame per distinct picture, with a list of steps. Each step names a frame, how long it shows, whether its wait is cancellable, and the sounds and track starts or fades that come with it. Random choices are made once at import with a fixed seed. Frames with no wait last 1/60 s. Because the frames stay in palette indexes, the palette's animation and the player's figure colors apply as in the view. The simulation names an effect by its flipbook in an `effect` event, with the creature it is placed by, or the creature's cell when the creature is gone (a caster that leaves no corpse). The presentation pauses the world, draws the flipbook in a framed box placed as above, kept inside the visible view, and plays it. When it ends or is cut short, the world goes on. What the game does after an effect waits for it: the simulation does it at the start of its next update, which the presentation holds back while the flipbook plays. That covers a level change (pits, exit markers), the gem socket's wall opening, and the rest of a spell's effects (Mending's conversation). Effect 6's level change already waits for the conversation to end. Effects 16 and up have no flipbook and play nothing.

## Sparkles

Healing, cures and the sealed doors show a sparkle: small stars that grow and fade over a 48 × 48 area of the view while the game waits. It has no box or border. A sparkle has a sprite bank and a color: the bank's frames 0–4 are a star from smallest to largest, and its colors 60h–6Fh draw as `color + (c − 60h)`, or as themselves for color 60h.

| What | Bank | Color | Area's top-left |
| --- | --- | --- | --- |
| Spell 31 Cure Poison; power 15h (cure poison and confusion) | RESOURCE 5Eh | 30h | the player's point on screen − (12, 10) |
| Spell 33 Heal; powers 1Dh (heal 6d6) and 45h (heal 3000) | RESOURCE 5Eh | 50h | the same |
| Spell 32 Restore | RESOURCE 60h | 60h | the same |
| A sealed door opening, stage 5 (rules.md) | RESOURCE 5Fh | 60h | the door's point on screen − (12, 20), only when on screen |
| Stage 6 | RESOURCE 5Eh | 80h | the same |
| Stage 7 | RESOURCE 5Eh | 30h | the same |

Positions are (row, column). The other healing and curing powers (24h, 28h, 44h, 46h) show nothing. The area is clipped to the view.

**The stars.** There are 96 stars, each with a phase and a step, both 0–15. Star `j` (from 0) starts at phase `(5 + 5j) mod 16`. Its step is `h_j`, where `h_0 = 9` and `h_(j+1) = (h_j + 7) mod 16`, taking 7 whenever that comes to 0. The stars are laid out in order from (0, 7) within the area. Each star's column is the last one's plus the last one's step. A column at 32 or beyond wraps back by 32, two rows lower. Stars from row 40 on are not drawn.

**The animation.** There are 25 frames, 2 ticks apart. Frame `k` (from 0) draws over the view as it was:

- each star at phase 8–15 whose phase is below `7 + k`, with frame 1, 2, 3, 4, 3, 2, 1, 0 for phases 8–15. The stars are drawn in passes, first phases 8 and 15, then 9 and 14, 10 and 13, and 11 and 12;
- then every star's phase goes up by 1, mod 16. In the last ten frames only stars already at phase 7 or above go up, so no new star starts and the rest finish.

**Engine model.** The importer renders each row of the table into a flipbook as for the view effects, its frames transparent where no star is drawn. The presentation plays it over the view at the area's place with the world paused. A `sparkle` event names the flipbook and the thing (the player) or the feature and cell (a sealed door) it is placed by. The engine keeps a thing's ground point rather than the game's entity position, so it places by the top-left of the thing's sprite box, its ground point less its sprite's anchor (entities.md), which is the game's "point on screen" (world.md, Projection), whatever the thing holds; view effects placed by a creature the same way. Nothing waits for a sparkle in the simulation: the healing, curing and door opening are done when it starts.

## Level transitions

When the player changes level, the new view is drawn hidden and then shown by one of:

- **Scroll**: the new view scrolls in from the side, 16 columns a step. Used for exits in walls, a conversation's travel and return, and the conversation after effect 6.
- **Rise**: the new view comes up from the bottom while the old one squeezes away above it, in five steps with no wait. At each step the new view, with rows dropped evenly to keep 1/6, 1/3, 1/2, 2/3 and then 5/6 of them, fills the view from row 107, 86, 64, 43 and 21 down. The old view, squeezed the other way, fills the rows above. Used for pits and open trapdoors, after effect 0 or 2.
- **Descend**: as rise, the other way up. The new view comes down from the top and the old one squeezes toward the bottom. Used for exit markers, after effect 1.

Within a level:

- **Teleport** (an armed teleporter): sound 16h. Circles grow around the player's point on screen, radius 0 to 21, a tick apart, in colors `83h + ((22 − r) / 2 & 0Fh)`. The view slides in, slides out flipped top to bottom, holds 9 ticks, slides in again and slides out. At the destination the circles shrink from 22 to 1 around the player, a tick apart.
- **Ripple** (ripple teleport, spells.md): the player is placed first. The view's rows shift sideways in a zigzag whose size grows over 23 frames and shrinks back over 22.

**Engine model.** Each change of level or teleport names its cue, by what moved the player: `teleporter`, `ripple` (a rune mark, by spell or power), `exit_wall`, `exit_floor` (an exit marker), `fall` (a pit or an open trapdoor) and `travel` (a conversation's travel or return, a portal); and `load`, the engine's own, for a saved game loaded during play (quick load or the disk menu), from the view as it was to the loaded world. The pack gives each cue a style (pack.md, Effects); a cue it leaves out shows the new view at once. The presentation holds the world while a transition plays, and draws it over the whole view, from the view as last drawn to the new one, scaled to its own view's size; what it draws over the view (glows, links, damage) waits for it to end. The styles:

- `iris`: a circle about the player's middle closes on the old view to the color, holds closed for `hold_ms`, and opens about the player's middle on the new view. It stands in for the game's circles and slides.
- `slide`: the new view pushes the old one out, coming in from a side (`from`, right by default), unstretched. It is scroll.
- `squeeze`: the new view comes in from a side (bottom by default) squeezed into the part it has taken, and the old one squeezes into the rest. Rise is from the bottom and descend from the top, in 5 steps: with `steps` n the transition moves in n even steps, the part taken going 1/(n+1), 2/(n+1), … n/(n+1).
- `ripple`: the view's rows shift sideways by a triangle wave, zigzagging over `rows` rows each way, whose size grows to `size` view pixels at the middle and shrinks back as a half sine; the old view shows until the middle, and the new view from it.
- `fade`: the old view fades to the color, and the color to the new view, each over half the time.
- `curtains`: the color closes in from the left and right over the old view until the two sides meet in the middle, holds closed for `hold_ms`, and parts again from the middle on the new view. Closing and parting each take half of what is left, easing in and out.
- `cut`: the new view at once.

The importer gives the teleporter an iris of 700 ms holding 150 ms in black; the ripple 750 ms, size 8, rows 16; exits in walls and travel a slide from the right of 350 ms; falls a squeeze from the bottom, and exit markers one from the top, of 300 ms in 5 steps; and loading curtains of 700 ms holding 150 ms in black, as the teleporter's iris.

## The game screen

The view sits in a frame of panels: the character's portrait and gauges, the hands, the spell panel, the message line. Panels are drawn over a stone texture made from the INIT texture shades. The engine draws its own panels (design.md); the original layout is reference only.

The message line shows the game's messages: what the player takes, refusals, combat, spells. A new message replaces the last. In memorizing past the limit the game shows messages 68h and 69h together, as two lines of one sentence. The pause key stops the game with message 79h ("Pausing, press a key ...") on the line until the next key or click.

**Status icons.** The right end of the message line holds a row of icons, frames of RESOURCE 52h (16 × 12), one for each of these conditions that holds, laid from the right in this order, 13 pixels apart, the first with its left edge at column 299 and row 1:

| Frame | Condition |
| --- | --- |
| 0 | Confused (confusion over 25, player.md) |
| 1 | Lightning shield (its timed effect running) |
| 2 | Battle rage (its timed effect running) |
| 3 | Poisoned |
| 4 | Fire shield (its timed effect running) |

The game draws them only when it redraws the line, at each message and when confusion ends.

**Engine model.** The pack's inventory record lists the status icons in order, each a status and an image (pack.md, Inventory). The presentation draws those whose status holds, every frame, in a row from the screen's top-right corner leftward within the margin, at the hotbar's scale, one pixel of that scale between them; below the debug panel while it shows. The pointer resting on one names its status beside the pointer: the icon's `name` text, or the engine's own name of the status. The statuses: `confused`, `poisoned`, `fire_shield`, `lightning_shield`, `battle_rage`, `haste`, `invisible`, `sight` (the timed ones while their effect runs), `regenerating`, `mana_regenerating`, `levitating`, `death_protection`. The importer gives the five above, in the game's order, from RESOURCE 52h.

**Engine model.** The presentation shows each message on one line centered just above the hotbar (and above a panel open over it) for three seconds; messages of the same update are joined on the line. A mouth's message is shown beside the mouth instead (dialogue.md). The presentation's own notices (saved, loaded, the sound setting) use the same line.

## The frontend

The game starts at a main menu: create a character, restore a game, or select a premade character.

**Engine model.** On an interactive start with the main menu enabled, the engine first shows its own splash artwork for seven seconds, in place of the game's opening picture, and starts the opening picture's track 3 (audio.md), which plays on into the menu. The artwork fades in from black over one second, as the opening picture's colors do. Then its title fills in from the left over a second and a half, each column showing as the fill's soft edge (three tenths of the title's width) passes it, much as the opening picture's lettering colors slide in one at a time; then the prompt to go on fades in over half a second. Any key or mouse click dismisses it to the menu. Returning to the menu after play does not show the splash again. The main menu, restoring and the screens of creating a character show over the splash artwork darkened, the main menu with its lettering; leaving the splash, the artwork darkens over half a second. The main menu's buttons start three eighths of the way down the screen, under the lettering.

**Creating a character** goes through:

1. **The descendant**: one of F's portrait rows (formats.md, F), shown as its portrait; the arrows step through them and R picks one at random. The row gives the character's sex, other-sex portrait and talk portrait.
2. **The name**, at most 25 characters. An empty name takes a stock name by sex.
3. **The attributes**, by one of three methods:
   - **Rolled**: each attribute 3d4 + 2d5, at most 20; rolled again until kept.
   - **Spent**: every attribute starts at 0, with 28d5 points to spend.
   - **Rolled and spent**: each attribute 2d5, with 14d5 points to spend.

   Spent points raise an attribute up to 20, and can be taken back down to its starting value; the character is kept only when every point is spent. Hit points `STR + END + 20` and magic points `TAL + POW + 20` show as the attributes change. **Engine model.** Each attribute has − and + buttons, and Left and Right move the chosen one by a point. With Shift held, a click on − or +, or Left or Right, moves it by 10 at once, or as far as its starting value, the maximum and the points left allow. R rolls the attributes, and any points to spend, again; Shift and R rolls again and again, 100 rolls an update, until the total (the attributes and the points still to spend) is 100 or more and can all be spent (at most 6 × the maximum), or once when the method's dice and maximum cannot reach that. A count of the rolls shows under the attributes while they roll and for two seconds after, then fades over one. Leaving the screen stops the rolling.
4. **The training**: a weapon class (0 sword, 1 axe, 2 spear, 3 bow) and a magic school.

**A premade character** comes from F's premade records: its name, portrait, attributes, school and weapon class.

## The story, in short

The game plays a new character's story from the moment Create Character is chosen (scenes-plan.md lists its seven parts): track 1 starts then, looping, and plays through the story and the creation screens within it, fading out when the story ends. A premade character sees the story too. Until the story's parts are specified, the engine plays it in short, in words of its own, ending with the two pictures of Rowena's farewell.

**Engine model.** Choosing Create Character starts the creation record's story track (the importer gives track 1), which plays on through the creation screens; going back to the main menu fades it out. Choosing a premade character starts it too. When the character is made (or a premade one chosen), the creation record's story scene plays, then the new game's door. The story track fades out with the slow fade as the story scene ends. Escape skips the rest of the story scene; any other key or a click ends a page of words (below) early, and skips the scene from any other step.

The story scene is drawn on the frontend's screen of 320 × 400, whose pixels are half as tall as wide (the presentation shows it in the same place as a 320 × 200 scene, each row half as tall). The game letters its pages in gold; the engine's own lettering does the same. In order:

1. **The summary**, three pages on black, each centered in the box (16, 60)–(304, 340), held its reading time (2 seconds and 45 ms a letter) or until a key or click:

   | Page | Words |
   | --- | --- |
   | 1 | War has spread across the land, and the black mage Shadow Weaver grows stronger with every season. In a fortress hidden on a lonely hill, a Council of mages led by the archmage Rowena searched for one of the bloodline of the hero who long ago overcame the tower of DarkSpyre. Their search found you. |
   | 2 | Brought before the Council, you swore to stand against Shadow Weaver. Through the months that followed, the weapon master Khail trained your arm, and the mage Jairus taught you to bend magic to your will. |
   | 3 | On the night your training ended, a gateway opened within the fortress and Shadow Weaver's horde poured through. Khail and his warriors fell holding the hall. Jairus held the enemy back with fire and lightning until an arrow found him. |

2. **Rowena's farewell**, the first picture: frontend resources (INIT's frontend base +) 16h at (132, 26), 17h at (132, 226), 18h at (164, 68), 1Ah at (164, 198), and the weapon class's picture 26h + class at (204, 266), each a strip drawn with its top-left at the point (x, y) given, over black, in palette 0. Over it, in the column (0, 30)–(132, 370) left of the picture, its lines from the column's top, each centered across it, two pages: "Rowena turns to you, and for the first time you see fear in her face." Then, by weapon class: "She presses into your hands **a falchion** (**a war axe**, **a spear**, **a bow and a quiver of arrows**), an apple, a worn old parchment and a scroll of your school's magic."

3. **The conversation**, over the same picture darkened: two framed panels across the screen, Rowena's above with her portrait (F's Rowena portrait, its header's third word, a people bank) at its left, and the character's below with its portrait at its right, mirrored when the portrait row says so. Each line shows in its speaker's panel, and clears the other's; `@` is the character's name:

   | Speaker | Line |
   | --- | --- |
   | Rowena | @, the gateway will not stay open for long. Take these, and listen. |
   | Rowena | We are lost here. The Council and I will flee and find shelter where we can, but you must go on alone. |
   | The character | Then I will go, and finish what my ancestor began. |
   | Rowena | I will send you to the entrance of the labyrinth. A man named Malachi waits for you there, and you can trust him. Be sure to ask him about the coins: they will help you on your way. |
   | Rowena | The way to destroy Shadow Weaver lies somewhere in the labyrinth. Find it. Now, be ready... |

4. **The second picture**: as the first, with resource 19h (Rowena raising her hands) at (164, 54) in place of 18h; 17h is the lower part of her robe, and 1Ah (the hands holding the gifts) and the weapon go in front as before. Over it, in the column: "Rowena lifts her hands, the air around you begins to shimmer, and the fortress falls away." After the page's reading time the hands and gifts are gone: 16h, 17h and 19h alone, the page still showing, for 2 seconds. Then the picture fades to black over a second.

## New game

The new character starts as player.md (Starting a new game) says. A new game shows the door: the closed door (RESOURCE 3Eh), the door pictures drawn over it at column 135, each held 5 ticks (RESOURCE 55h frames 0–3 at row 0, then 56h frames 0–9 at row 0 with frame `n + 10` below it at row 100), and the door opening (RESOURCE 54h's halves moving apart over the dungeon behind, RESOURCE 3, in 20 steps, each playing sound effect 17h, audio.md; the first six steps, spread `s` = 0..5, wait `11 − 2s` ticks more than the rest), which then holds open for 18 ticks. Then the player arrives on the starting level. No conversation starts: conversation 7Dh, which variable 34 starts on arrival (dialogue.md), belongs to play going on after ending G.

The door pictures 3Eh and 54h are 320 × 200 pictures stored as 160 tiles of 8 × 50 pixels, four rows of forty, row by row. Drawn with a spread `s`, the columns within `s` of either edge are left out and the rest move `s` columns outward from the middle, so the halves part.

**Engine model.** The pack's creation record (`creation.json`) gives what character creation offers: the portraits to choose from, the attribute methods, the weapon classes and schools with their names and what each adds to the starting inventory, the objects every character starts with, the body armor picked among at random, the premade characters, and what a new game does (the counter it sets, the conversation that opens it, the scene it shows). The engine's frontend runs these rules with its own menus; the game's own captions and stock names are in its program, not its data files, so a pack gives its own or does without.

## Death

When the player's hit points reach 0 at a status tick, or a death blow lands, play ends and the game returns to the main menu. The engine shows the player dead for a moment, then the main menu.

## Endings

Power 5Eh (items.md) chooses an ending by the first of these the player holds in either hand:

- the Chesschantra Skull (F7h): ending F, NPC record 4;
- WarMonger (64h): ending G, NPC record 5;
- the Necklace of Runes (59h): ending E, NPC record 3.

The record's number is stored in interaction variable 32, conversation 79h runs, and when it ends play ends and the frontend plays the chosen ending's end scene. Ending G first takes the objects from both hands, runs level procedure 27h, drops the Shadow Weaver's Cloak, Mask and the Black Boots at (15h, 1Fh), awards 600000 experience and saves the game so it can go on after the ending.

Conversation action 21h plays the closing, below, and then the game ends; nothing after it in the conversation runs. The closing shows the closing picture spread in (the door closing: RESOURCE 54h's halves coming together over the view, then 3Eh), a fade to white, the final scene (RESOURCE 5Ah, then frames 0–4 of 5Bh), and the closing texts (TEXT records 157h–170h), each shown for 2Dh ticks on the banner RESOURCE 5Ch. Then the frontend plays the chosen end scene (End scenes).

**Endings, engine model.** The creation record's `ending` gives the ending power's choices (the item held, the counter's value, a name and an optional scene), the counter, the conversation that runs before play ends, and the closing scene of the `ending` action. When play ends the simulation stops and reports the ending's name and scene; the presentation plays the scene and returns to the main menu. Ending G's changes to the world and its save are not done.

**Scenes, engine model.** A scene is a record of steps the presentation plays in order on a picture of 320 × 200, or of 320 × 400 shown with half-height rows. A step can show a picture (mirrored or not), part or close a picture's halves over a background in a number of steps, or show a caption. It can fade to a color, or in from one, or fade some palette colors only. It can wait, show a page of words in a box, show a line in a speaker's panel, or play the step chosen by the character's weapon class. It can draw a frame of pictures over the scene's pictures in place of the last, switch the palette the scene draws in (its effects running from then), or start, fade or stop a track. Escape skips the scene; another key or a click ends a page of words or a line early, and ends a run of steps marked cancellable. In any other step it skips the scene, unless only Escape skips it. The importer writes the new-game door, the closing, the story in short and the end scenes as scenes. The intro, whose captions are in the game's program, is left out.

## End scenes

The frontend plays an end scene after the closing: ending E plays **the god of Magic**, ending F **Chesschantra's return**, and ending G **Borel's farewell**. Each fades the music out as it ends, and the game then returns to the main menu (or, after ending G, goes on with the saved game).

**The screen.** End scenes draw on the frontend's screen of 320 × 400, whose pixels are half as tall as wide. Positions are (row, column) from its top-left. `+n` is RESOURCE `end-scene base + n` (INIT). A picture drawn *whole* is all its frames stacked, each its height below the last, with its top-left at the position. A frame of `a × b` pieces at (r, c) is drawn from the pieces of `+12h`, each 16 rows by 8 columns: piece (i, j), for i = 0 … a − 1 down and j = 0 … b − 1 across, goes at (r + 16i, c + 8j). The corners are piece 0 and the pieces next to them piece 1. On the left and right sides, the pieces past the first third and before the last third are piece 2, and the rest a random piece 4–12. On the top and bottom, they are piece 3, and the rest a random piece 13–21. The inside is left as it was.

**Colors.** The scenes draw in COLORS palette 7 (formats.md), coming in from black at the start. The captions' lettering is the font's colors 50h–5Fh, moved up by 70h to 50h + 70h = C0h–CFh, the palette's gold. While the scene *cycles*, every 6 ticks colors E0h–EFh rotate as one group of 16 and F0h–F7h as one of 8, each color taking the value of the next as in the view (Palette animation). Whatever is drawn in them shimmers: red for E0h–EFh, gold for F0h–F7h. A *fade* takes eight steps with no wait. A fade from a color brings the palette in from it, and a fade to a color takes it there.

**Captions.** A caption is centered in the box of rows 340–389 across the whole width, below the frame, in place of the caption before. A caption *held* stays its reading time before the scene goes on: 3 seconds for each whole 320 pixels of its lettering's width, 1 more for each whole 150 pixels of the rest, and 3 more.

**The opening.** All three scenes open the same way:

1. The palette goes black. A frame of 15 × 40 pieces is drawn at (80, 0) and the scene's picture `+0` whole at (100, 8).
2. The framed picture 0 is drawn. The *framed picture* `k` is `+1` drawn whole at (151, 212) and `+2 + k` at (126, 227), in place of the framed picture before.
3. The track starts with its loop: 6 (16 bars) for the god of Magic and Chesschantra's return, 5 (24 bars) for Borel's farewell. The first two fade out whatever was playing first.
4. The colors fade in from black, and the scene starts cycling. Nothing more happens for 90 ticks.
5. The framed picture 1 is drawn.

**The glow.** `+0Ch` drawn whole, with only its colors B0h–EFh. Each color `c` draws as `E0h + (c − B0h) mod 16`, so it takes the shimmering reds. At *strength* `m` only the first `m` of those colors draw (B0h … B0h + m − 1). The glow *comes up* in 16 frames with no wait, strength 4, 8, … 64, and *dies down* in 17, strength 64, 60, … 0.

**Calling up a figure.** A figure is called up from a box at (r, c) with a body, perhaps an extra drawn over it, a place for both at (row, col), its footing at a place of its own, and a caption:

1. The glow comes up at (r + 8, c + 20).
2. The caption shows.
3. The figure grows, in ten frames with no wait, `n` = 1 … 10. The body and the extra are drawn scaled by `n`/10 (Scaling) at (row + R/2 − h/2, col + W/2 − w/2). Here R × W is the body's full size, h × w its size scaled, and every halving rounds down. The last frame also draws the footing.
4. The glow dies down, with the figure, the extra and the footing at full size over it.
5. The caption clears.

**Scaling.** A picture scaled by `n`/`d` keeps, along each axis, source pixel `i` drawn `⌊(d/2 + (i + 1)n)/d⌋ − ⌊(d/2 + in)/d⌋` times, so its size is `⌊(s·n + d/2)/d⌋` for a side of `s`. Divisions here round down.

**Sparkles over a figure.** Stars twinkle over the 180 × 96 area at (r, c). The area has a list of regions, (row, column) offsets and sizes from the area's top-left. There are six slots. A star shows the frames 0, 1, 2, 3, 4, 3, 2, 1, 0 of `+16h`, one a frame, with that frame's top-left 4 rows above and 8 columns left of the star's point. In each frame with no wait, slot by slot:

- If no pair is waiting, and this slot and the next are free (slots 0–4), a pair is waiting from now on. On a chance of 20 in 100, the two slots light together, at (r + pr, c + pc₀) and (r + pr, c + pc₁).
- A lit slot draws its star and moves to the next frame. A free one lights on a chance of 25 in 100 at a random point of a random region: an offset of 0 up to the region's size less 1 along each axis.
- A star past its last frame frees its slot. When that slot is the first of the waiting pair, no pair is waiting any longer.

After 180 ticks, or at a key or click, no slot lights any more. From then on a count, starting at 0, goes up by 1 each frame and down by 1 for each slot still lit. The sparkles end when it reaches 2.

| Area | Regions: offset (size) | Pair (pr; pc₀, pc₁) |
| --- | --- | --- |
| The god of Magic's | (4, 16) (16 × 16), (44, 4) (34 × 18), (48, 67) (30 × 20), (28, 56) (22 × 2), (48, 38) (6 × 16), (4, 52) (22 × 24) | 12; 42, 46 |
| Chesschantra's | (20, 50) (24 × 12), (5, 60) (15 × 19), (20, 80) (30 × 13), (68, 53) (42 × 12), (58, 80) (50 × 5) | 34; 70, 75 |

**The blast.** `+7` drawn whole grows toward the player in `d` frames with no wait, `d` = 20 for one copy and 10 for two. Frame `i` (from 0) scales it by (i + 1)/d (Scaling) to h × w, w taken up to a multiple of 8. It draws it at row max(100, 152 − h/2), column max(8, 59 + 5i − w/2). A second copy goes at column max(8, 139 + 5i − w/2). The earlier frames' copies stay, so the blast leaves a trail. A caption shows as it starts and clears at the end. Then the scene stops cycling, the colors fade to (255, 255, 0), and the screen is cleared.

**The god of Magic** (ending E):

1. The opening.
2. The god of Magic is called up from the box at (100, 8): body `+4` and extra `+5` at (108, 41), with footing `+0Dh` at (282, 50). Caption: the god of Magic appears.
3. Sparkles over the god of Magic's area at (108, 41).
4. The extra gives way to `+6` at (108, 41).
5. The framed picture 0.
6. The blast, one copy. Caption: the player's end.
7. The world's end, the god of Magic and Abighael.

**Chesschantra's return** (ending F):

1. The opening.
2. Chesschantra is called up from the box at (100, 8): body `+8` and extra `+9` at (122, 60), with footing `+0Fh` at (280, 60). Caption: Chesschantra appears.
3. Colors F0h–F7h turn green: each takes its green alone. Caption: the player held, the staff taken.
4. Sparkles over Chesschantra's area at (100, 8). The F0h–F7h ramp is gold again after it.
5. The god of Magic is called up from the box at (100, 88): body `+4` and extra `+5` at (108, 121), with footing `+0Dh` at (282, 130). Caption: mother and daughter call up the god of Magic.
6. Sparkles over the god of Magic's area at (108, 121).
7. The extras give way: `+0Ah` at (122, 60) and `+6` at (108, 121).
8. The framed picture 0.
9. The blast, two copies. Caption: the player's end.
10. The world's end with Chesschantra.

**The world's end.** It is drawn on its own screen:

1. A frame of 21 × 40 pieces at (0, 0), and `+14h` whole at (20, 8).
2. The music stops. The colors, all (255, 255, 0) after the blast, fade to black.
3. `+15h` whole at (220, 8).

The palette is 7 again, and it no longer cycles. Three figures take part:

| Figure | Home (row, col) | Rows × width | Faces | Body (+n, rows down) | Poses 0–2 (+n, rows down) | Held 0–1 | Holds at the end |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Chesschantra | (54, 278) | 157 × 120 | left | 24h, 0 | 26h, 27h, 28h; 0 | none | 0 |
| Abighael | (72, 299) | 149 × 112 | left | 18h, 38 | 21h, 22h, 23h; 2 | 1Fh, 20h | 1 |
| The god of Magic | (53, 18) | 177 × 136 | right | 17h, 34 | 1Bh, 1Ch, 1Dh; 22 | 19h, 1Ah | 0 |

A figure at (row, col) is drawn in a pose and with a held picture, in order: the body, the pose and the held picture. Each is drawn whole, its rows down from `row`. A figure facing right draws each picture from column `col`. A figure facing left draws each picture mirrored, ending at column `col` (from `col − width`). A figure *wanders*: moving toward a target, it steps one pixel along the axis farther from it, or diagonally when the distances are equal, and stops there. Standing, it picks a new target of its home plus 0–14 rows and 0–14 columns, at random.

**A battle** of some of the figures, with a caption and a last caption:

1. The figures are drawn, pose 0, holding picture 0. The caption shows. In the first battle of a world's end, track 7 starts with its loop of 8 bars, and the colors fade in from black.
2. Each frame with no wait, every figure wanders a step and is drawn holding picture 0. With no caster chosen, every figure is in pose 0. With one chosen, the caster is in pose 1 and the others in pose 2, or pose 0 once the battle is ending.
3. At its time the battle chooses a caster at random, and in the next frame the caster casts. The first time comes 18, 36 or 54 ticks from the start, at random, and each later one the same from the end of the cast before. A cast stops the figures where they are. A bolt, the frames of `+25h` taken at random each frame (mirrored when the caster faces right), flies at row `row + 44` of the caster. It starts at column `col − width` for a caster facing left, or `col + width − 24` facing right, and moves 4 columns a frame in the caster's facing. It flies until its column meets a figure facing the other way: `col` to `col + width − 32` of a figure facing right, or, its column taken 24 further, `col − width + 32` to `col` of one facing left. There `+2Dh` and then `+2Eh` show for a frame each, whole at (bolt row − 32, the bolt's column rounded down to a multiple of 8, plus 8 when it flew right). Then the bolt and the burst are gone. After the second cast the caption clears.
4. After 360 ticks, or at a key or click, the battle is ending. The last caption shows, and a figure holding 0 at the end is chosen at random to cast at once. After that cast every figure is drawn once in pose 0 with its own picture held at the end. Then the caption and the figures are gone.

**A flight.** The figure is drawn, pose 0, holding picture 0, and the caption shows. Each frame with no wait, it wanders ten steps toward the target given, while its column is short of the target's. When it is there, or at a key or click, it is gone.

**The world's end, the god of Magic and Abighael** (ending E): a battle of the god of Magic and Abighael. Caption: the battle; last caption: Abighael's strength fails. Then the god of Magic's flight toward (his home row − 16, column 90). Caption: a dark day.

**The world's end with Chesschantra** (ending F): Chesschantra and Abighael both face left, mirrored, and Chesschantra holds nothing. The god of Magic holds 1 at the end, and the other two 0.

1. A battle of the god of Magic, Chesschantra and Abighael. Caption: the battle; last caption: the two powers prevail.
2. Chesschantra wanders toward (72, 18), ten steps a frame, with Abighael standing where the battle left her, and the god of Magic gone. Caption: Chesschantra turns on her daughter. Once her column is below 170, she faces right and her column becomes 100. A key or click stops her where she is.
3. Chesschantra stands at (72, 18) as her home, facing right. She holds picture 0 at the end, and Abighael picture 1.
4. A battle of Chesschantra and Abighael. Caption: the heavens shake; last caption: Abighael's strength fails.
5. Chesschantra's flight toward (56, 90). Caption: a dark day.

**The world's end's close**: `+1Eh` for the god of Magic, or `+29h` for Chesschantra's return, whole at (32, 92) or (32, 110). Caption: the world's end. After 54 ticks colors 00h–BFh fade to black. After 54 more, D0h–DFh. After 54 more, the rest.

**Borel's farewell** (ending G):

1. The opening.
2. Borel is called up from the box at (100, 8): body `+0Bh` at (114, 52), no extra, footing `+0Eh` at (282, 55). Caption: Borel appears.
3. After 54 ticks, three captions, each held: Abighael sees her father; Borel thanks the player; father and daughter will leave this world.
4. The portal clears. The framed picture is gone, leaving the scene's picture at rows 126–319, columns 208–271. For a frame with no wait, Abighael steps out: `+2Ch` at (192, 201), `+10h` at (126, 192), `+13h` at (284, 208) and `+11h` at (268, 220). Then she walks to Borel in four frames 3 ticks apart, frame `i` drawing `+2Bh` (`i` even) or `+2Ch` (`i` odd) at (192, 181 − 20i), `+2Ah` at (126, 172 − 20i) and `+13h` at (284, 188 − 20i). `+11h` stays at (268, 220). She stays as the last frame draws her.
5. Caption, held: the player's duty now.
6. The caption clears. The glow comes up at (108, 52), with Borel and Abighael drawn over it.
7. Caption: the farewell. The two go into the portal in ten frames with no wait, `n` = 9 … 0. The area at rows 100–299, columns 48–175 shows the scene's picture, with the glow at full strength at (108, 52). Over it go `+0Bh`, `+2Ch` and `+2Ah`, scaled by `n`/10, around (208, 110). Write `s(v)` for `⌊(v·n + 5)/10⌋`, with `b`, `a`, `e` the scaled widths `s` of `+0Bh`'s and `+2Ah`'s widths and `s(4)`, and `t = b + a − e`. Then:
   - `+0Bh` goes at (208 − ⌊(R·n + 5)/10⌋/2, 110 − t/2), for its R rows;
   - `+2Ah` at (208 − s(160)/2, that column + a);
   - `+2Ch` at (`+2Ah`'s row + s(66), `+2Ah`'s column + s(9)).

   At `n` = 0 none is drawn.
8. The glow dies down over that area.
9. Two captions, each held: the WarMonger is gone too; the day is won.
10. The scene stops cycling. After 18 ticks the colors fade to black.

**End scenes, engine model.** The importer writes each end scene as a scene of 320 × 400 (Scenes, engine model), and gives the choices of the creation record's `ending` its scene: `scene.end.e`, `scene.end.f` and `scene.end.g`. It plays each scene by the rules above, with random choices made once at a fixed seed, and no key or click. Frames with no wait last 1/60 s, so a fade takes 8/60 s. What moves, it writes as frames of pictures over the scene's picture. The frame, the scene's picture and `+14h`/`+15h` are its pictures. The glow at each strength, and the scaled figures and blasts, are sprites of their own. The figures of the world's end are each one sprite, with a frame per pose and held picture, mirrored to face left. The colors are palettes: `palette.end` is palette 7 cycling, `palette.end.green` the same with F0h–F7h green, and palette 7 unmoving for the world's end. A fade of some colors only fades those palette colors. A key or a click during the sparkles, a battle or a flight ends that part early, where the game's would have ended it sooner. The scene goes on from where the uninterrupted part would have ended. Only Escape skips an end scene. A caption held stays the story in short's reading time (2 seconds and 45 ms a letter). The captions are the engine's own words, written for each slot, as the story in short's are:

| Scene | Slot | Words |
| --- | --- | --- |
| E, F | the god of Magic appears | The portal flares, and the god of Magic steps forth. |
| E, F | the player's end | Fire and fury break over you, and your life is spent. |
| E, F | the battle | Gods and powers clash across the heavens. |
| E, F | Abighael's strength fails | Abighael fights on bravely, but at last her power fails. |
| E, F | a dark day | Darkness falls upon all who live. |
| E, F | the world's end | The end of the world is at hand. |
| F | Chesschantra appears | Out of the portal steps the witch Chesschantra. |
| F | the player held, the staff taken | The Staff is wrenched from your hands, and you cannot move. |
| F | mother and daughter call up the god of Magic | Mother and daughter join their power to call up the god of Magic. |
| F | the two powers prevail | Against the two together, Magic cannot prevail. |
| F | Chesschantra turns on her daughter | Then Chesschantra turns on her own daughter. |
| F | the heavens shake | The sky trembles as the battle rages on. |
| G | Borel appears | Out of the portal steps Borel. |
| G | Abighael sees her father | Abighael weeps to see her father once more. |
| G | Borel thanks the player | Borel turns to you and thanks you for all you have done. |
| G | father and daughter will leave this world | He and his daughter will leave this world, to seek a place where they can live in peace. |
| G | the player's duty now | Bringing peace back to this land, he tells you, is now your task. |
| G | the farewell | You say farewell as Borel and Abighael pass into the portal. |
| G | the WarMonger is gone too | Only when they are gone do you see that the WarMonger has gone with them. |
| G | the day is won | The day is yours... for now. |

## Unknowns

- The frontend's captions and its stock names, which are in the game's program rather than its data files; the save and load screens.
- How fast the game drew an effect's or an end scene's frames with no wait, and its fades; the engine gives them 1/60 s a frame or a fade's step.
- Which track plays in which scene and level.
- Which closing texts belong to which ending.
- The ripple's exact zigzag, and how long each slide of the teleport takes. The engine's styles stand in for them.
