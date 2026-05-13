package learnmarker_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/engram/internal/learnmarker"
)

func TestStateDirFromHome_DefaultsToLocalState(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := learnmarker.StateDirFromHome("/Users/joe", func(string) string { return "" })

	g.Expect(dir).To(gomega.Equal("/Users/joe/.local/state/engram"))
}

func TestStateDirFromHome_RespectsXDGStateHome(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	getenv := func(key string) string {
		if key == "XDG_STATE_HOME" {
			return "/custom/state"
		}
		return ""
	}

	dir := learnmarker.StateDirFromHome("/Users/joe", getenv)

	g.Expect(dir).To(gomega.Equal("/custom/state/engram"))
}

func TestMarkerPath_JoinsStateDirAndSlug(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := learnmarker.MarkerPath("/state/engram", "Users-joe-repos-foo")

	g.Expect(path).To(gomega.Equal("/state/engram/projects/Users-joe-repos-foo/last-learn-at"))
}
