//go:build targ && integration

package dev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test855a3r1a_AStatusSaveRequestWritesNoNote reads the learn skill the repository ships: a save request for status
// is not durable, so the skill says it is not saved and where it belongs.
func Test855a3r1a_AStatusSaveRequestWritesNoNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, "agent-instructions/skills/learn/SKILL.md")

	g.Expect(text).To(ContainSubstring("A save-request for status gets none (the durable test above): tell the user."))
	g.Expect(text).To(ContainSubstring("say that it was not saved to the vault and where it belongs " +
		"(the change's docs or the project's memory)"))
}

// Test855a3s1a_ADifferentClaimOnTheSameSituationIsAbsentAndNotNear reads the recall skill the repository ships: a
// cluster whose principle is a different claim is absent, however close its situation.
func Test855a3s1a_ADifferentClaimOnTheSameSituationIsAbsentAndNotNear(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, "agent-instructions/skills/recall/SKILL.md")

	g.Expect(text).To(ContainSubstring("A cluster whose principle is a different claim is never near, " +
		"however close the situation: it is **Absent**"))
}

// Test855a3s1b_AClusterThatIsNotDurableWritesNothing reads the recall skill the repository ships: a cluster is written
// as a note only when its principle is durable.
func Test855a3s1b_AClusterThatIsNotDurableWritesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, "agent-instructions/skills/recall/SKILL.md")

	g.Expect(text).To(ContainSubstring("Only when the principle is **durable**"))
	g.Expect(text).To(ContainSubstring("otherwise write nothing"))
}

// Test855a3t1a_AMessageThatArrivedWithADeliveredBlockIsNotQueriedForAgain reads the shim guidance the repository ships:
// a message that arrived with a delivered block is not queried for again.
func Test855a3t1a_AMessageThatArrivedWithADeliveredBlockIsNotQueriedForAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, "agent-instructions/guidance/shim.md")

	g.Expect(text).To(ContainSubstring("a block opening `[engram memory]` attached to the user's message means the mod " +
		"already looked up memory for that message, so do not run this query again for that message."))
}

// shippedText returns the text of a file the repository ships, white space collapsed to single spaces.
func shippedText(t *testing.T, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot, relativePath))
	if err != nil {
		t.Fatalf("reading %s: %v", relativePath, err)
	}

	return strings.Join(strings.Fields(string(content)), " ")
}
