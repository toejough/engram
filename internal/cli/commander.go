package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/toejough/engram/internal/update"
)

// unexported variables.
var (
	// Compile-time interface conformance (internal — the thin-api checker
	// does not walk internal/).
	_ update.Commander = primCommander{}
)

// primCommander is the production update.Commander: it composes the
// injected raw run primitive with output collection, contextual %w
// wrapping, and the platform-not-found → update.ErrCommandNotFound
// translation (doctrine flag C-1). cmd/engram contributes only the
// exec.CommandContext closure and the exec.ErrNotFound sentinel value;
// ALL policy lives here (#700).
type primCommander struct {
	prims Primitives
}

// Run executes name with args in dir (empty dir inherits the process
// cwd), returning captured stdout and stderr. A failure whose chain
// matches the injected NotFoundErr is additionally tagged
// update.ErrCommandNotFound per the Commander contract; errors.Is with
// a nil target matches no non-nil error, so an unwired NotFoundErr
// merely disables translation.
func (c primCommander) Run(
	ctx context.Context, dir, name string, args ...string,
) ([]byte, []byte, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	runErr := c.prims.RunCommand(ctx, dir, name, args, stdout, stderr)
	if runErr != nil {
		if errors.Is(runErr, c.prims.NotFoundErr) {
			return stdout.Bytes(), stderr.Bytes(),
				fmt.Errorf("%s %v: %w: %w", name, args, update.ErrCommandNotFound, runErr)
		}

		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, runErr)
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}
