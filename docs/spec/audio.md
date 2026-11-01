# Audio

The game's music and sound effects are songs for an OPL2 FM synthesizer (an AdLib card), stored in RESOURCE as type-3 records. The importer renders each song through an OPL2 emulator to an audio file with its loop point; the engine plays those. This file says how a song plays: which OPL registers it writes, with which values, and when.

## Songs

Track `n` is RESOURCE `music base + n` (formats.md, INIT). Sound effect `n` is track `11h + n`.

- A sound effect plays only when no music track is playing and no higher-numbered effect is still sounding. Otherwise it starts like any song, stopping a lower- or equal-numbered effect. A track or effect that has ended no longer counts as playing.
- Track 10 plays on a level-up, track 11 on a weapon skill rising and track 12 on a school of magic rising (player.md). A rise that brings a level-up plays track 11 or 12, then track 10 over it.
- **The music key** (F9) cycles three settings, showing message A6h, A7h or A8h for the new one: music and effects (the start), effects only, and silence. Leaving the first stops the song playing. In "effects only" the game starts no music track, so effects are never held back by one; in silence nothing plays. Coming back to the first starts a looping track again from its start if one was stopped by the key.
- One song plays at a time; starting a song stops the one playing (Stopping).
- A song plays once unless the game starts it with a loop of `B` bars. Then its loop point is `B × beats per bar × ticks per beat − 1` song ticks, and it restarts from its start at the loop point or at its end, whichever comes first (Timing).

The game starts these tracks with a loop; every other track plays once. The front end's music base is the same RESOURCE number as the game's.

| Track | Bars |
|---|---|
| 0 | 40 |
| 1 | 48 |
| 4 | 8 |
| 5 | 24 |
| 6 | 16 |
| 7 | 8 |
| 8 | 12 |
| 15 | 8 |

**Sound effects.** The effects of the world's mechanisms:

| Effect | When |
|---|---|
| 9 | A lever is pulled |
| 16h | A sealed door opens; the player is teleported |
| 17h | A moving door steps while between open and closed |

**Footsteps.** The player keeps a step count, 0..5. It is 1 when the game starts, and is set back to 1 when the player turns to its standing frame after 8 idle updates, when an action's frames end, and on every step while levitating. Otherwise each walk step that moves the player adds 1 to it, wrapping 5 to 0, and a count of 1 or 4 plays effect 0, or effect 1 on levels 21h–23h. So walking from standing sounds on the third step and every third step after.

**Music in play.**

- A new game starts track 0 (40 bars) as play begins. Restoring a saved game starts none.
- Once per pass over the creatures, the game checks whether a creature of NPC record 0Dh is on screen. When one first is, it starts track 8 (12 bars); when none is any longer, it stops the music track playing with the slow fade.
- A change of level, and restoring a game, stop the music track playing with the slow fade.

## Song descriptor (24 bytes)

| Offset | Type | Meaning |
| --- | --- | --- |
| 0 | u8 | Ticks per beat (240) |
| 1 | u8 | Beats per bar (4 or 5) |
| 2 | u32 | Length in song ticks, as written; usually the song's length. Not read when playing |
| 6 | u32 | Event stream length in bytes |
| 10 | u32 | Number of events, counting the end and each delay extension |
| 14 | u8 | Rhythm mode: nonzero turns on the OPL percussion section |
| 15 | u8 | Pitch bend range in semitones (clamped 1..12) |
| 16 | u16 | Tempo in beats per minute |
| 18 | u8[5] | PC speaker and Tandy channel choices (not used) |
| 23 | u8 | Instrument count `P` |

The payload is `P` instruments of 58 bytes, then the event stream.

## Instruments (58 bytes)

| Offset | Content |
| --- | --- |
| 0 | u16: a General MIDI program for MIDI devices (not used) |
| 2 | Modulator: 13 u16 parameters |
| 28 | Carrier: 13 u16 parameters |
| 54 | u16: modulator waveform |
| 56 | u16: carrier waveform |

Only the low byte of each parameter word counts; some instruments hold out-of-range values, and the rules below say which bits survive. The 13 operator parameters, in order:

