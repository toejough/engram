package render

import (
	"html/template"
	"os"
	"sync"
	"time"
)

type cacheEntry struct {
	tmpl    *template.Template
	modTime time.Time
}

// TemplateCache holds compiled templates by path and recompiles one when its file changes.
type TemplateCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	stat    func(string) (os.FileInfo, error)
	compile func(string) (*template.Template, error)
}

// NewTemplateCache returns an empty cache that reads templates from disk.
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		entries: map[string]cacheEntry{},
		stat:    os.Stat,
		compile: func(path string) (*template.Template, error) { return template.ParseFiles(path) },
	}
}

// Get returns the compiled template for path, dropping and recompiling the cached entry when the
// file's modification time differs from the one it was compiled at.
func (c *TemplateCache) Get(path string) (*template.Template, error) {
	info, err := c.stat(path)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[path]; ok && entry.modTime.Equal(info.ModTime()) {
		return entry.tmpl, nil
	}

	delete(c.entries, path)

	tmpl, err := c.compile(path)
	if err != nil {
		return nil, err
	}

	c.entries[path] = cacheEntry{tmpl: tmpl, modTime: info.ModTime()}

	return tmpl, nil
}
