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
	// inFlightEvent marks the event currently traversing the wrapper chain so
	// deeper wrappers reuse the pending state instead of re-absorbing it.
	inFlightEvent    *tcell.EventKey
	OnPendingChanged func(rune)
	// SequencesDisabled is set for text inputs and vim insert mode where
	// every rune must reach the inner handler verbatim.
	SequencesDisabled func() bool
}

func (cs *sequenceState) HasPending() bool { return cs.pending != 0 }
func (cs *sequenceState) GetPending() rune { return cs.pending }

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
	cs.inFlightEvent = nil
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

// WrapInputCapture wraps tview InputCapture handler so sequences (e.g. `gg`)
// are absorbed first. Non-rune events and unmatched runes propagate further
// down the wrapper chain (app → main → data → ...).
func (cs *sequenceState) WrapInputCapture(inner func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// Stale in-flight event: previous sequence (e.g. `gq`) found no match.
		if cs.inFlightEvent != nil && cs.inFlightEvent != ev {
			cs.inFlightEvent = nil
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
			// Escape cancels a pending prefix without forwarding the event,
			if cs.pending != 0 && ev.Key() == tcell.KeyEsc {
				cs.Reset()
				return nil
			}
			cs.Reset()
			return inner(ev)
		}
		if cs.pending != 0 {
			// Mark in-flight so deeper wrappers reuse the pending state.
			if cs.inFlightEvent == nil {
				cs.inFlightEvent = ev
			}
			result := inner(ev)
			if result == nil {
				// Sequence consumed by a matching k.Match call.
				cs.pending = 0
				cs.inFlightEvent = nil
				cs.notifyPending(0)
				return nil
			}
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
