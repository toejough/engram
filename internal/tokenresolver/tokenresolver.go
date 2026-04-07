// Package tokenresolver resolves an API token by checking environment variables
// first, then falling back to the macOS Keychain on darwin systems.
package tokenresolver

import (
	"context"
	"encoding/json"
)

// Resolver resolves an API token using env var and Keychain strategies.
type Resolver struct {
	getenv  func(string) string
	execCmd func(ctx context.Context, name string, args ...string) ([]byte, error)
	goos    string
}

// New creates a Resolver with the provided I/O dependencies.
func New(
	getenv func(string) string,
	execCmd func(ctx context.Context, name string, args ...string) ([]byte, error),
	goos string,
) *Resolver {
	return &Resolver{
		getenv:  getenv,
		execCmd: execCmd,
		goos:    goos,
	}
}

// Resolve returns the API token. It checks the environment variable first,
// then falls back to the macOS Keychain on darwin only. Keychain errors and
// parse failures return ("", nil) rather than propagating errors.
func (r *Resolver) Resolve(ctx context.Context) (string, error) {
	if token := r.getenv(envKey); token != "" {
		return token, nil
	}

	if r.goos != "darwin" {
		return "", nil
	}

	return r.resolveFromKeychain(ctx)
}

func (r *Resolver) resolveFromKeychain(ctx context.Context) (string, error) {
	output, err := r.execCmd(ctx, "security", "find-generic-password", "-s", keychainService, "-w")
	if err != nil {
		return "", nil //nolint:nilerr // Keychain unavailable is not a fatal error
	}

	var payload keychainPayload

	jsonErr := json.Unmarshal(output, &payload)
	if jsonErr != nil {
		return "", nil //nolint:nilerr // malformed JSON is not a fatal error
	}

	return payload.ClaudeAiOauth.AccessToken, nil
}

// unexported constants.
const (
	envKey          = "ENGRAM_API_TOKEN"
	keychainService = "Claude Code-credentials"
)

type keychainPayload struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"` //nolint:gosec // G117: keychain deserialization, not logged
	} `json:"claudeAiOauth"`
}
