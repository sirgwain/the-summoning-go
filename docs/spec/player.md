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

The character is made in the frontend (scenes.md): name, attributes, sex, portrait, one weapon skill and one magic school, each starting at level 1. Each attribute's maximum starts at its value.

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

Practice adds points. When they reach the threshold the level rises by 1 (with a fanfare, audio.md, and message 44h), the threshold is subtracted, the threshold doubles (up to 1400h), and the player gains 57 experience. Weapons practice on hits (combat.md); schools practice on casting and on a spell's hits (spells.md). Lowering a skill shows message 45h.

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

**View.** The view does not follow the player step by step. While the player stands within the middle of the view it stays put; a move that leaves the player outside it, horizontally or vertically, scrolls the view along that axis by as far as the player moved on screen. The engine takes the middle half of the view each way. While the minimap shows (world.md, The map), the middle's top and left edges move down and right, as far as needed to keep two cells' width (64 view pixels) between them and the minimap's corner, its name included, but never past the view's middle, so the view scrolls before the player walks under the minimap. The hotbar covers the bottom of the screen, so the view here, for its middle and for centering, is the part of the screen above the hotbar's frame. A player outside the view altogether (arriving on a level, teleported) is centered.

**Figure.** Out of water the player's figure is drawn in four layers from the character animation banks (INIT base `C`), each over the last, for frame `f` of group `g = f / 8` and step `s = f mod 8`:

1. the **torso**, bank `C` frame `f` at `(0, 13)`, its colors shifted by the body look (below);
2. the **legs**, bank `C + 1` frame `f` at `(0, 18)`, recolored by the feet look;
3. the **body look** `B`, bank `C + B` frame `h[g] + s` at `(0, 5)`;
4. the **head look** `H`, bank `C + H` frame `s` at `(8, 0)`;

with `h` = 0, 8, 16, 24, 16, 8, 32, 40, 48, 56 for groups 0–9. The strike groups (6 and up) take the torso's and legs' frame `s` rather than `f`. The figure's top-left is the player's position, so its anchor is `(18, 19)`. The figure's colors do not animate (scenes.md, Palette animation), and its colors E0h–E3h are the portrait's (below).

**Worn looks.** What is worn chooses the looks. Each worn object's look is byte 11's low seven bits (items.md) for categories 0–2:

- **Head** (slot 20): the look of the object worn there, or 2 with none; a female player's is one more. The Chain Coif's look is 6, the Wizard's Hat's 18h.
- **Body** (slot 26): the look of the object worn there, or with none 25h (26h for a female player). An object of look 0 leaves the body look as it was. The body look also shifts the torso's colors: its colors 3..12 become colors `k`..`k + 9`, where `k` is the shift below for body looks 18h..2Bh (3 leaves them as they are).

| Body look | 18h | 19h | 1Ah | 1Bh | 1Ch | 1Dh | 1Eh | 1Fh | 20h | 21h | 22h | 23h | 24h | 25h | 26h | 27h | 28h | 29h | 2Ah | 2Bh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Torso shift `k` | 80h | C6h | 06h | 03h | 03h | C6h | 03h | 25h | 80h | 00h | C6h | 60h | 03h | CEh | CEh | 80h | C6h | C6h | 80h | 00h |

- **Feet** (slot 24): the look of the object worn there, or 8 with none, chooses a set of colors for the legs; set 0 leaves them as they are. Set `n` (1..8) draws the legs' color B1h + `i` in color `c[i]` of its row, for `i` = 0..9 except 8:

| Set | c[0..9] |
| --- | --- |
| 1 | 00h 01h 02h 03h 04h 05h 06h 07h 08h 08h |
| 2 | 00h 01h 02h 03h 04h CDh CFh D0h D2h D2h |
| 3 | 01h 02h 03h 04h 05h 06h 07h 90h 92h 92h |
| 4 | 02h 03h 04h 05h 06h 07h 08h 09h 0Ah 0Ah |
| 5 | 00h 01h 02h 03h 04h 05h 06h 07h 37h 37h |
| 6 | E8h EAh 51h 52h 53h 54h 55h 56h 58h 58h |
| 7 | E4h E6h 53h 54h 55h 56h 57h 58h 59h 59h |
| 8 | CCh CDh CFh D0h D1h D2h D3h D5h D7h D7h |

