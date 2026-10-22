package cli_test

import (
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// Test792b2c10a1_ShowParent_AsksTheParentOnly verifies 792b2c10a1: with ENGRAM_PARENT set, whatever the
// local vault holds (the ref present with any text, or absent), `engram show <ref> --parent` makes exactly
// one request, to the parent, for that ref and prints the parent's answer without a label.
func Test792b2c10a1_ShowParent_AsksTheParentOnly(t *testing.T) {
	t.Parallel()

	scratch := t.TempDir()

	var iteration atomic.Int64

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		ref := rapid.StringMatching(`[a-z0-9][a-z0-9.-]{0,20}`).Draw(rt, "ref")
		localPresent := rapid.Bool().Draw(rt, "localPresent")
		localBody := rapid.String().Draw(rt, "localBody")
		parentAnswer := rapid.String().Draw(rt, "parentAnswer")

		vault := filepath.Join(scratch, strconv.FormatInt(iteration.Add(1), 10))
		g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

		if localPresent {
			g.Expect(os.WriteFile(filepath.Join(vault, ref+".md"), []byte(localBody), 0o600)).To(Succeed())
		}

		recorder := &cls1FetchRecorder{answer: parentAnswer}

		stdout, stderr := executeCapturingBoth(rt, []string{"engram", "show", ref, "--parent", "--vault", vault},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Fetch = recorder.fetch
			})

		g.Expect(stderr).To(BeEmpty())
		g.Expect(recorder.calls()).To(Equal([]string{parentURL + "/show?note=" + ref}))
		g.Expect(stdout).To(Equal(parentAnswer), "the parent's answer, unlabeled")
	})
}
