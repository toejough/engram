//go:build targ

package eval_test

import (
	"os"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestParseBashCommands_ExtractsInOrder(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	cmds := eval.ParseBashCommands(raw)
	want := []string{"go test ./...", "targ test"}
	if len(cmds) != len(want) {
		t.Fatalf("got %d cmds %v, want %d", len(cmds), cmds, len(want))
	}
	for i := range want {
		if cmds[i] != want[i] {
			t.Fatalf("cmd %d: got %q want %q", i, cmds[i], want[i])
		}
	}
}

func TestParseBashCommands_IgnoresMalformedLines(t *testing.T) {
	t.Parallel()

	cmds := eval.ParseBashCommands([]byte("garbage\n{\"type\":\"assistant\"}\n"))
	if len(cmds) != 0 {
		t.Fatalf("got %v, want empty", cmds)
	}
}
