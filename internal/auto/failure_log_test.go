package auto_test

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d36b4_ALogThatCannotBeWrittenChangesNeitherTheWarningNorTheJob(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Unwritable[world.LogPath] = true
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{}, errNonZero }

	done := failureLookup(t, world)

	g.Expect(done.Block).To(BeEmpty())
	g.Expect(world.Warnings).To(Equal([]string{skipped("prompt lookup failed: exit status 70", failureNext)}))
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("auto.log"))
}

func Test793d36b4_AFailureTheModHandsOverIsWordedAndLoggedLikeAnyOther(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	secret := "ghp_" + strings.Repeat("a1B2", 8)

	done := world.Do(t, map[string]any{
		"kind": "failure", "step": "indexing, curation and backlog capture", "sessionId": "s1",
		"text": "engram auto run exited 70: boom " + secret,
	})

	g.Expect(done.Block).To(BeEmpty())
	g.Expect(world.Warnings).To(Equal([]string{skipped(
		"indexing, curation and backlog capture failed: engram auto run exited 70: boom [redacted]", retried)}))
	g.Expect(world.Written[world.LogPath]).To(Equal(
		`2023-11-14T22:13:20Z session="s1" operation="indexing, curation and backlog capture" class=error ` +
			`reason="engram auto run exited 70: boom [redacted]"` + "\n"))
}

func Test793d36b4_EveryContainedFailureIsAppendedToTheLogWithItsTimeSessionOperationClassAndReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["ingest"] = func([]string) (auto.Output, error) {
		return auto.Output{}, errors.New("ingest: acquiring manifest lock: another ingest holds the manifest lock; skipped")
	}
	world.Files["/vault/"+curateOffer] = curateOpenFile
	world.Missing["/vault/walk"] = true

	world.Do(t, map[string]any{"kind": "housekeeping", "sessionId": "s1"})

	lines := strings.Split(strings.TrimSuffix(world.Written[world.LogPath], "\n"), "\n")
	g.Expect(lines).To(HaveLen(2))
	g.Expect(lines[0]).To(Equal(`2023-11-14T22:13:20Z session="s1" operation="indexing" class=busy ` +
		`reason="ingest: acquiring manifest lock: another ingest holds the manifest lock; skipped"`))
	g.Expect(lines[1]).To(HavePrefix(`2023-11-14T22:13:20Z session="s1" operation="curation" class=unavailable reason="`))
}

func Test793d36b4_TheLogIsCappedByDroppingItsOldestLines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	old := strings.Repeat(`2023-01-01T00:00:00Z session="-" operation="indexing" class=error reason="x"`+"\n", 14000)
	world.Files[world.LogPath] = old
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{}, errNonZero }

	g.Expect(len(old)).To(BeNumerically(">", 1<<20))

	failureLookup(t, world)

	log := world.Written[world.LogPath]
	g.Expect(len(log)).To(BeNumerically("<=", 768*1024))
	g.Expect(log).To(HavePrefix("2023-01-01T00:00:00Z session"))
	g.Expect(log).To(HaveSuffix(`class=error reason="query failed: exit status 70"` + "\n"))
	g.Expect(strings.Count(log, "\n")).To(BeNumerically("<", 14000))
}

func Test793d36b4_TheReasonInTheLogIsRedacted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	secret := "ghp_" + strings.Repeat("a1B2", 8)
	world.Process["query"] = func([]string) (auto.Output, error) {
		return auto.Output{}, errors.New("denied for " + secret)
	}

	failureLookup(t, world)

	g.Expect(world.Written[world.LogPath]).NotTo(ContainSubstring(secret))
	g.Expect(world.Written[world.LogPath]).To(ContainSubstring("denied for [redacted]"))
}

func Test793d36b4_TheWarningEndsWithThePathOfTheLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, home, logPath, shown string
	}{
		{"under the home directory", "/home/u", "/home/u/.local/share/engram/auto.log", "~/.local/share/engram/auto.log"},
		{"under XDG_DATA_HOME", "/home/u", "/data/engram/auto.log", "/data/engram/auto.log"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Home, world.LogPath = test.home, test.logPath
			world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{}, errNonZero }

			failureLookup(t, world)

			g.Expect(world.Warnings).To(HaveLen(1))
			g.Expect(world.Warnings[0]).To(HaveSuffix(" (details: " + test.shown + ")"))
			g.Expect(world.Written).To(HaveKey(test.logPath))
		})
	}
}
