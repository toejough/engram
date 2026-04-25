//go:build targ

package dev

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestT1_TargsRegistered(t *testing.T) {
	t.Parallel()

	want := []string{"c4-audit", "c4-l1-build", "c4-l1-externals", "c4-history"}
	out, err := exec.CommandContext(context.Background(), "targ").CombinedOutput()
	if err != nil {
		t.Fatalf("targ: %v\n%s", err, out)
	}
	for _, name := range want {
		if !strings.Contains(string(out), name) {
			t.Errorf("targ list missing %q\noutput: %s", name, out)
		}
	}
}
