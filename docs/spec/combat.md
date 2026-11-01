# Combat

Combat is a d20 roll against 12: a score of 12 or more hits. Scores below 1 count as 1.

## Damage codes

A damage code byte `c` means `(c >> 4) + 1` dice of `(c & 0Fh) + 1` sides. Code 00h (1d1) means no damage. Bare hands do 2d2.

## Damage kinds

| Kind | Meaning |
| --- | --- |
| 0 | Physical |
| 1 | Fire |
| 2 | Lightning |
| 3 | Arrow |
| 4 | Poison |

## The player attacks

The player swings the object in a hand with one of its attack actions (items.md), striking whatever stands one step ahead. The player's swing pose shows frame `30h + 8 × swing + direction` for two updates. Only living creatures can be struck (message 5Ah otherwise); animated objects refuse with message 5Dh.

**Behind.** The player strikes from behind when the creature's four-way direction equals the player's direction halved.

**To-hit score:**

```
score = (AGI − creature accuracy) / 2
      + d20
      + (END − 10)               if END < 10
      + (ACC − 12) / 2
      + 2                        if status "accurate"
      + skill − 1                with a weapon, its class's skill level
      + 4                        under battle rage
      + 4                        from behind
```

A miss shows message 5Bh with the score.

**Damage**, rolled from the action's damage code:

```
d = roll + skill                 (skill of the weapon's class; none bare-handed)
d = d × 2                        from behind
d = d + d / 2                    under battle rage
d = d + 2                        if status "accurate"
d = d + d / 4                    if the weapon's category is the creature's vulnerability
d = d + STR − 10                 with a weapon (not bare hands)
d = the Silver Sword's action-1 roll, or 0 for any other weapon
                                 against creatures with behavior 0400h
d = max(0, d − creature resistance)
```

