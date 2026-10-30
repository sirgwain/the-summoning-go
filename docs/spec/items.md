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
| 10 | Image within the object banks |
| 11 | Flags: 80h never chosen as a random object; other bits select the worn look |
| 12 | Use handler: 80h set means it has a power, the low bits the power number |
| 13 | Category: bits 0–5 the category; 40h a weapon; 80h two-handed |
| 14 | Scroll: the spell a scroll teaches |

**Categories.** 1 body armor ("shirts"), 4 quiver, 7 arrows, 8 bottles (break into Broken Glass, template 0Ch); categories 9..12 are weapon classes 0..3 (a weapon's class is its category − 9). Weapons have the 40h flag.

**Template 0** in a level is a random object: any template that is not never-random and is a weapon or of category 6.

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
| 22 | Left hand |
| 23 | Right hand |
| 24 | Feet |
| 25 | Quiver: an empty slot takes only a quiver; a filled one takes what fits in the quiver |
| 26 | Body |
| 27 | Neck |
| 28+ | Pack: the contents of the bag held in the right hand, one slot per capacity |

Pack slots exist only up to the right-hand bag's capacity and take what fits in it. Putting an object in a worn slot (20–27) starts its power if it lasts while worn, and adds its armor; taking it off ends both. Picking an object up puts it in the first empty shelf; with none, message 7Fh. An arrow picked up goes into the quiver when it fits.

Objects given by the game (conversations, powers) go into the first empty shelf, or are dropped at the player's feet.

## Floor

Each cell holds a list of objects on its floor; at most eight are drawn. Objects dropped or thrown land on the floor. Objects left on a level stay there.

## Hands and actions

Each hand shows the actions of the object it holds (bare hands can strike). An action icon:

| Icon | Meaning |
| --- | --- |
| 100–109 | Attack, available when the weapon's class skill is at least `icon % 10` |
| 110 | Two-handed attack |
| 120 | Fire a bow |
| 9, 10 | Use the object up |
| 11–20 | Use one charge |

Using an action makes the hand wait its delay before the next action; each player update counts the delays down. A weapon swing is an attack (combat.md). An action that needs a quantity is unavailable when none is left.

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

## Special objects

- **Transmuting** (spells.md) turns the held object into: Amethyst → Teiwaz, Emerald → Isa, Ruby → Algit, Diamond → Snake Oil, NightShade Leftovers → Isa, anything else → Jera.
- **Mending** joins the two staff halves D8h and D9h held in the hands into DAh.
- **Gems** C0h–C5h are gems 1–6 for the gem socket (rules.md).

## Unknowns

- Which bank the thrown-look frames index.
- The worn-look flags in byte 11 and the image tables; specified with character drawing.
- Category 6, and the hand delay formula beyond what is given.
