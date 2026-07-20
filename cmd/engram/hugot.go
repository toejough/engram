// Thin hugot capability wrappers. This file (plus its _test siblings) is
// the only place in the repo outside internal/embed's _test files that
// imports hugot — and it holds NO logic: hugotRuntime is an EMPTY struct
// whose methods are single-call / simple-error-wrapper bodies. The
// session/pipeline lifecycle, config policy, output mapping, and error
// wrapping all live in internal/embed (#700).
// (Blank line below keeps this file comment from doubling as a second
// package godoc — main.go owns the package comment.)

package main

import (
	"context"

	"github.com/knights-analytics/hugot"

	"github.com/toejough/engram/internal/embed"
)

// hugotRuntime implements embed.Runtime over the real hugot library.
type hugotRuntime struct{}

// NewPipeline opens a feature-extraction pipeline on session and returns
// its run function, erasing hugot's pipeline type via closure capture
// (doctrine flag E-1): the closure body is the sanctioned
// trivially-sequenced single-call shape — run on the captured pipe,
// err-check, selector return.
func (hugotRuntime) NewPipeline(
	session embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	//nolint:forcetypeassert // production invariant: sessions come from NewSession
	pipe, err := hugot.NewPipeline(session.(*hugot.Session), hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         name,
		OnnxFilename: onnxFilename,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // raw error contract: internal/embed wraps once (D-1)
	}

	return func(ctx context.Context, inputs []string) ([][]float32, error) {
		out, runErr := pipe.RunPipeline(ctx, inputs)
		if runErr != nil {
			return nil, runErr //nolint:wrapcheck // raw error contract: internal/embed wraps once (D-1)
		}

		return out.Embeddings, nil
	}, nil
}

// NewSession opens a Go-backend hugot session. *hugot.Session satisfies
// embed.RawSession structurally.
func (hugotRuntime) NewSession(ctx context.Context) (embed.RawSession, error) {
	return hugot.NewGoSession(ctx) //nolint:wrapcheck // raw error contract: internal/embed wraps once (D-1)
}
