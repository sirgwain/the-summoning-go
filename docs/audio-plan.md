# Audio plan

Status: 2026-11-03. Phases 1–5 done; phase 6 but for the scenes still to be specified. How music and sound effects get from the game's files to
the speakers, and the order to build it in. Update this file as work lands, and fold what's
settled into [design.md](design.md#audio), [pack.md](pack.md) and the spec; delete it once audio
is done.

## Shape

```
RESOURCE type 3 ──► song reader ──► song player ──► OPL2 chip ──► resample ──► WAV + loop sidecar
   (game files)     (importer)      (importer)      internal/opl2   44.1 kHz     (content pack)
                                                                                     │
                                         engine/audio: decode, loop, mix, fade ◄─────┘
```

- **The importer renders, the engine plays.** Each song is rendered once, at import, to an
  ordinary audio file. The engine plays audio files and knows nothing about FM synthesis, so a mod
  can replace any track with a recording.
- **All Go, no cgo, no other languages.**
- **The song player is ours**, written from [spec/audio.md](spec/audio.md). It turns a song into
  timed OPL register writes. It is the only part that is about the game.
- **The chip is a port.** `internal/opl2` is a Go port of the OPL2 part of
  [opal](https://github.com/RealBitdancer/opal) (MIT, C11), an OPL3 core that matches Nuked-OPL3
  sample for sample at the chip's own rate. The chip is Yamaha hardware, not the game, so it is
  not specified in `docs/spec/`.

## Decisions

| Question | Choice | Why |
| --- | --- | --- |
| Where the chip runs | At import | Keeps the engine generic; tracks are plain files |
| Emulator | Port opal's OPL2 subset to Go | No Go OPL2 emulator with a license exists; opal is MIT and accurate |
| Pack audio format | WAV, mono, 16-bit, 44100 Hz; the engine also accepts Ogg | Ebitengine decodes both; Go has no Ogg Vorbis encoder |
| Pack size | About 65 MB for the 42 songs in WAV | Acceptable for now; a smaller format can come later (Open questions) |
| Loops | Sidecar JSON with intro and loop lengths in samples | Ebitengine's `audio.NewInfiniteLoopWithIntro` plays exactly that |
| Repository license | Stays MIT | opal's MIT notice goes into `internal/opl2/LICENSE` |

## Phases

Each phase lands with its tests and leaves `go vet ./...` and `go test ./...` passing.

### 1. Settle the spec

Done. [spec/audio.md](spec/audio.md) now gives the chip's state at game start and song start
(default instruments, operator resets, carriers fully attenuated until a volume is set), the
operator slots and the exact bits each instrument parameter writes, the exact clock rate,
restarting (no notes stopped, the first delay skipped), stopping (keys off, carriers attenuated),
both fades, and the register-write list as the song player's output. It also corrects two
things: the descriptor's offset-2 length is not the loop, the game gives loops in bars when it
starts a track; and a song that ends cuts its carriers rather than ringing on.

### 2. The chip: `internal/opl2`

Done. A port of opal cut down to the OPL2: one register array, nine two-operator channels,
waveforms 0–3, rhythm mode, mono output, no four-operator pairs, no timers. `New`, `Write`
(queued, one write landing per sample, so a key-off then key-on restarts a note), `Sample` at
`Rate`, `Silent` (every operator died away, nothing pending) and `Resampler`. With register 01h
bit 5 clear, waveform writes are ignored, as on the OPL2; C0h bits 4–7 are ignored. opal's MIT
notice is in `internal/opl2/LICENSE`.

Tests use hand-made register writes: frequency from F-number and block, instant attack and the
sustain level, release rates doubling, retriggering, the four waveforms, waveform select, each
drum bit, drums needing rhythm mode, the resampler, and a golden hash of a script with feedback,
vibrato, tremolo, additive voices and drums. The hash matches opal's own build playing the same
writes at the same samples.

### 3. The song player: `importers/summoning/music.go`

Done. `DecodeSong` reads the descriptor, instruments and event stream. `songPlayer` starts from
the driver's state at game start, starts a song (with an optional loop in bars) and plays it
step by step into a timeline of register writes and clock-rate changes. A malformed stream
(no FCh, a bad status byte) stops the song with an error for the import report.

Fixture tests build songs of a few bytes and check the writes the spec gives: the frequency
rows, the game-start and song-start writes, program changes on melodic and percussion
channels, note on and off, a second note on without key-off, volume, pitch bend, rhythm mode,
the tempo command and the slowest clock, delays with F8h, running status, ignored events,
voiceless channels, missing instruments, bad streams, and loops restarting at the loop point
and at FCh.

Checked against the game: every register write of all 42 songs, in value and order, matches
the game's driver, and every write during playback lands on the same microsecond. Checking
the timing corrected the spec: the timer's divisor comes from 1193180, but it counts at
1193182 Hz.

### 4. Rendering and the pack

- [x] Render a song: the song player's writes go to the chip at their times, each write taking
      80 µs as the game's driver does (spec, Register writes), then the result is resampled to
      44100 Hz (`importers/summoning/render.go`).
- [x] Songs that play once keep their release until the chip is silent or nearly flat, up to
      5 seconds.
- [x] Looping songs keep two passes: the first is the intro, the second the loop, so the loop
      starts with the first pass's notes still ringing.
- [x] Mono 16-bit WAV in `audio/music` and `audio/sfx`, with a record of `file`, `intro` and
      `loop` beside each ([pack.md](pack.md#audio); `engine/content/audio.go`).
- [x] `import --no-audio` skips rendering.
- [x] The dev pack has a generated tune with a loop and one effect.

### 5. The engine: `engine/audio`

- [x] The mixer (`engine/audio`): a music channel with both fades, effects layered over it (not
      the game's one sound at a time; [design.md](design.md#audio)), the music key's three
      settings, music and effect volumes. Tested with a fake backend.
- [x] The Ebitengine backend (`engine/audio/speaker`): decodes WAV and Ogg to the context's rate,
      loops with `NewInfiniteLoopWithIntro`.
- [x] Cues: the simulation names its sounds (`lever`, `door`, `unseal`, `teleport`, `level_up`,
      `weapon_skill_up`, `magic_skill_up`); `audio/cues.json` maps them to sounds.
- [x] `summoning play` plays them; F9 cycles the music key, `--mute` silences it.

### 6. Which track plays where

- [x] The effects of combat, the bow, used-up objects, missiles, thrown bottles, handling
      objects and gestures (spec, Sound effects).
- [x] The view effects' effects and tracks (their flipbooks' steps), and the new game's door
      opening (effect 17h a step, the part's `step_sounds`).
- [x] Footsteps, a new game's track 0, the theme track 8 of the creature of NPC record 0Dh while
      it is in sight, and the slow fade on a change of level or a restore; leaving play
      quick-fades.
- [x] Track 3 under the splash, in place of the opening picture (creation's `music`).
- [ ] The intro's and endings' tracks (spec **Unknowns**, with scenes): 1 (a new character's
      story, 48 bars), 2 (the intro's crossing), 4 (the closing, 8 bars), 5, 6 and 7 (end
      scenes). Levels have no music of their own in the game, and the engine keeps it so.
- [x] Which tracks loop when started, and with how many bars (spec, Songs).

## Checking

- Fixture tests check the song player against the spec, and the chip against known behavior.
- Imported audio is checked by ear against the game running in DOSBox. Mismatches become spec
  corrections and new fixture tests.
- No game audio, rendered or raw, goes into the repository or its tests.
- 2026-11-01: the imported music and effects checked by ear, and they sound right.

## Open questions

- **Smaller packs.** If 80 MB matters: QOA (a simple lossy format, about 5:1; we would write the
  encoder and decoder, a few hundred lines each), FLAC through a pure-Go library (about 2:1), or
  Ogg when `ffmpeg` is on the PATH.
- **Rendering at the chip's rate.** The pack could keep 49716 Hz and let Ebitengine resample when
  it decodes, instead of resampling in the importer.
- **Exact fades.** A gain ramp differs slightly from the game lowering each carrier's level. If it
  is audible, render fades too, or move the song player into the engine.