| # | Parameter | Register | Value written |
| --- | --- | --- | --- |
| 0 | Key scale level | 40h | `p << 6` (bits 6–7) |
| 1 | Frequency multiplier | 20h | `p & 0Fh` |
| 2 | Feedback | C0h | `p << 1`, unmasked |
| 3 | Attack rate | 60h | `p << 4` (bits 4–7) |
| 4 | Sustain level | 80h | `p << 4` (bits 4–7) |
| 5 | Sustaining (EG type) | 20h | 20h when nonzero |
| 6 | Decay rate | 60h | `p & 0Fh` |
| 7 | Release rate | 80h | `p & 0Fh` |
| 8 | Output level (attenuation) | 40h | from `p & 3Fh` and the volume (Volume) |
| 9 | Tremolo (AM) | 20h | 80h when nonzero |
| 10 | Vibrato | 20h | 40h when nonzero |
| 11 | Key scale rate | 20h | 10h when nonzero |
| 12 | Connection: 0 FM, nonzero additive | C0h | 1 when the parameter is 0 |

Each register is the OR of its parts, kept to a byte. Waveforms go to register E0h as `w & 3`.

C0h belongs to a voice, and only an operator in a voice's modulator slot writes it: `(feedback << 1) | connection`, to C0h + the voice. Bits 4–7 can be set by a large feedback value; the OPL2 ignores them.

## Operators and voices

The OPL2 has 18 operator slots. Slot `s` has register offset `s / 6 × 8 + s % 6` (added to 20h, 40h, 60h, 80h and E0h). Melodic voice `v` (0..8) has its modulator in slot `v / 3 × 6 + v % 3` and its carrier three slots later.

MIDI channel `c` drives OPL voice `c`. Without rhythm mode, channels 0–8 are the nine melodic voices. With it, channels 0–5 are melodic and 6–10 are the percussion instruments:

| Channel | Instrument | Slots | Its volume goes to | Key bit in BDh |
| --- | --- | --- | --- | --- |
| 6 | Bass drum | 12 (modulator), 15 (carrier) | slot 15 | 10h |
| 7 | Snare drum | 16 | slot 16 | 08h |
| 8 | Tom-tom | 14 | slot 14 | 04h |
| 9 | Cymbal | 17 | slot 17 | 02h |
| 10 | Hi-hat | 13 | slot 13 | 01h |

Events on channels without a voice (9 and up without rhythm mode, 11 and up with it) are read and ignored.

A program change loads instrument `n` of the song (an `n` at or past `P` is ignored). On a melodic voice it sets the modulator slot from the instrument's modulator and the carrier slot from its carrier. On channel 6 it sets slot 12 from the modulator and slot 15 from the carrier. On channels 7–10 it sets the instrument's one slot from the modulator. Slots 12, 13 and 14 are modulator slots of voices 6, 7 and 8, so the bass drum, hi-hat and tom-tom also write C0h of those voices.

Setting an operator writes, in order: BDh (Rhythm register), 08h (0), 40h, C0h (modulator slots only), 60h, 80h, 20h, E0h. A program change does not touch A0h or B0h: a sounding note keeps sounding with the new sound.

## Default instruments

Before a song's first program change on a voice, the voice has a default instrument. Its 13 parameters, in the order of the parameter table, all with waveform 0:

| Default | Parameters |
| --- | --- |
| Melodic modulator | 1, 1, 3, 15, 5, 0, 1, 3, 15, 0, 0, 0, 1 |
| Melodic carrier | 0, 1, 1, 15, 7, 0, 2, 4, 0, 0, 0, 1, 0 |
| Bass drum modulator | 0, 0, 0, 10, 4, 0, 8, 12, 11, 0, 0, 0, 1 |
| Bass drum carrier | 0, 0, 0, 13, 4, 0, 6, 15, 0, 0, 0, 0, 1 |
| Snare drum | 0, 12, 0, 15, 11, 0, 8, 5, 0, 0, 0, 0, 0 |
| Tom-tom | 0, 4, 0, 15, 11, 0, 7, 5, 0, 0, 0, 0, 0 |
| Cymbal | 0, 1, 0, 15, 11, 0, 5, 5, 0, 0, 0, 0, 0 |
| Hi-hat | 0, 1, 0, 15, 11, 0, 7, 5, 0, 0, 0, 0, 0 |

