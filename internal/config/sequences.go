package config

import (
	"github.com/gdamore/tcell/v2"
)

// sequenceState holds the vim-mode sequence prefix state machine. Embedded
// anonymously in KeyBindings so its methods are promoted.
type sequenceState struct {
	vimMode          bool
	pending          rune
	sequencePrefixes map[rune]struct{}
	// sequenceEvent identifies the event currently being dispatched as a sequence
	// completion attempt. tview routes input through the focus chain
	// (Pages → page → focused primitive), so several wrappers in that chain
	// see the same event. Tracking the event lets the first wrapper mark the
	// dispatch as in-progress and lets later wrappers in the same chain reuse
	// the pending state without re-entering the absorption logic.
	sequenceEvent    *tcell.EventKey
	OnPendingChanged func(rune)
	// SequencesDisabled is set for text inputs and vim insert mode where
	// every rune must reach the inner handler verbatim.
	SequencesDisabled func() bool
}

func (cs *sequenceState) HasPending() bool { return cs.pending != 0 }

func (cs *sequenceState) IsSequencePrefix(r rune) bool {
	_, ok := cs.sequencePrefixes[r]
	return ok
}

func (cs *sequenceState) SetPending(r rune) {
	cs.pending = r
	cs.notifyPending(r)
}

// Reset clears any pending sequence prefix. Call on focus change or mode switch.
func (cs *sequenceState) Reset() {
	cs.sequenceEvent = nil
	if cs.pending != 0 {
		cs.pending = 0
		cs.notifyPending(0)
	}
}

func (cs *sequenceState) notifyPending(r rune) {
	if cs.OnPendingChanged != nil {
		cs.OnPendingChanged(r)
	}
}

// WrapInputCapture wraps tview InputCapture handler so that sequences can be
// properly absorbed by sequenceState firstly, if no match or it's not rune
// key is being propagated further chain of wrappers (app -> main -> data -> etc)
func (cs *sequenceState) WrapInputCapture(inner func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// Cleanup sequenceEvent if previous sequence-completion didn't find match, eg: `gq` -> don't match any key
		if cs.sequenceEvent != nil && cs.sequenceEvent != ev {
			cs.sequenceEvent = nil
			if cs.pending != 0 {
				cs.pending = 0
				cs.notifyPending(0)
			}
		}

		if cs.SequencesDisabled != nil && cs.SequencesDisabled() {
			cs.Reset()
			return inner(ev)
		}
		if ev.Key() != tcell.KeyRune {
			cs.Reset()
			return inner(ev)
		}
		if cs.pending != 0 {
			// Mark event as "in-progress", so deeper wrappers see the same
			// sequenceEvent and skip re-marking
			if cs.sequenceEvent == nil {
				cs.sequenceEvent = ev
			}
			result := inner(ev)
			if result == nil {
				// Inner matched and consumed the sequence.
				cs.pending = 0
				cs.sequenceEvent = nil
				cs.notifyPending(0)
				return nil
			}
			// Inner didn't match. Propagate so a deeper wrapper can try.
			return result
		}
		if _, ok := cs.sequencePrefixes[ev.Rune()]; ok {
			cs.pending = ev.Rune()
			cs.notifyPending(ev.Rune())
			return nil
		}
		return inner(ev)
	}
}
