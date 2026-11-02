# Dialogue

Conversations are scripts in INTERACT. A conversation is held with an NPC record: its script is byte 0 of the record (creatures.md) and its portrait byte 1. Some objects speak too, through powers that start a conversation with a given record (items.md), and so do some events of the game (a broken Decanter, Mending, the opening). Record 0's script is the disk menu (save, restore, delete, rename), opened with the O key. The engine also opens it with Esc when nothing is open over the view: Esc first closes what is (a conversation, the full map, the spell panel, a scroll's picture, the inventory panel), one press each, and opens the menu only from the plain view.

## Talking

The player talks with the talk key or by clicking a creature. The key talks to the nearest creature that can be talked to (behavior 0008h, which turning hostile clears) or mouth, among those in the view; a mouth says its level message instead (rules.md). Conversations pause the game.

**The pointer.** Over the view, the pointer is RESOURCE 2 frame 0 (an arrow), or the small image of the object in hand. Over a creature that can be talked to, or a mouth, it becomes frame 9, a speech bubble: when the pointer, 8 pixels down and right of its hot spot, is inside the 32×32 box at the top-left of where the thing is drawn. A click there talks to that creature, or hears that mouth. The pointer's hot spot is its top-left pixel; frame 10 is shown while the game waits.

**Engine model.** The presentation shows a mouth's message beside the mouth, rather than as a page of its own, and keeps it until the player acts or a few seconds pass.

## Keywords

KEYWORDS lists the words the player can ask about (formats.md). A typed word matches a keyword by its first four characters, ignoring case, up to a space or punctuation. Each keyword has an answer text in TEXT.

A conversation keeps an **offered list** of at most six keywords, shown as buttons beside a field where the player can type a word; with none offered, a single "Bye" button leaves. A typed word that is not a keyword is refused: the field clears and nothing is asked. Typing "Bye", Esc, a right click or an empty answer leaves. In conversation texts, keywords are marked and drawn underlined; while a text shows, clicking one, or typing its initial, asks it next (at most nine marked words per text).

## Interaction variables

99 byte variables (0..98) hold the story's state across the whole game. Scripts set, count and test them. The game itself uses:

| Var | Use |
| --- | --- |
| 14 | Set to 1 when the player takes the look of NPC bank +4Dh (spells.md); cleared when the transformation ends |
| 18 | Set to 1 when the Decanter breaks (items.md) |
| 19 | The result of power 51h's conversation: the object is used only if it is nonzero |
| 27 | WarMonger's level (combat.md) |
| 32 | The chosen ending (scenes.md) |
| 34 | Set to 1 when play goes on after ending G, from the game it saved (scenes.md, Endings); conversation 7Dh then starts on arrival. A new game leaves it 0 |
| 96 | Nonzero hides combat messages |
| 97 | Cleared after every conversation |
| 98 | Sight: read when drawing objects (scenes.md) |

Variable number 99 is the "print transcript" option of the disk menu: while it is set, every line said and question asked goes to the transcript. Action 1Ch prints a heading: with `first ≠ 0` TEXT records 74h, 75h, 76h, 78h, otherwise 77h, 75h, 76h, 78h, one to a line.

## Scripts

A script is a 20-byte name and a list of 16-bit words: clause pairs, each a **condition** and an **action list**:

```
script  = pair* −1
pair    = −2 condition −4 −3 action* −4
```

### Running a conversation

1. The offered list is emptied. The conversation window opens with the record's portrait (the player's talk portrait if it is the player's own).
2. **Greeting pass**: the clauses run with no input.
3. Then for each word the player asks, a pass runs with it, until an action ends the conversation or the player leaves (Esc, a right click, or an empty answer).
4. The NPC record gets behavior bit 0001h ("talked to").

**A pass** evaluates the conditions in order, from the first clause, and runs the action list of the first one that holds. If none holds, the pass ends, saying nothing. An action list that jumps (28h) makes the pass continue evaluating from the named clause (counting from 0); otherwise the pass ends after the list. Each text said waits for a key or click; Esc there stops the rest of the pass and leaves the conversation.

After a conversation, interaction variable 97 is cleared, and a stored level change happens (actions 1Dh with effect 6, and 2Ah).

### Conditions

| Op | Operands | Holds when |
| --- | --- | --- |
| 4 | | Greeting pass, and the player has never talked to this NPC |
| 5 | | Greeting pass |
| 6 | keywords… | The asked keyword is one of the listed. Then the question is shown ("Q: word"), the listed keywords leave the offered list, and the keyword's answer text is said if not empty |
| 7 | | Never; evaluation stops |
| 8 | templates… | The player carries at least one of each listed template (in any slot, inside containers too) |
| 9 | var, n | Variable > n |
| 0Ah | var, n | Variable = n |
| 0Bh | var, n | Variable < n |
| 0Ch | npc | NPC record `npc` is dead (behavior 0002h) |
| 0Dh | | The player's hit points are full |
| 0Eh | n | The player carries at least `n` Gold Coins (template 9) |
| 0Fh | slot | The inventory slot is empty |
| 10h | slot, template | The slot holds an object of the template |
| 11h | | Always |

