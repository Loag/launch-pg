package plan

import (
	"errors"
	"fmt"
)

// Plan is an ordered set of actions against one server. Order matters:
// the Planner is responsible for dependency ordering.
type Plan struct {
	Actions []Action
}

func (p *Plan) Add(actions ...Action) {
	p.Actions = append(p.Actions, actions...)
}

func (p *Plan) Empty() bool { return len(p.Actions) == 0 }

// Validate checks every action, returning all problems at once.
func (p *Plan) Validate() error {
	var errs []error
	for i, a := range p.Actions {
		if err := a.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("action %d (%s): %w", i+1, a.Kind(), err))
		}
	}
	return errors.Join(errs...)
}
