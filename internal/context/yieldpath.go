// Package context provides context generation and yield path management.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
)

// GenerateYieldPath creates a unique file path for skill yield output.
// Uses timestamp + UUID pattern for parallel execution safety.
//
// Path pattern (parallel): .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
// Path pattern (sequential): .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml
//
// Returns absolute path with parent directories created (mode 0755).
func GenerateYieldPath(projectDir, phase, taskID string) (string, error) {
	// Verify project directory exists
	info, err := os.Stat(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("project directory does not exist: %s", projectDir)
		}
		return "", fmt.Errorf("failed to stat project directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", projectDir)
	}

	// Read or create project UUID from state.toml
	projectUUID, projectName, err := getOrCreateProjectUUID(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to get project UUID: %w", err)
	}

	// Generate file-level UUID
	fileUUID := uuid.New().String()

	// Format timestamps
	now := time.Now()
	date := now.Format("2006-01-02")              // YYYY-MM-DD
	datetime := now.Format("2006-01-02.15-04-05") // YYYY-MM-DD.HH-mm-SS

	// Build directory path: .claude/context/{date}-{project}-{projectUUID}
	dirName := fmt.Sprintf("%s-%s-%s", date, projectName, projectUUID)
	contextDir := filepath.Join(projectDir, ".claude", "context", dirName)

	// Build filename based on whether taskID is provided
	var filename string
	if taskID != "" {
		// Parallel: {datetime}-{phase}-{taskID}-{fileUUID}.toml
		filename = fmt.Sprintf("%s-%s-%s-%s.toml", datetime, phase, taskID, fileUUID)
	} else {
		// Sequential: {datetime}-{phase}-{fileUUID}.toml
		filename = fmt.Sprintf("%s-%s-%s.toml", datetime, phase, fileUUID)
	}

	// Construct full path
	fullPath := filepath.Join(contextDir, filename)

	// Convert to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	// Create parent directories with mode 0755
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directories (permission denied): %w", err)
	}

	return absPath, nil
}

// stateFile represents the minimal structure we need from state.toml
type stateFile struct {
	Project projectSection `toml:"project"`
}

type projectSection struct {
	Name string `toml:"name"`
	UUID string `toml:"uuid,omitempty"`
}

// getOrCreateProjectUUID reads project UUID from state.toml or generates one if missing.
// Returns (uuid, projectName, error).
func getOrCreateProjectUUID(projectDir string) (string, string, error) {
	statePath := filepath.Join(projectDir, ".claude", "state.toml")

	// Read existing state
	var state stateFile
	if _, err := toml.DecodeFile(statePath, &state); err != nil {
		return "", "", fmt.Errorf("failed to read state.toml: %w", err)
	}

	projectName := state.Project.Name
	if projectName == "" {
		return "", "", fmt.Errorf("state.toml missing project name")
	}

	// If UUID exists, return it
	if state.Project.UUID != "" {
		return state.Project.UUID, projectName, nil
	}

	// Generate and store new UUID
	newUUID := uuid.New().String()
	if err := addUUIDToStateFile(statePath, newUUID); err != nil {
		return "", "", err
	}

	return newUUID, projectName, nil
}

// addUUIDToStateFile atomically adds a UUID to the project section of state.toml.
// Preserves all other fields in the file.
func addUUIDToStateFile(statePath, newUUID string) error {
	// Read full state file to preserve all fields
	data, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state.toml for update: %w", err)
	}

	// Parse into generic map to preserve all fields
	var fullState map[string]interface{}
	if err := toml.Unmarshal(data, &fullState); err != nil {
		return fmt.Errorf("failed to parse state.toml: %w", err)
	}

	// Update project section with UUID
	project, ok := fullState["project"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("state.toml missing [project] section")
	}
	project["uuid"] = newUUID

	// Write back atomically using temp file + rename
	return writeStateFileAtomic(statePath, fullState)
}

// writeStateFileAtomic writes state data to file atomically using temp file + rename.
func writeStateFileAtomic(statePath string, state map[string]interface{}) error {
	tmpPath := statePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp state file: %w", err)
	}

	if err := toml.NewEncoder(f).Encode(state); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to encode updated state: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to update state.toml: %w", err)
	}

	return nil
}
