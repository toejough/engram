package cli_test

import (
	"bytes"
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestRunQuery_RunbookCompetesInMainMatchedSet proves runbook notes receive no
// exclusion treatment (unlike qa-question) and rank purely by situation
// similarity, identical to fact/feedback — design.md Decision 2/3,
// recall-runbook-surfacing spec.
func TestRunQuery_RunbookCompetesInMainMatchedSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.2026-08-24.release-runbook.md",
		"---\ntype: runbook\nsituation: releasing a Go module version\ndone_when: the tag is pushed\n---\n\n"+
			"1. Tag the release\n2. Push the tag\n")
	plantNoteWithSidecar(t, memFS, vault, "2.2026-08-24.release-fact.md",
		"---\ntype: fact\nsituation: releasing a Go module version\n---\n\n"+
			"Information learned: when releasing a Go module version, tags matter.\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"releasing a Go module version"}, VaultPath: vault, Limit: 20},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	var runbookFound, factFound bool

	var runbookScore, factScore float32

	for _, item := range parsed.Items {
		switch item.Kind {
		case "runbook":
			runbookFound = true
			runbookScore = item.Score
		case "fact":
			factFound = true
			factScore = item.Score
		}
	}

	g.Expect(runbookFound).To(BeTrue(), "runbook note must appear in the matched set, not excluded")
	g.Expect(factFound).To(BeTrue())

	// Identical situation text on both notes → identical score. No
	// kind-specific boost or penalty differentiates a runbook from a fact.
	g.Expect(runbookScore).To(BeNumerically("~", factScore, 0.0001))

	foundInCandidates := false

	for _, c := range parsed.Clusters {
		for _, cand := range c.CandidateL2s {
			if cand.Path == "1.2026-08-24.release-runbook.md" {
				foundInCandidates = true
			}
		}
	}

	g.Expect(foundInCandidates).To(BeTrue(), "runbook note must be nominated in candidate_l2s, like fact/feedback")
}
