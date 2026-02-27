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

// BackupsCreated returns whether backups have been created for this transaction.
func (t *Transaction) BackupsCreated() bool {
	return t.backupsCreated
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

// CreateBackups creates backup copies of all critical files before modifications.
// Removes any stale backups from previous failed runs before creating new ones.
func (t *Transaction) CreateBackups() error {
	if t.backupsCreated {
		return errors.New("backups already created")
	}

	// Remove stale backups from previous failed runs
	_ = t.Cleanup()

	// Backup embeddings.db
	if t.opts.DBPath != "" {
		if _, err := os.Stat(t.opts.DBPath); err == nil {
			err := copyFile(t.opts.DBPath, t.opts.DBPath+".bak")
			if err != nil {
				return fmt.Errorf("failed to backup database: %w", err)
			}
		}
	}

	// Backup CLAUDE.md
	if t.opts.ClaudeMDPath != "" {
		if _, err := os.Stat(t.opts.ClaudeMDPath); err == nil {
			err := copyFile(t.opts.ClaudeMDPath, t.opts.ClaudeMDPath+".bak")
			if err != nil {
				return fmt.Errorf("failed to backup CLAUDE.md: %w", err)
			}
		}
	}

	// Backup skills directory
	if t.opts.SkillsDir != "" {
		if _, err := os.Stat(t.opts.SkillsDir); err == nil {
			err := copyDir(osDirOps{}, t.opts.SkillsDir, t.opts.SkillsDir+".bak")
			if err != nil {
				return fmt.Errorf("failed to backup skills directory: %w", err)
			}
		}
	}

	t.backupsCreated = true

	return nil
}

// GetChanges returns all recorded changes in the transaction.
func (t *Transaction) GetChanges() []ChangeRecord {
	return t.changes
}

// RecordChange records a change operation in the transaction log.
func (t *Transaction) RecordChange(change ChangeRecord) {
	t.changes = append(t.changes, change)
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
			err := copyFile(bakPath, t.opts.DBPath)
			if err != nil {
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
			err := copyFile(bakPath, t.opts.ClaudeMDPath)
			if err != nil {
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
			err := copyDir(osDirOps{}, bakPath, t.opts.SkillsDir)
			if err != nil {
				return fmt.Errorf("failed to restore skills directory: %w", err)
			}
		}
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

// TransactionOpts holds options for creating a transaction.
type TransactionOpts struct {
	DBPath       string // Path to embeddings.db
	ClaudeMDPath string // Path to CLAUDE.md
	SkillsDir    string // Path to skills directory
}

// InitTestDB is a helper function for tests to initialize a test database.
// It wraps initEmbeddingsDB which is package-private.
func InitTestDB(path string) (*sql.DB, error) {
	return initEmbeddingsDB(path)
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

// dirOps abstracts filesystem operations used by copyDir to allow DI in tests.
type dirOps interface {
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadDir(name string) ([]os.DirEntry, error)
	Readlink(name string) (string, error)
	Symlink(oldname, newname string) error
}

// osDirOps implements dirOps using the real OS.
type osDirOps struct{}

func (osDirOps) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

func (osDirOps) ReadDir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }

func (osDirOps) Readlink(name string) (string, error) { return os.Readlink(name) }

func (osDirOps) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

func (osDirOps) Symlink(oldname, newname string) error { return os.Symlink(oldname, newname) }

// copyDir recursively copies a directory from src to dst.
func copyDir(ops dirOps, src, dst string) error {
	// Get source directory info
	srcInfo, err := ops.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := ops.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := ops.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Handle symlinks: recreate the link rather than copying the target
		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := ops.Readlink(srcPath)
			if err != nil {
				return err
			}

			if err := ops.Symlink(linkTarget, dstPath); err != nil {
				return err
			}
		} else if entry.IsDir() {
			// Recursively copy subdirectory
			err := copyDir(ops, srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			err := copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer func() { _ = destFile.Close() }()

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
