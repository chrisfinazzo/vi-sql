package config

import (
	"github.com/gdamore/tcell/v2"
)

// chordState holds the vim-mode chord prefix state machine. Embedded
// anonymously in KeyBindings so its methods are promoted.
type chordState struct {
	vimMode       bool
	pending       rune
	chordPrefixes map[rune]struct{}
	// chordEvent identifies the event currently being dispatched as a chord
	// completion attempt. tview routes input through the focus chain
	// (Pages → page → focused primitive), so several wrappers in that chain
	// see the same event. Tracking the event lets the first wrapper mark the
	// dispatch as in-progress and lets later wrappers in the same chain reuse
	// the pending state without re-entering the absorption logic.
	chordEvent       *tcell.EventKey
	OnPendingChanged func(rune)
	// ChordsDisabled is set for text inputs and vim insert mode where
	// every rune must reach the inner handler verbatim.
	ChordsDisabled func() bool
}

func (cs *chordState) HasPending() bool { return cs.pending != 0 }

func (cs *chordState) IsChordPrefix(r rune) bool {
	_, ok := cs.chordPrefixes[r]
	return ok
}

func (cs *chordState) SetPending(r rune) {
	cs.pending = r
	cs.notifyPending(r)
}

// Reset clears any pending chord prefix. Call on focus change or mode switch.
func (cs *chordState) Reset() {
	cs.chordEvent = nil
	if cs.pending != 0 {
		cs.pending = 0
		cs.notifyPending(0)
	}
}

func (cs *chordState) notifyPending(r rune) {
	if cs.OnPendingChanged != nil {
		cs.OnPendingChanged(r)
	}
}

// WrapInputCapture wraps tview InputCapture handler so that chords can be
// properly absorbed by chordState firstly, if no match or it's not rune
// key is being propagate further chain of wrappers (app -> main -> data -> etc)
func (cs *chordState) WrapInputCapture(inner func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// Cleanup chordEvent if previous chord-completion didn't find match, eg: `gq` -> don't match any key
		if cs.chordEvent != nil && cs.chordEvent != ev {
			cs.chordEvent = nil
			if cs.pending != 0 {
				cs.pending = 0
				cs.notifyPending(0)
			}
		}

		if cs.ChordsDisabled != nil && cs.ChordsDisabled() {
			cs.Reset()
			return inner(ev)
		}
		if ev.Key() != tcell.KeyRune {
			cs.Reset()
			return inner(ev)
		}
		if cs.pending != 0 {
			// Mark event as "in-progress", so deeper wrappers see the same
			// chordEvent and skip re-marking
			if cs.chordEvent == nil {
				cs.chordEvent = ev
			}
			result := inner(ev)
			if result == nil {
				// Inner matched and consumed the chord.
				cs.pending = 0
				cs.chordEvent = nil
				cs.notifyPending(0)
				return nil
			}
			// Inner didn't match. Propagate so a deeper wrapper can try.
			return result
		}
		if _, ok := cs.chordPrefixes[ev.Rune()]; ok {
			cs.pending = ev.Rune()
			cs.notifyPending(ev.Rune())
			return nil
		}
		return inner(ev)
	}
}