A hit shows message 5Ch with the score and damage. A hit for 0 does nothing more. Otherwise the weapon wears by the damage (except the Rusted Blade and WarMonger), the creature is hurt (physical; the Staff of the Serpent's (8Ah) hits poison it, unless it resists poison), and the weapon's skill gets practice: 1, or 2 when the blow kills a creature of record 50 or above that has no conversation (player.md). A blow that kills a creature whose record has no conversation shows message 8Bh with the creature's name, TEXT `AAh + record`.

The creature turns hostile (creatures.md).

## Creatures attack

A creature attacks the player when its walk is blocked by the player while it is hostile.

```
score = (creature accuracy − AGI) / 2
      + creature damage bonus
      + d20
      + 4      under battle rage
      − 4      if the player is invisible
      − 4      if the player has blow protection
      + 4      from behind (the creature's direction equals the player's direction halved)
```

A creature with the death blow (behavior 0800h) ends the game at once when it attacks, unless the player is warded against it.

On a hit, damage is its attack damage code rolled; less a quarter under blow protection, unless the d20 was a natural 20; doubled from behind; at least 0. The score and damage show as message 7Dh; a miss shows 7Eh. Damage above 0 hurts the player (physical), and poisons 2 times in 8 if the creature is poisonous.

Messages about combat are hidden while interaction variable "quiet combat" is set (dialogue.md).

## Hurting the player

For damage `d` of kind `k`:

1. Doubled under battle rage.
2. Status "resist quarter": `d − d/4`. Status "resist forty": `d − 4d/10`.
3. At least 1.
4. Fire (1) under a fire shield, or lightning (2) under a lightning shield: 1.
5. Non-physical damage (any kind but 0, arrows included) while the player has the "charges" status is absorbed by the charges of the object in the right hand whose power is 23h, else the left hand's: each charge used absorbs 1, until the damage or the charges run out, and message 3Ah shows how many it absorbed. Hit points then drop without the damage shown.
6. With body armor and armor class above 0: the armor wears by `min(d, AC)`, then `d − AC`, at least 1.
7. Hit points drop by `d`. The amount shows over the player for 6 updates (at most 99).
8. At 0 hit points or less with death protection: the protection is used up and hit points and magic points are restored to full.

At 0 hit points the player dies (scenes.md).

## Hurting a creature

For damage `d` of kind `k`, with `poison` set by poisoning attacks:

- Creature record 79h is not hurt: the player takes the damage instead, as if hurt by it. It only animates, but it can be struck.

- Animated objects, corpses and kinds other than creatures are not hurt. A conjured thing loses 1 of its strength instead.
- Behavior 1000h: immune. Record 31h: hurt only by exactly 3000.
- Behavior 0400h: non-physical damage does 0.
- Quarter damage: poison with 8000h, fire with 2000h, lightning with 4000h.
- Arrows (3) do `d + d/4` against creatures whose vulnerability is 0Ch (weapon class 3).
- The amount shows over the creature for 3 updates.
- Poison makes the creature poisoned unless it has 8000h (creatures.md).
- At 0 or less the creature dies (creatures.md).

## Wear

An object's wear value starts at its template's durability (items.md). Wearing by `n` takes `3n` when `n > 1`.

- **Weapons** (category flag set): durability 30000 never wears. When the wear exceeds what is left, the weapon breaks: message 8Ah with its name, and it is destroyed.
- **Armor** (category 1): wear is taken off the value, at most what is left. Each time the value falls past a multiple of 500 and the armor's effective AC drops, the player's armor class drops by 1.

## Special weapons

- **Rusted Blade (63h)** is charged by the damage it deals (after resistance, when above 0), one count the player keeps. A hit while the count is 0 starts conversation 45h; a hit with the count at 250 or more replaces the blade in its hand with a new **WarMonger (64h)**, starts conversation 46h and sets WarMonger's level, interaction variable 27, to 1. The damage is then added to the count.
- **WarMonger** adds the damage it deals to the count. When the count passes 1000 while its level is below 9, the level rises by 1, the count resets to 0, and conversation 46h starts. The level changes nothing in combat: the conversations read it.

## Engine model

Combat is a module (`engine/combat`) the simulation calls with plain numbers; the pack names the rule set, and the Summoning's is this file. The simulation finds who strikes whom and applies the results; the rule set rolls and computes.

- **Striking.** The player strikes the first creature whose footprint overlaps the player's moved one full step along its facing. A creature strikes the player when its own step is blocked by the player.
- **Behind.** In engine directions: the striker's direction, rounded down to an even one, equals the target's facing.
- **Dice.** An item action's damage and a creature's attack are dice in the pack. A damage code byte `c` imports as `((c >> 4) + 1)d((c & 0Fh) + 1)`, and 00h as no damage.
- **Rolls.** "d20" is 1 + rand(20). Every message above is an engine message (pack.md): `miss` (5Bh, the score), `hit` (5Ch, the score and the damage), `not_a_creature` (5Ah), `cannot_hurt` (5Dh), `creature_hits` (7Dh), `creature_misses` (7Eh), `weapon_breaks` (8Ah, the item's name).
- **Showing damage.** The amount a blow takes shows over the one it hurt: a creature for 3 of its updates, the player for 6 of its own, at most 99 shown. It is part of the game's picture, not of the debug overlay.
- **What waits.** Battle rage, invisibility, blow protection, the death-blow ward, resist quarter and forty, shields and charges come with statuses; until then those terms are 0. Poisoning, armor wear and weapon wear apply now.

## Engine model, special weapons and creatures

- An item's `poisons` makes its blows poison the creature hit.
- An item's `grows` is a counted weapon: the damage it deals goes into the player's weapon count; `first` names the thing template whose conversation the first hit starts, and at `at` the item is replaced by `into` with `grown` the conversation that follows and `counter` set to 1. An item's `levels` is the grown weapon: every `every` points while `counter` is below `max`, the counter rises, the count resets and `talk`'s conversation starts.
- A creature template with `mirrors` passes the damage it takes to the player, and can be struck though it only animates.
- A creature template's `name` is shown when the player's blow kills it, if it has no conversation; `kill_practice` is the practice that blow gives (2 for records 50 and above without a conversation).
- The player's weapon count is the player's state.

## Unknowns

- None besides those of the items and creatures.
