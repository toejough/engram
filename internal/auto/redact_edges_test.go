package auto_test

import (
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

// 793d36b4 (Security): a credential must not show in any line the System writes or shows, wherever a text was cut:
// the 200 characters of an operation's error, the stderr the mod cut, the window and the 80-character ends of a reply
// the mod could not read. A fragment of leakFragment characters of a secret is a leak.
func Test793d36b4_NoCredentialShowsInAnyLineWhereverTheTextWasCut(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		secret, context := drawSecret(rt)
		world := NewWorld()

		switch rapid.IntRange(0, 4).Draw(rt, "path") {
		case 0: // the error of an operation, cut at 200 characters before the failure is worded
			filler := strings.Repeat("x ", rapid.IntRange(0, 110).Draw(rt, "filler"))
			world.Process["query"] = func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New(filler + context + " tail")
			}

			failureLookup(t, world)
		case 1: // the text the mod cut, handed over as it was cut
			full := strings.Repeat("x ", rapid.IntRange(0, 110).Draw(rt, "filler")) + context + " tail"
			text := full[:rapid.IntRange(0, len(full)).Draw(rt, "cut")]

			world.Do(t, map[string]any{"kind": "failure", "step": "indexing", "text": text})
		default:
			world.Do(t, map[string]any{
				"kind": "failure", "step": "indexing", "text": "JSON Parse error: Unterminated string",
				"unreadable": drawReply(rt, secret, context),
			})
		}

		shown := strings.Join(append(append(world.Warnings, world.Logs...), world.Written[world.LogPath]), "\n")
		for start := 0; start+leakFragment <= len(secret); start++ {
			if fragment := secret[start : start+leakFragment]; strings.Contains(shown, fragment) {
				rt.Fatalf("the fragment %q of the credential %q shows in %q", fragment, secret, shown)
			}
		}
	})
}

// unexported constants.
const (
	leakFragment   = 16
	redactorWindow = 8192
)

// drawReply is a reply that holds the credential and was cut where the property draws: inside the credential at the
// end (the process cut it), or at an edge of the window the engine redacts.
func drawReply(rt *rapid.T, secret, context string) string {
	inside := rapid.IntRange(0, len(secret)).Draw(rt, "inside")
	body := func(n int) string { return strings.Repeat(`{"out":"m"},`, n/12+1)[:n] }

	switch rapid.IntRange(0, 2).Draw(rt, "side") {
	case 0: // the reply is cut in the credential
		return body(rapid.IntRange(0, 4*redactorWindow).Draw(rt, "before")) + context[:strings.Index(context, secret)+inside]
	case 1: // the credential straddles the end of the window the start of the reply is redacted in
		return body(redactorWindow-inside) + context + body(2*redactorWindow)
	}

	// the credential straddles the start of the window the end of the reply is redacted in
	return body(2*redactorWindow) + context + body(redactorWindow-(len(context)-inside))
}

func drawSecret(rt *rapid.T) (secret, context string) {
	mixed := rapid.StringMatching(`[A-Za-z0-9]{30,48}`).Draw(rt, "token")
	secret = "a1" + mixed

	switch rapid.IntRange(0, 4).Draw(rt, "shape") {
	case 0:
		return secret, secret
	case 1:
		return secret, "ghp_" + secret
	case 2:
		return secret, "api_key=" + secret
	case 3:
		return secret, "Bearer " + secret
	}

	return secret, "sk-" + secret
}
