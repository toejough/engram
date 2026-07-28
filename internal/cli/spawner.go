package cli

import (
	"fmt"

	"github.com/toejough/engram/internal/update"
)

// unexported variables.
var (
	// Compile-time interface conformance (internal — the thin-api checker
	// does not walk internal/).
	_ update.Spawner = primSpawner{}
)

// primSpawner is the production update.Spawner: the raw spawn primitive
// (cmd/engram's spawnPrimitives) already distinguishes spawn failure from a
// started child's exit code via cmd.Process/cmd.ProcessState — raw error
// out (nolint:wrapcheck contract, mirrors internal/embed's D-1); Run wraps
// it with context before it reaches the Spawner contract.
type primSpawner struct {
	prims Primitives
}

// Run spawns name with args and extraEnv appended to the process
// environment, stdio inherited from the parent process. A non-nil error
// means the process could not be started; a started child's exit code
// (including a non-zero code for a signal death) comes back via exitCode
// with a nil error.
func (s primSpawner) Run(name string, args []string, extraEnv []string) (int, error) {
	exitCode, runErr := s.prims.Spawn.RunInherited(name, args, extraEnv)
	if runErr != nil {
		return exitCode, fmt.Errorf("spawning %s: %w", name, runErr)
	}

	return exitCode, nil
}
