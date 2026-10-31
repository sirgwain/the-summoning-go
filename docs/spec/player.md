# The player

## Attributes

Six attributes, each with a current value and a maximum:

| Index | Attribute | Short |
| --- | --- | --- |
| 0 | Strength | STR |
| 1 | Agility | AGI |
| 2 | Endurance | END |
| 3 | Accuracy | ACC |
| 4 | Talent | TAL |
| 5 | Power | POW |

The character is made in the frontend (scenes.md): name, attributes, sex, portrait, one weapon skill and one magic school, each starting at level 1.

## Derived values

- Hit points: maximum `STR + END + 20`, starting full.
- Magic points: maximum `TAL + POW + 20`, starting full.
- Hit and magic points stay between 0 and their maxima. A heal of more than 1 shows message 4Ah with the amount.
- Armor class: the sum of the worn objects' armor (items.md), starting 0.

## Experience and levels

The first level needs 500 experience; each level's threshold is double the last, except that 512000 becomes 600000. The player goes up as many levels as the experience reaches, to level 11 at most. Each level adds its dice to hit points and magic points, current and maximum, with message A5h and a fanfare:

| Level | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Hit points | 1d10 | 1d10 | 2d10 | 2d10 | 3d10 | 3d10 | 3d10 | 4d10 | 4d10 | 4d10 | 5d10 |
| Magic points | 1d10 | 2d5 | 2d10 | 4d5 | 3d10 | 6d5 | 6d5 | 4d10 | 8d5 | 8d5 | 5d10 |

Titles by level and skill ratings by skill level are text from the game's manual and its TEXT records.

Experience comes from killing creatures (creatures.md), from raising skills, and from story events.

## Skills

Four weapon classes (0–3; class 3 is bows) and four magic schools (0 Wizardry, 1 Sorcery, 2 Enchantment, 3 Healing), each with a level, practice points and a practice threshold that starts at 10.

Practice adds points. When they reach the threshold the level rises by 1 (with a fanfare and message 44h), the threshold is subtracted, the threshold doubles (up to 1400h), and the player gains 57 experience. Weapons practice on hits (combat.md); schools practice on casting and on a spell's hits (spells.md). Lowering a skill shows message 45h.

## Facing and walking

The player faces one of eight directions, numbered clockwise from screen-up: 0 up, 1 up-right, 2 right, 3 down-right, 4 down, 5 down-left, 6 left, 7 up-left. Facing `p` walks along table direction `(p + 7) mod 8` (README.md). The keypad facings come from INIT (formats.md).

A walk step moves the player's speed (8 units) in the facing, or half that when walking straight left or right. Walking shows frame `p`; after 8 idle updates the player shows the standing frame `p + 8`. Actions show frames from 30h (combat.md, spells.md).

**Engine model.** The player is a thing of kind `player` named by the pack (pack.md): its sprite, `speed` (8), `interval_ms` (the base interval, 5 ticks), `footprint`, `weight` (100), hit points and magic points. Its animations are `walk.d` and `idle.d` for table direction `d`; the importer makes `idle.d` the standing frame `p + 8` and `walk.d` the frames `p + 8, p + 16, … p + 40, p`, one per step, for the facing `p` of `d`. The template's `stats` give the six attributes by name (`strength`, `agility`, `endurance`, `accuracy`, `talent`, `power`) and the starting `bonus` (`interval_bonus`, in ticks), and `interval_ms` the base interval. The bonus is the player's own state; powers change it. Each update the player's interval is worked out as Speed says, from the attributes and what it carries (hand, slots and the contents of containers, weightless bags excepted); the ticks added to the base are converted at the tick's length. Haste, battle rage, worn bonuses and confusion come with Fight. Until character creation, the importer starts the player with STR 18, AGI 20, END 18, ACC 16, TAL 16, POW 16, so 56 hit points and 52 magic points, and a bonus of −2 standing in for the Leather Boots it starts wearing (items.md, Inventory) until worn slots come with Fight: an interval of 2 ticks.

