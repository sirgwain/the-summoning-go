# Spells

## Schools and numbers

Four schools: 0 Wizardry, 1 Sorcery, 2 Enchantment, 3 Healing. Spell number = `school × 10 + slot`. Each spell has a magic point cost, power dice `(dice, sides, bonus sides)` and a gesture, all read from INIT (formats.md).

A spell exists when its gesture is one to nine of the letters A–L. Slots 7 and 8, 27–29 and 34–39 have none (or begin with X) and do nothing. Slot 9 of Wizardry has gesture `XX`: it cannot be learned, but creatures cast it (creatures.md).

## Gestures, learning and memorizing

A gesture is a string of up to nine icons, the letters A–L. The player **knows** some icons. A new character knows the icons of its school's first spell. Reading a scroll shows its picture and teaches icons (below), and so does a conversation's show-gesture action (dialogue.md). Only known icons can be entered.

The engine keeps a **spellbook**: the spells the player has learned. A new character's spellbook is empty.

The player memorizes on the memorization screen, which F10 opens and closes (the engine also opens and closes it with R, and closes it with Esc). The screen pauses the game: while it is open nothing else moves and no time passes. **Engine model.** The spell panel holds the world while open (items.md, Holding the world).

- **Learning.** The player enters a gesture of known icons, at most eight, with the letters A–L or by clicking the icons; the same icon cannot be entered twice in a row, and Backspace takes the last one off. Entering it then checks it. A gesture that matches a spell of any school puts that spell in the spellbook (if not already there), then memorizes it once and selects it. Reading the scroll first is not required: knowing the icons is enough to discover a spell. A gesture that matches nothing costs 1 magic point per icon.
- **Memorizing from the spellbook.** Choosing a learned spell memorizes it once more and selects it, exactly as entering its gesture would, with no gesture to enter.
- **The spellbook list.** The screen lists the spellbook by school, in school order, each school headed by its name and how many memorizations it holds of its limit (below). Within a school the spells keep the order they were learned. Each spell's row shows its icon, its name, gesture and cost, how many times it is memorized, and a Memorize button that memorizes it as above. The list shows eight rows at a time and scrolls with the mouse wheel over it, Page Up and Page Down, or a click on its scroll bar above or below the thumb. Every gesture that matches a spell, however it was found, adds the spell to the list.

Either way the game's limit applies. A school holds at most `(TAL − 10) / 2 + skill / 2` memorizations, clamped to 1..9. Memorizing when the school already holds that many first forgets one of its memorizations at random (each memorization equally likely), with messages 68h and 69h.

In the game itself there is no spellbook: every memorization means entering the gesture again. The spellbook changes only how a memorization is asked for, not what it costs or allows.

**Scrolls.** Byte 14 of a scroll's template is its scroll number `s`, 1..25. Its picture, which shows the gesture, is RESOURCE `scroll base + s − 1` (INIT), a strip of one-pixel frames (formats.md, Strips). Reading it teaches the icons of a spell gesture by this table (letters past L teach nothing, so a scroll may teach part of a gesture):

| Scroll | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 | 22 | 23 | 24 | 25 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Spell | 38 | 39 | none | 3 | 0 | 11 | 10 | 4 | 1 | 2 | 5 | 30 | 31 | 12 | 13 | 14 | 15 | 16 | 23 | 22 | 21 | 20 | 32 | 36 | 37 |

Spells 36–39 are the halves of the gestures of 25 and 26 (their X stands for the other half). The game reads a scroll when it shows the scroll's details: whenever an object comes onto the pointer, taken from an inventory slot or picked up from the floor by clicking the player (items.md, The mouse), the information panel shows it, and a scroll shows its picture there. Taking it to a shelf with the take key does not.

## Casting

Casting a school's selected spell needs, in order: water below depth 3 unless levitating (else message 71h), the spell memorized (else 46h), no gesture already playing, the spell's condition (below), and the magic points for its cost (else 29h). The cost is paid and one memorization used at once; the gesture then plays on the gesture panel, one icon per player update, and the spell takes effect after the last. When the school has no memorization of the spell left, its next memorized spell is selected.

**Power.** Unless the caster supplies a power:

```
extra = max(0, (POW − 10) / 2 + school skill)
power = (dice)d(sides) + (extra)d(bonus sides)
```

Every cast by the player practices its school by 1; some spells practice more (below). Creatures cast with a power of their own (creatures.md); so do objects (items.md) and level procedures, which cast at 6d10 from a cell (rules.md).

**Conditions**, and the message when one fails:

