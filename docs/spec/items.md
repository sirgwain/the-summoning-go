# Items

## Templates

OBJECTS holds 254 templates of 15 bytes. An object is an instance of a template with one value of its own (durability, charges or count) and, for a container, its contents.

| Byte | Meaning |
| --- | --- |
| 0 | Low nibble: size. High nibble: armor bonus |
| 1 | Name: TEXT id `name_base + byte` |
| 2 | Weight |
| 3 | Container: low nibble capacity − 1, high nibble the largest size it takes; 0 if not a container |
| 4, 5 | Action icons of actions 0 and 1 |
| 6, 7 | Damage codes of actions 0 and 1 (combat.md) |
| 8 | Damage code of action 2; its nibbles + 3 also give hand delays |
| 9 | Charges (the starting value of objects that have one) |
| 10 | Image `i`: the small image is resource small-object base + `i`, the tiny image resource tiny-object base + `i` (INIT) |
| 11 | Flags: 80h never chosen as a random object; other bits select the worn look |
| 12 | Use handler: the low bits are the power number; 80h set means the power also starts when the object is worn |
| 13 | Category: bits 0–5 the category; 40h a weapon; 80h two-handed |
| 14 | Scroll number, 1..25 (spells.md, Scrolls); 0 if not a scroll |

**Categories.** 1 body armor ("shirts"), 4 quiver, 7 arrows, 8 bottles (break into Broken Glass, template 0Ch); categories 9..12 are weapon classes 0..3 (a weapon's class is its category − 9). Weapons have the 40h flag.

**Template 0** ("Random Object") in a level is a random object: any template other than 0 that is not never-random and is a weapon or of category 6. Template 0 itself is never chosen.

**The Bag of Lightness** is template 05h.

**The Sack** (template 04h) shows how full it is: with `n` objects inside, its small image and its worn image are image `i + (n + 2) / 3` (integer division; `i + 0` when empty) instead of `i`. Its nine places give four pictures.

## Object values

A new object's value:

- Container: its (empty) contents instead.
- Weapon: 30000 (unbreakable) for the Rusted Blade (63h), WarMonger (64h) and the Silver Sword (BFh); 6000 at weight 10; otherwise weight × 250. This is its durability.
- Body armor: 7500.
- Anything else: the template's charges.

Templates 72h–7Bh also have one **shared count** per template, held by the game rather than by any object: creating an object of one of these templates sets its count to the template's charges, so every object of the template draws on the same charges and a new one refills them. Most of them are weapons, which keep their durability as their value besides.

An object's **quantity** is the first that applies:

- Templates 72h–7Bh: the template's shared count.
- Template 81h: always 1.
- Containers, weapons and body armor: none.
- Anything else: its value.

**Engine model.** An item with `shared_charges` keeps its charges as one count for the whole item, which the world holds and saves; making an object of the item sets the count to the item's `charges`. The importer gives it to templates 72h–7Bh, with their charges, and their durability when they are weapons. A weapon or body armor otherwise has no `charges`, so its use actions are never available.

## Armor

Worn objects add their armor bonus to the player's armor class. Body armor's effective bonus falls by 1 for every full 500 its value is below 7500, but never below half its bonus.

## Weight and containers

An object's weight is its template's, plus the weights of what it holds, except that a **Bag of Lightness** carries its contents weightless. A container takes an object whose size is at most its largest size, while it holds fewer than its capacity.

## Inventory

The player has slots:

| Slot | Holds |
| --- | --- |
| 0–19 | Shelves: anything |
| 20 | Head |
| 21 | Arms |
| 22 | Right hand |
| 23 | Left hand |
| 24 | Feet |
| 25 | Quiver: an empty slot takes only a quiver; a filled one takes what fits in the quiver |
| 26 | Body |
| 27 | Neck |
| 28+ | Pack: the contents of the bag held in the left hand, one slot per capacity |

Pack slots exist only up to the right-hand bag's capacity and take what fits in it. Putting an object in a worn slot (20–27, the hands included) adds its armor and, when its use handler has 80h, starts its power; taking it off ends both. Ending undoes the powers marked "lasts while worn" and the speed powers (09h–0Bh), so a worn object's speed counts only while worn. The Leather Boots (24h) every character starts in have power 0Ah with 80h: interval −2 while worn. Taking an object from the floor with the take key puts it in the first empty shelf and shows its name as a message; with none, message 7Fh and it stays on the floor. An arrow taken goes into the quiver when it fits, with message 99h (`%N put in your quiver`, the object's name). With nothing on the floor, message 80h. Taking an object onto the pointer shows its name too.

Objects given by the game (conversations, powers) go into the first empty shelf, or are dropped at the player's feet.

**The hand.** Besides its slots the player has a hand, the object held on the pointer. Taking an object from the floor with the mouse puts it in the hand; clicking a slot swaps the hand with the slot; putting down puts the hand's object on the floor (rules.md, The player and the features).

**The mouse.** With the left button:

- Clicking the player's figure with an empty pointer takes the top object of the player's cell onto the pointer (pick up).
- Clicking the player's figure with an object on the pointer puts it down on the player's cell (put down), so an object meant for a keyhole, a socket or a plate goes in by standing on it and clicking the player.
- Clicking a slot (a shelf, a hand, a worn slot, a pack slot) swaps the pointer's object with the slot's.

A click counts as on the player's figure for these, rather than walking, anywhere within 32 view pixels of the point 8 right and 2 down from the top-left of the player's sprite, measured from the pointer's hot spot as `|dx| + |dy| < 32`. The engine also counts the figure's own pixels and its cell's ground.

With the right button, an object on the pointer goes to the first empty shelf (with none, message 7Fh and it stays on the pointer); with an empty pointer, a click in the view does what the take key does: the top object underfoot goes to the first empty shelf, with its messages. (The engine adds right clicks on boxes: see Opening containers and Using from a slot.)

## Opening containers

In the game the pack is always the bag in the left hand: to look into a bag or chest, the player holds it there. The engine also lets the player open a container where it lies. **Engine model.** The player has an open slot, none at first. Opening a fixed slot (0–27) holding a container makes that container the pack: the pack slots are its contents, as for the bag in the left hand. Opening the open slot again closes it. While the open slot holds no container (the container was taken out of it), and while none is open, the pack is the bag in the left hand as in the game. The mouse opens and closes: a right click on a box holding a container, with nothing on the pointer, on the hotbar or the inventory panel, opens it (showing the inventory panel, which shows the pack) or closes it. Other right clicks are as The mouse says. The open container's box is marked. Closing the inventory panel closes the open container too, so it is not open when the panel next opens.

## The inventory panel

The game shows the inventory in panels below the view.

- **The paper doll.** RESOURCE 8, a strip 96 pixels wide and 128 high, is the figure. Each worn slot's object is drawn on it with its worn image: RESOURCE `worn base + i` (INIT), a strip, where `i` is the object's image byte (byte 10), at the worn-image offset `i` from INIT (row, column) from the doll's top-left. The quiver slot's object is drawn with worn image 35h instead, and a quiver anywhere else with 36h; in the left hand, a bag shows how full it is (Containers) and the chest (template 3) worn image `i + 1`. The right hand's image is mirrored left to right and drawn at column `96 − column − width`. They are drawn in the order quiver, body, head, feet, neck, arms, then the two hands (right, then left), and then again the two hands for the objects left out the first time: those of category 10 with armor above 0. A click on the doll picks the slot by the region it falls in: the top-left corner (rows below 38, columns below 38) is the quiver; the top (rows below 32) the head; the neck around (rows 31..45, columns 39..54) while something is worn there; rows from 102 the feet; from 79 the body; from 53 the right hand left of column 44 and the left hand from it, and from 39 too while that hand holds something; an empty hand's region is the arms, or the body when nothing is worn on the arms; anything else the body. With an object on the pointer the click goes instead to the slot its category takes (Wearing), a hand object to the right hand left of column 44 and the left hand from it, and a quiver clicked from row 49 down to the hands the same way; but an object of size 0 clicked in the quiver's region goes to the quiver, unless it is an arrow and no quiver is worn. The slot then takes it or refuses it as a click on its box would.
- **The hands.** Each hand has three action boxes, RESOURCE 31h: its first two actions and throwing. A box shows the object's small image with the action's icon over it, RESOURCE 30h frame `(icon − 100) / 10 + 1` for icons from 100 and frame `icon` below; the throw box frame 0. An empty hand shows RESOURCE 32h (frame 1 for the right hand, 0 for the left) with the strike icon, frame 8.
- **The pack.** A bag held in the left hand opens as the pack: its picture, a bar as long as its capacity (RESOURCE 27h), and its contents in two rows of slots.
- **The character page**, one of three: the six attributes as `STR: 18`, an attribute below or above its maximum drawn in another color with the maximum after it; the weapon classes with a level, highest first, each with its icon (RESOURCE 12h, frame = class) and its rating, TEXT 1..10 by level (at most 10); the schools the same way (RESOURCE 13h, TEXT 11..20).
- **The information panel** shows the object on the pointer: a scroll's picture (spells.md, Scrolls), or the object's small image, its name, "Weight: " (TEXT 81h) and its weight, for a weapon "Can do " (82h) its largest damage and " damage points" (83h), "AC bonus of " (84h) its armor (its effective armor when worn down, with the full bonus in parentheses), and "Requires two hands" (85h) for two-handed objects. With nothing on the pointer it shows the character's name, its level's title (TEXT `15h + level`) and the weight carried, and always the armor gauge and the experience. The armor gauge is 41 tiles: of `v = 30 − armor class` (kept within 0..30), `41 × v / 30` tiles show RESOURCE 0Fh, at least one while `v` is above 0 and one short of all while it is below 30, and the rest RESOURCE 10h; so the 10h part grows with the armor.

Weights are in half kilograms: a weight `w` shows as `w / 2` kg, with ".5" for an odd `w`.

**Engine model.** The pack's inventory record gives the doll's picture, the quiver slot's look, the empty hands' pictures, the strike and throw icons, the skills' icons and ratings, the level titles and the panel's labels; an item gives its worn look (`doll`, an image and where it goes on the doll) and each action its icon. The engine's panel shows the doll with the worn looks, clicked as above, and a box for each worn slot beside it, the pack's slots, the character's attributes (one below its maximum in a warm red, one above it in green, each with its maximum after it), skills, hit and magic points, the armor gauge with the armor class after it, level and experience, and the details of the object under the pointer or on it. Its boxes work as the hotbar's do (The mouse). The pointer resting on a box of either names its object beside the pointer; for a container (a bag, a chest) the names of what it holds follow, a line each in the order they were put in, repeats counted (`Arrow x4`), or "Empty". The hotbar's hands show three action boxes each, the third throwing; a box shows the action's icon alone, without the object's picture under it, since the hand's own box beside it shows the object.

**Holding the world.** The game does not pause for its inventory panels, and neither does the engine's inventory panel: the world runs while it is open, so what an object sets off in time (a missile cast by a potion or wand, a timed boost) goes on as it would with the panel closed. The engine's spell panel holds the world while it is open (spells.md). A held world takes the commands that only handle the player's objects and spells, and does them at once: picking up, putting down, swapping with a slot, opening a container, putting on, using from a slot, moving a slot, reading, learning, memorizing and selecting a spell. What these set off happens then too (a plate pressed, a slot filled, a potion drunk), and what waits for a view effect is done before the next of them. No time passes: the player, creatures, flights, doors and timers do not move, and the step does not advance. Any other command is refused.

**Engine model.** The player has a hand and slots numbered as above. Commands:

- **Pick up** takes the top object of the player's cell, as the take key does: an arrow into a worn quiver with room, anything else into the first empty shelf, else message 7Fh (`no_room`) and it stays on the floor; with nothing there, message 80h (`nothing_to_take`). Taken, the object is named by the engine message `taken` (`{item}`, the importer's own text, as the game shows the bare name), or `into_quiver` (99h) for an arrow put in the quiver. **Pick up into the hand**, as clicking the player with an empty pointer does, takes it onto the pointer instead; with an object already in hand it is a plain pick up. Taking it off the floor lifts its weight off a plate.
- **Put down** puts the hand's object down on the player's cell.
- **Swap** exchanges the hand with a slot. Shelves and the two hands take anything. A pack slot takes an object whose size is at most the bag's largest, and never the bag itself; putting an object in the first free pack slot adds it to the bag's contents, and taking one out removes it, so the contents stay in order. The worn slots (20, 21, 24–27) are added with Fight, along with what each takes.

## Floor

Each cell holds a list of objects on its floor; at most eight are drawn. Objects dropped or thrown land on the floor. Objects left on a level stay there.

An object on the floor is drawn with its tiny image (about 8×6 pixels; the small 16×16 image is the inventory icon), with its top-left at `(8, 13)` from its cell's tile box corner (world.md), whatever its size, so a tiny image sits a little left of the box's middle; on a pressed weight plate (terrain 20h) 4 pixels lower, at `(8, 17)`. The objects of a cell are drawn in list order, each over the last, after the cell's floor and overlays and before anything standing.

**Magnified.** Objects on the floor are drawn with their small image instead, at the same place, so they are easier to see and to pick up, while the player has timed sight (the Sight spell, spells.md) or while interaction variable 98 is set. The options menu's Magnify option sets and clears variable 98 (dialogue.md), so it lasts across saves.

**Engine model.** The pack names the counter that magnifies (`magnify`, pack.md); the presentation draws floor objects magnified while it is set or the player's timed sight lasts, or while its own magnify setting is on. Magnifying changes only how floor objects are drawn and where they can be clicked, never the simulation.

## Hands and actions

Each hand shows the actions of the object it holds (bare hands can strike). An action icon:

| Icon | Meaning |
| --- | --- |
| 100 and above | Attack, available when the weapon's class skill is at least `icon % 10`; `icon / 10` picks the picture, except as below |
| 110–119 | Two-handed attack |
| 120–129 | Fire a bow |
| 9, 10 | Use the object up |
| 11–20 | Use one charge |

Using an action makes the hand wait its delay before the next action; each player update counts the delays down. A weapon swing is an attack (combat.md). An action of icon 11–99 needs a quantity (Object values) and is unavailable when the object has none.

**Engine model.** An item has up to two **actions**, each a kind, damage dice and a delay:

| Kind | Icons | Does |
| --- | --- | --- |
| `attack` | 100 and above, but for these | Strikes with the hand; needs the weapon class's skill at least `icon % 10` |
| `attack_both` | 110–119 | Strikes with both hands; the same skill rule |
| `fire` | 120–129 | Fires an arrow from the quiver |
| `use_up` | 9, 10 | Uses the object up for its power |
| `use` | 11–20 | Uses a charge for its power |

The **delay** of action 0 is `(byte 8 >> 4) + 3` points, of action 1 `(byte 8 & 0Fh) + 3`; an item flagged two-handed (category 80h) delays both hands. A hand holding a two-handed object while the other hand holds anything is **blocked**: none of its actions can be taken (its attacks, uses and throw alike, by click or key), and its action boxes show no icons, as a hand whose delay is still running does; nothing is said. Emptying the other hand, or taking the object out, frees it. The other hand acts as usual. **Engine model.** The hand's boxes on the hotbar and the inventory panel show fully shaded while it is blocked, and its commands are refused (the hand is blocked). Each point is three player updates: a hand waits `3 × delay` of the player's updates before acting again, counted down by one on each. An attack adds `weight / 8` exertion (player.md). An empty hand strikes with bare hands, 2d2 damage and a delay of 2 points (6 updates). A pack's action delays are in player updates, so the importer gives `3 × ((byte 8 >> 4) + 3)` and `3 × ((byte 8 & 0Fh) + 3)`. Icon FFh is no action.

**Wearing.** An item's category gives the worn slot it goes in: 0 head, 1 body, 2 feet, 3 arms, 4 quiver, 5 neck; categories 6–12 go in a hand. The engine model calls this its `wear` slot. Weapons (40h) have a weapon class, `category − 9` for categories 9–12. Putting an item on, by swapping it into its own worn slot, adds its armor and starts its power; a worn slot refuses anything whose `wear` is not that slot. The quiver slot also takes arrows into a worn quiver.

**Value.** The importer writes the starting value rules above as the item's `durability` (weapons and body armor) or `charges`.

**Breaking.** An item's `on_break` are actions run when it breaks on landing: the Decanter's sets interaction variable 18 to 1, starts conversation 2Eh, removes every creature of record 2Eh and gives 72903 experience. They are the registry's `set_counter`, `converse`, `remove_all` (the creatures of a thing template on the current level) and `experience`.

**Flying looks.** An item's `flying` is its thrown look: a sprite with a frame for each direction, and with `fired_only` only when fired.

**Using up.** An object used up (icons 9, 10) leaves a spent object in its hand when it has a quantity: bottles (category 8) leave an Empty Bottle (template 7), other objects the next template. An object with no quantity, such as a rune, leaves the hand empty. A charged object (icons 11–20) is never replaced: at no charges it stays where it is, its actions unavailable.

**Engine model.** An item's `spent` is the item it leaves when used up. The importer gives it to items with a `use_up` action and nonzero `charges`; the others have none.

**Using from a slot.** In the game an object is used only from a hand. The engine also lets the player use an object where it lies, on a shelf or in the pack: **use slot** takes the object's first `use` or `use_up` action as the right hand would, with the right hand's delay (it refuses while the right hand is not ready, and does nothing for an object with neither action). The object is used up in its own place, the spent object taking that place, a pack slot's emptying when there is none. Used this way from a hand, it is that hand's action. A right click on the box of such an object, with nothing on the pointer, uses it; a right click on a container's box opens it, as before.

**Quick moves.** In the game every move goes through the pointer. The engine also moves an object from one slot to another in one click. **Engine model.** **Move slot** takes the object in a slot (a fixed slot or a pack slot) to a place of one of three kinds, while nothing is on the pointer:

- `pack`: the pack's next place, after what it holds. It is refused with no pack, when the pack is full or the object is larger than its largest size, and for the pack's own container.
- `shelf`: the first empty shelf; with none, message 7Fh and the object stays.
- `worn`: the slot its `wear` names. A hand item goes in the right hand, a shield (a hand item with armor and no weapon class) in the left. What the slot held takes the object's old place, so a shelf's sword and the right hand's axe change places. It is refused for an object with no `wear`, and for one already in its slot.

Each place takes the object as a swap with the pointer would: putting it on or taking it off starts and ends its armor and power, and it is not read. A refused move leaves everything where it was.

The mouse moves with Shift: Shift and a left click on a box, with nothing on the pointer, moves its object

1. from the pack to the first empty shelf;
2. else, while the inventory panel is open and shows a pack, into the pack (but the pack's own container);
3. else, from a worn slot or a hand to the first empty shelf;
4. else to its worn slot.

An object that none of these can take (one with no `wear` on a shelf, with the panel closed) is read instead, as Shift and a click on a box reads with an object on the pointer.

## Powers

An object's power starts when it is worn (if it lasts while worn) or used. Some powers check first that they can do something; when they cannot, they show a message and the object is not used up.

| Power | Effect | Lasts while worn |
| --- | --- | --- |
| 01h | Magic points +2d6 | |
| 02h | Magic points +4d6 | |
| 03h | Magic points +3000 | |
| 04h | Confusion +75 (player.md); 2d8 hit points lost, past every defense and the armor; poisoned | |
| 05h–08h | School 0–3 skill +1 | |
| 09h | Interval −1 | |
| 0Ah | Interval −2 | |
| 0Bh | Interval −2 | yes |
| 0Ch | Status: resist quarter | |
| 0Dh | Status: resist forty | |
| 0Eh | Status: blow protection | |
| 0Fh | Levitation (lifts the player off a plate) | yes |
| 10h | Weapon class 3 skill +3, quietly | |
| 11h, 26h, 1Ah | Teleport to the level's rune mark 21h, 22h or 23h; refused (message 43h) when the level has none | |
| 12h | 2000 damage to every creature near the player; refused (88h) when none | |
| 13h | Cast spell 3 at power 20 | |
| 14h | Half the time, change sex (message 35h, confusion +50, player.md); otherwise a random attribute and its maximum ±1 | |
| 15h | Cure poison and confusion; refused (41h) unless poisoned or confused over 25 | |
| 16h | Nothing | yes |
| 17h | Status: regenerating | yes |
| 18h | Shows "(cast smoke spell)" | |
| 19h | Cast spell 4 at the object's action-1 damage | |
| 1Bh | Turn the object into a random object | |
| 1Ch | Fire shield, 10 | |
| 1Dh | Heal 6d6 (the Jera potion: its value); refused (3Eh) at full | |
| 1Eh | Cast spell 0Bh | |
| 1Fh | Weapon class 1 skill +2, quietly | |
| 20h | A random school +1 | |
| 21h | Delay every other entity by 100 ticks; message 39h | |
| 22h | Show the level map, unless the level refuses it (world.md) | |
| 23h | Status: charges | |
| 24h | Cure poison; refused (40h) unless poisoned | |
| 25h | A random weapon class +1 | |
| 27h | Lightning shield, 10 | |
| 28h | Heal 5d10; refused at full | |
| 29h | Cast spell 0Eh at power 20 | |
| 2Ah–2Fh | Attribute 0–5 +2 for 10 status ticks (if already boosted, message 33h and the time extends) | |
| 30h–41h | Attribute `(n − 30h) / 3` +`(n − 30h) % 3 + 1`, quietly | yes |
| 42h | Shows "(open all doors)" | |
| 43h | Status: accurate | |
| 44h | Recover 1–3 fatigue; refused (3Fh) without fatigue | |
| 45h | Heal 3000; refused at full | |
| 46h | Heal 20d4; refused at full | |
| 47h | Invisibility | yes |
| 48h–4Bh | Weapon class 0–3 +1 | |
| 4Ch | Status: mana regenerating | yes |
| 4Dh | Death protection, message 3Ch; refused (21h) if already protected | |
| 4Eh | Cast spell 0Ah at power 10, message 3Bh | |
| 4Fh | Fire immunity and interval −2 | |
| 50h | Confusion +150 | |
| 51h | Conversation 30h; the object is used only if the conversation sets its result variable | |
| 52h | Conversation 46h | |
| 53h–5Ah | Conversation `n − 0Bh` | |
| 5Bh | Status: death-blow ward | |
| 5Ch | Cast spell 5 at the object's action-1 damage | |
| 5Dh | Cast spell 6 | |
| 5Eh | The ending, chosen by what the player holds (scenes.md); refused (8Eh) unless holding the Chesschantra Skull (F7h), WarMonger (64h) or the Necklace of Runes (59h) | |

Taking off a lasting object undoes its effect (interval, status, attribute).

**Timed objects.** An object in a worn slot (20–27, the hands included) whose use handler has 80h and whose power lasts while worn is timed. Each status tick (player.md) its value, or its template's shared count for 72h–7Bh, drops by 1 as a 16-bit number: a value of 0 becomes 65535 and so lasts that many ticks more. On reaching 0 the object comes off, ending its power and armor; message 32h shows with its name; it flashes on the figure and is destroyed. The flash takes the object off the figure and puts it back ten times, 3 ticks each way, while the game waits; a weapon that breaks (combat.md, Wear) flashes off its hand the same way. **Engine model.** In place of the flashing, the object is gone at once and the presentation shows it going by the pack's lost-item style (pack.md, Effects): in its box on the inventory panel while that is open, else on the hotbar, else over the player's head. With `inventory`, the inventory panel opens for it, if it is not open, and the world waits until it has gone; the panel then closes again if it opened for it. The importer gives The Summoning `shatter` for 1200 ms with `inventory`, so it shows on the figure as the game's flash does, in less time: the box flashes, and the image breaks into a 3 × 3 grid of pieces that fly out from its middle, rise a little and fall, turning and fading. Body armor counts down the same value that wear takes down, so a timed cloak lasts at most 7500 ticks.

**Engine model.** The engine counts timed objects down on each status tick before attribute boosts, and announces one that runs out with the engine message `vanishes` (32h, `{item}` its name).

## Using objects

An action of kind use up (icons 9, 10) or use (11–20) starts the object's power, whatever 80h says. A power that refuses (below) leaves the object as it was. Use up then replaces the object with its spent object, if any (Using up); use takes one from its quantity, and the object stays in the hand at 0. An object with no quantity loses nothing. Neither strikes. Icon 11h also swings the object, without a blow: the swing pose (combat.md, The player attacks) of both hands for a weapon (40h), else of its hand, whether or not the power starts.

**Engine model.** An item's `power` is its number and `power_worn` the 80h flag; `spent` is the item it leaves when used up. The engine runs the powers in the table that need only the player: hit and magic points, cures, skills and attributes, statuses and timed effects, speed, and teleporting to a rune mark (a `mark` feature whose rune is `1`, `2` or `3` for 21h, 22h, 23h). A power that casts a spell is the item's `casts`: the spell and its power (none for the spell's own roll), with powers 19h and 5Ch taking the item's action-1 damage dice; using the object casts it at once, with no gesture, cost or memorization, and the cast practices the school as any cast by the player does. A power that starts a conversation is the item's `conversation`, a thing template whose conversation it holds; with `use_result` (power 51h) the object is used only when the conversation leaves the pack's result counter nonzero, and otherwise never. Power 22h shows the level map and is never used up; power 5Eh chooses the ending (scenes.md, Endings), refused with `no_ending` (8Eh).

A scroll's `teaches` are the icons reading it teaches, and its `picture` the image shown when it is read. Each refusal is an engine message: `full_health` (3Eh), `full_magic` (42h), `not_poisoned` (40h), `not_poisoned_or_confused` (41h), `not_fatigued` (3Fh), `no_destination` (43h), `no_creatures` (88h), `already_have` (21h); others shown: `healed` (4Ah, `{damage}` the amount), `protected` (3Ch), `boost_extended` (33h), `poisoned` (59h), `slowed` (60h), `lost_weapon_skills` (64h), `lost_magic_skills` (67h).

## Throwing

The player throws the object in hand, or fires an arrow from a bow. A thrown object flies `(100 − weight) / 5 + 5` steps (weight above 100 counts as 100); an arrow `5 × STR` steps, at most 127; 8 units a step. It flies as an entity of kind 3 (entities.md). A bow cannot be drawn while anything but a corpse stands within a step ahead of the player: firing gives message 96h ("Not enough room to draw bow") and the player must back away first. A throw or shot from too near a wall gives 28h ("Too close to wall to throw").

**Damage.** Seven templates have a thrown look of their own, RESOURCE `held-item base + offset` (INIT), with a frame for each way it flies: frame `(d + 1) mod 8` along table direction `d` (the player's facing that walks `d`). The others fly showing their tiny image.

| Template | 67h | 68h | 75h | 6Dh | 97h | 98h | 99h |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Offset | 27h | 28h | 29h | 2Ah | 2Dh | 2Ch | 2Bh |
| Flagged | | | | | yes | yes | yes |

A flagged template thrown rather than fired flies showing its tiny image instead.

They roll their action-1 damage, or 1 when a flagged one is thrown rather than fired. Anything else does `weight / 10 + 1`. The player's arrows add `(ACC − 10) / 2`, at least 1 in all. A thrower touching its target hits at once.

**Hitting.** A thrown object that meets an entity hurts it (arrows as damage kind 3, everything else 0) and lands. Only the player's throws land; a creature's vanish. The Isa Potion (82h) and the Poison Arrow (97h), thrown or fired, also poison what they hurt: a creature unless it resists poison (combat.md, Hurting a creature), the player as a poisoning wound does. A thrown Isa Potion does not confuse: creatures have no confusion.

**Landing.** On a door it may open it (rules.md). An arrow breaks 1 time in 25. A bottle breaks into Broken Glass. If the **Decanter (90h)** breaks, every creature of record 2Eh is removed, conversation 2Eh starts, and the player gains 72903 experience.

Throwing is not possible in water of depth 3 or more.

**Engine model.** The player throws the object in a hand (that hand then waits a delay of 5 points, 15 updates) or, with a `fire` action, shoots (pack.md, Items). The importer gives a bow's `fire` action the quiver's arrows as its ammunition (the first arrow of the quiver worn in slot 25) and `needs_room`: message 48h `no_quiver` without a quiver; then 96h `no_room_to_shoot` when the player's box moved a full step along its facing, `speed` units even along directions 1 and 5, overlaps any thing but a corpse; then 49h `no_arrows` when the quiver is empty. So a bow cannot be fired at a creature within a step: the player must back off first. An arrow flies `5 × STR` steps, at most 127, as a shot's default range. A throw or shot whose flight would start in a wall gives 28h `too_close_to_wall` (entities.md, Things in flight). Throwing practices weapon class 3 by 1; firing does too, and shows the both-hands strike pose. In deep water of depth 3 or more throwing gives 70h (`cannot_throw`) and other actions 73h (`cannot_act`). A thing in flight (entities.md) carries the object; how far it flies and its damage are as above, the damage from the item's `missile` (its dice, and whether only firing rolls them). A landing bottle becomes the item's `breaks_into`. An item's `poisons_thrown` makes it poison what it hurts in flight; the importer gives it to 82h and 97h.

Arrows taken up go into a worn quiver when it has room; putting arrows on (in slot 25) with a quiver worn adds them to it.

## Special objects

- **Transmuting** (spells.md) turns the held object into: Amethyst → Teiwaz, Emerald → Isa, Ruby → Algit, Diamond → Snake Oil, NightShade Leftovers → Isa, anything else → Jera.
- **Mending** joins the two staff halves D8h and D9h held in the hands into DAh.
- **Gems** C0h–C5h are gems 1–6 for the gem socket (rules.md).

## Import

Each template becomes an item:

- **Id**: `item.` and its name in lowercase, with runs of other characters as `_` (`item.bag_of_lightness`). A name used by several templates numbers the later ones from 2 in template order (`item.rainbow_sphere.2`).
- **Name**: the name's text. **Weight** and **size** as given.
- **Container**: present when byte 3 is not 0, with its capacity and largest size; weightless for the Bag of Lightness.
- **Tags** by category: 1 `body_armor`, 4 `quiver`, 7 `arrows`, 8 `bottle`; `weapon` with the 40h flag, `two_handed` with 80h.
- **Random**: a template that may be picked for a random object (above).
- **Icon** and **floor** images: frame 0 of the small and tiny images.
- **Fill**, for the Sack: `per` 3 and, for each step from 1 to `(capacity + 2) / 3`, the small image and the worn look of image `i + step`. An object of an item with a fill shows step `(n + per − 1) / per` (at most the last) of its `n` contents in its boxes and on the doll, and its own icon and worn look when empty.

Template 0 in a level becomes a random object rather than the item itself.

## Unknowns

- The figure in water, and the figure frame shifts in INIT.
- Which hand objects count as armor drawn last on the doll; the engine draws those with an armor bonus last.
- Category 6, and the hand delay formula beyond what is given.
