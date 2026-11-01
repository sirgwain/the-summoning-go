package sim

import "github.com/sirgwain/the-summoning-go/engine/content"

// The ending (docs/spec/scenes.md, Endings; docs/spec/items.md, power 5Eh).

// powerEnding chooses the ending by what the player holds.
const powerEnding = 0x5E

// EventEnding reports that play has ended: Detail is the ending's name ("" for the closing the
// conversations play) and Text the scene to show, if any. The world runs no more.
const EventEnding = "ending"

// Ended is how play ended.
type Ended struct {
	Name, Scene string
}

// Ended returns how play ended, or nil while it goes on.
func (w *World) Ended() *Ended { return w.ended }

// ending returns the pack's ending, or nil.
func (w *World) ending() *content.Ending {
	if c := w.pack.Creation; c != nil {
		return c.Ending
	}
	return nil
}

// endingChoice returns the first of the ending's choices whose item the player holds in a hand.
func (w *World) endingChoice() *content.EndingChoice {
	e, p := w.ending(), w.player
	if e == nil || p == nil {
		return nil
	}
	for i, ch := range e.Choices {
		for _, s := range []int{SlotRightHand, SlotLeftHand} {
			if o := p.Slots[s]; o != nil && o.Item == ch.Holding {
				return &e.Choices[i]
			}
		}
	}
	return nil
}

// chooseEnding runs the ending power: the counter takes the choice's value, the ending's
// conversation runs, and play ends with the choice when it does.
func (w *World) chooseEnding() {
	ch, e := w.endingChoice(), w.ending()
	if ch == nil {
		return
	}
	if e.Counter != "" {
		w.setCounter(e.Counter, ch.Value)
	}
	end := Ended{Name: ch.Name, Scene: ch.Scene}
	if tmpl := w.pack.Things[e.Conversation]; tmpl != nil {
		if t := w.startTalk(tmpl, nil); t != nil {
			if w.talk == t {
				t.ending = &end
				return
			}
			// The conversation ended at once (or ended the game itself).
			if w.ended != nil {
				return
			}
		}
	}
	w.end(end)
}

// end ends play.
func (w *World) end(e Ended) {
	if w.ended != nil {
		return
	}
	w.ended = &e
	w.talk = nil
	w.emit(Event{Kind: EventEnding, Level: w.current, Detail: e.Name, Text: e.Scene})
}

// closing ends play with the ending's closing scene, as the conversations' ending action does.
func (w *World) closing() {
	scene := ""
	if e := w.ending(); e != nil {
		scene = e.Scene
	}
	w.end(Ended{Scene: scene})
}
