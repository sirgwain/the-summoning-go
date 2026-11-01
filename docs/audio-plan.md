# Audio plan

Status: 2026-11-01. Phases 1–3 done. How music and sound effects get from the game's files to
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

- [ ] Render a song: run the song player and the chip together, one song tick of samples at a
      time, then resample to 44100 Hz.
- [ ] Songs that play once: render to FCh and the stop that follows, then keep going until
      every voice is silent (with an upper limit): stopping attenuates the carriers, but an
      additive voice's modulator rings on through its release.
- [ ] Looping songs: render two passes and store both. The loop region is the second pass, because
      it starts with the first pass's notes still ringing, so the loop is seamless. The intro is
      the first pass. A pass ends at the restart: the loop point (from the track's bars) or FCh,
      whichever comes first.
- [ ] Write mono 16-bit WAV (a small writer in the importer, no dependency).
- [ ] Write `audio/music/<id>.wav` or `audio/sfx/<id>.wav`, plus `<id>.json` with
      `{ "rate": 44100, "intro": samples, "loop": samples }` (`loop` 0 for songs that play once).
      Ids: `music.<n>` for track `n`, `sfx.<n>` for sound effect `n`.
- [ ] Document the audio files and sidecar in [pack.md](pack.md), with the content structs and
      validation in `engine/content`.
- [ ] Update [design.md](design.md#audio): WAV or Ogg in packs; the chip in `internal/opl2`;
      `internal/` in the repository layout.
- [ ] `summoning import` renders audio by default; `--no-audio` skips it for quick imports during
      development.
- [ ] The dev pack gets one short generated tone with a loop, so the engine's tests have audio
      without the game's files.

### 5. The engine: `engine/audio`

Ebitengine is allowed here; `engine/sim` only emits events.

- [ ] One `audio.Context` at 44100 Hz. Decode WAV and Ogg; loop with
      `NewInfiniteLoopWithIntro`.
- [ ] Music: one track at a time; starting one stops the last; a fade lowers the gain to silence
      over the time the spec's fade takes (3 of 127 per song tick, stopping at 30), then stops.
- [ ] Effects: the game's rule, as a mixer setting. An effect plays only when no music is playing
      and no higher-numbered effect is still sounding.
- [ ] Wire the simulation's events: level-up plays track 10, a skill rising plays track 11
      ([spec/player.md](spec/player.md)), and `sound` actions from the rules.
- [ ] Music and effect volume settings, and a mute key.
- [ ] Tests without a display: the mixer's choices (what plays, what is refused, what fades) as
      plain logic behind an interface, with the Ebitengine player as a thin layer.

### 6. Which track plays where

- [ ] Track assignment for levels, menus, character creation, death and the endings (spec
      **Unknowns**, with scenes). Fill [spec/audio.md](spec/audio.md) and
      [spec/scenes.md](spec/scenes.md), then the importer writes the assignments into the pack
      (level and scene fields).
- [ ] Which tracks loop when started, and with how many bars; the importer needs this to render
      loops, since songs don't carry it.

## Checking

- Fixture tests check the song player against the spec, and the chip against known behavior.
- Imported audio is checked by ear against the game running in DOSBox. Mismatches become spec
  corrections and new fixture tests.
- No game audio, rendered or raw, goes into the repository or its tests.

## Open questions

- **Smaller packs.** If 65 MB matters: QOA (a simple lossy format, about 5:1; we would write the
  encoder and decoder, a few hundred lines each), FLAC through a pure-Go library (about 2:1), or
  Ogg when `ffmpeg` is on the PATH.
- **Rendering at the chip's rate.** The pack could keep 49716 Hz and let Ebitengine resample when
  it decodes, instead of resampling in the importer.
- **Exact fades.** A gain ramp differs slightly from the game lowering each carrier's level. If it
  is audible, render fades too, or move the song player into the engine.
