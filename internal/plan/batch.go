package plan

// batch is a run of consecutive actions that execute on the same database
// connection. Transactional batches run inside one BEGIN/COMMIT.
type batch struct {
	database      string
	transactional bool
	actions       []Action
}

// batches groups consecutive actions without reordering them. Each
// non-transactional action gets its own batch.
func batches(actions []Action) []batch {
	var out []batch
	for _, a := range actions {
		n := len(out)
		if n > 0 && a.Transactional() && out[n-1].transactional && out[n-1].database == a.Database() {
			out[n-1].actions = append(out[n-1].actions, a)
			continue
		}
		out = append(out, batch{database: a.Database(), transactional: a.Transactional(), actions: []Action{a}})
	}
	return out
}
