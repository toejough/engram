package registry_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

func TestClaudeMDExtractor_Extract(t *testing.T) {
	t.Parallel()

	t.Run("splits on headings and bullets", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := `# Project Rules

## Code Quality

- Use targ for all builds
- Never run go test directly

## Testing

- Always use t.Parallel()
`
		ext := registry.ClaudeMDExtractor{
			Content:    content,
			SourcePath: "CLAUDE.md",
		}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(3))
		g.Expect(entries[0].ID).To(Equal("claude-md:CLAUDE.md:use-targ-for-all"))
		g.Expect(entries[0].SourceType).To(Equal("claude-md"))
		g.Expect(entries[0].SourcePath).To(Equal("CLAUDE.md"))
		g.Expect(entries[0].Title).To(Equal("Use targ for all builds"))
		g.Expect(entries[0].ContentHash).To(Equal(contentHash("Use targ for all builds")))

		g.Expect(entries[1].ID).To(Equal("claude-md:CLAUDE.md:never-run-go-test"))
		g.Expect(entries[2].ID).To(Equal("claude-md:CLAUDE.md:always-use-tparallel"))
	})

	t.Run("empty content produces zero entries", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.ClaudeMDExtractor{Content: "", SourcePath: "CLAUDE.md"}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(BeEmpty())
	})

	t.Run("stable IDs across extractions", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := "- Always wrap errors with context\n"
		ext := registry.ClaudeMDExtractor{Content: content, SourcePath: "CLAUDE.md"}

		entries1, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		entries2, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries1[0].ID).To(Equal(entries2[0].ID))
	})

	t.Run("handles bold prefixed bullets", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := "- **DI everywhere:** No function calls os directly\n"
		ext := registry.ClaudeMDExtractor{Content: content, SourcePath: "CLAUDE.md"}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(1))
		g.Expect(entries[0].ID).To(Equal("claude-md:CLAUDE.md:di-everywhere-no-function"))
	})
}

func TestMemoryMDExtractor_Extract(t *testing.T) {
	t.Parallel()

	t.Run("extracts bullets with memory-md source type", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := `# Memory

## Preferences

- Always use bun for JS
- Never auto-commit
`
		ext := registry.MemoryMDExtractor{
			Content:    content,
			SourcePath: "MEMORY.md",
		}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(2))
		g.Expect(entries[0].ID).To(Equal("memory-md:MEMORY.md:always-use-bun-for"))
		g.Expect(entries[0].SourceType).To(Equal("memory-md"))
		g.Expect(entries[1].ID).To(Equal("memory-md:MEMORY.md:never-autocommit"))
	})

	t.Run("empty content produces zero entries", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.MemoryMDExtractor{Content: "", SourcePath: "MEMORY.md"}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(BeEmpty())
	})
}

func TestRuleExtractor_Extract(t *testing.T) {
	t.Parallel()

	t.Run("one entry per rule file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.RuleExtractor{
			Filename: "go.md",
			Content:  "## Go Rules\n\nUse gofmt always.\n",
		}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(1))
		g.Expect(entries[0].ID).To(Equal("rule:go.md"))
		g.Expect(entries[0].SourceType).To(Equal("rule"))
		g.Expect(entries[0].SourcePath).To(Equal("go.md"))
		g.Expect(entries[0].Title).To(Equal("go.md"))
		g.Expect(entries[0].ContentHash).
			To(Equal(contentHash("## Go Rules\n\nUse gofmt always.\n")))
	})

	t.Run("empty content produces zero entries", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.RuleExtractor{Filename: "go.md", Content: ""}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(BeEmpty())
	})
}

func TestSkillExtractor_Extract(t *testing.T) {
	t.Parallel()

	t.Run("one entry per skill", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.SkillExtractor{
			SkillName: "commit",
			Content:   "Create a git commit with proper formatting.",
		}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(1))
		g.Expect(entries[0].ID).To(Equal("skill:commit"))
		g.Expect(entries[0].SourceType).To(Equal("skill"))
		g.Expect(entries[0].SourcePath).To(Equal("commit"))
		g.Expect(entries[0].Title).To(Equal("commit"))
		g.Expect(entries[0].ContentHash).
			To(Equal(contentHash("Create a git commit with proper formatting.")))
	})

	t.Run("empty content produces zero entries", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ext := registry.SkillExtractor{SkillName: "commit", Content: ""}

		entries, err := ext.Extract()
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(BeEmpty())
	})
}

func contentHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
