//go:build targ

package eval_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestOSConfigBuilder_NothingArm_NoSkillsDir(t *testing.T) {
	t.Parallel()

	if !eval.KeychainCredentialAvailable() {
		t.Skip("no Claude Code keychain credential; skipping config-builder integration test")
	}

	root := t.TempDir()
	b := eval.NewOSConfigBuilder("/path/to/fake/engram")

	arm, _ := eval.LookupArm("nothing")
	cfgDir, pathPrefix, err := b.Build(context.Background(), arm, root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if pathPrefix != "" {
		t.Fatalf("nothing arm should have empty PATH prefix, got %q", pathPrefix)
	}
	if _, statErr := os.Stat(filepath.Join(cfgDir, "skills")); !os.IsNotExist(statErr) {
		t.Fatal("nothing arm config must not contain a skills/ dir")
	}
	if _, statErr := os.Stat(filepath.Join(cfgDir, ".credentials.json")); statErr != nil {
		t.Fatalf("config must contain replicated credentials: %v", statErr)
	}
}
