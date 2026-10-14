//go:build refactorred

package auto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/toejough/engram/internal/auto"

	. "github.com/onsi/gomega"
)

func Test793d24b2_RefactorredCompactingASubagentLeavesTheLedgerOfTheMainAgentAndEmptiesItsOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := rrHoldNotes(t)

	world.Do(t, map[string]any{"kind": "compact", "agent": "a1"})

	ledgers := rrLedgers(t, world)
	g.Expect(ledgers["main"]).NotTo(BeEmpty(), "the main agent's ledger was emptied by the subagent's compaction")
	g.Expect(ledgers["a1"]).To(BeEmpty(), "the subagent's ledger kept the notes it lost")
}

func Test793d24b2_RefactorredCompactingTheMainAgentLeavesTheLedgerOfASubagent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := rrHoldNotes(t)

	world.Do(t, map[string]any{"kind": "compact"})

	ledgers := rrLedgers(t, world)
	g.Expect(ledgers["main"]).To(BeEmpty())
	g.Expect(ledgers["a1"]).NotTo(BeEmpty(), "the subagent's ledger was emptied by the main agent's compaction")
}

func rrHoldNotes(t *testing.T) *World {
	t.Helper()

	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	for _, agent := range []string{"", "a1"} {
		job := map[string]any{"kind": "lookup", "moment": "prompt", "text": "Is the rename done?", "step": "the prompt lookup"}
		if agent != "" {
			job["agent"] = agent
		}

		done := world.Do(t, job)
		if !strings.Contains(done.Block, "note-n") {
			t.Fatalf("the note was not delivered to %q", agent)
		}
	}

	return world
}

// 793d24b2: the job `compact` empties the ledger of the main agent whoever was compacted: after a subagent's
// compaction a note the main agent still holds is delivered to it a second time, and the subagent's own ledger keeps
// notes it lost. The goal: a job `compact` has an `agent` field (empty means main) and empties that agent's ledger.
func rrLedgers(t *testing.T, world *World) map[string][]string {
	t.Helper()

	ledgers := map[string][]string{}
	if err := json.Unmarshal(world.Ledger, &ledgers); err != nil {
		t.Fatalf("the ledger cannot be read: %v", err)
	}

	return ledgers
}