**Held objects.** The object in each hand is drawn with the figure. Its look is byte 11's low seven bits (items.md), for categories other than 0–2; 0 draws nothing. Look `L` uses the held-item bank `held base + L − 1` (INIT). For the figure's frame `f` and hand `h` (0 right, 1 left), the held-item table is `t = h`, or `h + 2` when `L` is above 11, and the entry is number `(t xor 1) × 96 + f`: its frame of the bank is drawn at the entry's row and column offsets from the figure's top-left, behind the figure's layers (layer 0) or in front of them (layer 1). Held objects are not drawn in water 3 deep or more.

**Engine model.** An item's `held` gives, for each hand, a sprite to draw behind the player and one in front, each with a frame for every frame of the player's sprite and the same anchor (pack.md). The importer composes them from the table above for each look the hand items use.

**Skills and experience** are the player's state: four weapon classes and four schools, each with a level (starting 1 for the chosen class and school, else 0), practice points and a threshold (10); experience, the player's level, and the threshold for the next (500). Practice and levels work as above; the dice per level are the importer's rules for the Summoning, kept in the combat rule set. Until character creation the importer starts the player at weapon class 0 and school 0, level 1.

**Statuses** are the player's state: the lasting conditions above as flags, the timed effects as counts of status ticks (at most 255), and each attribute's timed boost (its amount and its ticks). The combat rule set reads them. Levitating keeps the player off plates and floor effects and over pits and trapdoors, and stops it pushing corpses; fire-immune stops fire floors; invisible and transformed stop creatures chasing; haste and battle rage each take 1 from the interval. Regenerating and mana regenerating roll each player update. Sight and transforming wait for their spells.

**Exertion and fatigue** are as above: each strike adds `weight / 8` (at least 1), each push 1 three times in eight.

**Starting inventory.** The importer gives the player template its starting objects as the pack's `inventory`, slot by slot (pack.md): for weapon class 0 and school 0, the Apple of Vigor and the Palimpsest on shelves 0 and 1, Leather Trews (the first of templates 14–17) on the body, Leather Boots on the feet, the Falchion in the right hand and the school's scroll on shelf 2. Worn objects start as if put on, so the boots give the −2 bonus and the template needs no `interval_bonus`. Character creation will choose the class, school and body armor.

**Dying.** At 0 hit points the player is dead: it no longer acts, and the world reports it; what follows comes with scenes.

**View.** The view does not follow the player step by step. While the player stands within the middle of the view it stays put; a move that leaves the player outside it, horizontally or vertically, scrolls the view along that axis by as far as the player moved on screen. The engine takes the middle half of the view each way. A player outside the view altogether (arriving on a level, teleported) is centered.

**Figure.** The importer composes the player's sprite from the character animation banks (INIT base `C`), for the figure out of water: for frame `f` of group `g = f / 8` and step `s = f mod 8`, bank `C` frame `f` at `(0, 13)`, bank `C + 1` frame `f` at `(0, 18)`, the body look bank `C + 25h` (unarmored) frame `h[g] + s` at `(0, 5)`, and the head look bank `C + 2` frame `s` at `(8, 0)`, each later one over the earlier, with `h` = 0, 8, 16, 24, 16, 8 for groups 0–5. The figure's top-left is the player's position, so its anchor is `(18, 19)`. Worn looks, held items and colors come with character drawing.

**Confusion.** A count; over 25 the player is confused, and each step has a `count`% chance to veer 45° to either side. Walking into a wall adds 6. The count falls by 1 each move.

When a step is blocked, the player pushes what can be pushed (entities.md), enters a level exit marker or a Gateway, or touches a creature, which hurts the player if it hurts on contact (creatures.md). A boulder rolling across the step hurts 2d3. Moving ends a Shape Change.

## Speed

The player's update interval in ticks:

