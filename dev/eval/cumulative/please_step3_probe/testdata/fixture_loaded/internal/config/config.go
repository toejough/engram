package config

import "os"

// Config holds sweeper runtime configuration read from the environment. The
// reconciliation cadence is not part of Config yet — it is the
// cmd/sweeper.reconcileEvery compile-time constant.
type Config struct {
	QueueDSN string
	StoreDSN string
}

// Load reads sweeper configuration from the environment, falling back to
// local defaults for standalone runs.
func Load() Config {
	return Config{
		QueueDSN: envOr("SWEEP_QUEUE_DSN", "queue://local"),
		StoreDSN: envOr("SWEEP_STORE_DSN", "store://local"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
