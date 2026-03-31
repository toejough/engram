// Package correct implements the Remember & Correct pipeline (ARCH-1).
// Stubbed during SBIA migration — will be rebuilt in Step 2.
package correct

import "context"

// Corrector orchestrates the correction pipeline. Stubbed during SBIA migration.
type Corrector struct{}

// New creates a stubbed Corrector.
func New(_ ...any) *Corrector {
	return &Corrector{}
}

// Run is stubbed during SBIA migration. Returns empty string.
func (c *Corrector) Run(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// SetProjectSlug is stubbed during SBIA migration.
func (c *Corrector) SetProjectSlug(_ string) {}
