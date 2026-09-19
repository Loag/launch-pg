package config

import (
	"fmt"
	"regexp"
)

const (
	DefaultPort          = 5432
	DefaultSSLMode       = "prefer"
	DefaultMaintenanceDB = "postgres"
)

var serverNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}$`)

var validSSLModes = map[string]bool{
	"disable": true, "allow": true, "prefer": true,
	"require": true, "verify-ca": true, "verify-full": true,
}

// Server describes a Postgres server the tool administers. The admin
// password is never kept here; CredentialStore names where it lives.
type Server struct {
	Name            string `yaml:"name"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	SSLMode         string `yaml:"sslmode"`
	MaintenanceDB   string `yaml:"maintenanceDatabase"`
	CredentialStore string `yaml:"credentialStore"`
}

// WithDefaults fills unset optional fields.
func (s Server) WithDefaults() Server {
	if s.Port == 0 {
		s.Port = DefaultPort
	}
	if s.SSLMode == "" {
		s.SSLMode = DefaultSSLMode
	}
	if s.MaintenanceDB == "" {
		s.MaintenanceDB = DefaultMaintenanceDB
	}
	return s
}

// Validate checks the fields required to connect.
func (s Server) Validate() error {
	if !serverNamePattern.MatchString(s.Name) {
		return fmt.Errorf("invalid server name %q: use letters, digits, '-' or '_', starting with a letter", s.Name)
	}
	if s.Host == "" {
		return fmt.Errorf("server %q: host is required", s.Name)
	}
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("server %q: port %d out of range", s.Name, s.Port)
	}
	if s.User == "" {
		return fmt.Errorf("server %q: user is required", s.Name)
	}
	if !validSSLModes[s.SSLMode] {
		return fmt.Errorf("server %q: unknown sslmode %q", s.Name, s.SSLMode)
	}
	if s.CredentialStore == "" {
		return fmt.Errorf("server %q: credential store is required", s.Name)
	}
	return nil
}