Lists (6 and 8) run to the clause's −4.

### Actions

| Op | Operands | Effect |
| --- | --- | --- |
| 12h | text | Say TEXT `text + text_base` |
| 13h | keyword | Add the keyword to the offered list (replacing its old place; at most six) |
| 14h | template | Take one object of the template from the player |
| 15h | | End the conversation |
| 16h | var, n | Set the variable to `n` |
| 17h | var | Add 1 to the variable |
| 18h | template | Give the player a new object of the template |
| 19h | | Remove the speaking creature |
| 1Ah | | Cure poison and confusion, heal fully |
| 1Bh | | Empty the offered list |
| 1Ch | first | Print the transcript heading (first heading if `first ≠ 0`) |
| 1Dh | effect | Play view effect `effect` at the creature (scenes.md); effect 6 also takes the player to level 37 at (61, 30) when the conversation ends |
| 1Eh | | The creature and all creatures of its record turn hostile |
| 1Fh | p | Run level procedure `p` (rules.md) |
| 20h | | Nothing |
| 21h | | Play the ending (scenes.md) |
| 22h | | Ask whether to quit the game |
| 23h | | Save the game |
| 24h | | Restore a game |
| 25h | | Delete a saved game |
| 26h | frame | Set the creature's frame |
| 27h | portrait, text | Say TEXT `text + text_base` beside portrait `portrait` |
| 28h | clause | Jump: continue evaluating from clause `clause` |
| 29h | template, t1..t5 | Give a new container of the template holding new objects t1..t5 (FFh: none) |
| 2Ah | | When the conversation ends, return the player to the level and cell stored on entering level 30 (spells.md, Gateway) |
| 2Bh | spell | Show the spell's gesture, teaching its icons; from 200, show and teach the single icon `spell − 200` (0 = A) |
| 2Ch | portrait | Change the portrait, for this conversation and in the NPC record |
| 2Dh | | Ask whether to leave the game for the main menu |
| 2Eh | | Rename a saved game |

Objects given go to the first empty shelf, or to the floor at the player's feet.

## Showing text

Conversation text is shown word by word in the portrait window. `@` is replaced by the player's name. Asked questions and said texts also go to the **transcript**, which the disk menu can send to a printer (the engine: a text file), wrapped at 80 columns.

## Engine model

- **Keywords** are a pack-wide list: each a word and the text of its answer. A typed word matches the first keyword whose first four letters it shares, ignoring case, up to a space or punctuation.
- A **conversation** is a record: a name and its clauses, each a condition and a list of actions. Conditions: `first_meeting`, `greeting`, `asked` (keywords), `never`, `carries` (items, one of each), `counter` (a name and `above`, `equals` or `below` a value), `dead` (a thing template), `full_health`, `carries_count` (an item and how many objects of it), `slot_empty` and `slot_holds` (an inventory slot), `always`. Actions are the registry's: `say`, `offer`, `take`, `end`, `set_counter`, `add_counter`, `give`, `remove_speaker`, `heal_fully`, `forget_offers`, `transcript`, `effect`, `hostile`, `run` (a procedure of the current level), `ending`, `menu` (quit, save, restore, delete, rename, leave), `set_frame`, `say_as` (a portrait and a text), `jump`, `give_container`, `return`, `teach` (a spell's gesture, or icons), `set_portrait` and `travel` (after the conversation).
- **Counters** are the interaction variables, named, held by the world; the importer names variable `n` `var.n`. The pack names the counters the engine itself reads and writes: the one cleared after every conversation, the result of a conversation an object starts, and the one Shape Change sets.
- A thing template names its `conversation` and the `talk_portrait` shown; `talks` marks a creature that can be talked to. The world keeps, per template, whether the player has talked to it, whether one of its creatures has died, and a changed portrait.
- Commands: **talk** (to a thing, or the nearest), **ask** (a word), **leave**, and **menu**, which opens the conversation of the thing template the pack names as its `menu` (the importer: NPC record 0's, the disk menu). While a conversation is open the world does not advance; the presentation shows its lines one at a time and the offered keywords.
- The **transcript** goes to a text file, `transcript.txt` beside the saves: each line and question wrapped at 80 columns, `@` replaced by the player's name, and under each line holding a keyword a line of `_` under its letters. The pack names the counter that turns it on (`transcript`) and the two headings as the engine messages `transcript_heading` and `transcript_continued`; the `transcript` action prints `heading` or `continued`.
- A `menu` action waits like a line, of kind `menu`. The presentation opens that menu: save (a typed name, or a save to replace), restore, delete or rename a save, or a yes-or-no question for quit and leave (to the main menu); when it closes, going on runs the rest of the conversation. A save made then leaves the conversation out, and restoring replaces the world with none open.

## Unknowns

- The full list of interaction variables the game itself reads, by number.

