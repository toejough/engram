package agent_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/agent"
)

func TestActiveWorkerCount_CountsActiveAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "ACTIVE"},
			{Name: "exec-2", State: "SILENT"},
			{Name: "exec-3", State: "ACTIVE"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(2))
}

func TestActiveWorkerCount_CountsStartingAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "STARTING"},
			{Name: "exec-2", State: "STARTING"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(2))
}

func TestActiveWorkerCount_EmptyStateFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(agent.ActiveWorkerCount(agent.StateFile{})).To(Equal(0))
}

func TestActiveWorkerCount_IgnoresSilentDeadUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "SILENT"},
			{Name: "exec-2", State: "DEAD"},
			{Name: "exec-3", State: "UNKNOWN"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(0))
}

func TestAddAgent_AppendsToEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{}
	rec := agent.AgentRecord{Name: "planner-1", State: "STARTING"}
	got := agent.AddAgent(stateFile, rec)
	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].Name).To(Equal("planner-1"))
	g.Expect(stateFile.Agents).To(BeEmpty()) // original unchanged
}

func TestAddHold_AppendsHold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{}
	got := agent.AddHold(stateFile, agent.HoldEntry{HoldID: "h1", Target: "executor-1"})
	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("h1"))
}

func TestParseStateFile_Empty_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile, err := agent.ParseStateFile(nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stateFile.Agents).To(BeEmpty())
	g.Expect(stateFile.Holds).To(BeEmpty())
}

func TestParseStateFile_WithAgents_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	spawnedAt := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	original := agent.StateFile{
		Agents: []agent.AgentRecord{
			{
				Name:           "executor-1",
				PaneID:         "main:1.2",
				SessionID:      "abc123",
				State:          "ACTIVE",
				SpawnedAt:      spawnedAt,
				ArgumentWith:   "",
				ArgumentCount:  0,
				ArgumentThread: "",
			},
		},
	}

	data, marshalErr := agent.MarshalStateFile(original)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	got, parseErr := agent.ParseStateFile(data)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].Name).To(Equal("executor-1"))
	g.Expect(got.Agents[0].PaneID).To(Equal("main:1.2"))
	g.Expect(got.Agents[0].State).To(Equal("ACTIVE"))
	g.Expect(got.Agents[0].SpawnedAt.UTC()).To(Equal(spawnedAt))
}

func TestParseStateFile_WithHolds_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	original := agent.StateFile{
		Holds: []agent.HoldEntry{
			{
				HoldID:     "uuid-1234",
				Holder:     "lead",
				Target:     "executor-1",
				Condition:  "lead-release:phase3",
				Tag:        "phase3",
				AcquiredTS: acquiredAt,
			},
		},
	}

	data, marshalErr := agent.MarshalStateFile(original)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	got, parseErr := agent.ParseStateFile(data)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("uuid-1234"))
	g.Expect(got.Holds[0].AcquiredTS.UTC()).To(Equal(acquiredAt))
}

func TestRemoveAgent_MissingName_NoChange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{Agents: []agent.AgentRecord{{Name: "executor-1"}}}
	got := agent.RemoveAgent(stateFile, "nonexistent")
	g.Expect(got.Agents).To(HaveLen(1))
}

func TestRemoveAgent_RemovesNamedAgent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "executor-1"},
			{Name: "reviewer-1"},
		},
	}
	got := agent.RemoveAgent(stateFile, "executor-1")
	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].Name).To(Equal("reviewer-1"))
}

func TestRemoveHold_RemovesById(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Holds: []agent.HoldEntry{{HoldID: "h1"}, {HoldID: "h2"}},
	}
	got := agent.RemoveHold(stateFile, "h1")
	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("h2"))
}
