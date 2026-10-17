//go:build targ

package dev

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// checkModuleSyntax parses each module, a path and its source, and returns an error that names node when it is not
// found, and the path and the complaint of each module that does not parse. A check that cannot run does not pass
// (793b55). Not written yet: it passes every module.
func checkModuleSyntax(
	_ context.Context,
	_ map[string]string,
	_ func() (string, error),
	_ func(context.Context, string, string) ([]byte, error),
) error {
	return nil
}

// findNode looks up the node executable.
func findNode() (string, error) {
	path, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("looking up node: %w", err)
	}

	return path, nil
}

// parseModule has node check the syntax of a JavaScript module: the source goes to node on standard input and is read
// as an ES module, so the result does not depend on how a release of node detects the syntax of a `.js` file. It
// returns what node printed, and an error when node found the module wrong or could not be run.
func parseModule(ctx context.Context, node, source string) ([]byte, error) {
	command := exec.CommandContext(ctx, node, "--input-type=module", "--check")
	command.Stdin = strings.NewReader(source)

	out, err := command.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("checking a module with node: %w", err)
	}

	return out, nil
}
