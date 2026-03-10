package instruct_test

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/instruct"
)

// T-215: Scanner extracts instructions from memory source only.
func TestScanAll_ExtractsMemorySources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/data/memories/mem1.toml": "memory one",
		"/data/memories/mem2.toml": "memory two",
		"/data/memories/mem3.toml": "memory three",
	}

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			switch pattern {
			case "/data/memories/*.toml":
				return []string{
					"/data/memories/mem1.toml",
					"/data/memories/mem2.toml",
					"/data/memories/mem3.toml",
				}, nil
			default:
				return nil, nil
			}
		},
		EffData: map[string]float64{},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// 3 memories only
	const expectedCount = 3
	g.Expect(items).To(HaveLen(expectedCount))

	for _, item := range items {
		g.Expect(item.Source).To(Equal(instruct.SourceMemory))
		g.Expect(item.Path).NotTo(BeEmpty())
		g.Expect(item.Content).NotTo(BeEmpty())
	}
}

// T-216: Scanner joins effectiveness data to instructions.
func TestScanAll_JoinsEffectivenessData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/data/memories/mem1.toml": "memory with data",
		"/data/memories/mem2.toml": "memory without data",
	}

	const mem1Score = 85.5

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{
					"/data/memories/mem1.toml",
					"/data/memories/mem2.toml",
				}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{
			"/data/memories/mem1.toml": mem1Score,
		},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Find items by path
	var mem1, mem2 *instruct.InstructionItem

	for idx := range items {
		switch items[idx].Path {
		case "/data/memories/mem1.toml":
			mem1 = &items[idx]
		case "/data/memories/mem2.toml":
			mem2 = &items[idx]
		}
	}

	g.Expect(mem1).NotTo(BeNil())

	if mem1 != nil {
		g.Expect(mem1.EffectivenessScore).To(Equal(mem1Score))
	}

	g.Expect(mem2).NotTo(BeNil())

	if mem2 != nil {
		g.Expect(mem2.EffectivenessScore).To(Equal(0.0))
	}
}

// T-216b: Scanner skips files with empty content.
func TestScanAll_SkipsEmptyFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("   "), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/mem1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(items).To(BeEmpty())
}

// T-216a: Scanner skips files that fail to read.
func TestScanAll_SkipsUnreadableFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scanner := &instruct.Scanner{
		ReadFile: func(_ string) ([]byte, error) {
			return nil, errors.New("read error")
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{"/data/memories/mem1.toml"}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(items).To(BeEmpty())
}
