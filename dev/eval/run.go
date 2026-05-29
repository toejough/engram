//go:build targ

package eval

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrNotImplemented = errors.New("not implemented")
	ErrUnknownArm     = errors.New("unknown arm")
)

// Run executes every scenario under the named arm and writes results.
// Orchestration only; all I/O goes through deps.
func Run(_ context.Context, armName string, _ RunConfig, _ Deps) error {
	if _, ok := LookupArm(armName); !ok {
		return fmt.Errorf("%w: %q", ErrUnknownArm, armName)
	}

	return ErrNotImplemented // completed in Task 9
}
