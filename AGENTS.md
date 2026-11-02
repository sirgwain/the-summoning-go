# AGENTS.md

A Go engine, over Ebitengine, for isometric real-time RPGs in the style of _The Summoning_, with an
importer that converts that game's data files into the engine's content format. `docs/design.md`
describes the engine; `docs/spec/` describes the game and the files.

If `AGENTS.private.md` exists at the repository root, read it and follow it too. It is not part of the
repository.

## Rules

1. **The spec is the source of truth.** Code implements `docs/spec/`. When a change needs behavior
   the spec does not describe, update the spec first, then the code.
2. **The spec stands alone.** It describes behavior as rules, tables and math in its own words, and
   cites no other implementation of the game. Someone with only the spec and the game's data files
   must be able to build from it.
3. No original game data in the repository: no art, text, levels, music or extracted files. Tests use
   small hand-made fixtures. Tests may also use the game's own files when given them through the 
   environment (SUMMONING_DATA, SUMMONING_SAVES), and skip without them, so go test ./... passes without the game.
4. **`engine/sim` does not import Ebitengine**, so the simulation, the importer and their tests run
   without a display.
5. Format with `gofmt`, and keep `go vet ./...` and `go test ./...` passing.