**Resetting the operators** sets every slot to the melodic default for its position (carrier slots 3–5, 9–11 and 15–17 the melodic carrier, the others the melodic modulator), in slot order. In rhythm mode it then sets slot 12 (bass drum modulator), 15 (bass drum carrier), 16 (snare), 14 (tom-tom), 17 (cymbal) and 13 (hi-hat).

## The chip before a song

**The Rhythm register (BDh)** is 20h when rhythm mode is on, OR the keys of the percussion instruments sounding. Deep tremolo and deep vibrato (bits 7, 6) stay off.

**When the game starts** (once), the driver sets every operator's volume to 127 (Volume), turns rhythm mode off (no keys, reset the operators, write BDh), writes BDh again and 08h = 0, writes A0h and B0h of voices 0–8 as 0, sets the pitch bend range to 1 and every voice's bend to the center, then writes E0h = 0 for every slot and 01h = 20h (waveform selection on). Last, it applies the master volume 127 (Volume), with every channel's volume 0.

**When a song starts**, after stopping any song playing:

1. Every channel's volume is set to 0 (stored only).
2. Rhythm mode is set from the descriptor, clearing the keys. When it is on, A0h and B0h of voices 6, 7 and 8 are written 0, then voice 8 is tuned to note 24 and voice 7 to note 31, keyed off (Notes). In either mode the operators are then reset, and BDh is written.
3. The pitch bend range comes from the descriptor (clamped 1..12).
4. The song clock is set from the tempo (Timing).
5. The master volume is applied: every voice's volume is rewritten (Volume), which, with the channel volumes at 0, puts every carrier at full attenuation until the song sets a volume.

A voice's bend is not reset when a song starts.

## Timing

The song clock is the PC's timer. With `r = ticks_per_beat × tempo / 60` (integer division, at least 19), the timer's divisor is `floor(1193180 / r)`; the timer counts at 1193182 Hz, so the clock runs at `1193182 / divisor` ticks per second. The tempo command (`F0 7F 00 i f F7`) sets the tempo to `T × i + (T × f) / 128` (16-bit, integer), with `T` the descriptor's tempo; it does not change `T`.

The event stream is: a delay, then an event, repeated. A delay byte `d` waits `d` ticks; F8h waits 240 ticks plus the delay byte that follows, which can itself be F8h. Events at the same time have delays of 0 between them. FCh ends the song.

The player waits a number of clock ticks, then takes a step. The song's first delay is one byte, without F8h, and its wait is at least 1 tick. Each step:

1. A fading song takes its fade step (Fades).
2. The position advances by the delay just waited.
3. If the song has a loop point and the position has reached it, the song restarts.
4. Events play until a nonzero delay, which is the wait before the next step.

A tempo change sets the clock rate for the ticks after the step.

**Restarting** moves to the first event, with the position 0. It neither stops notes nor resets instruments, volumes or bends, so notes from the end of the song ring into its start. The first delay byte is skipped: the first events play in the tick of the restart. A restart at the loop point drops the events that were due at that tick. FCh restarts a song with a loop point, and stops one without (Stopping).

## Events

| Status | Data | Effect |
| --- | --- | --- |
| 8xh | note, velocity | Note off |
| 9xh | note, velocity | Note on; velocity 0 is note off. A velocity different from the channel's volume sets the channel volume to it first |
| Axh | volume | Set the channel volume (0..127) |
| Bxh | 2 bytes | Ignored |
| Cxh | instrument | Program change (Operators and voices) |
| Dxh | 1 byte | Ignored |
| Exh | low 7 bits, high 7 bits | Pitch bend, 0..3FFFh, centered at 2000h |
| F0h | … F7h | The tempo command, or ignored |
| FCh | | End |

A data byte below 80h in the status position repeats the last status (running status). Any other status byte is an error.

## Notes

