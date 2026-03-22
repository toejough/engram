package toolgate_test

import (
	"testing"

	"engram/internal/toolgate"

	. "github.com/onsi/gomega"
)

func TestSurfaceProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// count 0 → 1.0
	g.Expect(toolgate.SurfaceProbability(0)).To(BeNumerically("~", 1.0, 0.001))

	// count 1 → 1/(1+ln(2)) ≈ 0.59
	g.Expect(toolgate.SurfaceProbability(1)).To(BeNumerically("~", 0.59, 0.01))

	// count 10 → 1/(1+ln(11)) ≈ 0.29
	g.Expect(toolgate.SurfaceProbability(10)).To(BeNumerically("~", 0.29, 0.01))

	// count 100 → 1/(1+ln(101)) ≈ 0.18
	g.Expect(toolgate.SurfaceProbability(100)).To(BeNumerically("~", 0.18, 0.01))

	// monotonically decreasing
	prev := toolgate.SurfaceProbability(0)
	for _, count := range []int{1, 2, 5, 10, 50, 100, 1000} {
		probability := toolgate.SurfaceProbability(count)
		g.Expect(probability).To(BeNumerically("<", prev), "probability should decrease with count")
		prev = probability
	}
}

func TestCommandKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{name: "two tokens subcommand", cmd: "go test ./...", want: "go test"},
		{name: "targ subcommand", cmd: "targ check-full", want: "targ check-full"},
		{name: "flag second token dropped", cmd: "grep -r foo src/", want: "grep"},
		{name: "leading env var stripped", cmd: "FOO=bar git push origin main", want: "git push"},
		{name: "multiple env vars stripped", cmd: "A=1 B=2 npm install", want: "npm install"},
		{name: "single token command", cmd: "ls", want: "ls"},
		{name: "flag only second token", cmd: "ls -la", want: "ls"},
		{name: "empty string", cmd: "", want: ""},
		{name: "whitespace only", cmd: "   ", want: ""},
		{name: "env var only", cmd: "FOO=bar", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(toolgate.CommandKey(tt.cmd)).To(Equal(tt.want))
		})
	}
}
