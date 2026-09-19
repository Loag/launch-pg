package model

import (
	"net/url"
	"strconv"
)

// Credential is everything an app needs to connect as a role.
type Credential struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// URL returns a postgres:// connection URL with all parts escaped.
func (c Credential) URL() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + strconv.Itoa(c.Port),
		Path:   "/" + c.Database,
	}
	if c.SSLMode != "" {
		u.RawQuery = url.Values{"sslmode": {c.SSLMode}}.Encode()
	}
	return u.String()
}
