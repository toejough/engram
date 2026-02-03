package yield_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/yield"
)

func TestValidateContent(t *testing.T) {
	t.Run("valid complete yield", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
type = "complete"
timestamp = 2026-02-02T11:30:00Z

[payload]
artifact = "docs/requirements.md"

[context]
phase = "pm"
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeTrue())
		g.Expect(result.Errors).To(BeEmpty())
	})

	t.Run("missing type field", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
timestamp = 2026-02-02T11:30:00Z
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeFalse())
		g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [yield].type")))
	})

	t.Run("missing timestamp field", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
type = "complete"
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeFalse())
		g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [yield].timestamp")))
	})

	t.Run("invalid type", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
type = "not-a-real-type"
timestamp = 2026-02-02T11:30:00Z
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeFalse())
		g.Expect(result.Errors).To(ContainElement(ContainSubstring("invalid yield type")))
	})

	t.Run("resumable type without context", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-02T11:30:00Z

[payload]
question = "What is your name?"
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeFalse())
		g.Expect(result.Errors).To(ContainElement(ContainSubstring("requires [context] section")))
	})

	t.Run("resumable type with context", func(t *testing.T) {
		g := NewWithT(t)
		content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-02T11:30:00Z

[payload]
question = "What is your name?"

[context]
phase = "pm"
awaiting = "user-response"
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeTrue())
	})

	t.Run("all producer types are valid", func(t *testing.T) {
		g := NewWithT(t)
		for _, yieldType := range yield.ValidProducerTypes {
			content := `
[yield]
type = "` + yieldType + `"
timestamp = 2026-02-02T11:30:00Z

[context]
phase = "pm"
`
			result, err := yield.ValidateContent(content)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(result.Errors).ToNot(ContainElement(ContainSubstring("invalid yield type")),
				"type %s should be valid", yieldType)
		}
	})

	t.Run("all QA types are valid", func(t *testing.T) {
		g := NewWithT(t)
		for _, yieldType := range yield.ValidQATypes {
			content := `
[yield]
type = "` + yieldType + `"
timestamp = 2026-02-02T11:30:00Z

[context]
phase = "pm"
`
			result, err := yield.ValidateContent(content)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(result.Errors).ToNot(ContainElement(ContainSubstring("invalid yield type")),
				"type %s should be valid", yieldType)
		}
	})

	t.Run("invalid TOML syntax", func(t *testing.T) {
		g := NewWithT(t)
		content := `[yield
type = "complete"
`
		result, err := yield.ValidateContent(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Valid).To(BeFalse())
		g.Expect(result.Errors).To(ContainElement(ContainSubstring("TOML parse error")))
	})
}
