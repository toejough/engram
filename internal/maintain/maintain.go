// Package maintain provides maintenance operations for memories.
// NOTE: Stubbed during SBIA migration (Step 1). Will be rebuilt in Step 5.
package maintain

import "errors"

// Exported variables.
var (
	ErrUserQuit = errors.New("user quit")
)

// Confirmer prompts the user for confirmation during maintenance operations.
type Confirmer interface {
	Confirm(prompt string) (bool, error)
}
