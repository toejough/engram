package internal_test

// Tests for ARCH-9: Hook Shell Scripts
// Traces through ARCH-9 to REQ-1, REQ-13, DES-6, DES-8

import "testing"

// T-42: The Stop hook script invokes both "engram extract" and "engram catchup"
// with the session transcript path.
// Traces: ARCH-9
func TestHookScript_StopInvokesExtractAndCatchup(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-43: The UserPromptSubmit hook script invokes "engram correct" with the
// user's message.
// Traces: ARCH-9
func TestHookScript_UserPromptSubmitInvokesCorrect(t *testing.T) {
	t.Skip("RED: not implemented")
}
