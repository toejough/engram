package keyword_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/keyword"
)

func TestNormalizeAll_NilInputReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(keyword.NormalizeAll(nil)).To(gomega.BeNil())
}

func TestNormalizeAll_NormalizesSlice(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	input := []string{"prefixed-ids", "prefixed_IDs", "collision-avoidance"}
	result := keyword.NormalizeAll(input)

	g.Expect(result).To(gomega.Equal([]string{"prefixed_ids", "prefixed_ids", "collision_avoidance"}))
}

func TestNormalize_LowercasesAndReplacesHyphens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cases := []struct {
		input    string
		expected string
	}{
		{"prefixed-ids", "prefixed_ids"},
		{"prefixed_IDs", "prefixed_ids"},
		{"collision-avoidance", "collision_avoidance"},
		{"collision_avoidance", "collision_avoidance"},
		{"already_normalized", "already_normalized"},
		{"UPPER_CASE", "upper_case"},
		{"mixed-Case_Thing", "mixed_case_thing"},
		{"", ""},
	}

	for _, tc := range cases {
		g.Expect(keyword.Normalize(tc.input)).To(gomega.Equal(tc.expected), "input: %q", tc.input)
	}
}
