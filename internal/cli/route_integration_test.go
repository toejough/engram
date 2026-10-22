//go:build integration

package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
)

// Test792a17n1a_RouteLookup_AggregateSurfacesAmongTwentyEvidenceNotes runs the
// route skill's aggregate lookup, `engram query --lazy-chunks
// --phrase="route evidence <work-kind> tier tally"`, through the real binary
// and the bundled embedder over a vault on disk that holds a work-kind's
// aggregate fact and 20 near-identical evidence notes (same slug, situation,
// subject and predicate, differing in the dispatch tier and outcome), the
// shapes `engram learn fact` writes for route's two write sites. It asserts the
// aggregate's path surfaces in the items or in a cluster's candidate_l2s, the
// two places route's drowning audit reads.
//
// The vault is composed by hand and embedded with one `embed apply --missing`
// because learning 21 notes through the binary loads the embedding model 21
// times (recorded under 861d317).
func Test792a17n1a_RouteLookup_AggregateSurfacesAmongTwentyEvidenceNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	aggregateBasename := route1WriteWorkKindVault(g, vault)

	binPath := sharedEngramBinary(t)
	chunksDir := t.TempDir()

	embedRun := exec.Command(binPath, "embed", "apply", "--missing", "--vault", vault)
	embedRun.Env = append(envWithoutEngramRemotes(), isolatedHomeEnv(t)...)

	var embedOut bytes.Buffer

	embedRun.Stdout = &embedOut
	embedRun.Stderr = &embedOut
	g.Expect(embedRun.Run()).To(Succeed(), "embed apply failed: %s", embedOut.String())

	lookup := exec.Command(binPath, "query", "--lazy-chunks",
		"--phrase=route evidence "+route1WorkKind+" tier tally",
		"--vault", vault, "--chunks-dir", chunksDir)
	lookup.Env = append(envWithoutEngramRemotes(), isolatedHomeEnv(t)...)

	var stdout, stderr bytes.Buffer

	lookup.Stdout = &stdout
	lookup.Stderr = &stderr
	lookupErr := lookup.Run()
	g.Expect(lookupErr).NotTo(HaveOccurred(), "query failed: %s", stderr.String())

	if lookupErr != nil {
		return
	}

	var parsed queryParsed

	parseErr := yaml.Unmarshal(stdout.Bytes(), &parsed)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Budget.TotalNotes).To(Equal(route1EvidenceNotes + 1))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(route1EvidenceNotes + 1))

	surfaced := route1SurfacedBasenames(parsed)

	g.Expect(surfaced).To(ContainElement(aggregateBasename),
		"aggregate %s missing from the lookup results %v", aggregateBasename, surfaced)
}

// unexported constants.
const (
	route1CreatedDate   = "2026-10-22"
	route1EvidenceNotes = 20
	route1FileMode      = 0o600
	route1WorkKind      = "go-package-implementation"
)

// route1AggregateNote is the frontmatter of the aggregate fact, in the key
// order `engram learn fact` writes; it carries no tags.
type route1AggregateNote struct {
	Type      string `yaml:"type"`
	Situation string `yaml:"situation"`
	Subject   string `yaml:"subject"`
	Predicate string `yaml:"predicate"`
	Object    string `yaml:"object"`
	Luhmann   string `yaml:"luhmann"`
	Created   string `yaml:"created"`
	Source    string `yaml:"source"`
}

// route1EvidenceNote is the frontmatter of one dispatch's evidence fact, in the
// key order `engram learn fact` writes, with the three categorical tags.
type route1EvidenceNote struct {
	Type      string   `yaml:"type"`
	Situation string   `yaml:"situation"`
	Subject   string   `yaml:"subject"`
	Predicate string   `yaml:"predicate"`
	Object    string   `yaml:"object"`
	Luhmann   string   `yaml:"luhmann"`
	Created   string   `yaml:"created"`
	Source    string   `yaml:"source"`
	Tags      []string `yaml:"tags"`
}

