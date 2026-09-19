package credstore

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileStore keeps passwords in a 0600 file, one "server:password" per line.
// Server names cannot contain ':', so the first ':' is the separator.
type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (f *FileStore) Get(server string) (string, error) {
	entries, err := f.read()
	if err != nil {
		return "", err
	}
	pw, ok := entries[server]
	if !ok {
		return "", fmt.Errorf("%w: %q in %s", ErrNotFound, server, f.path)
	}
	return pw, nil
}

func (f *FileStore) Set(server, password string) error {
	if strings.ContainsAny(password, "\r\n") {
		return errors.New("file store cannot hold passwords containing newlines")
	}
	entries, err := f.read()
	if err != nil {
		return err
	}
	entries[server] = password
	return f.write(entries)
}

func (f *FileStore) Delete(server string) error {
	entries, err := f.read()
	if err != nil {
		return err
	}
	if _, ok := entries[server]; !ok {
		return nil
	}
	delete(entries, server)
	return f.write(entries)
}

func (f *FileStore) read() (map[string]string, error) {
	entries := map[string]string{}
	file, err := os.Open(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return entries, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", f.path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, pw, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		entries[name] = pw
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", f.path, err)
	}
	return entries, nil
}

func (f *FileStore) write(entries map[string]string) error {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("# launch-pg admin credentials — server:password\n")
	for _, name := range names {
		fmt.Fprintf(&b, "%s:%s\n", name, entries[name])
	}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("create dir for %s: %w", f.path, err)
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, f.path); err != nil {
		return fmt.Errorf("replace %s: %w", f.path, err)
	}
	return nil
}
