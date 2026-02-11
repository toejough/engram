package memory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TransactionOpts holds options for creating a transaction.
type TransactionOpts struct {
	DBPath       string // Path to embeddings.db
	ClaudeMDPath string // Path to CLAUDE.md
	SkillsDir    string // Path to skills directory
}

// ChangeRecord represents a single change made during a transaction.
// Used for transaction logging and potential undo operations.
type ChangeRecord struct {
	Type   string `json:"type"`   // "prune", "decay", "consolidate", "split", "promote", "demote"
	Tier   string `json:"tier"`   // "embeddings", "skills", "claude-md"
	Target string `json:"target"` // ID, file path, or entry text
	Before string `json:"before"` // State before change
	After  string `json:"after"`  // State after change
}

// Transaction manages atomic backup, rollback, and cleanup operations
// for memory maintenance across all three tiers.
type Transaction struct {
	opts           TransactionOpts
	changes        []ChangeRecord
	backupsCreated bool
}

// NewTransaction creates a new transaction manager.
func NewTransaction(opts TransactionOpts) *Transaction {
	return &Transaction{
		opts:    opts,
		changes: make([]ChangeRecord, 0),
	}
}

// CreateBackups creates backup copies of all critical files before modifications.
// Returns an error if backups already exist to prevent accidental overwrites.
func (t *Transaction) CreateBackups() error {
	if t.backupsCreated {
		return errors.New("backups already created")
	}

	// Backup embeddings.db
	if t.opts.DBPath != "" {
		if _, err := os.Stat(t.opts.DBPath); err == nil {
			if err := copyFile(t.opts.DBPath, t.opts.DBPath+".bak"); err != nil {
				return fmt.Errorf("failed to backup database: %w", err)
			}
		}
	}

	// Backup CLAUDE.md
	if t.opts.ClaudeMDPath != "" {
		if _, err := os.Stat(t.opts.ClaudeMDPath); err == nil {
			if err := copyFile(t.opts.ClaudeMDPath, t.opts.ClaudeMDPath+".bak"); err != nil {
				return fmt.Errorf("failed to backup CLAUDE.md: %w", err)
			}
		}
	}

	// Backup skills directory
	if t.opts.SkillsDir != "" {
		if _, err := os.Stat(t.opts.SkillsDir); err == nil {
			if err := copyDir(t.opts.SkillsDir, t.opts.SkillsDir+".bak"); err != nil {
				return fmt.Errorf("failed to backup skills directory: %w", err)
			}
		}
	}

	t.backupsCreated = true
	return nil
}

// RecordChange records a change operation in the transaction log.
func (t *Transaction) RecordChange(change ChangeRecord) {
	t.changes = append(t.changes, change)
}

// GetChanges returns all recorded changes in the transaction.
func (t *Transaction) GetChanges() []ChangeRecord {
	return t.changes
}

// BackupsCreated returns whether backups have been created for this transaction.
func (t *Transaction) BackupsCreated() bool {
	return t.backupsCreated
}

// Rollback restores all files from backups, undoing all changes.
// Returns an error if no backups exist.
func (t *Transaction) Rollback() error {
	if !t.backupsCreated {
		return errors.New("no backups to restore")
	}

	// Restore embeddings.db
	if t.opts.DBPath != "" {
		bakPath := t.opts.DBPath + ".bak"
		if _, err := os.Stat(bakPath); err == nil {
			// Remove current file
			_ = os.Remove(t.opts.DBPath)
			// Restore from backup
			if err := copyFile(bakPath, t.opts.DBPath); err != nil {
				return fmt.Errorf("failed to restore database: %w", err)
			}
		}
	}

	// Restore CLAUDE.md
	if t.opts.ClaudeMDPath != "" {
		bakPath := t.opts.ClaudeMDPath + ".bak"
		if _, err := os.Stat(bakPath); err == nil {
			// Remove current file
			_ = os.Remove(t.opts.ClaudeMDPath)
			// Restore from backup
			if err := copyFile(bakPath, t.opts.ClaudeMDPath); err != nil {
				return fmt.Errorf("failed to restore CLAUDE.md: %w", err)
			}
		}
	}

	// Restore skills directory
	if t.opts.SkillsDir != "" {
		bakPath := t.opts.SkillsDir + ".bak"
		if _, err := os.Stat(bakPath); err == nil {
			// Remove current directory
			_ = os.RemoveAll(t.opts.SkillsDir)
			// Restore from backup
			if err := copyDir(bakPath, t.opts.SkillsDir); err != nil {
				return fmt.Errorf("failed to restore skills directory: %w", err)
			}
		}
	}

	return nil
}

// Cleanup removes all backup files after successful commit.
// Idempotent - can be called multiple times safely.
func (t *Transaction) Cleanup() error {
	// Remove embeddings.db backup
	if t.opts.DBPath != "" {
		_ = os.Remove(t.opts.DBPath + ".bak")
	}

	// Remove CLAUDE.md backup
	if t.opts.ClaudeMDPath != "" {
		_ = os.Remove(t.opts.ClaudeMDPath + ".bak")
	}

	// Remove skills directory backup
	if t.opts.SkillsDir != "" {
		_ = os.RemoveAll(t.opts.SkillsDir + ".bak")
	}

	return nil
}

// SaveLog saves the transaction log to a file.
func (t *Transaction) SaveLog(path string) error {
	data, err := json.MarshalIndent(t.changes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transaction log: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write transaction log: %w", err)
	}

	return nil
}

// LoadTransactionLog loads a transaction log from a file.
func LoadTransactionLog(path string) ([]ChangeRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction log: %w", err)
	}

	var changes []ChangeRecord
	if err := json.Unmarshal(data, &changes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction log: %w", err)
	}

	return changes, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// InitTestDB is a helper function for tests to initialize a test database.
// It wraps initEmbeddingsDB which is package-private.
func InitTestDB(path string) (*sql.DB, error) {
	return initEmbeddingsDB(path)
}
