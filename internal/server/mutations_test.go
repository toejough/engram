package server_test

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/server"
)

func TestDeleteMemory_ArchivesFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn:     func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		mkdirAllFn: func(_ string, _ fs.FileMode) error { return nil },
		renameFn: func(src, dst string) error {
			g.Expect(src).To(HaveSuffix("/memories/test-mem.toml"))
			g.Expect(dst).To(HaveSuffix("/archived/test-mem.toml"))

			return nil
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/test-mem", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["slug"]).To(Equal("test-mem"))
	g.Expect(result["status"]).To(Equal("archived"))
}

func TestDeleteMemory_CreatesArchivedDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mkdirCalled := false
	ops := &mockFileOps{
		statFn: func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		mkdirAllFn: func(path string, _ fs.FileMode) error {
			g.Expect(path).To(HaveSuffix("/archived"))

			mkdirCalled = true

			return nil
		},
		renameFn: func(_, _ string) error { return nil },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/some-slug", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(mkdirCalled).To(BeTrue())
}

func TestDeleteMemory_InvalidSlug(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/..%2Fetc", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestDeleteMemory_MkdirError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn:     func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		mkdirAllFn: func(_ string, _ fs.FileMode) error { return errMkdirFailed },
		renameFn:   func(_, _ string) error { return nil },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/test-mem", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestDeleteMemory_NotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn: func(_ string) (fs.FileInfo, error) { return nil, fs.ErrNotExist },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/nonexistent", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNotFound))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("memory not found"))
}

func TestDeleteMemory_RenameError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn:     func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		mkdirAllFn: func(_ string, _ fs.FileMode) error { return nil },
		renameFn:   func(_, _ string) error { return errRenameFailed },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/memories/test-mem", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestRestoreMemory_InvalidSlug(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memories/..%2Fetc/restore", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestRestoreMemory_MovesFromArchive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn: func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		renameFn: func(src, dst string) error {
			g.Expect(src).To(HaveSuffix("/archived/test-mem.toml"))
			g.Expect(dst).To(HaveSuffix("/memories/test-mem.toml"))

			return nil
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memories/test-mem/restore", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["slug"]).To(Equal("test-mem"))
	g.Expect(result["status"]).To(Equal("restored"))
}

func TestRestoreMemory_NotInArchive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn: func(_ string) (fs.FileInfo, error) { return nil, fs.ErrNotExist },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memories/nonexistent/restore", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNotFound))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("memory not found in archive"))
}

func TestRestoreMemory_RenameError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn:   func(_ string) (fs.FileInfo, error) { return nil, nil }, //nolint:nilnil
		renameFn: func(_, _ string) error { return errRenameFailed },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memories/test-mem/restore", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestRestoreMemory_StatError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := &mockFileOps{
		statFn: func(_ string) (fs.FileInfo, error) { return nil, errStatFailed },
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithFileOps(ops),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memories/test-mem/restore", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestUpdateMemory_InvalidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := &mockModifier{
		readModifyWriteFn: func(_ string, _ func(*memory.MemoryRecord)) error {
			return nil
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithModifier(modifier),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/memories/test-mem", strings.NewReader("{invalid"))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("invalid JSON body"))
}

func TestUpdateMemory_InvalidSlug(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/memories/..%2Fhack", strings.NewReader("{}"))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestUpdateMemory_NotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := &mockModifier{
		readModifyWriteFn: func(_ string, _ func(*memory.MemoryRecord)) error {
			return fs.ErrNotExist
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithModifier(modifier),
	)

	rec := httptest.NewRecorder()
	body := `{"situation":"s","behavior":"b","impact":"i","action":"a"}`
	req := httptest.NewRequest(http.MethodPut, "/api/memories/nonexistent", strings.NewReader(body))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNotFound))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("memory not found"))
}

