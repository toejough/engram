package cli_test

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/toejough/engram/internal/cli"
)

// unexported constants.
const (
	fakeSkillDirPerm  fs.FileMode = 0o755
	fakeSkillFilePerm fs.FileMode = 0o644
)

type fakeSkillNodeKind int

// fakeSkillNodeKind values.
const (
	fakeSkillDir fakeSkillNodeKind = iota
	fakeSkillFile
	fakeSkillLink
)

// unexported variables.
var (
	errFakeNotDir     = errors.New("fake: not a directory")
	errFakeNotFile    = errors.New("fake: is a directory")
	errFakeNotLink    = errors.New("fake: not a symlink")
	errFakeReadDenied = fmt.Errorf("fake: %w", fs.ErrPermission)
)

type cliSkillScanResult = cli.SkillScanResult

// fakeSkillFS is an in-memory cli.SkillSourceFS for the source-scanner tests.
// It models real directories, regular files, symlinks (to dirs, to files, and
// dangling), and per-path read errors distinct from not-exist. Like the real
// scanners' contract, ReadDir/ReadFile are exact-path lookups: a path through
// an unresolved symlink is not found, so a scanner that forgets to resolve a
// link sees ErrNotExist rather than silently succeeding.
type fakeSkillFS struct {
	nodes map[string]*fakeSkillNode
}

func (f *fakeSkillFS) Lstat(path string) (fs.FileInfo, error) {
	node, ok := f.nodes[filepath.Clean(path)]
	if !ok {
		return nil, &fs.PathError{Op: "lstat", Path: path, Err: fs.ErrNotExist}
	}

	if node.lstatErr != nil {
		return nil, &fs.PathError{Op: "lstat", Path: path, Err: node.lstatErr}
	}

	return fakeSkillInfo{name: filepath.Base(path), node: node}, nil
}

func (f *fakeSkillFS) ReadDir(path string) ([]fs.DirEntry, error) {
	clean := filepath.Clean(path)

	node, ok := f.nodes[clean]
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: path, Err: fs.ErrNotExist}
	}

	if node.kind != fakeSkillDir {
		return nil, &fs.PathError{Op: "readdir", Path: path, Err: errFakeNotDir}
	}

	if node.readErr != nil {
		return nil, &fs.PathError{Op: "readdir", Path: path, Err: node.readErr}
	}

	entries := make([]fs.DirEntry, 0)

	for child, childNode := range f.nodes {
		if child == clean || filepath.Dir(child) != clean {
			continue
		}

		entries = append(entries, fs.FileInfoToDirEntry(fakeSkillInfo{name: filepath.Base(child), node: childNode}))
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return entries, nil
}

func (f *fakeSkillFS) ReadFile(path string) ([]byte, error) {
	node, ok := f.nodes[filepath.Clean(path)]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	if node.kind != fakeSkillFile {
		return nil, &fs.PathError{Op: "read", Path: path, Err: errFakeNotFile}
	}

	if node.readErr != nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: node.readErr}
	}

	return node.content, nil
}

func (f *fakeSkillFS) Readlink(path string) (string, error) {
	node, ok := f.nodes[filepath.Clean(path)]
	if !ok {
		return "", &fs.PathError{Op: "readlink", Path: path, Err: fs.ErrNotExist}
	}

	if node.kind != fakeSkillLink {
		return "", &fs.PathError{Op: "readlink", Path: path, Err: errFakeNotLink}
	}

	return node.target, nil
}

// dir creates a directory (and its parents).
func (f *fakeSkillFS) dir(path string) *fakeSkillFS {
	clean := filepath.Clean(path)
	if clean == "/" {
		return f
	}

	f.dir(filepath.Dir(clean))

	if _, ok := f.nodes[clean]; !ok {
		f.nodes[clean] = &fakeSkillNode{kind: fakeSkillDir}
	}

	return f
}

// failLstat makes Lstat of path fail with a non-not-exist error.
func (f *fakeSkillFS) failLstat(path string) *fakeSkillFS {
	if node, ok := f.nodes[filepath.Clean(path)]; ok && node != nil {
		node.lstatErr = errFakeReadDenied
	}

	return f
}

// failRead makes ReadDir/ReadFile of an existing path fail with a
// non-not-exist error.
func (f *fakeSkillFS) failRead(path string) *fakeSkillFS {
	if node, ok := f.nodes[filepath.Clean(path)]; ok && node != nil {
		node.readErr = errFakeReadDenied
	}

	return f
}

// file creates a regular file (and its parent dirs).
func (f *fakeSkillFS) file(path, content string) *fakeSkillFS {
	clean := filepath.Clean(path)
	f.dir(filepath.Dir(clean))
	f.nodes[clean] = &fakeSkillNode{kind: fakeSkillFile, content: []byte(content)}

	return f
}

// link creates a symlink at path pointing to target (absolute or relative),
// which need not exist.
func (f *fakeSkillFS) link(path, target string) *fakeSkillFS {
	clean := filepath.Clean(path)
	f.dir(filepath.Dir(clean))
	f.nodes[clean] = &fakeSkillNode{kind: fakeSkillLink, target: target}

	return f
}

type fakeSkillInfo struct {
	name string
	node *fakeSkillNode
}

func (i fakeSkillInfo) IsDir() bool { return i.node.kind == fakeSkillDir }

func (i fakeSkillInfo) ModTime() time.Time { return time.Time{} }

func (i fakeSkillInfo) Mode() fs.FileMode {
	switch i.node.kind {
	case fakeSkillDir:
		return fs.ModeDir | fakeSkillDirPerm
	case fakeSkillLink:
		return fs.ModeSymlink | fakeSkillFilePerm
	case fakeSkillFile:
		return fakeSkillFilePerm
	default:
		return fakeSkillFilePerm
	}
}

func (i fakeSkillInfo) Name() string { return i.name }

func (i fakeSkillInfo) Size() int64 { return int64(len(i.node.content)) }

func (i fakeSkillInfo) Sys() any { return nil }

type fakeSkillNode struct {
	kind    fakeSkillNodeKind
	content []byte
	target  string
	// readErr fails ReadDir (dirs) or ReadFile (files) with a non-not-exist
	// error, e.g. fs.ErrPermission.
	readErr error
	// lstatErr fails Lstat itself.
	lstatErr error
}

// candidateNames returns the candidates' Names, in order.
func candidateNames(result cliSkillScanResult) []string {
	names := make([]string, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		names = append(names, candidate.Name)
	}

	return names
}

func newFakeSkillFS() *fakeSkillFS {
	fsys := &fakeSkillFS{nodes: map[string]*fakeSkillNode{}}
	fsys.nodes["/"] = &fakeSkillNode{kind: fakeSkillDir}

	return fsys
}

// rootScanned returns the scanned flag recorded for root, and whether root
// was recorded at all.
func rootScanned(result cliSkillScanResult, root string) (scanned, found bool) {
	for _, recorded := range result.Roots {
		if recorded.Path == root {
			return recorded.Scanned, true
		}
	}

	return false, false
}

// warningsMention reports whether any warning contains substr.
func warningsMention(result cliSkillScanResult, substr string) bool {
	for _, warning := range result.Warnings {
		if strings.Contains(warning, substr) {
			return true
		}
	}

	return false
}
