package embed_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/embed"
)

// TestContentHash_EpisodeSituationSensitivityProperty asserts that for an
// episode-shaped note (Text embeds the trimmed situation:), changing only the
// situation: frontmatter field changes the hash even when the body is
// byte-identical. Targets are space-free so the trim in Text cannot collapse
// two distinct draws to the same embedded text.
func TestContentHash_EpisodeSituationSensitivityProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		situationA := genFieldValue(rt, "situationA")
		situationB := genFieldValue(rt, "situationB")

		if situationA == situationB {
			return
		}

		body := genFieldValue(rt, "body")
		noteA := fmt.Appendf(nil, "---\ntype: episode\nsituation: %s\n---\n%s\n", situationA, body)
		noteB := fmt.Appendf(nil, "---\ntype: episode\nsituation: %s\n---\n%s\n", situationB, body)

		g.Expect(embed.ContentHash(noteA)).NotTo(Equal(embed.ContentHash(noteB)))
	})
}

// TestContentHash_FactBodySensitivityProperty asserts that for a fact- or
// feedback-shaped note (Text embeds the body, never the situation:), changing
// only the body changes the hash while the frontmatter is held fixed. Bodies
// are newline-free so ExtractBody's single leading-newline strip cannot
// collapse two distinct draws.
func TestContentHash_FactBodySensitivityProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		noteType := rapid.SampledFrom([]string{"fact", "feedback"}).Draw(rt, "type")
		bodyA := genFieldValue(rt, "bodyA")
		bodyB := genFieldValue(rt, "bodyB")

		if bodyA == bodyB {
			return
		}

		frontmatter := fmt.Sprintf("---\ntype: %s\nluhmann: \"1\"\n---\n", noteType)
		noteA := []byte(frontmatter + bodyA + "\n")
		noteB := []byte(frontmatter + bodyB + "\n")

		g.Expect(embed.ContentHash(noteA)).NotTo(Equal(embed.ContentHash(noteB)))
	})
}

// TestContentHash_IdempotentProperty asserts that hashing the same raw note
// bytes twice yields the identical sha256:-prefixed string, across the note
// shapes the embedder actually sees (episode, fact, feedback, and bare body).
func TestContentHash_IdempotentProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		raw := genRawNote(rt)

		first := embed.ContentHash(raw)
		second := embed.ContentHash(raw)

		g.Expect(second).To(Equal(first))
		g.Expect(first).To(HavePrefix("sha256:"))
	})
}

// genFieldValue draws a non-empty, space-free, newline-free token suitable for
// either a frontmatter field value or a body. Avoiding spaces sidesteps the
// trim in Text; avoiding newlines keeps frontmatter and body shapes intact.
func genFieldValue(rt *rapid.T, label string) string {
	return rapid.StringMatching(`[a-zA-Z0-9]{1,12}`).Draw(rt, label)
}

// genRawNote draws a raw note across the shapes the embedder sees: an
// episode (with situation:), a fact/feedback body note, or a bare body with no
// frontmatter at all.
func genRawNote(rt *rapid.T) []byte {
	shape := rapid.SampledFrom([]string{"episode", "fact", "feedback", "bare"}).Draw(rt, "shape")
	body := genFieldValue(rt, "rawBody")

	if shape == "bare" {
		return []byte(body + "\n")
	}

	if shape == "episode" {
		situation := genFieldValue(rt, "rawSituation")

		return fmt.Appendf(nil, "---\ntype: episode\nsituation: %s\n---\n%s\n", situation, body)
	}

	return fmt.Appendf(nil, "---\ntype: %s\nluhmann: \"1\"\n---\n%s\n", shape, body)
}