| Spell | Needs | Else |
| --- | --- | --- |
| 12, 18 | A walkable cell (17h–23h) ahead: half a cell for 12, a cell for 18 | 6Ah, 6Fh |
| 15 | A rune mark 22h on the level | 43h |
| 17 | A spell in flight within 96 of the player | 61h |
| 19 | The two staff halves in the hands | 6Dh |
| 25 | A creature within 128 | 89h |
| 30 | An Empty Bottle (template 7) in a hand | 6Ch |
| 31 | Poisoned | 40h |
| 32 | Fatigue | 3Fh |
| 33 | Missing hit points | 3Eh |

## Missiles

A missile spell flies as an entity of kind 3. The player's starts ahead of the player along its facing, a creature's ahead of the creature, 8 units a step (half along directions 1 and 5), and it flies up to 255 steps. A caster already touching something ahead (within 4 units) hits it at once. A step that meets an entity, or a wall, closed door or the edge of the map, ends the flight there; running out of range ends it without a hit. An ended missile shows its last four frames (1–4) and vanishes, except types 0 and 5, which end as a cloud of their own (resources 57 and 58) that shows those frames the same way.

On hitting an entity (the player included):

| Type | Effect | Practice |
| --- | --- | --- |
| 0, 4 | Fire damage (kind 1) of the power | School 0 by 1, by 2 when the power is 50 or more |
| 5 | Lightning damage (kind 2) | As 0 |
| 3 | Poison damage (kind 4) | As 0 |
| 0Bh | Starts a boulder rolling the way the missile flew | School 1 by 5 |
| 0Eh | Moves its target, a boulder, trap, conjured thing or walking creature (not the player, nor a creature that only animates or stands and casts), to a random floor cell the player has seen | School 1 by 3 |
| 1Ah | 3000 damage to creature record 31h; nothing to others | School 2 by 20 |

Only the player's missiles practice, and a creature hit by one turns hostile. Type 0Bh meeting a door whose bit 20h is set (rules.md) opens it (school 1 by 5); a door without it gives message 34h (school 1 by 1); a wall gives message 93h. Bouncing missiles turn back off walls (a diagonal one turning a quarter when it meets a wall along one axis) and fly on; meeting an entity ends them.

| Type | Sprite | Frame in flight |
| --- | --- | --- |
| 0 | held-item base + 2Eh (INIT) | The direction it flies |
| 5 | held-item base + 2Fh | The direction it flies |
| 3 | 65 | 0 |
| 4 | 41 | 0 |
| 0Bh | 73 | 0 |
| 0Eh | 61 | 0 |
| 1Ah | 66 | 0 |

## Effects

| No. | Name | Effect |
| --- | --- | --- |
| 0 | Flame Arrow | Missile type 0 |
| 1 | Four-way Fireball | Four type-4 missiles: the caster turns a quarter between them (creatures) or casts along the four diagonals of its facing (player) |
| 2 | Bounce Fireball | A bouncing type-4 missile |
| 3 | Poison Gas | Missile type 3 |
| 4 | Fireball | Missile type 4 |
| 5 | Lightning | Missile type 5 |
| 6 | Battle Rage | STR +2, END +2, AGI −2 for `power` status ticks (message 5Eh); if already raging, message 92h and the time extends |
| 9 | (none) | Fires a new Arrow (template 99h) from the caster |
| 10 | Sight | Timed sight for `power` ticks |
| 11 | Kano | Missile type 0Bh |
| 12 | Magic Wall | Practices Sorcery 5 more. Conjures a wall entity (kind 8, resource 59) half a cell ahead, its frame Sorcery skill / 3 (at most 2). It blocks and can be pushed, updates every 18 ticks and lasts `power` updates; each blow on it takes one. Message 6Bh when there is no room |
| 13 | Swiftness | Haste for `power` ticks (message 5Fh) |
| 14 | Zap Away | Missile type 0Eh |
| 15 | Teleport | The player ripples to a rune mark 22h on the level |
| 16 | Invisibility | Practices Sorcery 3 more. Invisible for `power` ticks |
| 17 | Dispel | The spells in flight within 96 of the player end where they are |
| 18 | Gateway | Practices Sorcery 5 more. Opens a portal (kind 9, resource 60, frames 0, 1, 2, 1 every 9 ticks) a cell ahead, replacing any other; message 6Fh when there is no room. Walking into it removes it, stores the player's level and cell, and takes the player to level 30 at (60, 26) |
| 19 | Mending | Joins the staff halves D8h and D9h in the hands into DAh, in the right hand (Sorcery 20 more); sets interaction variable 32 to 1 and starts conversation 79h |
| 20 | Freeze | Practices Enchantment 1 more. Delays every other entity's update by `power × 10` ticks (message 39h) |
| 21 | Lightning Shield | Practices Enchantment 1 more. Lightning shield for `power` ticks (message 38h) |
| 22 | Fire Shield | Practices Enchantment 1 more. Fire shield for `power` ticks (message 37h) |
| 23 | Weapon Skill | Each weapon class +1 for `power` ticks (message 62h); if already active, message 63h and the time extends |
| 24 | Magic Skill | Each school +1 for `power` ticks (message 65h); if active, message 66h |
| 25 | Shape Change | The player takes the look of the nearest creature within 128 (transformed); message 89h if none. Taking the look of NPC bank +4Dh sets interaction variable 14. The transformed figure is that creature's sprite alone, frame `p >> 1` for the player's facing `p` (player.md), modulo its frames: no held objects and no armor. It lasts until the player's next step that moves it |
| 26 | Alteration | Missile type 1Ah |
| 30 | Liquify | Healing 1 more. With an Empty Bottle in one hand, the other hand's object fills it: Amethyst (ABh) → 80h, Emerald (ACh) → 82h, Ruby (ADh) → 83h, Diamond (AEh) → DDh, NightShade Leftovers (0Bh) → 82h, using the object up; anything else, or nothing, makes a Jera potion (81h) whose value is the power, and stays. The bottle's hand holds the result; message 6Eh names it |
| 31 | Cure Poison | Practices Healing 3 more. Cures poison (message 7Ah) |
| 32 | Restore | Recovers `power` fatigue |
| 33 | Heal | Practices Healing 5 more. Heals 3000 |

