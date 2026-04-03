package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"engram/internal/memory"
)

func (s *Server) handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if !isValidSlug(slug) {
		writeJSONError(w, "invalid slug", http.StatusBadRequest)

		return
	}

	if s.fileOps == nil {
		writeJSONError(w, "delete not configured", http.StatusInternalServerError)

		return
	}

	srcPath := filepath.Join(s.dataDir, "memories", slug+".toml")

	_, err := s.fileOps.Stat(srcPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSONError(w, "memory not found", http.StatusNotFound)

			return
		}

		writeJSONError(w, "failed to check memory file", http.StatusInternalServerError)

		return
	}

	archivedDir := filepath.Join(s.dataDir, "archived")

	mkdirErr := s.fileOps.MkdirAll(archivedDir, archivedDirPerm)
	if mkdirErr != nil {
		writeJSONError(w, "failed to create archive directory", http.StatusInternalServerError)

		return
	}

	dstPath := filepath.Join(archivedDir, slug+".toml")

	renameErr := s.fileOps.Rename(srcPath, dstPath)
	if renameErr != nil {
		writeJSONError(w, "failed to archive memory", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(mutationResponse{Slug: slug, Status: "archived"})
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleRestoreMemory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if !isValidSlug(slug) {
		writeJSONError(w, "invalid slug", http.StatusBadRequest)

		return
	}

	if s.fileOps == nil {
		writeJSONError(w, "restore not configured", http.StatusInternalServerError)

		return
	}

	srcPath := filepath.Join(s.dataDir, "archived", slug+".toml")

	_, err := s.fileOps.Stat(srcPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSONError(w, "memory not found in archive", http.StatusNotFound)

			return
		}

		writeJSONError(w, "failed to check archived file", http.StatusInternalServerError)

		return
	}

	dstPath := filepath.Join(s.dataDir, "memories", slug+".toml")

	renameErr := s.fileOps.Rename(srcPath, dstPath)
	if renameErr != nil {
		writeJSONError(w, "failed to restore memory", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(mutationResponse{Slug: slug, Status: "restored"})
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleUpdateMemory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if !isValidSlug(slug) {
		writeJSONError(w, "invalid slug", http.StatusBadRequest)

		return
	}

	if s.modifier == nil {
		writeJSONError(w, "update not configured", http.StatusInternalServerError)

		return
	}

	var req updateMemoryRequest

	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)

		return
	}

	memoryPath := filepath.Join(s.dataDir, "memories", slug+".toml")
	now := s.now().UTC().Format(time.RFC3339)

	modifyErr := s.modifier.ReadModifyWrite(memoryPath, func(rec *memory.MemoryRecord) {
		rec.Situation = req.Situation
		rec.Behavior = req.Behavior
		rec.Impact = req.Impact
		rec.Action = req.Action
		rec.ProjectScoped = req.ProjectScoped
		rec.ProjectSlug = req.ProjectSlug
		rec.UpdatedAt = now
	})
	if modifyErr != nil {
		if errors.Is(modifyErr, fs.ErrNotExist) {
			writeJSONError(w, "memory not found", http.StatusNotFound)

			return
		}

		writeJSONError(w, "failed to update memory", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(mutationResponse{Slug: slug, UpdatedAt: now})
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// unexported constants.
const (
	archivedDirPerm = 0o750
)

type mutationResponse struct {
	Slug      string `json:"slug"`
	Status    string `json:"status,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type updateMemoryRequest struct {
	Situation     string `json:"situation"`
	Behavior      string `json:"behavior"`
	Impact        string `json:"impact"`
	Action        string `json:"action"`
	ProjectScoped bool   `json:"projectScoped"`
	ProjectSlug   string `json:"projectSlug"`
}
