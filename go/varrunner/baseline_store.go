package varrunner

import (
	"os"
	"path/filepath"
)

// FileBaselineStore is the filesystem drift baseline: var.lock.json at the
// project root. The core owns the format; this adapter moves only raw text. Port
// of baseline_store.py / FileBaselineStore. Implements varcore.BaselineStore.
type FileBaselineStore struct {
	path string
}

// NewFileBaselineStore returns a store reading/writing <root>/var.lock.json.
func NewFileBaselineStore(root string) *FileBaselineStore {
	return &FileBaselineStore{path: filepath.Join(root, "var.lock.json")}
}

func (s *FileBaselineStore) Read() (string, bool) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (s *FileBaselineStore) Write(contents string) {
	_ = os.WriteFile(s.path, []byte(contents), 0o644)
}
