package auto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/toejough/engram/internal/auto"

	. "github.com/onsi/gomega"
)

func Test793d24b2_CompactingASubagentLeavesTheLedgerOfTheMainAgentAndEmptiesItsOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := holdNotesOfTwoAgents(t)

	world.Do(t, map[string]any{"kind": "compact", "agent": "a1"})

	ledgers := ledgersOf(t, world)
	g.Expect(ledgers["main"]).NotTo(BeEmpty(), "the main agent's ledger was emptied by the subagent's compaction")
	g.Expect(ledgers["a1"]).To(BeEmpty(), "the subagent's ledger kept the notes it lost")
}

func Test793d24b2_CompactingTheMainAgentLeavesTheLedgerOfASubagent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := holdNotesOfTwoAgents(t)

	world.Do(t, map[string]any{"kind": "compact"})

	ledgers := ledgersOf(t, world)
	g.Expect(ledgers["main"]).To(BeEmpty())
	g.Expect(ledgers["a1"]).NotTo(BeEmpty(), "the subagent's ledger was emptied by the main agent's compaction")
}

func holdNotesOfTwoAgents(t *testing.T) *World {
	t.Helper()

	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	for _, agent := range []string{"", "a1"} {
		job := map[string]any{
			"kind": "lookup", "moment": "prompt", "text": "Is the rename done?", "step": "the prompt lookup",
		}
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

// ledgersOf is the ledger of each agent, as the job handed it back.
func ledgersOf(t *testing.T, world *World) map[string][]string {
	t.Helper()

	ledgers := map[string][]string{}
	if err := json.Unmarshal(world.Ledger, &ledgers); err != nil {
		t.Fatalf("the ledger cannot be read: %v", err)
	}

	return ledgers
}
