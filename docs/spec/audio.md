# Audio

The game's music and sound effects are songs for an OPL2 FM synthesizer (an AdLib card), stored in RESOURCE as type-3 records. The importer renders each song through an OPL2 emulator to an audio file with its loop point; the engine plays those. This file says how a song plays.

## Songs

Track `n` is RESOURCE `music base + n` (formats.md, INIT). Sound effect `n` is track `11h + n`.

- A sound effect plays only when no music track is playing and no higher-numbered effect is still sounding.
- Track 10 plays on a level-up and track 11 on a skill rising (player.md).
- A song started with a loop length repeats from its start; otherwise it plays once.

## Song descriptor (24 bytes)

| Offset | Type | Meaning |
| --- | --- | --- |
| 0 | u8 | Ticks per beat (240) |
| 1 | u8 | Beats per bar (4) |
| 2 | u32 | Loop length in song ticks: a whole number of bars |
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

Only the low byte of each parameter word counts. The 13 operator parameters, in order:

| # | Parameter | OPL register and bits |
| --- | --- | --- |
| 0 | Key scale level | 40h, bits 6–7 |
| 1 | Frequency multiplier | 20h, bits 0–3 |
| 2 | Feedback | C0h, bits 1–3 (modulator only) |
| 3 | Attack rate | 60h, bits 4–7 |
| 4 | Sustain level | 80h, bits 4–7 |
| 5 | Sustaining (EG type) | 20h, bit 5 |
| 6 | Decay rate | 60h, bits 0–3 |
| 7 | Release rate | 80h, bits 0–3 |
| 8 | Output level (attenuation) | 40h, bits 0–5 |
| 9 | Tremolo (AM) | 20h, bit 7 |
| 10 | Vibrato | 20h, bit 6 |
| 11 | Key scale rate | 20h, bit 4 |
| 12 | Connection: 0 FM, nonzero additive | C0h, bit 0 = 1 when the parameter is 0 |

Waveforms go to register E0h; waveform selection is enabled.

## Channels and voices

MIDI channel `c` drives OPL voice `c`. Without rhythm mode, channels 0–8 are the nine melodic voices. With it, channels 0–5 are melodic and 6–10 are the percussion instruments in OPL order: 6 bass drum, 7 snare, 8 tom-tom, 9 cymbal, 10 hi-hat. Events on channels without a voice are ignored.

In rhythm mode, a program change on channel 6 sets both operators of the bass drum; channels 7–10 set their single operator from the instrument's modulator.

## Timing

The song clock runs at `tempo × ticks_per_beat / 60` ticks per second (at least 19). The tempo command (`F0 7F 00 i f F7`) changes the tempo to `T × i + (T × f) / 128`, with `T` the descriptor's tempo.

The event stream is: a delay, then an event, repeated. A delay byte `d` waits `d` ticks; F8h adds 240 and another delay byte follows. Events at the same time have delays of 0 between them. FCh ends the song; a looping song restarts from its first event. A song with a loop length also restarts when its position reaches that length.

## Events

| Status | Data | Effect |
| --- | --- | --- |
| 8xh | note, velocity | Note off |
| 9xh | note, velocity | Note on; velocity 0 is note off. A velocity different from the channel's volume sets the channel volume to it first |
| Axh | volume | Set the channel volume (0..127) |
| Bxh | 2 bytes | Ignored |
| Cxh | instrument | Load instrument `n` of the song into the voice |
| Dxh | 1 byte | Ignored |
| Exh | low 7 bits, high 7 bits | Pitch bend, 0..3FFFh, centered at 2000h |
| F0h | … F7h | The tempo command, or ignored |
| FCh | | End |

A data byte below 80h in the status position repeats the last status (running status).

## Notes

A note `n` sounds as `n − 12` (at least 0). With the voice's bend semitones added and the result clamped to 0..95, the OPL block is `n / 12` and the F-number comes from the voice's frequency row at `n % 12`. Note on writes A0h and B0h with key-on (B0h bit 5); note off rewrites them without it.

For percussion, a note on sets the instrument's bit in register BDh (bass drum 10h, snare 08h, tom 04h, cymbal 02h, hi-hat 01h); the bass drum is tuned by voice 6, and the tom by voice 8 with voice 7 at `note + 7`. Note off clears the bit.

**Frequency rows.** 25 rows, row `r` tuned `4r` cents sharp. Row `r`:

```
v = ((10000 + 24r) × 52088) / 250000
v = (v × 16384 × 9) / 111875
for k in 0..11:
    fnum[k] = (v + 4) >> 3
    v = v × 106 / 100
```

(integer arithmetic). Row 0 is 343, 364, 385, 408, 433, 459, 486, 515, 546, 579, 614, 650.

**Pitch bend.** `steps = (bend − 2000h) × range × 25 / 2000h`, truncated. The voice plays `floor(steps / 25)` semitones off, on row `steps mod 25` (floored modulo). A bend retunes a sounding note at once. Percussion voices above 6 do not bend.

## Volume

Volume applies to each voice's carrier (the bass drum's carrier; the single operator of the other percussion). For an operator with output level `L` and volume `V` (0..127, the channel volume × master volume / 128):

```
level = 63 − ((63 − L) × V × 2 + 127) / 254
```

written with its key scale level to register 40h. Modulators keep their own level (volume 127).

The master volume is 0..127. A fade lowers it by 3 each song tick and stops the song at 30 or below.

## Unknowns

- Which tracks loop when the game starts them, and with what loop length when it differs from the descriptor's.
- Which track plays where (levels, menus, endings); specified with scenes.
