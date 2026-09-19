package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONLFile appends one JSON object per line to a file.
type JSONLFile struct {
	path string
}

func NewJSONLFile(path string) *JSONLFile {
	return &JSONLFile{path: path}
}

func (j *JSONLFile) Record(e Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(j.path), 0o700); err != nil {
		return fmt.Errorf("create audit dir: %w", err)
	}
	f, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
