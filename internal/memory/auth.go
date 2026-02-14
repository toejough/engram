package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

var ErrAuthUnavailable = errors.New("auth unavailable")

type KeychainAuth struct {
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewKeychainAuth() *KeychainAuth {
	return &KeychainAuth{
		CommandRunner: defaultCommandRunner,
	}
}

func (k *KeychainAuth) GetToken(ctx context.Context) (string, error) {
	user := os.Getenv("USER")
	out, err := k.CommandRunner(ctx, "security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-a", user,
		"-w",
	)
	if err != nil {
		return "", fmt.Errorf("%w: keychain read failed: %v", ErrAuthUnavailable, err)
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   string `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("%w: failed to parse keychain credentials: %v", ErrAuthUnavailable, err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("%w: no accessToken in keychain credentials", ErrAuthUnavailable)
	}

	// Check expiry if present
	if creds.ClaudeAiOauth.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, creds.ClaudeAiOauth.ExpiresAt)
		if err == nil && time.Now().After(expiry) {
			return "", fmt.Errorf("%w: token expired at %s", ErrAuthUnavailable, creds.ClaudeAiOauth.ExpiresAt)
		}
	}

	return creds.ClaudeAiOauth.AccessToken, nil
}
