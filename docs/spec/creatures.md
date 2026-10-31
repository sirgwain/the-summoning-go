# Creatures

Creatures (entity kind 2) are made from NPC records. So are people and animated objects such as flames and fountains. The level's entity stream places them by record id (formats.md).

## NPC record

Sixteen bytes:

| Byte | Meaning |
| --- | --- |
| 0 | Conversation script (INTERACT record); 0 for none |
| 1 | Portrait shown in conversation |
| 2 | Hit point bonus: adds 4 × this |
| 3 | Hit dice: d10 each |
| 4 | Damage resistance |
| 5 | Damage and to-hit bonus |
| 6 | Starting direction |
| 7 | Sprite bank (+ NPC base) |
| 8 | Frame count, or an animation pattern when 100 or more |
| 9 | Attack damage code (combat.md) |
| 10 | Accuracy |
| 11 | Update interval in ticks |
| 12 | Magic attack: a spell number, 0 for none |
| 13 | Weapon vulnerability: a weapon class that does extra damage |
| 14–15 | Behavior bits (u16) |

Behavior bits:

| Bit | Meaning |
| --- | --- |
| 0001h | The player has talked to it (set by the game) |
| 0002h | Dead (set by the game when one of its creatures dies) |
| 0004h | Never turns hostile |
| 0008h | Cleared by the game when it turns hostile; meaning otherwise unknown |
| 0010h | Leaves a corpse |
| 0040h | Poisonous: its hits poison 2 times in 8; worth 123 more experience |
| 0080h | Hurts on contact: 1 damage when it walks into the player |
| 0100h | Hostile from the start |
| 0200h | Resists pits and teleporters |
| 0400h | Hurt only by physical damage, and of weapons only by the Silver Sword |
| 0800h | Death blow: its every attack on the player kills outright, unless the player is warded (combat.md) |
| 1000h | Never turns hostile, and cannot be hurt |
| 2000h | Fire does a quarter damage |
| 4000h | Lightning does a quarter damage |
| 8000h | Poison does a quarter damage |

## Spawning

A creature gets `(byte 3)d10 + 4 × (byte 2)` hit points. With 0 it is a corpse from the start. Its speed is 8 units per step, or 2 for sprite bank 12h (NPC base + 12h); animated objects keep their pattern number as speed. Its direction and frame start at byte 6. A frame count of 2 marks a stationary caster, which starts facing direction 1. Hostile-from-start creatures begin hostile.

Objects listed with a creature in the level go into its inventory and drop when it dies.

## Animated objects

When byte 8 is 100 or more the entity never moves and only animates, each update:

| Pattern | Frames |
| --- | --- |
| 100 | Frame = direction (fixed) |
| 101 | Rests on frame 0, then flickers 1, 2, 1, 2 … for a while; cycles of 15–29 updates |
| 102 | As 101, with random frames 1–3 |
| 103 | Toggles between two frames every 2–9 updates |
| 104 | A random frame 0–3 each update |

## Walkers

Creatures move in four directions: 0..3 use direction table entries 0, 2, 4, 6. Direction 4 means resting.

The frames of a creature with `n` frames: walking direction `d` shows frames `d` and `d + 4` alternately; with more than 12 frames the cycle is `d, d + 4, d, d + 8`. Attacking shows `d + n − 4`. Frames wrap modulo `n`.

Each update a creature counts down its current walk. When the count runs out it picks a new one:

- **Hostile**, and the player is visible (not invisible and not transformed): head toward the player for one step. The heading picks the four-way direction that best closes the larger of the two axis gaps.
- **Otherwise**: a random direction 0..4 for 5–9 steps; direction 4 rests 10 steps longer.

Then, if it is not resting, it steps 8 units in its direction:

- **The step succeeds.** With a magic attack, 1 in 10 steps it casts it: a hostile creature turns to the player first (only if it is chasing this very step), others face a random direction; the spell's power is 5d5. Otherwise it advances its walk frame.
- **Blocked by the player, while hostile:** it attacks (combat.md), facing the player.
- **Blocked by the player, when it hurts on contact:** 1 damage, poisoning if the creature is poisonous.
- **Blocked otherwise:** it turns around and rests 5–9 steps (a hostile one picks a random direction instead).

After an attack or a cast it rests 10–14 steps.

A creature that is poisoned loses 1–2 hit points with a 1 in 40 chance each update.

**Stationary casters** (frame count 2) with a magic attack: every 15–24 updates they turn (frame and direction flip) and, 9 updates later, cast spell 1 at power 15d10.

