package tokenresolver_test

import (
	"context"
	"errors"
	"testing"

	"engram/internal/tokenresolver"

	. "github.com/onsi/gomega"
)

func TestResolver_EnvVarPresent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	executorCalled := false
	resolver := tokenresolver.New(
		func(key string) string {
			if key == "ENGRAM_API_TOKEN" {
				return "env-token"
			}

			return ""
		},
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			executorCalled = true

			return nil, nil
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal("env-token"))
	g.Expect(executorCalled).To(BeFalse(), "executor should never be called when env var is present")
}

func TestResolver_KeychainEmptyAccessToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"claudeAiOauth":{"accessToken":""}}`), nil
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal(""))
}

func TestResolver_KeychainError_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("security command failed")
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal(""))
}

func TestResolver_KeychainFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"claudeAiOauth":{"accessToken":"keychain-token"}}`), nil
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal("keychain-token"))
}

func TestResolver_KeychainMalformedJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`not valid json`), nil
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal(""))
}

func TestResolver_KeychainMissingField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"someOtherField":{"accessToken":"token"}}`), nil
		},
		"darwin",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal(""))
}

func TestResolver_KeychainSkippedOnNonDarwin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	executorCalled := false
	resolver := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			executorCalled = true

			return []byte(`{"claudeAiOauth":{"accessToken":"keychain-token"}}`), nil
		},
		"linux",
	)

	token, err := resolver.Resolve(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(token).To(Equal(""))
	g.Expect(executorCalled).To(BeFalse(), "executor should never be called on non-darwin")
}
