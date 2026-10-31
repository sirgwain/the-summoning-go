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
| 14 | Scroll: the spell a scroll teaches |

**Categories.** 1 body armor ("shirts"), 4 quiver, 7 arrows, 8 bottles (break into Broken Glass, template 0Ch); categories 9..12 are weapon classes 0..3 (a weapon's class is its category − 9). Weapons have the 40h flag.

**Template 0** ("Random Object") in a level is a random object: any template that is not never-random and is a weapon or of category 6.

**The Bag of Lightness** is template 05h.

## Object values

A new object's value:

- Container: its (empty) contents instead.
- Weapon: 30000 (unbreakable) for the Rusted Blade (63h), WarMonger (64h) and the Silver Sword (BFh); 6000 at weight 10; otherwise weight × 250. This is its durability.
- Body armor: 7500.
- Anything else: the template's charges.

Templates 72h–7Bh share one count per template across all their objects. Template 81h always has 1.

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

Pack slots exist only up to the right-hand bag's capacity and take what fits in it. Putting an object in a worn slot (20–27, the hands included) adds its armor and, when its use handler has 80h, starts its power; taking it off ends both. Ending undoes the powers marked "lasts while worn" and the speed powers (09h–0Bh), so a worn object's speed counts only while worn. The Leather Boots (24h) every character starts in have power 0Ah with 80h: interval −2 while worn. Taking an object from the floor with the take key puts it in the first empty shelf; with none, message 7Fh and it stays on the floor. An arrow taken goes into the quiver when it fits.

Objects given by the game (conversations, powers) go into the first empty shelf, or are dropped at the player's feet.

**The hand.** Besides its slots the player has a hand, the object held on the pointer. Taking an object from the floor with the mouse puts it in the hand; clicking a slot swaps the hand with the slot; putting down puts the hand's object on the floor (rules.md, The player and the features).

**The mouse.** With the left button:

- Clicking the player's figure with an empty pointer takes the top object of the player's cell onto the pointer (pick up).
- Clicking the player's figure with an object on the pointer puts it down on the player's cell (put down), so an object meant for a keyhole, a socket or a plate goes in by standing on it and clicking the player.
- Clicking a slot (a shelf, a hand, a worn slot, a pack slot) swaps the pointer's object with the slot's.

With the right button, an object on the pointer goes to the first empty shelf; with an empty pointer, clicking the player's figure takes the top object underfoot straight to the first empty shelf. With no empty shelf the right button does nothing.

**Engine model.** The player has a hand and slots numbered as above. Commands:

- **Pick up** takes the top object of the player's cell, as the take key does: an arrow into a worn quiver with room, anything else into the first empty shelf, else message 7Fh (`no_room`) and it stays on the floor. **Pick up into the hand**, as clicking the player with an empty pointer does, takes it onto the pointer instead; with an object already in hand it is a plain pick up. Taking it off the floor lifts its weight off a plate.
- **Put down** puts the hand's object down on the player's cell.
- **Swap** exchanges the hand with a slot. Shelves and the two hands take anything. A pack slot takes an object whose size is at most the bag's largest, and never the bag itself; putting an object in the first free pack slot adds it to the bag's contents, and taking one out removes it, so the contents stay in order. The worn slots (20, 21, 24–27) are added with Fight, along with what each takes.

## Floor

Each cell holds a list of objects on its floor; at most eight are drawn. Objects dropped or thrown land on the floor. Objects left on a level stay there.

An object on the floor is drawn with its tiny image (about 8×6 pixels; the small 16×16 image is the inventory icon), centered in its cell's tile box: an image `w × h` has its top-left at `(16 − ⌊w/2⌋, 16 − ⌊h/2⌋)` from the box's corner (world.md). The objects of a cell are drawn in list order, each over the last, after the cell's floor and overlays and before anything standing.

**Magnified.** With magnifying on, objects on the floor are drawn with their small image instead, centered the same way, so they are easier to see and to pick up.

**Engine model.** Magnifying is a game setting the player turns on or off; it changes only how floor objects are drawn and where they can be clicked, never the simulation.

## Hands and actions

Each hand shows the actions of the object it holds (bare hands can strike). An action icon:

| Icon | Meaning |
| --- | --- |
| 100 and above | Attack, available when the weapon's class skill is at least `icon % 10`; `icon / 10` picks the picture, except as below |
| 110–119 | Two-handed attack |
| 120–129 | Fire a bow |
| 9, 10 | Use the object up |
| 11–20 | Use one charge |

Using an action makes the hand wait its delay before the next action; each player update counts the delays down. A weapon swing is an attack (combat.md). An action that needs a quantity is unavailable when none is left.

**Engine model.** An item has up to two **actions**, each a kind, damage dice and a delay:

| Kind | Icons | Does |
| --- | --- | --- |
| `attack` | 100 and above, but for these | Strikes with the hand; needs the weapon class's skill at least `icon % 10` |
| `attack_both` | 110–119 | Strikes with both hands; the same skill rule |
| `fire` | 120–129 | Fires an arrow from the quiver |
| `use_up` | 9, 10 | Uses the object up for its power |
| `use` | 11–20 | Uses a charge for its power |

The **delay** of action 0 is `(byte 8 >> 4) + 3` player updates, of action 1 `(byte 8 & 0Fh) + 3`; an item flagged two-handed (category 80h) delays both hands. A hand waits its delay before acting again, counted down on each of the player's updates. An attack adds `weight / 8` exertion (player.md). An empty hand strikes with bare hands, 2d2 damage and a delay of 2. Icon FFh is no action.

**Wearing.** An item's category gives the worn slot it goes in: 0 head, 1 body, 2 feet, 3 arms, 4 quiver, 5 neck; categories 6–12 go in a hand. The engine model calls this its `wear` slot. Weapons (40h) have a weapon class, `category − 9` for categories 9–12. Putting an item on, by swapping it into its own worn slot, adds its armor and starts its power; a worn slot refuses anything whose `wear` is not that slot. The quiver slot also takes arrows into a worn quiver.

**Value.** The importer writes the starting value rules above as the item's `durability` (weapons and body armor) or `charges`.

**Using up.** When the last use of an object goes, it becomes a spent object: bottles (category 8) become an Empty Bottle (template 7); other objects become the next template.

## Powers

An object's power starts when it is worn (if it lasts while worn) or used. Some powers check first that they can do something; when they cannot, they show a message and the object is not used up.

| Power | Effect | Lasts while worn |
| --- | --- | --- |
| 01h | Magic points +2d6 | |
| 02h | Magic points +4d6 | |
| 03h | Magic points +3000 | |
| 04h | Confusion +75 and 2d8 damage | |
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
| 14h | Half the time, change sex; otherwise a random attribute and its maximum ±1 | |
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

Timed objects in worn slots count down their value each status tick and end when it runs out.

## Using objects

An action of kind use up (icons 9, 10) or use (11–20) starts the object's power, whatever 80h says. A power that refuses (below) leaves the object as it was. Use up then replaces the object with its spent object; use takes one charge from its value, and at 0 it becomes its spent object the same way. Neither strikes, except icon 11h, which also strikes with the object.

**Engine model.** An item's `power` is its number and `power_worn` the 80h flag; `spent` is the item it becomes. The engine runs the powers in the table that need only the player: hit and magic points, cures, skills and attributes, statuses and timed effects, speed, and teleporting to a rune mark (a `mark` feature whose rune is `1`, `2` or `3` for 21h, 22h, 23h). Powers that cast spells, start conversations, change the ending or show the map wait for those milestones and report so. Each refusal is an engine message: `full_health` (3Eh), `full_magic` (42h), `not_poisoned` (40h), `not_poisoned_or_confused` (41h), `not_fatigued` (3Fh), `no_destination` (43h), `no_creatures` (88h), `already_have` (21h); others shown: `healed` (4Ah, `{damage}` the amount), `protected` (3Ch), `boost_extended` (33h), `poisoned` (59h), `slowed` (60h), `lost_weapon_skills` (64h), `lost_magic_skills` (67h).

## Throwing

The player throws the object in hand, or fires an arrow from a bow. A thrown object flies `(100 − weight) / 5 + 5` steps (weight above 100 counts as 100); an arrow `5 × STR` steps, at most 127; 8 units a step. It flies as an entity of kind 3 (entities.md).

**Damage.** Seven templates have a thrown look, their own flying frame:

| Template | 67h | 68h | 75h | 6Dh | 97h | 98h | 99h |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Frame | 27h | 28h | 29h | 2Ah | 2Dh | 2Ch | 2Bh |
| Flagged | | | | | yes | yes | yes |

They roll their action-1 damage, or 1 when a flagged one is thrown rather than fired. Anything else does `weight / 10 + 1`. The player's arrows add `(ACC − 10) / 2`, at least 1 in all. A thrower touching its target hits at once.

**Hitting.** A thrown object that meets an entity hurts it (arrows as damage kind 3, everything else 0) and lands. Only the player's throws land; a creature's vanish.

**Landing.** On a door it may open it (rules.md). An arrow breaks 1 time in 25. A bottle breaks into Broken Glass. If the **Decanter (90h)** breaks, every creature of record 2Eh is removed, conversation 2Eh starts, and the player gains 72903 experience.

Throwing is not possible in water of depth 3 or more.

**Engine model.** The player throws the object in a hand (that hand then waits 5 updates) or, with a `fire` action, the first arrow of the quiver worn in slot 25 (message 48h `no_quiver` without one, 49h `no_arrows` when it is empty; 96h `no_room_to_shoot` when something stands a step ahead). Throwing practices weapon class 3 by 1; firing does too, and shows the both-hands strike pose. In deep water of depth 3 or more throwing gives 70h (`cannot_throw`) and other actions 73h (`cannot_act`). A thing in flight (entities.md) carries the object; how far it flies and its damage are as above, the damage from the item's `missile` (its dice, and whether only firing rolls them). A landing bottle becomes the item's `breaks_into`.

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

Template 0 in a level becomes a random object rather than the item itself.

## Unknowns

- Whether the game also magnifies floor objects on its own, for example while the player has timed sight (player.md) or by interaction variable 98 (dialogue.md).
- Which bank the thrown-look frames index.
- The worn-look flags in byte 11 and the image tables; specified with character drawing.
- Category 6, and the hand delay formula beyond what is given.