**Corpses** (no hit points) do not update.

**Teleported creatures** spin through their four directions for a few updates, then arrive (rules.md).

## Special records

- **31h** takes damage only from the Alteration spell, which does exactly 3000 (spells.md).
- **79h** passes all damage it takes to the player.
- **5Bh** hurts on contact only while the player has no fire shield.

## Hostility

A creature turns hostile when the player attacks it, unless behavior bit 0004h or 1000h is set. A conversation can turn a creature and every creature of the same record hostile; creatures of records 23h and 2Bh then also get speed 16.

## Death

At 0 hit points a creature drops what it carries (in its cell). A stationary caster dies with a view effect (scenes.md). With bit 0010h it stays as a corpse showing its bank's last frame (pushable, no updates); otherwise it is removed. Its record gets bit 0002h. The player gains its experience (player.md):

```
xp = (byte 3)d10 + 4 × byte 2 + 13 × byte 5 + byte 10 + byte 4
     + 247 if byte 12 ≠ 0
     + 123 if behavior & 0040h
```

## Import

Each NPC record `n` becomes a thing template `thing.npc.n` of kind creature:

- **Sprite**: NPC base + byte 7, with the creature anchor (entities.md). **Speed** as under Spawning, 0 for animated objects. **Interval**: byte 11 in milliseconds (`ticks × 1000 / 18.2`, rounded). **Weight** 100.
- **Hit points**: `(byte 3)d10 + 4 × byte 2`.
- **Facing**: for walkers, direction table entry `2 × byte 6`; for stationary casters, 2.
- **Behaviors**: `hostile` (0100h), `never_hostile` (0004h or 1000h), `leaves_corpse` (0010h), `resists_teleport` (0200h), `invulnerable` (1000h).
- **Stats**: `resistance` (byte 4), `bonus` (byte 5), `accuracy` (byte 10).
- **Animations**, the template's own (the sprite is shared by records that start on different frames), named by engine direction `e = 2d` for a creature direction `d`:
  - walkers with `n` frames: `idle.e` frame `d`; `walk.e` frames `d, d + 4` (with more than 12 frames, `d, d + 4, d, d + 8`); `attack.e` frame `d + n − 4`, all modulo `n`; `dead` frame `n − 1`;
  - stationary casters: `idle.2` frame 0 and `idle.6` frame 1;
  - animated objects: `idle`, a still frame for now: byte 6 for pattern 100, 0 for the others.

Each frame lasts the interval.

## Engine model

A creature template carries what fighting needs: `attack` (dice, byte 9), `vulnerability` (byte 13 holds a weapon category, 9–12, or 0 for none: the weapon class `byte 13 − 9`; arrows do more against class 3, the bows' category 0Ch), `spell` (byte 12, used with Magic), `experience` (dice, the formula above with its hit dice rolled as dice), and the behaviors `poisonous` (0040h), `hurts_on_contact` (0080h), `death_blow` (0800h), `physical_only` (0400h), `resists_fire` (2000h), `resists_lightning` (4000h), `resists_poison` (8000h), `animated` (animated objects) and `stationary` (stationary casters); stats `resistance`, `bonus` and `accuracy` as before.

- **Walking.** Creature direction `d` is engine direction `2d`. A walker's step tests its ground point like the player's (entities.md, Moving) and refuses avoided floors, and pits and open trapdoors unless it resists teleporting; it then backs off from other things by 2 units per axis. A step that moves nothing is blocked; what it met is the blocker.
- **Waking.** A creature whose ground point is farther than 320 units (half the Manhattan distance) from the player does not move.
- **Counting.** The walk count and rests count the creature's updates (`interval_ms`).
- **Animated objects** (pattern 100 or more) and stationary casters do not walk; the casters' casting comes with Magic.
- **Showing.** A walking creature shows `walk.e`, stepped by its moves; attacking it shows `attack.e` for its next update; a corpse shows `dead`.
- **Dying.** Its items drop on its cell, the player gains its experience, and it becomes a corpse (`leaves_corpse`) or is removed. Corpses weigh on plates and can be pushed.
- **Poison.** A poisoned creature is a state of the thing, not of the template.

## Unknowns

- Behavior bit 0008h.
- The exact heading rule toward the player at shallow angles: within 4 units on `v` it goes along `u`; otherwise it compares the slope of the gap. An implementation may use any rule that heads along the larger gap; refine if play shows a difference.
