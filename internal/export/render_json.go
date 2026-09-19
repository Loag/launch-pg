package export

import (
	"encoding/json"

	"launch-pg/internal/model"
)

// JSONRenderer writes an array of credential objects.
type JSONRenderer struct{}

func (JSONRenderer) Kind() string             { return "json" }
func (JSONRenderer) NeedsPassword() bool      { return true }
func (JSONRenderer) Check(opts Options) error { return nil }

type jsonCredential struct {
	Role     string `json:"role"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslmode"`
	URL      string `json:"url"`
}

func (JSONRenderer) Render(creds []model.Credential, _ Options) ([]byte, error) {
	out := make([]jsonCredential, len(creds))
	for i, c := range creds {
		out[i] = jsonCredential{
			Role: c.User, Host: c.Host, Port: c.Port, Database: c.Database,
			Username: c.User, Password: c.Password, SSLMode: c.SSLMode, URL: c.URL(),
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
