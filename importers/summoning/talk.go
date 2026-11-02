package summoning

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// KEYWORDS and INTERACT (formats.md), and the conversations made from them (dialogue.md).

// ReadKeywords reads KEYWORDS: the keywords, without the sentinel at the end.
func ReadKeywords(data []byte) ([]string, error) {
	r := &reader{data: data}
	n := int(r.u16())
	offsets := make([]int, n)
	for i := range offsets {
		offsets[i] = int(r.u16())
	}
	if r.err != nil {
		return nil, fmt.Errorf("keywords: %w", r.err)
	}
	var words []string
	for i, off := range offsets {
		if off < 0 || off >= len(data) {
			return nil, fmt.Errorf("keywords: keyword %d at offset %d outside the file", i, off)
		}
		end := off
		for end < len(data) && data[end] != 0 {
			end++
		}
		words = append(words, decode437(data[off:end]))
	}
	if len(words) > 0 {
		words = words[:len(words)-1]
	}
	return words, nil
}

// Script is an INTERACT record: its name and its words.
type Script struct {
	Name  string
	Words []int16
}

// Interact is INTERACT: the scripts and the TEXT id their texts count from.
type Interact struct {
	TextBase int
	Scripts  []Script
}

// ReadInteract reads INTERACT.
func ReadInteract(data []byte) (*Interact, error) {
	r := &reader{data: data}
	n, base := int(r.u16()), int(r.u16())
	offsets := make([]int, n)
	for i := range offsets {
		offsets[i] = int(r.u16()) | int(r.u16())<<16
	}
	if r.err != nil {
		return nil, fmt.Errorf("interact: %w", r.err)
	}
	in := &Interact{TextBase: base, Scripts: make([]Script, n)}
	for i, off := range offsets {
		if off < 0 || off > len(data) {
			return nil, fmt.Errorf("interact: record %d at offset %d outside the file", i, off)
		}
		rr := &reader{data: data, pos: off}
		name := cString(rr.bytes(20))
		size := int(rr.u16())
		words := make([]int16, size/2)
		for j := range words {
			words[j] = int16(rr.u16())
		}
		if rr.err != nil {
			return nil, fmt.Errorf("interact: record %d: %w", i, rr.err)
		}
		in.Scripts[i] = Script{Name: decode437([]byte(name)), Words: words}
	}
	return in, nil
}

// Script markers (dialogue.md, Scripts).
const (
	markEnd     = -1
	markClause  = -2
	markActions = -3
	markClose   = -4
)

// actionOperands are how many operands each action takes (dialogue.md, Actions).
var actionOperands = map[int16]int{
	0x12: 1, 0x13: 1, 0x14: 1, 0x15: 0, 0x16: 2, 0x17: 1, 0x18: 1, 0x19: 0, 0x1A: 0, 0x1B: 0,
	0x1C: 1, 0x1D: 1, 0x1E: 0, 0x1F: 1, 0x20: 0, 0x21: 0, 0x22: 0, 0x23: 0, 0x24: 0, 0x25: 0,
	0x26: 1, 0x27: 2, 0x28: 1, 0x29: 6, 0x2A: 0, 0x2B: 1, 0x2C: 1, 0x2D: 0, 0x2E: 0,
}

// The view effect that also changes level, and where it takes the player (dialogue.md, 1Dh).
const effectLeaves = 6

var effectTo = content.Destination{Level: levelID(37), Cell: content.Cell{61, 30}}

// gestureSingles is where 2Bh's single icons start (dialogue.md, Actions).
const gestureSingles = 200

// keywords adds keywords.json: keyword k's answer is TEXT record T − 2 − k (formats.md).
func (im *importer) keywords(words []string, texts int) {
	k := &content.Keywords{ID: "keywords"}
	used := map[string]int{}
	im.keywordIDs = make([]string, len(words))
	for i, word := range words {
		id := "keyword." + slug(word)
		if id == "keyword." {
			id = fmt.Sprintf("keyword.%d", i)
		}
		if used[id]++; used[id] > 1 {
			id = fmt.Sprintf("%s.%d", id, used[id])
		}
		im.keywordIDs[i] = id
		kw := content.Keyword{ID: id, Word: id + ".word"}
		im.strings[kw.Word] = word
		if a := texts - 2 - i; a >= 0 && im.strings[msgID(a)] != "" {
			kw.Answer = msgID(a)
		}
		k.Keywords = append(k.Keywords, kw)
	}
	im.pack.Keywords = k
}

// conversations adds a conversation for every INTERACT record, and the NPC templates objects and
// scripts talk through.
func (im *importer) conversations(in *Interact) {
	im.interact = in
	for i, s := range in.Scripts {
		c, err := im.conversation(i, s)
		if err != nil {
			im.report.addf("conversation %d (%s): %v; left out", i, s.Name, err)
			continue
		}
		im.pack.Conversations[c.ID] = c
	}
}

