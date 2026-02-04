// Package id provides ID generation for traceability artifacts.
package id

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidPrefixes contains the allowed ID prefixes.
var ValidPrefixes = map[string]bool{
	"REQ":   true,
	"DES":   true,
	"ARCH":  true,
	"TASK":  true,
	"ISSUE": true,
}

// Next returns the next sequential ID for the given type prefix.
// It scans markdown files in the directory and docs/ subdirectory
// to find the highest existing ID, then returns the next one.
// If no IDs of that type exist, returns TYPE-001.
func Next(dir, prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("prefix cannot be empty")
	}

	if !ValidPrefixes[prefix] {
		return "", fmt.Errorf("invalid prefix %q: must be one of REQ, DES, ARCH, TASK, ISSUE", prefix)
	}

	// Build regex pattern for the specific prefix
	// Match any number of digits (backward compatible with 3-digit padded IDs)
	pattern := regexp.MustCompile(regexp.QuoteMeta(prefix) + `-(\d+)`)

	maxNum := 0

	// Scan root directory
	if err := scanDir(dir, pattern, &maxNum); err != nil {
		return "", err
	}

	// Scan docs/ subdirectory
	docsDir := filepath.Join(dir, "docs")
	if err := scanDir(docsDir, pattern, &maxNum); err != nil {
		return "", err
	}

	// Format next ID as simple number (no padding)
	nextNum := maxNum + 1
	return fmt.Sprintf("%s-%d", prefix, nextNum), nil
}

// scanDir scans markdown files in a directory for IDs matching the pattern.
// Updates maxNum if a higher number is found.
func scanDir(dir string, pattern *regexp.Regexp, maxNum *int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, that's fine
		}
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", name, err)
		}

		// Find all matches in the file
		matches := pattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) >= 2 {
				num, err := strconv.Atoi(match[1])
				if err == nil && num > *maxNum {
					*maxNum = num
				}
			}
		}
	}

	return nil
}
