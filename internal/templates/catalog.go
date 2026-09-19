package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// Source lists and fetches templates.
type Source interface {
	List() ([]Template, error)
	Get(name string) (Template, error)
}

// Catalog merges built-in templates with user templates from a directory.
// A user template with the same name overrides the built-in one.
type Catalog struct {
	userDir string
}

func NewCatalog(userDir string) *Catalog {
	return &Catalog{userDir: userDir}
}

func (c *Catalog) List() ([]Template, error) {
	byName := map[string]Template{}
	builtin, err := fs.Sub(builtinFS, "builtin")
	if err != nil {
		return nil, err
	}
	if err := loadDir(builtin, byName); err != nil {
		return nil, fmt.Errorf("built-in templates: %w", err)
	}
	if err := loadDir(os.DirFS(c.userDir), byName); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("templates in %s: %w", c.userDir, err)
	}

	out := make([]Template, 0, len(byName))
	for _, t := range byName {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Catalog) Get(name string) (Template, error) {
	all, err := c.List()
	if err != nil {
		return Template{}, err
	}
	for _, t := range all {
		if t.Name == name {
			return t, nil
		}
	}
	return Template{}, fmt.Errorf("unknown template %q", name)
}

func loadDir(fsys fs.FS, into map[string]Template) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		ext := path.Ext(e.Name())
		if e.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return err
		}
		var t Template
		if err := yaml.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		if t.Name == "" {
			t.Name = strings.TrimSuffix(e.Name(), ext)
		}
		into[t.Name] = t
	}
	return nil
}
