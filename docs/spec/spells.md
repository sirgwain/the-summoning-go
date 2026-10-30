# Spells

## Schools and numbers

Four schools: 0 Wizardry, 1 Sorcery, 2 Enchantment, 3 Healing. Spell number = `school × 10 + slot`. Each spell has a magic point cost, power dice `(dice, sides, bonus sides)` and a gesture, all read from INIT (formats.md).

## Gestures, learning and memorizing

A gesture is a string of up to nine icons, the letters A–L. The player **knows** some icons. A new character knows the icons of its school's first spell. Reading a scroll shows its spell's gesture and teaches those icons, and so does a conversation's show-gesture action (dialogue.md). Only known icons can be entered.

The engine keeps a **spellbook**: the spells the player has learned. A new character's spellbook is empty.

- **Learning.** In the spell-learning UI the player enters a gesture of known icons, at most eight. A gesture that matches a spell of any school puts that spell in the spellbook (if not already there), then memorizes it once and selects it. Reading the scroll first is not required: knowing the icons is enough to discover a spell. A gesture that matches nothing costs 1 magic point per icon.
- **Memorizing from the spellbook.** Choosing a learned spell memorizes it once more and selects it, exactly as entering its gesture would, with no gesture to enter.

Either way the game's limit applies. A school holds at most `(TAL − 10) / 2 + skill / 2` memorizations, clamped to 1..9. Memorizing past that first forgets one of the school's memorized spells at random.

In the game itself there is no spellbook: every memorization means entering the gesture again. The spellbook changes only how a memorization is asked for, not what it costs or allows.

## Casting

Casting a school's selected spell needs: the spell memorized, no gesture already playing, the spell's condition (below), the magic points for its cost, and water below depth 3 (unless levitating). The cost is paid and one memorization used at once; the gesture then plays on the gesture panel, one icon per player update, and the spell takes effect after the last.

**Power.** Unless the caster supplies a power:

```
extra = max(0, (POW − 10) / 2 + school skill)
power = (dice)d(sides) + (extra)d(bonus sides)
```

Every cast by the player practices its school by 1; some spells practice more (below).

**Conditions:**

| Spell | Needs |
| --- | --- |
| 12, 18 | A walkable cell (17h–23h) ahead |
| 15 | A rune mark 22h on the level |
| 17 | A lingering spell near the player |
| 19 | The two staff halves in the hands |
| 25 | A creature within 128 |
| 30 | An Empty Bottle (template 7) in hand |
| 31 | Poisoned |
| 32 | Fatigue |
| 33 | Missing hit points |

## Missiles

A missile spell flies as an entity of kind 3 from the caster, 8 units a step, until its range runs out or it hits. On hitting an entity:

| Type | Effect |
| --- | --- |
| 0, 4 | Fire damage (kind 1) of the power |
| 5 | Lightning damage (kind 2) |
| 3 | Poison damage (kind 4) |
| 0Bh | Opens doors with bit 20h (rules.md), starts boulders |
| 0Eh | Moves its target to a random seen floor cell |
| 1Ah | 3000 damage to creature record 31h; nothing to others |

Each hit by the player's missile practices the player's school 0 (damage types), 1 (0Bh, 0Eh) or 2 (1Ah). Types 0 and 5 leave a trail and, when they end, a lingering cloud that goes off when the player comes near. Bouncing missiles turn back off walls.

## Effects

| No. | Name | Effect |
| --- | --- | --- |
| 0 | Flame Arrow | Missile type 0, with trail |
| 1 | Four-way Fireball | Four type-4 missiles: the caster turns a quarter between them (creatures) or casts along the four diagonals of its facing (player) |
| 2 | Bounce Fireball | A bouncing type-4 missile |
| 3 | Poison Gas | Missile type 3 |
| 4 | Fireball | Missile type 4 |
| 5 | Lightning | Missile type 5, with trail |
| 6 | Battle Rage | STR +2, END +2, AGI −2 for `power` status ticks (message 5Eh); if already raging, message 92h and the time extends |
| 9 | (none) | Fires a new arrow from the caster |
| 10 | Sight | Timed sight for `power` ticks |
| 11 | Kano | Missile type 0Bh |
| 12 | Magic Wall | Practices Sorcery 5 more. Conjures a wall entity (kind 8) half a cell ahead, its look by Sorcery skill / 3 (at most 2), holding `power` strength. Message 6Bh when there is no room |
| 13 | Swiftness | Haste for `power` ticks (message 5Fh) |
| 14 | Zap Away | Missile type 0Eh |
| 15 | Teleport | The player ripples to a rune mark 22h on the level |
| 16 | Invisibility | Practices Sorcery 3 more. Invisible for `power` ticks |
| 17 | Dispel | Sets off the lingering spells within 96 of the player |
| 18 | Gateway | Practices Sorcery 5 more. Opens a portal (kind 9) on the wall ahead, replacing any other; message 6Fh when there is no room. Walking into it removes it, stores the player's level and cell, and takes the player to level 30 at (60, 26) |
| 19 | Mending | Joins the staff halves D8h and D9h into DAh |
| 20 | Freeze | Practices Enchantment 1 more. Delays every other entity's update by `power × 10` ticks (message 39h) |
| 21 | Lightning Shield | Practices Enchantment 1 more. Lightning shield for `power` ticks (message 38h) |
| 22 | Fire Shield | Practices Enchantment 1 more. Fire shield for `power` ticks (message 37h) |
| 23 | Weapon Skill | Each weapon class +1 for `power` ticks (message 62h); if already active, message 63h and the time extends |
| 24 | Magic Skill | Each school +1 for `power` ticks (message 65h); if active, message 66h |
| 25 | Shape Change | The player takes the look of the nearest creature within 128 (transformed); message 89h if none. Taking the look of NPC bank +4Dh sets an interaction variable |
| 26 | Alteration | Missile type 1Ah |
| 30 | Liquify | Transmutes the held object (items.md) |
| 31 | Cure Poison | Practices Healing 3 more. Cures poison (message 7Ah) |
| 32 | Restore | Recovers `power` fatigue |
| 33 | Heal | Practices Healing 5 more. Heals 3000 |

Spells 7, 8 and 27–29 do nothing.

**Ripple teleport** (spell 15, powers 11h, 1Ah, 26h): the player moves to a cell of the level whose terrain is the given rune mark, with a ripple effect.

## Spell panel

The spell panel shows, for each school, its selected memorized spell and how many times it is memorized. Selecting cycles through the school's memorized spells. The spellbook lists the learned spells by school, each with its gesture, cost and memorized count.

## Unknowns

- Whether anything besides scrolls, conversations and the starting school teaches icons.
- The lingering cloud's lifetime and the distance at which it goes off.
- Missile range and sprite per type.
