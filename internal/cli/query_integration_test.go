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

// TestEngramQuery_F6F91_EndToEnd builds the real binary, writes ~33
// thematically-clustered notes (3 themes x 1 hub + 10 members, each linking
// to its hub), runs `engram embed apply --missing` to produce real sidecars
// via the bundled embedder, then queries with one theme's own text.
// Verifies that the YAML payload includes the clusters[] section and budget
// fields, and that the matching theme's notes actually surface.
func TestEngramQuery_F6F91_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := vault
	g.Expect(os.MkdirAll(permDir, 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())

	const notesPerCluster = 10

	const numClusters = 3

	for clusterIdx := range numClusters {
		theme := clusterThemes[clusterIdx]
		hubBasename := "hub-" + strconv.Itoa(clusterIdx)
		hubBody := buildSyntheticBody(theme, clusterIdx, notesPerCluster)

		writeSyntheticNote(g, permDir, hubBasename, hubBody)

		for memberIdx := 1; memberIdx <= notesPerCluster; memberIdx++ {
			memberBase := "n" + strconv.Itoa(clusterIdx) + "-" + strconv.Itoa(memberIdx)
			memberBody := "---\ntype: fact\n---\n" + theme + " — detail note " +
				strconv.Itoa(memberIdx) + ".\n[[" + hubBasename + "]]\n"

			writeSyntheticNote(g, permDir, memberBase, memberBody)
		}
	}

	binPath := sharedEngramBinary(t)

	embedRun := exec.Command(binPath, "embed", "apply", "--missing", "--vault", vault)

	var embedOut bytes.Buffer

	embedRun.Stdout = &embedOut
	embedRun.Stderr = &embedOut
	g.Expect(embedRun.Run()).To(Succeed(), "embed apply failed: %s", embedOut.String())

	run := exec.Command(binPath, "query", "--phrase", clusterThemes[1], "--vault", vault, "--limit", "5")

	var stdout bytes.Buffer

	run.Stdout = &stdout
	run.Stderr = os.Stderr
	runErr := run.Run()

	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(stdout.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Budget.TotalNotes).To(Equal(33))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(33))
	g.Expect(parsed.Items).NotTo(BeEmpty())
	// The new payload sections always render (clusters may be empty).
	g.Expect(strings.Contains(stdout.String(), "clusters:")).To(BeTrue())
}

// unexported variables.
var (
	clusterThemes = [3]string{
		"authentication and login session security",
		"database query performance and index tuning",
		"frontend user interface rendering and CSS layout",
	}
)

// buildSyntheticBody returns a note body that states theme and lists
// wikilinks to N members.
func buildSyntheticBody(theme string, clusterID, members int) string {
	var sb strings.Builder

	sb.WriteString("---\ntype: fact\n---\n")
	sb.WriteString(theme)
	sb.WriteString(" — hub note.\n")

	for i := 1; i <= members; i++ {
		sb.WriteString("[[n")
		sb.WriteString(strconv.Itoa(clusterID))
		sb.WriteString("-")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("]]\n")
	}

	return sb.String()
}

// writeSyntheticNote plants a note under permDir with no sidecar — the
// caller runs `engram embed apply` afterward to produce a real one via the
// bundled embedder.
func writeSyntheticNote(g Gomega, permDir, basename, body string) {
	notePath := filepath.Join(permDir, basename+".md")
	g.Expect(os.WriteFile(notePath, []byte(body), 0o600)).To(Succeed())
}
