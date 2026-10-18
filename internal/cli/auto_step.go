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
	encoded, marshalErr := json.Marshal(step)
	if marshalErr != nil {
		return fmt.Errorf("%s: encoding the request: %w", command, marshalErr)
	}

	_, writeErr := fmt.Fprintln(stdout, string(encoded))
	if writeErr != nil {
		return fmt.Errorf("%s: writing the request: %w", command, writeErr)
	}

	return nil
}