Leather Boots have look 0, so they leave the legs' own colors; bare feet take set 8.

**Engine model.** The player template's `figure` names its layers bottom up, each with its own look: a sprite, and the sprite for a female player when it differs (pack.md). An item's `looks` give, by layer name, the look it puts on a layer while worn in a worn slot (not a hand). The player's figure is drawn as each layer's look: the worn object's, else the layer's own, the female sprite for a player whose portrait is female. Each layer sprite has a frame for every frame of the player's sprite and its own anchor, and is drawn between the held objects behind and in front. The importer makes the layers `torso`, `legs`, `body` and `head` from the banks above, one sprite for each torso shift, leg color set, body look and head look the items and the bare figure use, and gives each body armor (looks `body` and `torso`), headgear (`head`, with the female look one more) and footwear (`legs`) its looks. The player's own sprite stays the bare figure composed whole, for anything that draws the player without its layers. Water and a transformed or invisible figure are drawn as before.

**Portrait.** The player has a portrait (formats.md, F), chosen in character creation. It is shown in the player's panel, flipped when its row says so, and it gives the figure's hair and skin: the figure's pixels in colors E0h–E3h are drawn in the portrait's four figure colors, in order. Changing sex swaps the portrait for the other-sex portrait its row names.

**Engine model.** Each of F's rows becomes a portrait `portrait.<i>`: its image (the strip stacked into one frame), whether it is mirrored, its `figure_colors`, its sex, and its other-sex and talk portraits (pack.md). The player template names its `portrait`; the player keeps its own, which the presenter reads to look the figure's colors 224–227 up in the figure colors. Until character creation the importer gives the player portrait 0.

**Confusion.** A count, 0 at first. While it is over 25 the player is confused: each step asked for has a `count`% chance to turn 45° to one side or the other, either side as likely. The count falls by 1 at every one of the player's updates while it is above 0, whether the player steps, stands, acts or is posed, so confusion wears off with time, not with steps: at an interval of 2 ticks a count of 75 falls to 25 in 50 updates, about 5½ seconds. It rises by:

| Cause | Adds | Says |
| --- | --- | --- |
| A blocked step that meets no entity and is not into a pit (walking into a wall) | 6 | |
| Power 04h (items.md) | 75 | nothing of it |
| Power 50h | 150 | |
| A change of sex (power 14h) | 50 | message 35h only |
| A ripple teleport (spells.md) | 40 | |
| An armed teleporter (world.md), unless interaction variable 34 is set | 30, or 5 on level 19 | |

When the count goes from 25 or less to over 25, message 36h ("You are confused") shows, except by power 04h and a change of sex. A wall bump is judged at the end of its update, after the fall: a count of 20 bumped ends the update at 25, not confused, and says nothing. When it falls back to 25 the game clears its message line. The status icons show the condition (scenes.md, The game screen).

**Engine model.** The teleporter's 30 is the level's `teleporter_confusion` when the level gives one (pack.md, Levels); the importer gives level 19 5. Variable 34 is the pack's new-game `counter` (pack.md, Creation). Message 36h is the engine message `confused` and 35h `sex_changed`. The engine's messages leave the line on their own, so nothing is cleared at 25.

When a step is blocked, the player pushes what can be pushed (entities.md), enters a level exit marker or a Gateway, or touches a creature, which hurts the player if it hurts on contact (creatures.md). A boulder rolling across the step hurts 2d3. Moving ends a Shape Change.

## Speed

The player's update interval in ticks:

```
interval = 5 + bonus + load
         + (10 − END) / 2 + 1     if END < 10
         − 1                      under haste
         − 1                      under battle rage
         − 2                      while running (Running)
interval = max(1, interval)
```

`bonus` comes from worn objects and powers (items.md). Running is the engine's; the game's player only walks.

**Load.** Carried weight `w` (the sum of every carried object's weight, containers' contents included except inside a Bag of Lightness) against the limit `L = 10 × STR`:

- `w < L / 10`: agility +1.
- `w ≥ L`: agility − `p`, with `p = (w − L) / 20 + 1`, and `load` gets `p + 1`.
- Then, from agility after that: `load` gets −1 at 15 or more, or `10 − AGI` below 10.

The load gauge shows `5w / L` of 5 steps.

## Exertion and fatigue

