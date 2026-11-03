# Scenes plan

Status: 2026-11-03. The new character's story plays in short (spec/scenes.md, The story, in
short): an engine summary, then part 7, Rowena's farewell, with words of the engine's own; the
story's track plays. Scenes have gained a tall size, pages of words, speech panels and steps by
weapon class. The intro and parts 1–6 are deferred; the end scenes are next. This plan covers writing the spec for the frontend's
cinematics: the intro, the new character's story and the end scenes. It also says how they fit
the scene and flipbook model of [spec/scenes.md](spec/scenes.md). Update this file as each phase
lands, fold what's settled into the spec, and delete it once the frontend's scenes are specified.
The code that follows (importer, pack format, presentation) gets its own plan when the spec is done.

## What there is

The game plays three groups of cinematics outside play, on the full 320 × 200 screen. All their art
is in RESOURCE, from the frontend base in INIT. Their captions are in the game's program instead.

- **The intro**, played when the game starts. Figures cross the screen, then come pictures
  between palette fades, the title growing in from small to full size, and a war story of four
  captions. It ends on the opening picture. Every step can be skipped.
- **A new character's story**, played when Create Character is chosen. It has seven parts in
  order:
  1. the fortress story;
  2. character creation (descendant, name, attributes);
  3. the training: two fighters spar, the weapon master and the trees walk in, and the player
     chooses a weapon class and a school here;
  4. the training's end;
  5. the battle, with Khail;
  6. Jairus's death, a talking portrait;
  7. Rowena's farewell, which shows and names the weapon chosen.

  Captions are word-wrapped in boxes or centered, a page at a time, and fade in and out. A
  scrolling border and palette effects (rotation, lightning flicker, fades toward a color) run
  on their own clocks. Skipping a part leaves out the parts after it. Track 1 plays through,
  looping.
- **The end scenes**, played after play ends in an ending: ending E's, ending F's (the world's
  end, with Chesschantra), and ending G's (Borel's return and farewell). Each opens in a frame
  with its colors shifted by 70h. It calls up figures with a glow, shows framed pictures,
  sparkles and captions under a cycling palette, and has figures wander to random places and
  walk off. The music fades at the end.

The new game's door and the ending's closing are already specified (spec/scenes.md, New game and
Endings) and fit the same model.

## Shape

```
RESOURCE (frontend base) ──► importer plays each animation ──► flipbooks ─┐
                                                                         ├──► scene records ──► frontend plays them
spec: steps, layering, palette, captions, choices ───────────────────────┘
```

- **A scene stays a list of steps** the presentation plays, as now. It gains steps to:
  - play a flipbook at a place, waiting for it or alongside the steps after it;
  - change the palette: fade, fade toward a color, rotate a range, flicker, shift;
  - start or fade a track;
  - choose, run by the frontend's own screens (the weapon class, the school);
  - branch, on the character or on the ending.
- **The flipbook is the shared unit.** The view effects and sparkles are already flipbooks. The
  frontend's moving parts become flipbooks too: crossing figures, the title growing, sparring,
  wandering figures, twinkles, talking portraits. A view effect is then a scene of one flipbook
  in a framed box.
- **Layers.** A scene step draws on a layer: a picture under, flipbooks over it, captions on top.
  Several flipbooks can run at once.
- **Palette shifts are baked at import.** The end scenes' colors shifted by 70h are drawn into
  their frames, so a scene needs no palette field for them; only cycling the palette by the clock
  would need one, and it too can be baked into frames if the flipbooks stay small enough.
- **Randomness is baked at import** with a fixed seed, as for the view effects. That covers the
  wandering targets, the twinkles and the portrait's blinking.
- **Captions are slots.** The spec gives each caption's place, box, alignment, timing and fades,
  and an id. The words are the engine's own, written for each slot, as the story in short's are;
  nothing is read from the game's program.

## Phases

Each phase is a spec change with its own commit. Check each against the game running in DOSBox,
and turn any mismatch into a correction.

### 1. The scene model

Write the scene's engine model in spec/scenes.md:

- the step kinds above and their fields;
- layers and running in parallel;
- the palette steps;
- caption slots;
- the frontend's skip rules (which key or button skips, and whether a skip ends one step, one
  part or the rest).

Then restate the new game's door, the closing and the view effects in it, to prove it covers them.

### 2. The intro

Specify each step: the crossing figures, the pictures and their fades, the title's sizes and how
it grows in, the war story's four caption slots, the opening picture, the palette's fades and
rotations, and the music. Specify where each step can be skipped.

### 3. The new character's story

Specify the seven parts in order:

- the scrolling border and the palette effects' clocks;
- the training's sparring, walking and trees, and where the weapon and school choices come in;
- the battle;
- Jairus's talking portrait (head, mouth and eyes);
- Rowena's farewell, with its weapon picture by class (a branch);
- the caption slots and paging;
- which skips leave out the rest.

This is the largest phase and the only one that is interactive.

### 4. The end scenes

Specify the three scenes:

- the frame, the color shift and the palette cycle;
- the glow that calls figures up;
- the framed pictures and their sparkles;
- the wandering and walking off;
- the caption slots;
- which ending plays which scene.

Settle "which closing texts belong to which ending" from spec/scenes.md's Unknowns here.

### 5. Fold in

Move the tracks each scene plays into spec/audio.md, closing the audio plan's last open item for
the frontend. Clear the spec's Unknowns these phases answer. Update design-notes.md, and write the
code plan.

## Open questions

- **Where captions come from.** Settled: the importer writes words of the engine's own for each
  slot, as for the story in short, and does not read the game's program (CODE.1).
- **Frames with no wait.** As for the view effects, their speed was the machine's. The view effects
  use 1/60 s, and the frontend should match unless it looks wrong next to the game.
- **The character in the story.** Parts after the choices show the chosen weapon. Whether anything
  else depends on the character (portrait, sex, name) is to be checked in phase 3.
- **Baking against playing.** Wandering figures walk to random targets with a walking cycle.
  Baking one run into a flipbook is simplest. If a full-screen flipbook gets too large, the scene
  could instead carry a "wander" step with its rules.
