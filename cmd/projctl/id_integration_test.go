//go:build integration

package main_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/id"
)

// TEST-181 traces: TASK-004
// Test idNext command outputs next REQ ID
func TestIdNext_REQ(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a file with existing REQ IDs
	content := `# Requirements

## REQ-001: First requirement

## REQ-002: Second requirement
`
	err := os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// The CLI will call id.Next internally
	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-3"))
}

// TEST-182 traces: TASK-004
// Test idNext command outputs next TASK ID
func TestIdNext_TASK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a file with existing TASK IDs
	content := `# Tasks

## TASK-042: Some task
`
	err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "TASK")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("TASK-43"))
}

// TEST-183 traces: TASK-004
// Test idNext command returns TYPE-001 when no IDs exist
func TestIdNext_NoExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	result, err := id.Next(dir, "DES")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("DES-1"))
}

// TEST-184 traces: TASK-004
// Test idNext command returns error for invalid type
func TestIdNext_InvalidType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := id.Next(dir, "INVALID")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid"))
}

// TEST-185 traces: TASK-004
// Test idNext command handles ARCH prefix
func TestIdNext_ARCH(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Architecture

## ARCH-010: Some decision
`
	err := os.WriteFile(filepath.Join(dir, "architecture.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "ARCH")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("ARCH-11"))
}

// TEST-186 traces: TASK-004
// Test idNext command handles ISSUE prefix
func TestIdNext_ISSUE(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Issues

## ISSUE-5: A bug
`
	err := os.WriteFile(filepath.Join(dir, "issues.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "ISSUE")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("ISSUE-6"))
}
