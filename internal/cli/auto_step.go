package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/toejough/engram/internal/auto"
)

// printStep prints the request of a step and its carry as one line of JSON, which the mod knows by its first
// characters (855d1b), and reports a failure under the name of the command.
func printStep(stdout io.Writer, command string, step auto.Step) error {
	encodeErr := json.NewEncoder(stdout).Encode(step)
	if encodeErr != nil {
		return fmt.Errorf("%s: writing the request: %w", command, encodeErr)
	}

	return nil
}
