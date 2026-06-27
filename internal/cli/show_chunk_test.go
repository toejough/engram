package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestShowChunkEmptyRefErrors(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.ShowChunkDeps{
		ListIndexes: func(string) ([]string, error) { return nil, nil },
		ReadFile:    (&memFS{files: map[string][]byte{}}).read,
	}

	var out strings.Builder

	err := cli.RunShowChunk(context.Background(), cli.ShowChunkArgs{
		Ref: "   ", ChunksDir: "/chunks",
	}, deps, &out)

	g.Expect(err).To(gomega.HaveOccurred())
}

func TestShowChunkNotFoundErrors(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := chunkIndexFS(t, []chunk.Record{
		{Source: "/s/a.jsonl", Anchor: "turn-1", ContentHash: "sha256:aa", Text: "only chunk"},
	})
	deps := cli.ShowChunkDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/s1.jsonl"}, nil },
		ReadFile:    fs.read,
	}

	var out strings.Builder

	err := cli.RunShowChunk(context.Background(), cli.ShowChunkArgs{
		Ref: "/s/a.jsonl#turn-9", ChunksDir: "/chunks",
	}, deps, &out)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("chunk not found: /s/a.jsonl#turn-9")))
}

func TestShowChunkPrintsMatchingText(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := chunkIndexFS(t, []chunk.Record{
		{Source: "/s/a.jsonl", Anchor: "turn-1", ContentHash: "sha256:aa", Text: "first chunk evidence"},
		{Source: "/s/a.jsonl", Anchor: "turn-2", ContentHash: "sha256:bb", Text: "second chunk evidence"},
	})
	deps := cli.ShowChunkDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/s1.jsonl"}, nil },
		ReadFile:    fs.read,
	}

	var out strings.Builder

	err := cli.RunShowChunk(context.Background(), cli.ShowChunkArgs{
		Ref: "/s/a.jsonl#turn-2", ChunksDir: "/chunks",
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("second chunk evidence"))
	g.Expect(out.String()).NotTo(gomega.ContainSubstring("first chunk evidence"))
}
