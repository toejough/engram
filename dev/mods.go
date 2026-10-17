//go:build targ

package dev

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"
)

// unexported variables.
var (
	errModulesDoNotParse = errors.New("these modules do not parse")
	errNodeMissing       = errors.New("node is required to check the JavaScript of the mods")
)

// checkModuleSyntax parses each module, a path and its source, and returns an error that names node when it is not
// found, and the path and the complaint of each module that does not parse. A check that cannot run does not pass
// (793b55).
func checkModuleSyntax(
	ctx context.Context,
	modules map[string]string,
	find func() (string, error),
	parse func(context.Context, string, string) ([]byte, error),
) error {
	node, findErr := find()
	if findErr != nil {
		return fmt.Errorf("%w: %w", errNodeMissing, findErr)
	}

	var failures []string

	for _, path := range slices.Sorted(maps.Keys(modules)) {
		complaint, parseErr := parse(ctx, node, modules[path])
		if parseErr != nil {
			failures = append(failures, path+": "+cmp.Or(strings.TrimSpace(string(complaint)), parseErr.Error()))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("%w:\n%s", errModulesDoNotParse, strings.Join(failures, "\n"))
	}

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