func conversationID(n int) string { return fmt.Sprintf("conversation.%d", n) }

// conversation translates script n (dialogue.md, Scripts).
func (im *importer) conversation(n int, s Script) (*content.Conversation, error) {
	w := s.Words
	at := 0
	next := func() (int16, error) {
		if at >= len(w) {
			return 0, fmt.Errorf("ends early")
		}
		at++
		return w[at-1], nil
	}
	c := &content.Conversation{ID: conversationID(n), Name: s.Name}
	for {
		m, err := next()
		if err != nil {
			return nil, err
		}
		if m == markEnd {
			break
		}
		if m != markClause {
			return nil, fmt.Errorf("word %d: %d where a clause starts", at-1, m)
		}
		var cond []int16
		for {
			x, err := next()
			if err != nil {
				return nil, err
			}
			if x == markClose {
				break
			}
			cond = append(cond, x)
		}
		if m, err := next(); err != nil || m != markActions {
			return nil, fmt.Errorf("word %d: no action list", at-1)
		}
		var acts []int16
		for {
			x, err := next()
			if err != nil {
				return nil, err
			}
			if x == markClose && len(acts) >= im.operandsDone(acts) {
				break
			}
			acts = append(acts, x)
		}
		ifc, err := im.condition(cond)
		if err != nil {
			return nil, fmt.Errorf("clause %d: %w", len(c.Clauses), err)
		}
		do, err := im.talkActions(acts)
		if err != nil {
			return nil, fmt.Errorf("clause %d: %w", len(c.Clauses), err)
		}
		c.Clauses = append(c.Clauses, content.Clause{If: ifc, Do: do})
	}
	return c, nil
}

// operandsDone returns how long acts must be for its last action's operands to be complete, so a
// −4 among operands is not taken for the list's end.
func (im *importer) operandsDone(acts []int16) int {
	i := 0
	for i < len(acts) {
		i += 1 + actionOperands[acts[i]]
	}
	return i
}

func intp(n int) *int { return &n }

// condition translates a clause's condition words.
func (im *importer) condition(w []int16) (content.Condition, error) {
	var c content.Condition
	if len(w) == 0 {
		return c, fmt.Errorf("no condition")
	}
	arg := func(i int) (int, error) {
		if i >= len(w) {
			return 0, fmt.Errorf("condition %02Xh: too few operands", w[0])
		}
		return int(w[i]), nil
	}
	two := func() (int, int, error) {
		a, err := arg(1)
		if err != nil {
			return 0, 0, err
		}
		b, err := arg(2)
		return a, b, err
	}
	switch w[0] {
	case 4:
		c.FirstMeeting = true
	case 5:
		c.Greeting = true
	case 6:
		for _, k := range w[1:] {
			if int(k) < 0 || int(k) >= len(im.keywordIDs) {
				return c, fmt.Errorf("no keyword %d", k)
			}
			c.Asked = append(c.Asked, im.keywordIDs[k])
		}
	case 7:
		c.Never = true
	case 8:
		for _, t := range w[1:] {
			id, ok := im.item(int(t), "conversation")
			if !ok {
				return c, fmt.Errorf("no object template %d", t)
			}
			c.Carries = append(c.Carries, id)
		}
	case 9, 0x0A, 0x0B:
		v, x, err := two()
		if err != nil {
			return c, err
		}
		t := &content.CounterTest{Counter: varID(v)}
		switch w[0] {
		case 9:
			t.Above = intp(x)
		case 0x0A:
			t.Equals = intp(x)
		default:
			t.Below = intp(x)
		}
		c.Counter = t
	case 0x0C:
		npc, err := arg(1)
		if err != nil {
			return c, err
		}
		c.Dead = npcID(npc)
		im.talkers[c.Dead] = true
	case 0x0D:
		c.FullHealth = true
	case 0x0E:
		gold, err := arg(1)
		if err != nil {
			return c, err
		}
		id, ok := im.item(templateGold, "conversation")
		if !ok {
			return c, fmt.Errorf("no gold coins")
		}
		c.CarriesCount = &content.ItemCount{Item: id, Count: gold}
	case 0x0F:
		slot, err := arg(1)
		if err != nil {
			return c, err
		}
		c.SlotEmpty = intp(slot)
	case 0x10:
		slot, t, err := two()
		if err != nil {
			return c, err
		}
		id, ok := im.item(t, "conversation")
		if !ok {
			return c, fmt.Errorf("no object template %d", t)
		}
		c.SlotHolds = &content.SlotHolds{Slot: slot, Item: id}
	case 0x11:
		c.Always = true
	default:
		return c, fmt.Errorf("unknown condition %02Xh", w[0])
	}
	return c, nil
}

// templateGold is the Gold Coin.
const templateGold = 0x09

