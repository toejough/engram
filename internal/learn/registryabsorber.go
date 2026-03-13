package learn

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// RegistryAbsorberFunc is a function that implements the RegistryAbsorber interface (UC-33).
type RegistryAbsorberFunc func(existingPath, candidateTitle, contentHash string, now time.Time) error

// RecordAbsorbed implements the RegistryAbsorber interface.
func (f RegistryAbsorberFunc) RecordAbsorbed(
	existingPath, candidateTitle, contentHash string,
	now time.Time,
) error {
	return f(existingPath, candidateTitle, contentHash, now)
}

// ComputeContentHash computes a hash of keywords for the Absorbed record (UC-33).
func ComputeContentHash(keywords []string) string {
	joined := strings.Join(keywords, ",")
	hash := sha256.Sum256([]byte(joined))

	return hex.EncodeToString(hash[:])[:16]
}