Actions add exertion (at least 1 each). When exertion exceeds `END + 3` (and END is above 1), END drops by 1, fatigue rises by 1, and exertion resets to 2.

On each status tick exertion falls by 1; when it is already 0 and there is fatigue, one point of fatigue returns to END and exertion becomes `END + 3`.

## Running

Running is the engine's own addition. Holding Shift with a walk runs: the player's interval is 2 ticks shorter (Speed), as Boots of Speed make it, and is still at least 1 tick. The player runs at an update that walks with Shift held, while no swing pose holds and no gesture plays (with a gesture playing it walks). A step that does not move, and letting go of Shift or of the walk, end the run at once.

Running tires: each running step that moves adds 1 exertion one time in four (Exertion and fatigue), besides what a push adds.

While running the player cannot attack or cast. Strikes and shots with either hand (bare hands too), throws, casting a school's spell and using an object that casts a spell are refused, and nothing is said. Other uses, such as eating and drinking, are taken as when walking. Shift's other meanings (a hand's second action, the next spell) hold when not walking; selecting the next spell is taken while running.

Running is not saved: a restored game walks until Shift is held again.

## Status

Lasting conditions:

| Status | Effect |
| --- | --- |
| Poisoned | 1–2 damage, 1 in 40 moves. Set by poisoning wounds and power 04h, with message 59h when not poisoned already |
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

## Napping

Napping is the engine's own addition, to wait for magic points without leaving the game running: holding N runs the whole world 64 times as fast. A press of N starts a nap unless the magic points are full ("Your magic points are full.") or a living creature is near the player (within the distance power 12h reaches, items.md) on a cell the player can see (world.md, Visibility; "A creature is near."), so creatures behind a closed door or a wall do not count. The nap goes on while N is held, and stops when N is released, the magic points become full, a creature comes near, the player is hurt, a message is shown or a conversation starts, or another key or a click is taken. While it runs, the view shows time passing quickly: what the presentation runs on time (the animations that run on time, palette effects and the lights' flicker) runs 8 times as fast. They do not follow the world's 64, which would move each a second or more per frame and leave them looking still. While it runs, the top of the screen shows "Waiting: " with the magic points and the time played (hours, minutes and seconds of world time). After a stop, N starts a nap again only once released and pressed again.

## The book

The book is the engine's own addition: a record of the creatures and objects the player has come across, with what the player has learned of each. The game has nothing like it, and no text that describes a creature or an object; what the book says it works out from the templates, with a pack's own texts beside.

**Engine model.** The world keeps an entry for each thing template of kind creature, other than the player's, and each item template, once the player has come across one of its creatures or objects. An entry has a stage, the level and step it was first reached on, and for a creature the kills, the items dropped and the defenses learned. Reaching a stage reaches those before it: a creature first met by being killed is entered at its last stage. The book is world state, kept in saves; a save without one, or imported from the game, starts with an empty book.

| Stage | A creature's, when one of its creatures | An item's, when one of its objects |
| --- | --- | --- |
| 1, seen | stands in a cell visible to the player at the end of an update | lies uppermost in a cell visible to the player at the end of an update (contents of what lies there are not seen) |
| 2, fought or handled | strikes at the player, hit or miss; hurts the player by touch or by any blow it is named for; or is hurt, on the current level | is on the pointer, in any inventory slot, or inside a container there at any depth, at the end of an update |
| 3, killed or used | dies on the current level, by any cause | is used, or put on, with a power, even when the power refuses (a refusal tells what it does); is in a worn slot (the hands included) at the end of an update, with a power that starts worn (`power_worn`); casts its spell; starts its conversation; or, thrown or fired, hurts a creature or the player |

A creature's defenses are learned only by meeting them, each at the moment it works on one of the template's creatures on the current level, which also reaches stage 2:

| Defense | Learned when |
| --- | --- |
| Its vulnerability | the player's blow hits it with a weapon of that class, or an arrow hurts it and the class is the bows' (3) |
| `resists_fire`, `resists_lightning`, `resists_poison` | it is hurt by fire, lightning or poison, or the player's blow hits it with a poisoning weapon (`poisons`) and it resists poison |
| `physical_only` | the player's blow hits it, or it is hurt by anything but a physical blow |
| `resists_teleport` | it stands on an armed teleporter, a pit or an open trapdoor and is not taken |

