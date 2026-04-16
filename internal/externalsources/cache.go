package externalsources

// FileCache memoizes file contents (and errors) for the duration of a single
// engram process invocation. There is no cross-invocation persistence — the
// cache is dropped when the process exits.
type FileCache struct {
	reader ReaderFunc
	cache  map[string]cachedRead
}

// NewFileCache creates a FileCache backed by the given ReaderFunc.
func NewFileCache(reader ReaderFunc) *FileCache {
	return &FileCache{
		reader: reader,
		cache:  make(map[string]cachedRead),
	}
}

// Read returns the file's bytes, reading through to the underlying reader
// only on first access for a given path. Errors are cached too — repeated
// reads of an unreadable file do not retry.
func (c *FileCache) Read(path string) ([]byte, error) {
	if entry, ok := c.cache[path]; ok {
		return entry.content, entry.err
	}

	content, err := c.reader(path)
	c.cache[path] = cachedRead{content: content, err: err}

	return content, err
}

// ReaderFunc reads a file's bytes given an absolute path. Wired at the edge
// to os.ReadFile in production; replaced by a fake in tests.
type ReaderFunc func(path string) ([]byte, error)

type cachedRead struct {
	content []byte
	err     error
}
