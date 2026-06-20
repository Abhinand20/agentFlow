package emit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// FS is an ordered in-memory file set for binding output.
type FS struct {
	files map[string][]byte
	order []string
}

// NewFS returns an empty file set.
func NewFS() *FS {
	return &FS{files: make(map[string][]byte)}
}

// Write stores content at a relative path, replacing any existing entry.
func (fs *FS) Write(path string, content []byte) {
	if _, ok := fs.files[path]; !ok {
		fs.order = append(fs.order, path)
		sort.Strings(fs.order)
	}
	cp := make([]byte, len(content))
	copy(cp, content)
	fs.files[path] = cp
}

// Get returns the content at path and whether it exists.
func (fs *FS) Get(path string) ([]byte, bool) {
	content, ok := fs.files[path]
	return content, ok
}

// Paths returns all stored paths in sorted order.
func (fs *FS) Paths() []string {
	out := make([]string, len(fs.order))
	copy(out, fs.order)
	return out
}

// Flush writes all files under dir, creating parent directories as needed.
func (fs *FS) Flush(dir string) error {
	for _, path := range fs.Paths() {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("emit: mkdir %s: %w", filepath.Dir(full), err)
		}
		content, _ := fs.Get(path)
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return fmt.Errorf("emit: write %s: %w", full, err)
		}
	}
	return nil
}
