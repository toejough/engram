package audit_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/audit"
)

func TestStripMarkdownFence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fence returns as-is",
			input: `[{"instruction":"mem","compliant":true}]`,
			want:  `[{"instruction":"mem","compliant":true}]`,
		},
		{
			name:  "json fence stripped",
			input: "```json\n[{\"instruction\":\"mem\"}]\n```",
			want:  `[{"instruction":"mem"}]`,
		},
		{
			name:  "bare fence stripped",
			input: "```\n[{\"x\":1}]\n```",
			want:  `[{"x":1}]`,
		},
		{
			name:  "fence only opening no newline",
			input: "```",
			want:  "```",
		},
		{
			name:  "nested fences preserves inner",
			input: "```json\nouter\n```inner```\n```",
			want:  "outer\n```inner```",
		},
		{
			name:  "partial fence no closing",
			input: "```json\ncontent without closing fence",
			want:  "content without closing fence",
		},
		{
			name:  "whitespace around fence",
			input: "  \n```json\n{\"a\":1}\n```\n  ",
			want:  `{"a":1}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(audit.StripMarkdownFence(tc.input)).To(Equal(tc.want))
		})
	}
}
