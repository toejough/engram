package cycle_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// realPersisterAdapter wraps tomlwriter to persist candidate memories to disk
// under dataDir. Used in the planted-token integration test.
type realPersisterAdapter struct {
	dataDir string
}

func (a *realPersisterAdapter) WriteFeedback(
	_ context.Context,
	situation, behavior, impact, action string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: 2,
		Source:        "agent",
		Situation:     situation,
		Type:          "feedback",
		Content: memory.ContentFields{
			Behavior: behavior,
			Impact:   impact,
			Action:   action,
		},
	}

	writer := tomlwriter.New()

	path, err := writer.Write(rec, situation, a.dataDir)
	if err != nil {
		return "", false, err
	}

	return memory.NameFromPath(path), true, nil
}

func (a *realPersisterAdapter) WriteFact(
	_ context.Context,
	situation, subject, predicate, object string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: 2,
		Source:        "agent",
		Situation:     situation,
		Type:          "fact",
		Content: memory.ContentFields{
			Subject:   subject,
			Predicate: predicate,
			Object:    object,
		},
	}

	writer := tomlwriter.New()

	path, err := writer.Write(rec, situation, a.dataDir)
	if err != nil {
		return "", false, err
	}

	return memory.NameFromPath(path), true, nil
}

// recallReportingFakeRecaller returns a canned report for known queries.
type recallReportingFakeRecaller struct {
	reports map[string]string
}

func (r *recallReportingFakeRecaller) Recall(
	_ context.Context,
	_, query string,
) (string, error) {
	report, ok := r.reports[query]
	if !ok {
		return "", errors.New("unknown query")
	}

	return report, nil
}

// TestE2E_LearnedMemoryPersistsAndPlantedTokenSurfacesInRecall verifies the
// full cycle.Cycle.Run path with real tomlwriter-based persistence and a stub
// LLM/recaller:
//  1. The learn step persists a candidate memory to disk via tomlwriter.
//  2. The recall step receives a query and includes a report mentioning the token.
//  3. The Output contains both the persisted learned memory and the recalled report.
func TestE2E_LearnedMemoryPersistsAndPlantedTokenSurfacesInRecall(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const plantedToken = "INTEGRATION-TOKEN-91827364"

	dataDir := t.TempDir()

	// Call 0 returns a learn-candidate JSON array; call 1 returns a query string.
	llmA := `[{"type":"feedback","situation":"asked about the integration token",` +
		`"behavior":"queried","impact":"got token","action":"return ` + plantedToken + `"}]`
	llmB := `query for integration token`
	runner := &fakeRunner{responses: []string{llmA, llmB}}

	transcripts := &fakeTranscript{content: "USER: query about integration\nASSISTANT: replying"}
	persister := &realPersisterAdapter{dataDir: dataDir}
	recaller := &recallReportingFakeRecaller{
		reports: map[string]string{
			"query for integration token": "Recalled memory: " + plantedToken + " was the planted token",
		},
	}

	cyc := cycle.New(runner, transcripts, persister, recaller)

	out, err := cyc.Run(context.Background(), "/tmp/projectdir")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The learn candidate was persisted via the real tomlwriter adapter.
	g.Expect(out.Learned).To(HaveLen(1))
	g.Expect(out.Learned[0].Content.Action).To(ContainSubstring(plantedToken))

	// The recall step produced a report containing the planted token.
	g.Expect(out.Recalled).To(HaveLen(1))
	g.Expect(out.Recalled[0].Report).To(ContainSubstring(plantedToken))
}
