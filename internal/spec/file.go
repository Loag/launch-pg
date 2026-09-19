// Package spec parses declarative project files for `launch-pg apply`.
package spec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// File is a project spec as written by the user. Only Project is required
// when Template supplies everything else.
type File struct {
	Server   string          `yaml:"server"`
	Project  string          `yaml:"project"`
	Template string          `yaml:"template"`
	Database DatabaseSection `yaml:"database"`
	Roles    []RoleSection   `yaml:"roles"`
	Export   []string        `yaml:"export"`
}

// DatabaseSection overrides template database settings. Extensions are
// added to the template's, not substituted.
type DatabaseSection struct {
	Name       string   `yaml:"name"`
	Schema     string   `yaml:"schema"`
	Encoding   string   `yaml:"encoding"`
	Locale     string   `yaml:"locale"`
	Extensions []string `yaml:"extensions"`
}

// RoleSection declares a role. A role with the same name as a template role
// replaces it. Login defaults to true.
type RoleSection struct {
	Name            string `yaml:"name"`
	Preset          string `yaml:"preset"`
	Login           *bool  `yaml:"login"`
	ConnectionLimit int    `yaml:"connectionLimit"`
}

// Load reads and parses a spec file.
func Load(path string) (File, error) {
	if path == "" {
		return File{}, errors.New("spec file path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return File{}, fmt.Errorf("open spec: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

// Parse decodes a spec, rejecting unknown fields so typos fail loudly.
func Parse(r io.Reader) (File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return File{}, fmt.Errorf("read spec: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var f File
	if err := dec.Decode(&f); err != nil {
		if errors.Is(err, io.EOF) {
			return File{}, errors.New("spec is empty")
		}
		return File{}, fmt.Errorf("parse spec: %w", err)
	}
	if f.Project == "" {
		return File{}, errors.New("spec: project is required")
	}
	return f, nil
}
