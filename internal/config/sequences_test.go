package config

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestHasPending(t *testing.T) {
	cs := sequenceState{}
	assert.False(t, cs.HasPending())

	cs.pending = 'g'
	assert.True(t, cs.HasPending())
}

func TestIsSequencePrefix(t *testing.T) {
	cs := sequenceState{sequencePrefixes: map[rune]struct{}{'g': {}}}
	assert.True(t, cs.IsSequencePrefix('g'))
	assert.False(t, cs.IsSequencePrefix('x'))
}

func TestSetPending_FiresNotification(t *testing.T) {
	var notified []rune
	cs := sequenceState{OnPendingChanged: func(r rune) { notified = append(notified, r) }}

	cs.SetPending('g')
	assert.Equal(t, rune('g'), cs.pending)
	assert.Equal(t, []rune{'g'}, notified)
}

func TestReset_ClearsPendingAndFires(t *testing.T) {
	var notified []rune
	cs := sequenceState{
		pending:          'g',
		OnPendingChanged: func(r rune) { notified = append(notified, r) },
	}

	cs.Reset()
	assert.Equal(t, rune(0), cs.pending)
	assert.Equal(t, []rune{0}, notified)
}

func TestReset_ClearsInFlightEvent(t *testing.T) {
	ev := mkRune('g')
	cs := sequenceState{pending: 'g', inFlightEvent: ev}

	cs.Reset()
	assert.Nil(t, cs.inFlightEvent, "Reset must clear inFlightEvent")
}

// When a sequence's second rune has no matching binding, inner returns the
// event (passes through). Pending stays — until the next, different event
// triggers the stale-event reset and clears it.
func TestWrapInputCapture_UnmatchedSecondRuneClearsOnNextEvent(t *testing.T) {
	kb := &KeyBindings{sequenceState: sequenceState{
		vimMode:          true,
		sequencePrefixes: map[rune]struct{}{'g': {}},
		pending:          'g',
	}}
	var notified []rune
	kb.OnPendingChanged = func(r rune) { notified = append(notified, r) }

	fn := kb.WrapInputCapture(func(ev *tcell.EventKey) *tcell.EventKey { return ev })

	// Second rune of an unmatched sequence: inner returns ev, so wrapper
	// passes it through and leaves pending in place.
	out := fn(mkRune('q'))
	assert.NotNil(t, out, "unmatched second rune must pass through")
	assert.Equal(t, rune('g'), kb.pending, "pending must remain after unmatched second rune")
	assert.NotNil(t, kb.inFlightEvent, "inFlightEvent must be set so deeper wrappers reuse pending")
	assert.Empty(t, notified, "no notification yet — pending unchanged")

	// Next event arrives: stale check clears pending.
	out = fn(mkRune('x'))
	assert.NotNil(t, out)
	assert.Equal(t, rune(0), kb.pending, "stale pending cleared on next event")
	assert.Nil(t, kb.inFlightEvent)
	assert.Equal(t, []rune{0}, notified, "OnPendingChanged(0) fires when stale pending is cleared")
}

// The inFlightEvent guard prevents the stale-reset from firing when the SAME
// event re-enters the wrapper (deeper wrappers in the chain share the state).
func TestWrapInputCapture_SameEventReusedDoesNotResetPending(t *testing.T) {
	ev := mkRune('g')
	kb := &KeyBindings{sequenceState: sequenceState{
		vimMode:          true,
		sequencePrefixes: map[rune]struct{}{'g': {}},
		pending:          'g',
		inFlightEvent:    ev,
	}}

	innerSawPending := rune(0)
	fn := kb.WrapInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		innerSawPending = kb.pending
		return nil
	})

	out := fn(ev)
	assert.Nil(t, out, "matched second rune must be consumed")
	assert.Equal(t, rune('g'), innerSawPending, "inner must observe pending intact (not reset by stale check)")
	assert.Equal(t, rune(0), kb.pending, "pending cleared after successful sequence match")
	assert.Nil(t, kb.inFlightEvent)
}

// Escape while a prefix is pending must cancel the prefix and consume the
// event (return nil) so the Escape does not trigger other handlers.
func TestWrapInputCapture_EscapeCancelsPendingAndConsumes(t *testing.T) {
	kb := &KeyBindings{sequenceState: sequenceState{
		vimMode:          true,
		sequencePrefixes: map[rune]struct{}{'g': {}},
		pending:          'g',
	}}
	var notified []rune
	kb.OnPendingChanged = func(r rune) { notified = append(notified, r) }

	innerCalled := false
	fn := kb.WrapInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		innerCalled = true
		return ev
	})

	out := fn(mkKey(tcell.KeyEsc))
	assert.Nil(t, out, "Escape while pending must be consumed, not forwarded")
	assert.False(t, innerCalled, "inner handler must not be called")
	assert.Equal(t, rune(0), kb.pending, "pending must be cleared")
	assert.Equal(t, []rune{0}, notified, "OnPendingChanged(0) must fire")
}

// Escape with no pending prefix must be forwarded to inner as normal.
func TestWrapInputCapture_EscapeWithNoPendingForwards(t *testing.T) {
	kb := &KeyBindings{sequenceState: sequenceState{
		vimMode:          true,
		sequencePrefixes: map[rune]struct{}{'g': {}},
	}}

	innerCalled := false
	fn := kb.WrapInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		innerCalled = true
		return ev
	})

	out := fn(mkKey(tcell.KeyEsc))
	assert.NotNil(t, out, "Escape with no pending must be forwarded")
	assert.True(t, innerCalled)
}
