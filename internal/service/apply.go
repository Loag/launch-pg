package service

import (
	"context"

	"launch-pg/internal/backup"
	"launch-pg/internal/config"
	"launch-pg/internal/plan"
)

// PlanApplier executes a plan against a server.
type PlanApplier interface {
	Apply(ctx context.Context, server config.Server, p *plan.Plan) (plan.Result, error)
}

// ApplyService applies prepared plans, taking any requested backup first.
type ApplyService struct {
	applier PlanApplier
	dumper  backup.Dumper
}

func NewApplyService(applier PlanApplier, dumper backup.Dumper) *ApplyService {
	return &ApplyService{applier: applier, dumper: dumper}
}

func (s *ApplyService) Apply(ctx context.Context, p Prepared) (plan.Result, error) {
	if err := p.Plan.Validate(); err != nil {
		return plan.Result{Skipped: p.Plan.Actions}, err
	}
	if p.Backup != nil {
		if err := s.dumper.Dump(ctx, p.Server, p.Backup.Database, p.Backup.Path); err != nil {
			return plan.Result{Skipped: p.Plan.Actions}, err
		}
	}
	return s.applier.Apply(ctx, p.Server, p.Plan)
}