A note `n` sounds as `n − 12` (at least 0). The voice remembers that note and whether it is keyed on. To tune a voice to a note: add the voice's bend semitones, clamp to 0..95; the block is `note / 12` and the F-number comes from the voice's frequency row at `note % 12`. A0h + voice gets the F-number's low byte; B0h + voice gets `key << 5 | block << 2 | (fnum >> 8 & 3)`.

**Melodic note on** tunes the voice keyed on. A note on to a voice already keyed on rewrites its pitch without keying off, so the chip does not restart its envelope. **Note off** tunes the voice to its remembered note, keyed off, whatever note the event names.

**Percussion note on** sets the instrument's bit and writes BDh. Before that, the bass drum tunes voice 6 to the note, keyed off; the tom-tom tunes voice 8 to the note and voice 7 to note + 7, keyed off. The snare, cymbal and hi-hat use no frequency of their own beyond what voices 7 and 8 hold. **Percussion note off** clears the bit and writes BDh.

**Frequency rows.** 25 rows, row `r` tuned `4r` cents sharp. Row `r`:

```
v = ((10000 + 24r) × 52088) / 250000
v = (v × 16384 × 9) / 111875
for k in 0..11:
    fnum[k] = (v + 4) >> 3
    v = v × 106 / 100
```

(integer arithmetic). Row 0 is 343, 364, 385, 408, 433, 459, 486, 515, 546, 579, 614, 650.

**Pitch bend.** `steps = (bend − 2000h) × range × 25 / 2000h`, truncated. The voice plays `floor(steps / 25)` semitones off, on row `steps mod 25` (floored modulo). A centered bend is row 0, no semitones. A bend retunes the voice at once, to its remembered note and key state. Percussion voices above 6 do not bend.

## Volume

Each operator slot has a volume, 0..127, kept across songs. Writing an operator's 40h:

```
level = 63 − ((63 − L) × V × 2 + 127) / 254
40h   = level | KSL << 6
```

with `L` its output level parameter (`& 3Fh`) and `V` its volume.

Setting a channel's volume `C` stores `C` and gives the channel's volume slot (a melodic voice's carrier; the percussion table) the volume `min(127, C × master / 128)`, then writes that slot's 40h at once, also on a sounding note. Modulators keep the volume they had, 127 from the start of the game unless a percussion instrument used the slot.

**The master volume** is 0..127. Changing it rewrites the volume of channels 0–8, or 0–10 in rhythm mode, from their stored volumes.

## Stopping

Stopping a song (at FCh without a loop point, when another starts, or when a fade ends):

1. Note off on channels 0–10, in order. Without rhythm mode, channels 9 and 10 still write A0h, B0h + 9 and + 10 (registers the OPL2 does not have).
2. The master volume is applied as 0, putting every carrier at full attenuation; the master volume is then restored without writing.

Notes released by the key-off are silenced by their carriers' attenuation; an additive voice's modulator rings on through its release.

## Fades

Both fades end by stopping the song, and leave the master volume as it was before the fade for the next song.

- **Slow fade** (stopping a track while play goes on): at each step, before the position advances, a master volume at 30 or below stops the song; otherwise it drops by 3 (at least 0). Steps come with the song's events, so the fade's speed follows them.
- **Quick fade** (before a scene or a change of level): the master volume drops by 3 every 1/18.2 second, from its value down to 0 or the last step above it, while the song keeps playing; then the song stops.

## Register writes

The song player's output, the thing to test, is the list of register writes in order, each with the song tick of the clock it happens on, and the clock rate whenever it changes. The chip's starting state is the one "When the game starts" leaves, followed by the song's start.

**Write timing.** The driver waits after each write: after the register number for a short delay, and after the value for seven times as long. It sizes both by counting the passes of a loop over one period of the timer when the game starts, so they take about the same time on any machine: about 10 µs and 70 µs. Each write takes about 80 µs, so the chip hears the writes of a step spread out: a program change (16 writes) over about 1.3 ms, a song's start over about 12 ms.

## Unknowns

- Which track plays where (levels, menus, endings); specified with scenes.
