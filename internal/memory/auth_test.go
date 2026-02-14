package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

func TestGetKeychainToken_ReturnsToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken":  "sk-ant-oat01-test-token",
			"refreshToken": "sk-ant-oart01-refresh",
			"expiresAt":    "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(token).To(Equal("sk-ant-oat01-test-token"))
}

func TestGetKeychainToken_ReturnsErrWhenKeychainFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.")
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnMalformedJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnEmptyAccessToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnExpiredToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-expired",
			"expiresAt":   "2020-01-01T00:00:00Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_MissingExpiresAtTreatedAsValid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-no-expiry",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(token).To(Equal("sk-ant-oat01-no-expiry"))
}

func TestGetKeychainToken_PassesCorrectSecurityArgs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedName string
	var capturedArgs []string
	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-x",
			"expiresAt":   "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedName = name
			capturedArgs = args
			return credsJSON, nil
		},
	}

	_, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(capturedName).To(Equal("security"))
	g.Expect(capturedArgs).To(ContainElements("find-generic-password", "-s", "Claude Code-credentials", "-w"))
}
