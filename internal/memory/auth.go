package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"
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
	username := os.Getenv("USER")
	if username == "" {
		if u, err := user.Current(); err == nil {
			username = u.Username
		}
	}
	out, err := k.CommandRunner(ctx, "security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-a", username,
		"-w",
	)
	if err != nil {
		return "", fmt.Errorf("%w: keychain read failed: %v", ErrAuthUnavailable, err)
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   any    `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("%w: failed to parse keychain credentials: %v", ErrAuthUnavailable, err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("%w: no accessToken in keychain credentials", ErrAuthUnavailable)
	}

	// Check expiry if present — keychain stores as Unix millis (number or string) or RFC3339
	if creds.ClaudeAiOauth.ExpiresAt != nil {
		var expiresAtStr string
		switch v := creds.ClaudeAiOauth.ExpiresAt.(type) {
		case string:
			expiresAtStr = v
		case float64:
			expiresAtStr = strconv.FormatInt(int64(v), 10)
		}

		var expiry time.Time
		if expiresAtStr != "" {
			if ms, err := strconv.ParseInt(expiresAtStr, 10, 64); err == nil {
				expiry = time.UnixMilli(ms)
			} else if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				expiry = t
			}
		}
		if !expiry.IsZero() && time.Now().After(expiry) {
			return "", fmt.Errorf("%w: token expired at %s", ErrAuthUnavailable, expiry.Format(time.RFC3339))
		}
	}

	return creds.ClaudeAiOauth.AccessToken, nil
}
