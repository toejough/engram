package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestTransaction_CopyDirWithSubdirs verifies CreateBackups copies nested directories.
func TestTransaction_CopyDirWithSubdirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	subDir := filepath.Join(skillsDir, "my-skill")

	err := os.MkdirAll(subDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("skill content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	txn := memory.NewTransaction(memory.TransactionOpts{
		SkillsDir: skillsDir,
	})

	err = txn.CreateBackups()

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(txn.BackupsCreated()).To(BeTrue())

	backupDir := skillsDir + ".bak"
	g.Expect(backupDir).To(BeADirectory())

	err = txn.Cleanup()
	g.Expect(err).ToNot(HaveOccurred())
}

// TestTransaction_CopyDirWithSymlink verifies CreateBackups preserves symlinks.
func TestTransaction_CopyDirWithSymlink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	targetFile := filepath.Join(tmpDir, "target.txt")

	err := os.MkdirAll(skillsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(targetFile, []byte("target"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.Symlink(targetFile, filepath.Join(skillsDir, "link.txt"))
	g.Expect(err).ToNot(HaveOccurred())

	txn := memory.NewTransaction(memory.TransactionOpts{
		SkillsDir: skillsDir,
	})

	err = txn.CreateBackups()

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(txn.BackupsCreated()).To(BeTrue())

	err = txn.Cleanup()
	g.Expect(err).ToNot(HaveOccurred())
}

// TestTransaction_LoadTransactionLog_InvalidJSON verifies error on malformed JSON.
func TestTransaction_LoadTransactionLog_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "txn.json")

	err := os.WriteFile(logPath, []byte("not valid json"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	changes, err := memory.LoadTransactionLog(logPath)

	g.Expect(err).To(HaveOccurred())
	g.Expect(changes).To(BeNil())
}

// TestTransaction_LoadTransactionLog_MissingFile verifies error on missing file.
func TestTransaction_LoadTransactionLog_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	changes, err := memory.LoadTransactionLog("/nonexistent/path/txn.json")

	g.Expect(err).To(HaveOccurred())
	g.Expect(changes).To(BeNil())
}

// TestTransaction_LoadTransactionLog_Success verifies loading a valid transaction log.
func TestTransaction_LoadTransactionLog_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "txn.json")

	txn := memory.NewTransaction(memory.TransactionOpts{})
	txn.RecordChange(memory.ChangeRecord{
		Type:   "prune",
		Tier:   "embeddings",
		Target: "42",
		Before: "old content",
		After:  "",
	})

	err := txn.SaveLog(logPath)
	g.Expect(err).ToNot(HaveOccurred())

	changes, err := memory.LoadTransactionLog(logPath)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(changes).ToNot(BeNil())
	g.Expect(changes).To(HaveLen(1))

	if len(changes) == 0 {
		t.Fatal("changes is empty")
	}

	g.Expect(changes[0].Type).To(Equal("prune"))
	g.Expect(changes[0].Tier).To(Equal("embeddings"))
}

// TestTransaction_Rollback_WithDB verifies Rollback restores all backed-up files.
func TestTransaction_Rollback_WithDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	err := os.WriteFile(dbPath, []byte("db content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(claudeMDPath, []byte("claude md content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	txn := memory.NewTransaction(memory.TransactionOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
	})

	err = txn.CreateBackups()
	g.Expect(err).ToNot(HaveOccurred())

	// Modify files after backup
	err = os.WriteFile(dbPath, []byte("modified db"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = txn.Rollback()
	g.Expect(err).ToNot(HaveOccurred())

	// Verify files are restored
	data, err := os.ReadFile(dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("db content"))
}

// TestTransaction_SaveLog_ErrorOnBadPath verifies SaveLog returns error on unwritable path.
func TestTransaction_SaveLog_ErrorOnBadPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	txn := memory.NewTransaction(memory.TransactionOpts{})

	err := txn.SaveLog("/dev/null/invalid/path/txn.json")

	g.Expect(err).To(HaveOccurred())
}

// TestTransaction_SaveLog_Success verifies SaveLog writes valid JSON.
func TestTransaction_SaveLog_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "txn.json")

	txn := memory.NewTransaction(memory.TransactionOpts{})
	txn.RecordChange(memory.ChangeRecord{
		Type:   "decay",
		Tier:   "embeddings",
		Target: "99",
		Before: "0.9",
		After:  "0.81",
	})
	txn.RecordChange(memory.ChangeRecord{
		Type:   "prune",
		Tier:   "skills",
		Target: "old-skill",
	})

	err := txn.SaveLog(logPath)

	g.Expect(err).ToNot(HaveOccurred())

	data, err := os.ReadFile(logPath)
	g.Expect(err).ToNot(HaveOccurred())

	var changes []memory.ChangeRecord

	err = json.Unmarshal(data, &changes)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(changes).To(HaveLen(2))
}