Each death on the current level adds one to its template's kills. The items of the objects a dying creature drops are added to its entry's drops, each once, in the order first dropped. An item with nothing to use (no power, spell or conversation) and no `poisons_thrown` has nothing more to learn after stage 2.

**Recipes.** The book also keeps the recipes the player has made, each once, in the order first made: a spell that fills a holder from the other hand (`transmute`, spells.md, Liquify) records the spell, the holder, the object it made from (none when the other hand held nothing it turns, and the result is `otherwise`) and what it made. A recipe reaches the entries of all three items, each at stage 1 at least, and each of them shows it, from first sight, after where it was first seen:

- the object it was made from: "Liquify, with an Empty Bottle, makes it into Teiwaz." — the spell's name, the holder's and the result's;
- what was made: "Made by Liquify from Amethyst in an Empty Bottle.", or with nothing turned, "Made by Liquify in an Empty Bottle from anything else.";
- the holder: "Liquify fills it from Amethyst with Teiwaz.", or "Liquify fills it with Jera from anything else.".

What an entry shows, stage by stage, each stage adding to the last:

- **Creature, seen:** its name (the template's `name`, else its conversation's name), its picture (the first frame of its idle animation for its starting facing, or the first frame `animate` shows), its `lore` text, that it can be talked to (`talks`), and where and when it was first seen (the level's name and the play time).
- **Creature, fought:** its blow's damage (`attack`); that its blows poison (`poisonous`), that touching it hurts (`hurts_on_contact`), that one blow can kill (`death_blow`), that it passes the damage it takes back (`mirrors`); the spell it casts, by name; and each defense learned: the weapon class it is vulnerable to, by name (Creation's `weapon_classes`, or else the name of the weapon that class starts with), that it resists fire, lightning, poison or teleporting, and that magic does not hurt it and of weapons only silver does (`physical_only`).
- **Creature, killed:** its hit points (dice); the experience it is worth (dice); its kills; and its drops, by name.
- **Item, seen:** its name, its inventory image, its `lore` text, and where and when it was first seen.
- **Item, handled:** its weight in half kilograms; each strike action, as the hotbar's boxes number them: its damage (dice), and for a second one whether it is slower than the first and the weapon skill it needs (`skill`); a single strike is just its damage; its armor; the slot it is worn in; that it needs two hands (`two_handed`); a container's capacity and largest size, and that its contents weigh nothing; its charges, when more than one.
- **Item, used:** its `effect` text; the spell it casts, by name; that it poisons what it hits when thrown (`poisons_thrown`).
- **Scrolls.** Reading a scroll to learn its spell is how the game is played, not something to find out, so an item with `teaches` is not shown by stages: from first sight its entry shows its name, image, `lore` and where it was first seen, then the spell whose gesture its `teaches` is, by name with its school, its cost in magic points, and the spell's `effect` text. When `teaches` is no spell's gesture, it names the gesture's icons instead. A scroll's entry shows no hint.

The book's window opens with J or the hotbar's Book box, and holds the world while open, as the conversation log does. Two tabs, Creatures and Items, each list their entries by name in alphabetical order, with the count found. A filter field beside the tabs, empty when the book opens, narrows both tabs to the entries whose names hold what is typed, ignoring case, and the counts to those shown; Backspace takes a letter off, and a tab with no entry that matches says "Nothing matches.". Escape clears the filter when it holds anything, and otherwise closes the book. Up, Down, the wheel and a click choose an entry, Page Up and Page Down move a page, Left, Right and a click on a tab switch tabs, and a right click or Close shut it. The chosen entry shows on the right with its picture, at twice its size, its lines, and below them, unless it is a creature that can be talked to (`talks`), what the next stage would show: "Fight one to learn more.", "Defeat one to learn more.", "Pick one up to learn more.", "Wear one to learn more." (an item whose power starts worn and that has no use action), "Use or throw one to learn more." (an item with `poisons_thrown` and something to use), "Throw one to learn more." (with `poisons_thrown` and nothing to use) or "Use one to learn more.".

**Import.** The importer writes an `effect` text, in its own words, for each item whose power the game's table gives (items.md, Powers), and none for powers that cast spells or start conversations, which the book names itself. It writes an `effect` text for each spell, also in its own words, from what the spell does (spells.md, Effects). It writes no `lore`; the game has none.

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
