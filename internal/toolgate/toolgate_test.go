package toolgate_test

import (
	"testing"

	"engram/internal/toolgate"

	. "github.com/onsi/gomega"
)

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
