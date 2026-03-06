package effectiveness_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/effectiveness"
)

// evalLine builds a minimal evaluation log JSON line for testing.
func evalLine(memoryPath, outcome string) string {
	return fmt.Sprintf(
		`{"memory_path":%q,"outcome":%q,"evidence":"x","evaluated_at":"2024-01-01T00:00:00Z"}`,
		memoryPath,
		outcome,
	)
}

// T-112: Aggregate computes effectiveness from evaluation logs.
func TestComputer_Aggregate_ComputesEffectiveness(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const memA = "memories/mem-a.toml"

	// 3 session files, memory A evaluated 5 times: 3 followed, 1 contradicted, 1 ignored.
	files := map[string]string{
		"session1.jsonl": strings.Join([]string{
			evalLine(memA, "followed"),
			evalLine(memA, "followed"),
		}, "\n"),
		"session2.jsonl": strings.Join([]string{
			evalLine(memA, "contradicted"),
			evalLine(memA, "ignored"),
		}, "\n"),
		"session3.jsonl": evalLine(memA, "followed"),
	}

	dirEntries := fakeDirEntries(files)

	computer := effectiveness.New("/eval",
		effectiveness.WithReadDir(func(string) ([]os.DirEntry, error) {
			return dirEntries, nil
		}),
		effectiveness.WithReadFile(func(name string) ([]byte, error) {
			for filename, content := range files {
				if name == "/eval/"+filename {
					return []byte(content), nil
				}
			}

			return nil, os.ErrNotExist
		}),
	)

	stats, err := computer.Aggregate()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stats).To(gomega.HaveKey(memA))
	stat := stats[memA]
	g.Expect(stat.FollowedCount).To(gomega.Equal(3))
	g.Expect(stat.ContradictedCount).To(gomega.Equal(1))
	g.Expect(stat.IgnoredCount).To(gomega.Equal(1))
	g.Expect(stat.EffectivenessScore).To(gomega.BeNumerically("~", 60.0, 0.001))
}

// T-113: Missing evaluations directory returns empty map, no error.
func TestComputer_Aggregate_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	computer := effectiveness.New("/eval",
		effectiveness.WithReadDir(func(string) ([]os.DirEntry, error) {
			return nil, os.ErrNotExist
		}),
		effectiveness.WithReadFile(func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
	)

	stats, err := computer.Aggregate()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stats).To(gomega.BeEmpty())
}

// T-114: Aggregate skips malformed JSONL lines.
func TestComputer_Aggregate_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const memB = "memories/mem-b.toml"

	content := strings.Join([]string{
		evalLine(memB, "followed"),
		"not valid json at all",
		evalLine(memB, "followed"),
		evalLine(memB, "ignored"),
	}, "\n")

	files := map[string]string{"session.jsonl": content}
	dirEntries := fakeDirEntries(files)

	computer := effectiveness.New("/eval",
		effectiveness.WithReadDir(func(string) ([]os.DirEntry, error) {
			return dirEntries, nil
		}),
		effectiveness.WithReadFile(func(name string) ([]byte, error) {
			if name == "/eval/session.jsonl" {
				return []byte(content), nil
			}

			return nil, os.ErrNotExist
		}),
	)

	stats, err := computer.Aggregate()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stats).To(gomega.HaveKey(memB))
	stat := stats[memB]
	// 3 valid lines parsed (1 malformed skipped): 2 followed + 1 ignored.
	g.Expect(stat.FollowedCount).To(gomega.Equal(2))
	g.Expect(stat.IgnoredCount).To(gomega.Equal(1))
	g.Expect(stat.ContradictedCount).To(gomega.Equal(0))
}

// fakeDirEntry implements os.DirEntry for testing.
type fakeDirEntry struct {
	name string
}

var errNotImplemented = errors.New("not implemented")

func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, errNotImplemented }

func (f fakeDirEntry) IsDir() bool { return false }

func (f fakeDirEntry) Name() string { return f.name }

func (f fakeDirEntry) Type() os.FileMode { return 0 }

func fakeDirEntries(files map[string]string) []os.DirEntry {
	entries := make([]os.DirEntry, 0, len(files))
	for name := range files {
		entries = append(entries, fakeDirEntry{name: name})
	}

	return entries
}