// route1SurfacedBasenames lists the basenames of the paths in the payload's
// items and in every cluster's candidate_l2s.
func route1SurfacedBasenames(parsed queryParsed) []string {
	basenames := make([]string, 0, len(parsed.Items))

	for _, item := range parsed.Items {
		basenames = append(basenames, filepath.Base(item.Path))
	}

	for _, cluster := range parsed.Clusters {
		for _, candidate := range cluster.CandidateL2s {
			basenames = append(basenames, filepath.Base(candidate.Path))
		}
	}

	return basenames
}

// route1WriteNote renders frontmatter and the fact body formula into
// <luhmann>.<date>.<slug>.md under the vault and returns the basename.
func route1WriteNote(
	g Gomega, vault, luhmann, slug, situation, subject, predicate, object string, frontmatter any,
) string {
	frontmatterText, err := yaml.Marshal(frontmatter)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return ""
	}

	body := "Information learned: when in " + situation + ", " + subject + " " + predicate + " " + object + ".\n"
	content := "---\n" + string(frontmatterText) + "---\n\n" + body
	basename := luhmann + "." + route1CreatedDate + "." + slug + ".md"

	g.Expect(os.WriteFile(filepath.Join(vault, basename), []byte(content), route1FileMode)).To(Succeed())

	return basename
}

// route1WriteWorkKindVault plants the aggregate `route-evidence-<work-kind>`
// and the evidence notes `route-dispatch-<work-kind>` for the work-kind and
// returns the aggregate's basename.
func route1WriteWorkKindVault(g Gomega, vault string) string {
	evidenceSlug := "route-dispatch-" + route1WorkKind
	evidenceWikilinks := make([]string, 0, route1EvidenceNotes)
	evidenceSituation := "routing " + route1WorkKind + " work"
	evidenceSubject := route1WorkKind + " dispatch at cheap (haiku @ dispatch)"
	evidencePredicate := "resolved as"

	// Luhmann ID 1 is the aggregate (position top); the evidence notes follow.
	const firstEvidenceID = 2

	for index := range route1EvidenceNotes {
		luhmann := strconv.Itoa(firstEvidenceID + index)
		outcome := "pass"

		if index%4 == 3 { // a mix of passes and fails, as dispatches produce
			outcome = "fail"
		}

		object := outcome + " per review verdict; why: cold-start default; escalation: none; duration: " +
			strconv.Itoa(1000+index) + " ms; cost: " + strconv.Itoa(2000+index) + " tok (no rate on hand)"

		basename := route1WriteNote(g, vault, luhmann, evidenceSlug,
			evidenceSituation, evidenceSubject, evidencePredicate, object,
			route1EvidenceNote{
				Type: "fact", Situation: evidenceSituation, Subject: evidenceSubject,
				Predicate: evidencePredicate, Object: object, Luhmann: luhmann,
				Created: route1CreatedDate, Source: "route dispatch record, engram, " + route1CreatedDate,
				Tags: []string{"work-kind/" + route1WorkKind, "tier/cheap", "outcome/" + outcome},
			})

		evidenceWikilinks = append(evidenceWikilinks, "[["+strings.TrimSuffix(basename, ".md")+"]]")
	}

	aggregateSituation := "routing " + route1WorkKind + " work: which tier the evidence supports"
	aggregateSubject := "route evidence for " + route1WorkKind
	aggregatePredicate := "tallies"
	aggregateObject := "cheap 15/20 as of " + route1CreatedDate + " — evidence: " + strings.Join(evidenceWikilinks, ", ")

	return route1WriteNote(g, vault, "1", "route-evidence-"+route1WorkKind,
		aggregateSituation, aggregateSubject, aggregatePredicate, aggregateObject,
		route1AggregateNote{
			Type: "fact", Situation: aggregateSituation, Subject: aggregateSubject,
			Predicate: aggregatePredicate, Object: aggregateObject, Luhmann: "1",
			Created: route1CreatedDate, Source: "route dispatch record, engram, " + route1CreatedDate,
		})
}
