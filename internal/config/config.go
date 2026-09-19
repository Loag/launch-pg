package config

import (
	"errors"
	"fmt"
)

var ErrServerNotFound = errors.New("server not found")

// Config is the user's persisted tool configuration.
type Config struct {
	DefaultServer   string   `yaml:"defaultServer,omitempty"`
	DefaultTemplate string   `yaml:"defaultTemplate,omitempty"`
	Servers         []Server `yaml:"servers"`
}

// Server returns the named server, or the default server when name is empty.
func (c *Config) Server(name string) (Server, error) {
	if name == "" {
		name = c.DefaultServer
	}
	if name == "" {
		return Server{}, errors.New("no server given and no default server configured")
	}
	for _, s := range c.Servers {
		if s.Name == name {
			return s.WithDefaults(), nil
		}
	}
	return Server{}, fmt.Errorf("%w: %q", ErrServerNotFound, name)
}

// AddServer appends a new server. The first server added becomes the default.
func (c *Config) AddServer(s Server) error {
	s = s.WithDefaults()
	if err := s.Validate(); err != nil {
		return err
	}
	if c.hasServer(s.Name) {
		return fmt.Errorf("server %q already exists", s.Name)
	}
	c.Servers = append(c.Servers, s)
	if c.DefaultServer == "" {
		c.DefaultServer = s.Name
	}
	return nil
}

// RemoveServer deletes a server and clears the default if it pointed at it.
func (c *Config) RemoveServer(name string) error {
	for i, s := range c.Servers {
		if s.Name == name {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			if c.DefaultServer == name {
				c.DefaultServer = ""
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrServerNotFound, name)
}

// SetDefaultServer marks an existing server as the default.
func (c *Config) SetDefaultServer(name string) error {
	if !c.hasServer(name) {
		return fmt.Errorf("%w: %q", ErrServerNotFound, name)
	}
	c.DefaultServer = name
	return nil
}

func (c *Config) hasServer(name string) bool {
	for _, s := range c.Servers {
		if s.Name == name {
			return true
		}
	}
	return false
}
