package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestResolveChunksDirPrecedence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	env := func(key string) string {
		return map[string]string{"ENGRAM_CHUNKS_DIR": "/from/env"}[key]
	}
	noEnv := func(string) string { return "" }

	g.Expect(cli.ResolveChunksDir("/from/flag", "/home/dev", env)).To(gomega.Equal("/from/flag"),
		"explicit flag wins")
	g.Expect(cli.ResolveChunksDir("", "/home/dev", env)).To(gomega.Equal("/from/env"),
		"env beats default")
	g.Expect(cli.ResolveChunksDir("", "/home/dev", noEnv)).To(
		gomega.Equal(filepath.Join("/home/dev", ".local", "share", "engram", "chunks")),
		"default mirrors the vault: XDG data dir")
}
