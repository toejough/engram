package memory_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TestTransactionCreateBackups verifies that backups are created for all critical files.
// Traces to: ISSUE-212 AC-4
func TestTransactionCreateBackups(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	// Create test files to backup
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	skillsDir := filepath.Join(tmpDir, "skills")

	g.Expect(os.WriteFile(dbPath, []byte("test db"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("test md"), 0644)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "test-skill"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "test-skill", "SKILL.md"), []byte("test skill"), 0644)).To(Succeed())

	// Create transaction
	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
	})

	// Create backups
	err := tx.CreateBackups()
	g.Expect(err).ToNot(HaveOccurred())

	// Verify backups exist
	g.Expect(dbPath + ".bak").To(BeAnExistingFile())
	g.Expect(claudeMDPath + ".bak").To(BeAnExistingFile())
	_, err = os.Stat(skillsDir + ".bak")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify backup contents match originals
	dbBakContent, err := os.ReadFile(dbPath + ".bak")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(dbBakContent).To(Equal([]byte("test db")))

	mdBakContent, err := os.ReadFile(claudeMDPath + ".bak")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(mdBakContent).To(Equal([]byte("test md")))

	skillBakContent, err := os.ReadFile(filepath.Join(skillsDir+".bak", "test-skill", "SKILL.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skillBakContent).To(Equal([]byte("test skill")))
}

// TestTransactionLogRecordsChanges verifies transaction log captures all operations.
// Traces to: ISSUE-212 AC-4
func TestTransactionLogRecordsChanges(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	g.Expect(os.WriteFile(dbPath, []byte("test"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("test"), 0644)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    tmpDir,
	})

	// Record various operations
	tx.RecordChange(memory.ChangeRecord{
		Type:   "prune",
		Tier:   "embeddings",
		Target: "entry-123",
		Before: "old content",
		After:  "",
	})

	tx.RecordChange(memory.ChangeRecord{
		Type:   "consolidate",
		Tier:   "claude-md",
		Target: "learning-1",
		Before: "learning A",
		After:  "learning A merged",
	})

	// Verify log contains both records
	changes := tx.GetChanges()
	g.Expect(changes).To(HaveLen(2))
	g.Expect(changes[0].Type).To(Equal("prune"))
	g.Expect(changes[0].Tier).To(Equal("embeddings"))
	g.Expect(changes[1].Type).To(Equal("consolidate"))
	g.Expect(changes[1].Tier).To(Equal("claude-md"))
}

// TestTransactionRollbackOnError verifies rollback restores original state.
// Traces to: ISSUE-212 AC-4
func TestTransactionRollbackOnError(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create original files
	originalDB := []byte("original db content")
	originalMD := []byte("original md content")
	g.Expect(os.WriteFile(dbPath, originalDB, 0644)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, originalMD, 0644)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "original-skill"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "original-skill", "SKILL.md"), []byte("original skill"), 0644)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
	})

	// Create backups
	g.Expect(tx.CreateBackups()).To(Succeed())

	// Modify files (simulating operations)
	g.Expect(os.WriteFile(dbPath, []byte("modified db"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("modified md"), 0644)).To(Succeed())
	g.Expect(os.RemoveAll(filepath.Join(skillsDir, "original-skill"))).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "new-skill"), 0755)).To(Succeed())

	// Rollback
	err := tx.Rollback()
	g.Expect(err).ToNot(HaveOccurred())

	// Verify files restored to original state
	dbContent, err := os.ReadFile(dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(dbContent).To(Equal(originalDB))

	mdContent, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(mdContent).To(Equal(originalMD))

	// Verify original skill restored
	_, err = os.Stat(filepath.Join(skillsDir, "original-skill"))
	g.Expect(err).ToNot(HaveOccurred())
	skillContent, err := os.ReadFile(filepath.Join(skillsDir, "original-skill", "SKILL.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skillContent).To(Equal([]byte("original skill")))

	// Verify new skill removed
	_, err = os.Stat(filepath.Join(skillsDir, "new-skill"))
	g.Expect(err).To(HaveOccurred())
}

// TestTransactionCleanupSuccess verifies backups removed on successful commit.
// Traces to: ISSUE-212 AC-4
func TestTransactionCleanupSuccess(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	skillsDir := filepath.Join(tmpDir, "skills")

	g.Expect(os.WriteFile(dbPath, []byte("test"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("test"), 0644)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
	})

	// Create backups
	g.Expect(tx.CreateBackups()).To(Succeed())

	// Verify backups exist
	g.Expect(dbPath + ".bak").To(BeAnExistingFile())
	g.Expect(claudeMDPath + ".bak").To(BeAnExistingFile())
	_, err := os.Stat(skillsDir + ".bak")
	g.Expect(err).ToNot(HaveOccurred())

	// Commit (cleanup)
	err = tx.Cleanup()
	g.Expect(err).ToNot(HaveOccurred())

	// Verify backups removed
	_, err = os.Stat(dbPath + ".bak")
	g.Expect(err).To(HaveOccurred())
	_, err = os.Stat(claudeMDPath + ".bak")
	g.Expect(err).To(HaveOccurred())
	_, err = os.Stat(skillsDir + ".bak")
	g.Expect(err).To(HaveOccurred())

	// Verify originals still exist
	g.Expect(dbPath).To(BeAnExistingFile())
	g.Expect(claudeMDPath).To(BeAnExistingFile())
	_, err = os.Stat(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestTransactionRollbackWithDBChanges verifies rollback works with database operations.
// Traces to: ISSUE-212 AC-4
func TestTransactionRollbackWithDBChanges(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Initialize test database
	db, err := memory.InitTestDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert test data
	_, err = db.Exec("INSERT INTO embeddings (content, source, confidence) VALUES (?, ?, ?)", "test entry", "test", 1.0)
	g.Expect(err).ToNot(HaveOccurred())
	db.Close()

	// Create file
	g.Expect(os.WriteFile(claudeMDPath, []byte("test"), 0644)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    tmpDir,
	})

	// Create backups
	g.Expect(tx.CreateBackups()).To(Succeed())

	// Modify database (simulating a transaction that should be rolled back)
	db, err = sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = db.Exec("DELETE FROM embeddings")
	g.Expect(err).ToNot(HaveOccurred())
	db.Close()

	// Verify deletion
	db, err = sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
	db.Close()

	// Rollback
	g.Expect(tx.Rollback()).To(Succeed())

	// Verify original data restored
	db, err = sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))

	var content string
	err = db.QueryRow("SELECT content FROM embeddings").Scan(&content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("test entry"))
}

// TestTransactionChangeRecordJSON verifies change records can be serialized.
// Traces to: ISSUE-212 AC-4
func TestTransactionChangeRecordJSON(t *testing.T) {
	g := NewWithT(t)

	record := memory.ChangeRecord{
		Type:   "prune",
		Tier:   "embeddings",
		Target: "entry-123",
		Before: "old content",
		After:  "",
	}

	// Serialize
	data, err := json.Marshal(record)
	g.Expect(err).ToNot(HaveOccurred())

	// Deserialize
	var decoded memory.ChangeRecord
	err = json.Unmarshal(data, &decoded)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(decoded.Type).To(Equal(record.Type))
	g.Expect(decoded.Tier).To(Equal(record.Tier))
	g.Expect(decoded.Target).To(Equal(record.Target))
	g.Expect(decoded.Before).To(Equal(record.Before))
	g.Expect(decoded.After).To(Equal(record.After))
}

// TestTransactionMultipleBackupsError verifies creating backups twice fails.
// Traces to: ISSUE-212 AC-4
func TestTransactionMultipleBackupsError(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	g.Expect(os.WriteFile(dbPath, []byte("test"), 0644)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		SkillsDir:    tmpDir,
	})

	// First backup should succeed
	g.Expect(tx.CreateBackups()).To(Succeed())

	// Second backup should fail
	err := tx.CreateBackups()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("backups already created"))
}