```
interval = 5 + bonus + load
         + (10 − END) / 2 + 1     if END < 10
         − 1                      under haste
         − 1                      under battle rage
interval = max(1, interval)
```

`bonus` comes from worn objects and powers (items.md).

**Load.** Carried weight `w` (the sum of every carried object's weight, containers' contents included except inside a Bag of Lightness) against the limit `L = 10 × STR`:

- `w < L / 10`: agility +1.
- `w ≥ L`: agility − `p`, with `p = (w − L) / 20 + 1`, and `load` gets `p + 1`.
- Then, from agility after that: `load` gets −1 at 15 or more, or `10 − AGI` below 10.

The load gauge shows `5w / L` of 5 steps.

## Exertion and fatigue

Actions add exertion (at least 1 each). When exertion exceeds `END + 3` (and END is above 1), END drops by 1, fatigue rises by 1, and exertion resets to 2.

On each status tick exertion falls by 1; when it is already 0 and there is fatigue, one point of fatigue returns to END and exertion becomes `END + 3`.

## Status

Lasting conditions:

| Status | Effect |
| --- | --- |
| Poisoned | 1–2 damage, 1 in 40 moves. Set by poisoning wounds (message 59h) |
| Regenerating | 2d2 hit points, 1 in 30 moves |
| Mana regenerating | 2d2 magic points, 1 in 30 moves |
| Resist quarter | Damage − 1/4 |
| Resist forty | Damage − 40% |
| Charges | Non-physical damage is absorbed by charges in the hands' objects |
| Fire-immune | No fire floor or fire-level damage |
| Death-blow ward | Creatures' death blows fail |
| Accurate | +2 to hit and to damage |
| Blow protection | Creatures' to-hit −4 and damage −1/4 |
| Levitating | Pits, plates, trapdoors and floor effects are ignored; cannot push corpses |
| Invisible | Creatures do not chase; their to-hit −4 |
| Transformed | Looks like a creature (Shape Change); creatures do not chase; ends on moving |
| Death protection | Revives once to full hit and magic points |
| Female | Sex, chosen at creation |

**Timed effects** count down once per status tick: sight, fire shield, lightning shield, battle rage, haste, invisibility, weapon skills, magic skills. Adding to one saturates at 255. Each attribute also has a timed boost.

## Status tick

Every 109 ticks:

1. At 0 hit points the player dies (scenes.md).
2. Deep water and fire levels hurt the player (world.md).
3. Every fourth tick, 1 magic point.
4. Timed objects in worn slots run down (items.md).
5. Attribute boosts run down; one ending takes its +2 back off.
6. Each timed effect counts down; on reaching 0:
   - sight ends (the view redraws);
   - the shields end;
   - battle rage ends: STR −2, END −2, AGI +2, and the interval is recomputed;
   - haste ends with message 60h, and the interval is recomputed;
   - invisibility ends;
   - weapon skills: each weapon skill −1, message 64h;
   - magic skills: each school −1, message 67h.
7. Exertion and fatigue recover (above).

## Every move

On each of the player's updates: the hands' delays count down (items.md), spell gestures play (spells.md), and regeneration, mana regeneration and poison roll their chances.

## Starting a new game

The player starts on the level and cell from INIT, with:

- Apple of Vigor (shelf slot 0), Palimpsest (slot 1);
- on the body, a random one of templates 14–17; Leather Boots on the feet;
- in the right hand, the weapon of the chosen class: Falchion (61h), War Axe (6Ah), Spear (67h), Bow (6Eh); with the bow, a Quiver holding 4 Arrows;
- in slot 2, the scroll of the chosen school: DFh, E5h, ECh, F4h;
- the gesture letters of the school's first spell known (spells.md).

Experience 0, next level at 500, interval base 5.

## Unknowns

- The weapon classes' names in the manual (sword, axe, spear, bow) and their mapping to classes 0–2; confirm against the character screen.
