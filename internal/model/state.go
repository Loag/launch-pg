package model

// ServerState is a snapshot of what exists on a server.
type ServerState struct {
	Info      ServerInfo
	Databases []DatabaseInfo
	Roles     []RoleInfo
}

func (s ServerState) Database(name string) (DatabaseInfo, bool) {
	for _, d := range s.Databases {
		if d.Name == name {
			return d, true
		}
	}
	return DatabaseInfo{}, false
}

func (s ServerState) Role(name string) (RoleInfo, bool) {
	for _, r := range s.Roles {
		if r.Name == name {
			return r, true
		}
	}
	return RoleInfo{}, false
}