Spells 7, 8 and 27–29 do nothing.

**Ripple teleport** (spell 15, powers 11h, 1Ah, 26h): the player moves to a cell of the level whose terrain is the given rune mark, with a ripple effect.

## Spell panel

The spell panel shows the four schools in boxes two by two, row by row: Wizardry top left, Sorcery top right, Enchantment bottom left, Healing bottom right. Each box shows its school's selected memorized spell's icon (RESOURCE 45h + school, frame = slot) and how many times it is memorized (at most 9 shown). Selecting cycles to the school's next memorized spell; with none other, message 46h (none memorized) or 47h names the school. The gesture panel shows the gesture icons as they play: RESOURCE 54, frame `icon + 1` (frame 0 is the empty panel).

## Engine model

A spell is a record: its school, cost, power (dice and the bonus die's sides), gesture (letters A–L), icon, an optional condition, extra practice, and a list of **effects**, actions from the same registry as procedures:

| Effect | Does |
| --- | --- |
| `missile` | Flies a missile as above: what it does on hitting (`hurts` a damage kind, `opens` doors and starts boulders, `scatters`, or `bane` a creature template for a fixed damage), its sprite and whether its frame follows its direction, the `cloud` it ends as, `bounces`, and `around` for the four-way cast |
| `fire_item` | Fires a new object of an item from the caster, as a bow fires |
| `timed` | A timed effect for `power` status ticks: `battle_rage`, `sight`, `haste`, `invisibility`, `fire_shield`, `lightning_shield`, `weapon_skills`, `magic_skills`, with their messages |
| `conjure` | Puts a new thing of a template half a cell ahead, lasting `power` of its updates |
| `portal` | Puts a new portal thing a cell ahead, replacing the level's other portal, that takes the player to a level and cell |
| `teleport_to_mark` | Ripples the player to a mark of a rune |
| `dispel` | Ends spells in flight within a distance of the player |
| `join` | Joins two items held in the hands into a third |
| `freeze` | Delays every other thing by `power ×` ticks |
| `shape_change` | Takes the look of the nearest creature within a distance |
| `transmute` | Fills an item held in a hand from the other hand's, by a table |
| `cure_poison`, `restore`, `heal` | Cure poison, recover `power` fatigue, heal an amount |
| `set_counter`, `converse` | Set an interaction variable; start a conversation (Mending) |

A spell's condition (`needs`) is one of: `floor_ahead` (a distance), `mark` (a rune), `spell_near` (a distance), `holding` (items, each in a hand), `creature_near` (a distance), `poisoned`, `fatigued`, `hurt`; each refusal is an engine message, which the spell may name (`else`) where the condition's own does not fit (Magic Wall, Gateway, Liquify). The player keeps the icons it knows, its spellbook, each spell's memorizations, each school's selected spell and the gesture playing.

Commands: **learn** (enter a gesture of icons), **memorize** (a spell in the spellbook), **cast** (a school's selected spell), **select** (a school's next memorized spell), and **read** (an object's details, which teaches a scroll's icons). An object coming into the hand is read; the presentation shows a scroll's picture in a small window above the hotbar until the player next acts.

## Unknowns

- Whether anything besides scrolls, conversations and the starting school teaches icons.
- Exactly how long each gesture icon shows: the game holds some icons over two player updates, by a table of which icon may follow which.
- What a magic wall weighs on a plate; the importer gives it a creature's weight.