// TestTransactionRollbackBeforeBackupsFails verifies rollback fails if no backups exist.
// Traces to: ISSUE-212 AC-4
func TestTransactionRollbackBeforeBackupsFails(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       filepath.Join(tmpDir, "embeddings.db"),
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		SkillsDir:    tmpDir,
	})

	// Rollback without backups should fail
	err := tx.Rollback()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no backups to restore"))
}

// TestTransactionCleanupIdempotent verifies cleanup can be called multiple times safely.
// Traces to: ISSUE-212 AC-4
func TestTransactionCleanupIdempotent(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	g.Expect(os.WriteFile(dbPath, []byte("test"), 0644)).To(Succeed())

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		SkillsDir:    tmpDir,
	})

	g.Expect(tx.CreateBackups()).To(Succeed())

	// First cleanup
	g.Expect(tx.Cleanup()).To(Succeed())

	// Second cleanup (should be no-op)
	g.Expect(tx.Cleanup()).To(Succeed())
}

// TestTransactionLogPersistence verifies transaction log can be saved and loaded.
// Traces to: ISSUE-212 AC-4
func TestTransactionLogPersistence(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transaction.log")

	tx := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       filepath.Join(tmpDir, "embeddings.db"),
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		SkillsDir:    tmpDir,
	})

	// Record changes
	tx.RecordChange(memory.ChangeRecord{Type: "prune", Tier: "embeddings", Target: "1"})
	tx.RecordChange(memory.ChangeRecord{Type: "consolidate", Tier: "skills", Target: "2"})

	// Save log
	err := tx.SaveLog(logPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(logPath).To(BeAnExistingFile())

	// Load log
	loaded, err := memory.LoadTransactionLog(logPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(loaded).To(HaveLen(2))
	g.Expect(loaded[0].Type).To(Equal("prune"))
	g.Expect(loaded[1].Type).To(Equal("consolidate"))
}

// TestTransactionPropertyBackupsBeforeModifications verifies backups are always created before changes.
// Traces to: ISSUE-212 AC-4
func TestTransactionPropertyBackupsBeforeModifications(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)
		tmpDir, err := os.MkdirTemp("", "transaction-test-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Generate random initial content
		dbContent := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "dbContent")
		mdContent := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "mdContent")

		g.Expect(os.WriteFile(dbPath, dbContent, 0644)).To(Succeed())
		g.Expect(os.WriteFile(claudeMDPath, mdContent, 0644)).To(Succeed())
		g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

		tx := memory.NewTransaction(memory.TransactionOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
		})

		// Property: Cannot record changes before backups created
		tx.RecordChange(memory.ChangeRecord{Type: "test", Tier: "embeddings", Target: "test"})
		g.Expect(tx.BackupsCreated()).To(BeFalse(), "backups should not be marked created yet")

		// Create backups
		g.Expect(tx.CreateBackups()).To(Succeed())
		g.Expect(tx.BackupsCreated()).To(BeTrue(), "backups should be marked created")

		// Property: Backup content matches original
		bakDB, err := os.ReadFile(dbPath + ".bak")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(bakDB).To(Equal(dbContent))

		bakMD, err := os.ReadFile(claudeMDPath + ".bak")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(bakMD).To(Equal(mdContent))
	})
}
