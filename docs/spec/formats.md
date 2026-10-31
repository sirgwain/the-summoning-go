# Data formats

The game's data files, as the importer reads them. File names are matched case-insensitively. All integers are little-endian.

## Indexed container (RESOURCE, TEXT, LEVELS)

```text
u32 count
u32 offsets[count]      // absolute file offsets; offsets[0] == 4 + 4*count
byte data[]
```

Record `i` runs from `offsets[i]` to `offsets[i+1]`, the last to the end of the file. There is no terminal offset. Reject offsets outside the file or out of order.

## RESOURCE

Each record starts:

| Offset | Type | Meaning |
| --- | --- | --- |
| 0 | `"EH"` | Magic |
| 2 | u8 | Type: 1 sprite bank, 2 font, 3 music |
| 3 | u16 | Descriptor length `H` |
| 5 | byte[H] | Descriptor |
| 5+H | … | Payload |

### Sprite bank (type 1)

Descriptor, seven bytes:

| Offset | Type | Meaning |
| --- | --- | --- |
| 0 | u16 | Frame count |
| 2 | u8 | Frame height in pixels |
| 3 | u8 | Frame width in units of 8 pixels |
| 4 | u8 | Transparent palette index; FFh: opaque |
| 5 | u8 | Compression: 0 raw, 1 RLE, 3 LZSS (2 is not used; reject it) |
| 6 | u8 | Unused (0) |

The payload decompresses to `frames * height * width*8` palette indexes: frames in order, each row-major, one byte per pixel. Decompress the whole payload as one stream; runs and dictionary state carry across frame boundaries.

- **RLE (1).** Read a signed byte `c`. If `c >= 0`, copy the next `c+1` bytes. If `c < 0`, repeat the next byte `1−c` times.
- **LZSS (3).** A 4096-byte ring buffer, bytes 0..FEDh filled with FEh; writing starts at FEEh. Read a flag byte and use its bits least significant first. A 1 bit: copy one literal byte to the output and the ring. A 0 bit: read bytes `a, b`; position `a | (b & F0h) << 4`, length `(b & 0Fh) + 3`; copy that many bytes from the ring at the position (wrapping at 4096), each written to the output and the ring as it is copied. Stop when the output is full.

Frames one pixel high exist: some banks are backgrounds stored as one frame per scanline, which are stacked to form the image.

### Font (type 2)

Descriptor: `u8 advance[128]` then a sprite descriptor (seven bytes). The payload is the sprite bank: 128 glyph frames, glyph n for character n. A glyph is drawn in its frame's cell; the pen advances by `advance[n]`. The fonts are resources 0, 1 and 68. The engine uses its own fonts; the importer does not need these.

### Music (type 3)

See audio.md for the descriptor, the patches and the event stream.

## TEXT

An indexed container of NUL-terminated strings. Every byte before the terminator is XORed with DAh. Text is code page 437. Markup inside strings:

- `*word*`-style asterisks mark a keyword (dialogue.md).
- `@` is replaced by the player's name.
- `%s` is replaced by a value the game supplies, such as an object's name (messages 87h and 8Fh).
- The last record is `END` and is not shown.

## KEYWORDS

```text
u16 count
u16 offsets[count]      // absolute
byte strings[]          // NUL-terminated
```

The last entry, `zzzzzzzzzz`, is a sentinel and not a keyword; keywords are `0 .. count−2`. The list is sorted, and keywords are matched on their first four characters, case-insensitively. Keyword `k`'s answer text is TEXT record `T − 2 − k`, where `T` is TEXT's record count: the keyword texts sit at the end of TEXT in reverse order, before `END`.

## LEVELS

An indexed container of 41 levels, addressed by index. Each level:

```text
u8  rows
u8  columns
byte header[64]
u8  terrain[rows][columns]
entity stream
u16 text_pool_size
byte text_pool[text_pool_size]
u16 script_pool_size
byte script_pool[script_pool_size]
```

At most 64 rows and 64 columns.

### Header

| Offset | Type | Meaning |
| --- | --- | --- |
| 00h | u16 | Wall bank (+ terrain base) |
| 02h | u16 | Floor bank (+ terrain base) |
| 04h | u16 | Plate/feature bank (+ terrain base) |
| 06h | u16 | Door bank (+ terrain base) |
| 08h | u16 | Floor overlay bank (+ terrain base) |
| 0Ah | u16 | Exit-wall bank (+ terrain base) |
| 0Ch | u8[10] | Wall decorations: five pairs; the first byte of pair `n` is decoration `n`'s bank (+ wall-decoration base) |
| 16h | u16 | Liquid kind: 0 none, 1 water-like, 2 fire |
| 18h | u16 | Trap sprite bank (+ terrain base) |
| 1Ah | u16 | Boulder sprite bank (+ terrain base) |
| 1Ch | u16[18] | Unknown |

The second byte of each decoration pair is unknown.

If the floor bank offset equals 1Bh (the floor bank is terrain base + 1Bh), the level is **deep water**: liquid depth applies (world.md).

### Entity stream

Repeated until a coordinate byte is FFh (both coordinate bytes are always present):

```text
u8 x
u8 y
// stop if x == FFh or y == FFh
u8 object_count
u8 templates[object_count]
u8 trap            // nonzero: a trap entity at the cell
u8 type
u8 extra[len(type)]
```

