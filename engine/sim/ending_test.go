package sim

import (
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/input"
)

// endingWorld is the study with an ending: holding the sword chooses ending "S" (value 7), whose
// conversation says a line and, when closing is set, plays the closing.
func endingWorld(t *testing.T, closing bool) *World {
	t.Helper()
	text := "talk.sage.hello"
	return studyWorld(t, devpack.StudyStart, 2, func(p *content.Pack) {
		do := []content.Action{{Say: &text}}
		if closing {
			do = append(do, content.Action{Ending: true}, content.Action{Say: &text})
		}
		p.Conversations["conversation.end"] = &content.Conversation{ID: "conversation.end",
			Clauses: []content.Clause{{If: content.Condition{Greeting: true}, Do: do}}}
		end := *p.Things[devpack.ThingSage]
		end.ID, end.Conversation = "thing.end", "conversation.end"
		p.Things["thing.end"] = &end
		p.Creation.Ending = &content.Ending{Scene: "", Counter: "var.ending", Conversation: "thing.end",
			Choices: []content.EndingChoice{{Name: "S", Holding: devpack.ItemSword, Value: 7}}}
		p.Items[devpack.ItemRock].Power, p.Items[devpack.ItemRock].Charges = powerEnding, 1
		p.Items[devpack.ItemRock].Actions = []content.ItemAction{{Kind: content.ActionUse, Delay: 3}}
	})
}

func TestEndingPower(t *testing.T) {
	w := endingWorld(t, false)
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject(devpack.ItemRock)
	// Without the sword in a hand, the power is refused.
	if ev := do(w, input.Strike{Hand: 0}); !has(ev, EventMessage, content.MsgNoEnding) || w.Talk() != nil {
		t.Fatalf("the ending without the sword: %v", ev)
	}
	p.Slots[SlotLeftHand] = w.newObject(devpack.ItemSword)
	p.Delays[0] = 0
	do(w, input.Strike{Hand: 0})
	if w.Talk() == nil || w.Counters()["var.ending"] != 7 || w.Ended() != nil {
		t.Fatalf("choosing: talk %v, counters %v, ended %v", w.Talk(), w.Counters(), w.Ended())
	}
	do(w, input.Continue{})
	ev := do(w, input.Leave{})
	if w.Ended() == nil || w.Ended().Name != "S" || !hasEvent(ev, EventEnding, "S") {
		t.Fatalf("the conversation's end did not end play: %v, %v", w.Ended(), ev)
	}
	step := w.Step()
	w.Update(nil)
	if w.Step() != step {
		t.Error("the world ran on after the ending")
	}
}

func TestEndingAction(t *testing.T) {
	w := endingWorld(t, true)
	p := w.Player()
	p.Slots[SlotRightHand] = w.newObject(devpack.ItemRock)
	p.Slots[SlotLeftHand] = w.newObject(devpack.ItemSword)
	do(w, input.Strike{Hand: 0})
	ev := do(w, input.Continue{})
	if e := w.Ended(); e == nil || e.Name != "" || w.Talk() != nil || !hasEvent(ev, EventEnding, "") {
		t.Fatalf("the ending action: ended %v, talk %v, %v", e, w.Talk(), ev)
	}
}
