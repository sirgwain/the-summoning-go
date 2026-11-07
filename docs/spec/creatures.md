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
| 0008h | Can be talked to; cleared by the game when it turns hostile |
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

- **Hostile**, and the player is visible (not invisible and not transformed): head toward the player for one step. The heading picks the four-way direction that best closes the larger of the two axis gaps. In the engine, Tactics below refine this.
- **Otherwise**: a random direction 0..4 for 5–9 steps; direction 4 rests 10 steps longer.

Then, if it is not resting, it steps 8 units in its direction:

- **The step succeeds.** With a magic attack, 1 in 10 steps it casts it: a hostile creature turns to the player first and casts only if it is chasing the player this very step; others face a random direction; the spell's power is 5d5. Otherwise it advances its walk frame.
- **Blocked by the player, while hostile:** it attacks (combat.md), facing the player. In the engine, a creature with a wind-up starts it instead and strikes when it ends (combat.md, Wind-up).
- **Blocked by the player, when it hurts on contact:** 1 damage, poisoning 2 times in 8 if the creature is poisonous, as its blows do.
- **Blocked otherwise:** it turns around and rests 5–9 steps (a hostile one picks a random direction instead).

After an attack or a cast it rests 10–14 steps (in the engine, after an attack it recovers as Tactics, After the blow, says).

A creature that is poisoned loses 1–2 hit points with a 1 in 40 chance each update.

**Stationary casters** (frame count 2) with a magic attack: every 15–24 updates they turn (frame and direction flip) and, 9 updates later, cast spell 1 at power 15d10.

**Corpses** (no hit points) do not update.

**Teleported creatures** spin on the teleporter for 30 ticks, turning to the next of their four directions each tick, and are then moved; they act again from there (rules.md, In-level teleporters).

**Animated objects, engine model.** A template's `animate` says how its frame changes on each of its updates: `flicker` (a count of 15–29 is set when it runs out and counts down; while above 15 the frame alternates between the flicker frames by the count's parity, or is a random one of them with `random`; otherwise the rest frame), `toggle` (between two frames, every 2–9 updates), `random` (a random frame of the list each update) and `cycle` (the next frame of the list each update). The importer gives patterns 101–104 `flicker` over frames 0, 1, 2; `flicker` random over 0, 1, 2, 3; `toggle` over 0, 1; and `random` over 0–3.

## Special records

- **31h** takes damage only from the Alteration spell, which does exactly 3000 (spells.md).
- Every record has a name, TEXT `AAh + record`.
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

- **Sprite**: NPC base + byte 7, with the creature anchor (entities.md). **Speed** as under Spawning, 0 for animated objects; records 23h and 2Bh have a hostile speed of 16 (Hostility). **Interval**: byte 11 in milliseconds (`ticks × 1000 / 18.2`, rounded). **Weight** 100.
- **Hit points**: `(byte 3)d10 + 4 × byte 2`.
- **Facing**: for walkers, direction table entry `2 × byte 6`; for stationary casters, 2.
- **Behaviors**: `hostile` (0100h), `never_hostile` (0004h or 1000h), `leaves_corpse` (0010h), `resists_teleport` (0200h), `invulnerable` (1000h), `talks` (0008h).
- **Conversation**: `conversation.n` for script `n` of byte 0 when the record talks; **talk portrait** `portrait.p` for byte 1, also only when it talks (the others hold 0 in both bytes). Portraits past F's rows are the people banks `people base + p`, imported as portraits with only an image.
- **Spell**: the spell of byte 12 (spells.md), when it is one.
- **Stats**: `resistance` (byte 4), `bonus` (byte 5), `accuracy` (byte 10).
- **Animations**, the template's own (the sprite is shared by records that start on different frames), named by engine direction `e = 2d` for a creature direction `d`:
  - walkers with `n` frames: `idle.e` frame `d`; `walk.e` frames `d, d + 4` (with more than 12 frames, `d, d + 4, d, d + 8`); `attack.e` frame `d + n − 4`, all modulo `n`; `dead` the bank's last frame (Death), whatever `n` is;
  - stationary casters: `idle.2` frame 0 and `idle.6` frame 1;
  - animated objects: `idle`, a still frame for now: byte 6 for pattern 100, 0 for the others.

Each frame lasts the interval.

## Engine model