| Type | Extra | Meaning |
| --- | --- | --- |
| 0 | 0 | Objects only |
| 1 | 1 | Wall decoration `b`: ornament frame `b % 10`, decoration `b / 10` |
| 2 | 1 | Container: the first object is a container, the rest go in it. The byte is unused |
| 3 | 5 | In-level teleporter record (rules.md) |
| 4 | 1 | Door, with its initial state byte (rules.md) |
| 5 | 0 | Unused |
| 6 | 2 | Boulder: start state (0 resting, 1 rolling, 2 shuttling), boulder direction 0..3 (entities.md) |
| 7 | 5 | Trigger: `unused, disarmed, threshold, procedure_a, procedure_b`; armed when `disarmed == 0` |
| 8 | 4 | Level exit: `marker, level, x, y`; `marker == 0` also places a visible exit marker |
| 9 | 1 | NPC record id |
| 10 | 1 | Weight plate (byte unused) |
| 11 | 1 | Mouth, with its message id in the level's text pool |

The exit's `marker` byte follows what the cell is: 0 a visible exit marker on a floor cell, 1 a pit (25h), 2 a closed trapdoor (27h), 3 an open trapdoor (26h), 4 an exit in a wall. Type 10 occurs only on the gem socket's cell, and type 2 containers only on rune marks (21h–23h).

Objects: each template id becomes an object. If the entry is an NPC, the objects go into its inventory. Otherwise, if the first object is a container, the rest go inside it (an object that does not fit is dropped); else they lie on the floor of the cell.

The trap flag places a trap entity (entities.md) at the cell.

### Text pool

NUL-separated strings, not XOR-coded, possibly padded with zeros. Level message `n` is the n-th string of the pool, counting from 0 (empty strings count). A mouth shows its message as a centered text page.

### Script pool

Level procedures, one after another, then zero padding:

```text
u8 count
{ u8 op; u8 arg[4]; } [count]
```

Procedure `n` is the n-th in the pool. A count of 0 is an empty procedure, which does nothing; empty procedures occur between others and still take their place in the numbering. The padding at the end reads as empty procedures. Ops are in rules.md.

## INTERACT

```text
u16 count
u16 text_base           // 370: conversation text id = word + text_base
u32 offsets[count]      // absolute
records[]
```

Record: `byte name[20]` (NUL-terminated), `u16 size` (bytes), then `size/2` `i16` words. The grammar and opcodes are in dialogue.md.

## OBJECTS

```text
u16 count               // 254
u16 name_base           // name text id = name_base + template byte 1
byte templates[count][15]
```

Field meanings are in items.md.

## NPC

```text
u16 count               // 127
byte records[count][16]
```

Field meanings are in creatures.md.

## INIT

Tables read in order:

| Bytes | Content |
| --- | --- |
| 6 | Three u16: frontend base (1232), view-effect base (1315), end-scene base (1440) |
| 22 | Eleven u16 resource bases, in order: small object (100), worn (333), tiny object (566), people (799), music (1487), scroll (894), character animation (919), held-item animation (963), terrain (1012), NPC (1080), wall decoration (1186) |
| 1 | `N` (233) |
| 2N | Worn-image offsets: N pairs `(col, row)` |
| 256 | Texture shades (panel stone texture; FFh ends the cycle) |
| 256 | Base color map |
| 96 | Figure frame shifts |
| 1536 | Held-item frames: four tables of 96 entries (player frames 0–95) of four bytes: layer (0 behind the figure, 1 in front), frame of the held-item bank, `i8` row offset, `i8` column offset (player.md, Held objects) |
| 40 | Spell costs: four schools × ten spells |
| 120 | Spell power: four schools × ten × `(dice, sides, bonus_sides)` |
| 360 | Spell gestures: four schools × ten × nine bytes (letters A–L, NUL-padded) |
| 16 | Direction deltas: eight `(i8 dx, i8 dy)` |
| 11 | Keypad facings: for the keys Home, Up, PgUp, (−), Left, (5), Right, (+), End, Down, PgDn, the player facing they walk (8: none; player.md) |
| 20 | Wall top parts: per wall shape 0..19, bit 1 the left cap, bit 0 the right cap (world.md) |
| 3 | New game start: level, x, y (shipped: level 0, cell (16, 15)) |
| 1 | Unused |

Values in parentheses are the shipped ones; read them from the file.

## COLORS

```text
u8 blocks
u8 rgb[blocks][16][3]   // 6-bit components, 0..63
```

Eight full 256-color palettes start at blocks 0, 10h, 20h, … 70h. Convert a component with `c * 255 / 63` rounded. Gameplay uses palette 0 unless a scene says otherwise.

## CODES

The copy-protection images: 1-bit PCX files (version 5, RLE). The engine does not use them.

## JAZ and saves

JAZ is the game's runtime cache of visited levels and holds no content. The importer ignores it and the game's save files.

## Fixed resources

Resources the game uses by a fixed number rather than through a level header or a record:

| Resource | Use |
| --- | --- |
| 56 | Mouth (entities.md) |
| 60 | Entities that cycle frames 0, 1, 2, 1 (entities.md) |
| 67 | Floor hazard tiles (world.md) |
| 99 | Exit marker (entities.md) |
| terrain base + 37h | Gem socket (world.md) |
| terrain base + 38h | Gems set in the socket (world.md) |
| terrain base + 39h | Sealed doors (world.md) |
| NPC base + 12h | The slow creature bank: speed 2 (creatures.md) |
| NPC base + 4Dh | The look the player takes when transformed (spells.md) |
| small-object base + image | An object's inventory image (items.md) |
| tiny-object base + image | An object's floor image (items.md) |

## Unknowns

- Header words 1Ch–3Fh, and the second byte of each decoration pair.
- The meaning of the INIT tables beyond the ones named (figure frame shifts, base color map): their use belongs to character drawing and is specified with it.
