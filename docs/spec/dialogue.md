# Dialogue

Conversations are scripts in INTERACT. A creature's script is byte 0 of its NPC record (creatures.md). Some objects speak too, through powers that start a conversation with a given record (items.md). Record 0's script is the disk menu (save, restore, delete, rename).

## Keywords

KEYWORDS lists the words the player can ask about (formats.md). A typed word matches a keyword by its first four characters, ignoring case, up to a space or punctuation. Each keyword has an answer text in TEXT.

A conversation keeps an **offered list** of at most six keywords, shown as buttons beside a field where the player can type any word. In conversation texts, keywords are marked and drawn underlined; clicking one, or typing its initial, asks it (at most nine marked words per text).

## Interaction variables

99 byte variables (0..98) hold the story's state across the whole game. Scripts set, count and test them. The game itself uses:

| Var | Use |
| --- | --- |
| 14 | Set to 1 when the player takes the look of NPC bank +4Dh (spells.md); cleared when the transformation ends |
| 18 | Set to 1 when the Decanter breaks (items.md) |
| 19 | The result of power 51h's conversation: the object is used only if it is nonzero |
| 27 | WarMonger's level (combat.md) |
| 32 | The chosen ending (scenes.md) |
| 34 | Set to 1 for a new game, which starts the opening conversation (7Dh) on arrival |
| 96 | Nonzero hides combat messages |
| 97 | Cleared after every conversation |
| 98 | Sight: read when drawing objects (scenes.md) |

Variable number 99 is the "print transcript" option of the disk menu.

## Scripts

A script is a 20-byte name and a list of 16-bit words: clause pairs, each a **condition** and an **action list**:

```
script  = pair* −1
pair    = −2 condition −4 −3 action* −4
```

### Running a conversation

1. The offered list is emptied. The conversation window opens with the creature's portrait (the player's talk portrait if the creature shares the player's).
2. **Greeting pass**: the clauses run with no input.
3. Then for each word the player asks, a pass runs with it, until an action ends the conversation or the player leaves (Esc, a right click, or an empty answer).
4. The NPC record gets behavior bit 0001h ("talked to").

**A pass** evaluates the conditions in order, from the first clause, and runs the action list of the first one that holds. If none holds, the pass ends. An action list that jumps (28h) makes the pass continue evaluating from the named clause (counting from 0); otherwise the pass ends after the list. A typed word that is not a keyword runs no clause.

### Conditions

| Op | Operands | Holds when |
| --- | --- | --- |
| 4 | | Greeting pass, and the player has never talked to this NPC |
| 5 | | Greeting pass |
| 6 | keywords… | The asked keyword is one of the listed. Then the question is shown ("Q: word"), the listed keywords leave the offered list, and the keyword's answer text is said if not empty |
| 7 | | Never; evaluation stops |
| 8 | templates… | The player carries at least one of each listed template |
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
| 2Bh | spell | Show the spell's gesture, teaching its icons |
| 2Ch | portrait | Change the portrait, for this conversation and in the NPC record |
| 2Dh | | Ask whether to leave the game for the main menu |
| 2Eh | | Rename a saved game |

Objects given go to the first empty shelf, or to the floor at the player's feet.

## Showing text

Conversation text is shown word by word in the portrait window. `@` is replaced by the player's name. Asked questions and said texts also go to the **transcript**, which the disk menu can send to a printer (the engine: a text file), wrapped at 80 columns.

## Unknowns

- The response, if any, when the player asks a word that is not a keyword or that no clause handles.
- The full list of interaction variables the game itself reads, by number.
