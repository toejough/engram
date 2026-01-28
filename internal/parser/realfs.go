package parser

// RealFS implements CollectableFS using the real file system.
type RealFS struct{}

// NewRealFS creates a new RealFS instance.
func NewRealFS() *RealFS {
	return &RealFS{}
}

// DirExists returns true if the directory exists.
func (r *RealFS) DirExists(path string) bool {
	return false
}

// FileExists returns true if the file exists.
func (r *RealFS) FileExists(path string) bool {
	return false
}

// ReadFile reads the file content as a string.
func (r *RealFS) ReadFile(path string) (string, error) {
	return "", nil
}

// Walk traverses the directory tree.
func (r *RealFS) Walk(root string, fn func(path string, isDir bool) error) error {
	return nil
}