func TestUpdateMemory_ReadModifyWriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := &mockModifier{
		readModifyWriteFn: func(_ string, _ func(*memory.MemoryRecord)) error {
			return errModifyFailed
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithModifier(modifier),
	)

	rec := httptest.NewRecorder()
	body := `{"situation":"s","behavior":"b","impact":"i","action":"a"}`
	req := httptest.NewRequest(http.MethodPut, "/api/memories/test-mem", strings.NewReader(body))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestUpdateMemory_SetsUpdatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	fixedTime := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	var capturedRecord *memory.MemoryRecord

	modifier := &mockModifier{
		readModifyWriteFn: func(path string, mutate func(*memory.MemoryRecord)) error {
			g.Expect(path).To(HaveSuffix("/memories/test-mem.toml"))

			rec := &memory.MemoryRecord{
				Situation: "old situation",
				Content:   memory.ContentFields{Behavior: "old behavior"},
			}

			mutate(rec)

			capturedRecord = rec

			return nil
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithModifier(modifier),
		server.WithNow(func() time.Time { return fixedTime }),
	)

	rec := httptest.NewRecorder()
	body := `{"situation":"new sit","behavior":"new beh",` +
		`"impact":"new imp","action":"new act",` +
		`"projectScoped":true,"projectSlug":"engram"}`
	req := httptest.NewRequest(http.MethodPut, "/api/memories/test-mem", strings.NewReader(body))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	g.Expect(capturedRecord).NotTo(BeNil())

	if capturedRecord == nil {
		return
	}

	g.Expect(capturedRecord.Situation).To(Equal("new sit"))
	g.Expect(capturedRecord.Content.Behavior).To(Equal("new beh"))
	g.Expect(capturedRecord.Content.Impact).To(Equal("new imp"))
	g.Expect(capturedRecord.Content.Action).To(Equal("new act"))
	g.Expect(capturedRecord.ProjectScoped).To(BeTrue())
	g.Expect(capturedRecord.ProjectSlug).To(Equal("engram"))
	g.Expect(capturedRecord.UpdatedAt).To(Equal("2026-04-03T12:00:00Z"))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["slug"]).To(Equal("test-mem"))
	g.Expect(result["updatedAt"]).To(Equal("2026-04-03T12:00:00Z"))
}

func TestUpdateMemory_WrappedNotFoundError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := &mockModifier{
		readModifyWriteFn: func(_ string, _ func(*memory.MemoryRecord)) error {
			return &wrappedError{msg: "reading /data/memories/gone.toml", inner: fs.ErrNotExist}
		},
	}

	srv := server.NewServer(
		&mockLister{memories: []*memory.Stored{}},
		"/data",
		server.WithModifier(modifier),
	)

	rec := httptest.NewRecorder()
	body := `{"situation":"s","behavior":"b","impact":"i","action":"a"}`
	req := httptest.NewRequest(http.MethodPut, "/api/memories/gone", strings.NewReader(body))

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNotFound))
}

// unexported variables.
var (
	errMkdirFailed  = errorf("mkdir failed")
	errModifyFailed = errorf("modify failed")
	errRenameFailed = errorf("rename failed")
	errStatFailed   = errorf("stat failed")
)

type mockFileOps struct {
	renameFn   func(src, dst string) error
	mkdirAllFn func(path string, perm fs.FileMode) error
	statFn     func(path string) (fs.FileInfo, error)
}

func (m *mockFileOps) MkdirAll(path string, perm fs.FileMode) error {
	return m.mkdirAllFn(path, perm)
}

func (m *mockFileOps) Rename(oldpath, newpath string) error {
	return m.renameFn(oldpath, newpath)
}

func (m *mockFileOps) Stat(path string) (fs.FileInfo, error) {
	return m.statFn(path)
}

type mockModifier struct {
	readModifyWriteFn func(path string, mutate func(*memory.MemoryRecord)) error
}

func (m *mockModifier) ReadModifyWrite(path string, mutate func(*memory.MemoryRecord)) error {
	return m.readModifyWriteFn(path, mutate)
}

type wrappedError struct {
	msg   string
	inner error
}

func (e *wrappedError) Error() string { return e.msg + ": " + e.inner.Error() }

func (e *wrappedError) Unwrap() error { return e.inner }