// menuActions are the actions that open the game's menus.
var menuActions = map[int16]string{
	0x22: content.MenuQuit, 0x23: content.MenuSave, 0x24: content.MenuRestore, 0x25: content.MenuDelete,
	0x2D: content.MenuLeave, 0x2E: content.MenuRename,
}

// talkActions translates a clause's action words.
func (im *importer) talkActions(w []int16) ([]content.Action, error) {
	var out []content.Action
	str := func(s string) *string { return &s }
	text := func(n int) (string, error) {
		id := msgID(n + im.interact.TextBase)
		if _, ok := im.strings[id]; !ok {
			return "", fmt.Errorf("no text %d", n+im.interact.TextBase)
		}
		return id, nil
	}
	for i := 0; i < len(w); {
		op := w[i]
		k, ok := actionOperands[op]
		if !ok {
			return nil, fmt.Errorf("unknown action %02Xh", op)
		}
		if i+1+k > len(w) {
			return nil, fmt.Errorf("action %02Xh: too few operands", op)
		}
		args := make([]int, k)
		for j := range k {
			args[j] = int(w[i+1+j])
		}
		i += 1 + k
		var a content.Action
		switch op {
		case 0x12:
			t, err := text(args[0])
			if err != nil {
				return nil, err
			}
			a.Say = &t
		case 0x13:
			if args[0] < 0 || args[0] >= len(im.keywordIDs) {
				return nil, fmt.Errorf("no keyword %d", args[0])
			}
			a.Offer = str(im.keywordIDs[args[0]])
		case 0x14, 0x18:
			id, ok := im.item(args[0], "conversation")
			if !ok {
				return nil, fmt.Errorf("no object template %d", args[0])
			}
			if op == 0x14 {
				a.Take = &id
			} else {
				a.Give = &content.Object{Item: id}
			}
		case 0x15:
			a.End = true
		case 0x16:
			a.SetCounter = &content.SetCounter{Counter: varID(args[0]), Value: args[1]}
		case 0x17:
			a.AddCounter = str(varID(args[0]))
		case 0x19:
			a.RemoveSpeaker = true
		case 0x1A:
			a.HealFully = true
		case 0x1B:
			a.ForgetOffers = true
		case 0x1C:
			name := "continued"
			if args[0] != 0 {
				name = "heading"
			}
			a.Transcript = &name
		case 0x1D:
			a.Effect = str(effectID(int(args[0])))
			if args[0] == effectLeaves && im.levelExists(effectTo.Level) {
				d := effectTo
				out = append(out, a, content.Action{Travel: &d})
				continue
			}
		case 0x1E:
			a.Hostile = true
		case 0x1F:
			a.Run = str(procName(args[0]))
		case 0x20:
			continue
		case 0x21:
			a.Ending = true
		case 0x22, 0x23, 0x24, 0x25, 0x2D, 0x2E:
			a.Menu = str(menuActions[op])
		case 0x26:
			a.SetFrame = intp(args[0])
		case 0x27:
			t, err := text(args[1])
			if err != nil {
				return nil, err
			}
			a.SayAs = &content.SayAs{Text: t}
			if args[0] != 0 {
				a.SayAs.Portrait = im.npcPortrait(args[0])
			}
		case 0x28:
			a.Jump = intp(args[0])
		case 0x29:
			id, ok := im.item(args[0], "conversation")
			if !ok {
				return nil, fmt.Errorf("no object template %d", args[0])
			}
			o := &content.Object{Item: id}
			for _, t := range args[1:] {
				if t == 0xFF {
					continue
				}
				if c, ok := im.item(t, "conversation"); ok {
					o.Contents = append(o.Contents, content.Object{Item: c})
				}
			}
			a.Give = o
		case 0x2A:
			a.Return = true
		case 0x2B:
			g := icons(im.gesture(args[0]))
			if args[0] >= gestureSingles {
				g = icons(string(rune('A' + args[0] - gestureSingles)))
			}
			if g == "" {
				continue
			}
			a.Teach = &g
		case 0x2C:
			p := im.npcPortrait(args[0])
			if p == "" {
				continue
			}
			a.SetPortrait = &p
		}
		out = append(out, a)
	}
	return out, nil
}

// npcPortrait returns the portrait of people bank n: one of F's, or else added with only its
// image (creatures.md, Import); "" when it cannot be.
func (im *importer) npcPortrait(n int) string {
	id := portraitID(n)
	if _, ok := im.pack.Portraits[id]; ok {
		return id
	}
	sid, err := im.stripSprite(int(im.init.Bases.People)+n, "sprite.portrait")
	if err != nil {
		im.report.addf("portrait %d: %v", n, err)
		return ""
	}
	im.pack.Portraits[id] = &content.Portrait{ID: id, Image: content.Image{Sprite: sid}}
	return id
}