A creature template carries what fighting needs: `attack` (dice, byte 9), `vulnerability` (byte 13 holds a weapon category, 9–12, or 0 for none: the weapon class `byte 13 − 9`; arrows do more against class 3, the bows' category 0Ch), `spell` (byte 12, used with Magic), `experience` (dice, the formula above with its hit dice rolled as dice), and the behaviors `poisonous` (0040h), `hurts_on_contact` (0080h), `death_blow` (0800h), `physical_only` (0400h), `resists_fire` (2000h), `resists_lightning` (4000h), `resists_poison` (8000h), `animated` (animated objects) and `stationary` (stationary casters); stats `resistance`, `bonus` and `accuracy` as before.

- **Walking.** Creature direction `d` is engine direction `2d`. A walker's step tests its ground point like the player's (entities.md, Moving) and refuses avoided floors, and pits and open trapdoors unless it resists teleporting; it then backs off from other things by 2 units per axis. A step that moves nothing is blocked; what it met is the blocker.
- **Waking.** A creature whose ground point is farther than 320 units (half the Manhattan distance) from the player does not move.
- **Counting.** The walk count and rests count the creature's updates (`interval_ms`).
- **Animated objects** (pattern 100 or more) and stationary casters do not walk.
- **Casting.** A creature's `spell` names the spell it casts (Walkers; Stationary casters). Casting turns it as the rules above say and casts at the creature's facing with the power given; casting needs no gesture, magic points or memorization.
- **Talking.** `conversation` names the conversation record of its NPC record, `talk_portrait` the portrait shown (byte 1), and the behavior `talks` (0008h) that it can be talked to; turning hostile takes `talks` away from that creature.
- **Showing.** A walking creature shows `walk.e`, stepped by its moves; attacking it shows `attack.e` for its next update; a corpse shows `dead`.
- **Dying.** Its items drop on its cell, the player gains its experience, and it becomes a corpse (`leaves_corpse`) or is removed. Corpses weigh on plates and can be pushed.
- **Poison.** A poisoned creature is a state of the thing, not of the template.

## Tactics

Tactics are the engine's own addition, like the wind-up (combat.md). The game's hostile creatures all head straight for the player and rest after every blow; in the engine a creature finds its way around walls, and its template may say how it fights around the player. A template with no tactics fights as the game's creatures do, but for noticing the player and finding its way.

**Noticing.** A hostile creature chases the player only once it has noticed the player; since it finds its way around walls, it would otherwise come for the player from anywhere within waking distance. Each of its updates, before anything else it does:

- While it has not noticed the player, it notices the player when the player is visible (not invisible, not transformed), the creature's cell is visible to the player (world.md, Engine model: in the same open space, not beyond a wall or closed door), and the larger of the two axis gaps between their ground points is at most 6 cells (192 units). Noticing ends its current walk, so it picks a new one at once.
- Once it has noticed the player, it keeps chasing while the player is visible and its cell visible to the player, at any distance. Otherwise it counts its updates, and after 40 it loses the player and has to notice the player again.
- A blow, a thrown or shot object or a spell of the player's that meets it makes it notice the player at once, wherever it is, as it turns hostile (combat.md).

Until it notices the player, and after it loses the player, a hostile creature walks, turns when blocked, and casts as one that is not hostile (Walkers), and does not shoot; blocked by the player, it still attacks. Turning hostile in a conversation does not notice the player. Whether it has noticed the player, and how many updates it has left before losing the player, are saved with it.

These rules replace the hostile heading of Walkers. Each applies when a hostile walker that has noticed the player picks a new walk while the player is visible (not invisible, not transformed); the walk lasts one step, as the heading does. Facing follows the step, unless a rule says the creature faces the player, in which case it steps backward or sideways with its face toward the player (the four-way direction of the heading).

**Finding the way.** A creature heading for a goal cell (the player's, unless said otherwise) counts the way to it in cells, four-way, over the cells its ground point may stand in (Engine model, Walking: not blocked, not an avoided floor, and no pit or open trapdoor unless it resists teleporting) within 24 cells of its own on each axis. The player's cell counts as one it may stand in, unless a rule says otherwise; other things are not counted. Its step is the heading's (Walkers) toward the goal, the player's ground point or else the middle of the goal cell, when the neighboring cell that way is one nearer the goal, or when it stands in the goal cell. Otherwise it steps toward a neighboring cell one nearer the goal, trying the direction of its last step, then the direction that closes the other axis gap, then the rest in direction order (0..3), so that it keeps to one way around. When no way reaches the goal within that window, it takes the heading.

**Keeping distance.** A template's `keep_distance`, `n` cells, makes a creature fight from afar. The gap is the larger of the two axis gaps between its ground point and the player's, in units; a cell is 32.

- **Too near** (gap below `32n`): it backs away, facing the player. It steps directly away along the axis of the larger gap; if the neighboring cell that way is not one it may stand in, along the other axis away from the player (either way when that gap is 0, the positive first); if neither, it heads for the player (Finding the way) and fights as any creature does.
- **Too far** (gap above `32(n + 2)`): it heads for the player (Finding the way).
- **Within reach:** when the smaller axis gap is below its step, it is lined up: it rests this step facing the player and, 1 time in 4, casts its spell at the player, then rests 10–14 steps as after any cast. Otherwise it steps along the smaller gap's axis toward lining up, facing the player.

**After the blow.** A template's `after_blow` says what a creature does after its blow, whether the blow struck or met nothing (combat.md, Wind-up), in place of resting 10–14 steps:

| `after_blow` | What it does |
| --- | --- |
| `rest`, or none | Rests 10–14 steps |
| `back_off` | For its next 2–3 steps it backs away from the player as Keeping distance's too near, facing the player; a step it cannot back away it rests. Then it heads for the player again |
| `flank` | For up to its next 20 steps it heads for the cell behind the player, the cell beside the player's against the player's four-way facing, counting the player's cell as one it cannot stand in (but for its own). In that cell it steps onto the line of the player's facing, until its gap across that line is below its step; then it stops flanking and heads for the player, and its blow strikes from behind (combat.md). With no way to that cell, it stops flanking and rests 10–14 steps; so does a creature whose blow was already from behind |

What remains of a creature's recovery is saved with it. A creature blocked by the player while flanking or backing off attacks as any hostile creature does, and recovers anew from that blow.

**Import.** The importer gives each walker's template tactics from its NPC record:

- With a magic attack (byte 12 a spell): `keep_distance` 4 and `after_blow` `back_off`.
- Otherwise, poisonous (0040h) with an attack (byte 9 above 0): `after_blow` `back_off`, to strike and step away.
- Otherwise, updating every 3 ticks or quicker (byte 11 of 3 or less) with an attack: `after_blow` `flank`.
- Otherwise none.

## Unknowns

- The exact heading rule toward the player at shallow angles: within 4 units on `v` it goes along `u`; otherwise it compares the slope of the gap. An implementation may use any rule that heads along the larger gap; refine if play shows a difference.
