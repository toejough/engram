package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestProperty_ParseAdoptFlags_RefusesAnyRepeatedKey covers the --adopt
// dedupe rule: for any list of well-formed "<key>=<note-ref>" entries, the
// parse succeeds exactly when no key repeats — mapping each key to its one
// ref — and fails with errDuplicateAdoptKey otherwise, whatever the refs
// and whatever the position of the repeat (a later flag never silently
// replaces an earlier one).
func TestProperty_ParseAdoptFlags_RefusesAnyRepeatedKey(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		keyGen := rapid.SampledFrom(
			[]string{"claude:curate", "claude:route", "claude:write-memory", "pi:learn"},
		)
		refGen := rapid.SampledFrom(
			[]string{"1036", "1045", "1049", "1053.2026-09-22.write-memory-worker"},
		)
		keys := rapid.SliceOfN(keyGen, 1, 6).Draw(rt, "keys")

		entries := make([]string, 0, len(keys))
		want := make(map[string]string, len(keys))
		repeated := false

		for _, key := range keys {
			ref := refGen.Draw(rt, "ref")
			entries = append(entries, key+"="+ref)

			if _, seen := want[key]; seen {
				repeated = true
			}

			want[key] = ref
		}

		got, err := cli.ExportParseAdoptFlags(entries)

		if repeated {
			g.Expect(err).To(MatchError(cli.ErrDuplicateAdoptKeyForTest))

			return
		}

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(got).To(Equal(want))
	})
}
