package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// EvaluationService implements the read (pull) path from
// docs/business/requirements/0007-feature-flag-evaluation-api.md: given an
// environment, return every requested flag's configured value in that
// environment. Unlike FeatureFlagValueService.List (admin-facing, one
// flag's values across environments), this is the reverse shape (one
// environment's values across many/all flags), and it's a separate service
// rather than a method on FeatureFlagValueService because it's reached
// through Service Key (Bearer) authentication instead of a session, and —
// per 0007 — it never writes to the audit log, unlike every write path in
// this package.
type EvaluationService struct {
	flags        repository.FeatureFlagRepository
	environments repository.EnvironmentRepository
	values       repository.FeatureFlagEnvironmentValueRepository
}

// NewEvaluationService returns an EvaluationService backed by the given
// repositories.
func NewEvaluationService(flags repository.FeatureFlagRepository, environments repository.EnvironmentRepository, values repository.FeatureFlagEnvironmentValueRepository) *EvaluationService {
	return &EvaluationService{flags: flags, environments: environments, values: values}
}

// EvaluatedFlag is a single flag's configured state in the requested
// environment: Enabled, and — for non-Boolean types — its Value. A flag
// with no configured value for the environment evaluates as disabled, per
// 0005/0007, and is returned the same way rather than omitted.
type EvaluatedFlag struct {
	Key     string
	Enabled bool
	Value   *string
}

// Evaluate returns the evaluated state of every flag in keys (or every
// flag, if keys is empty) for environmentID. Unknown ids in keys are
// silently omitted from the result rather than treated as an error — this
// is a high-volume evaluation path, not an admin lookup, per 0007.
func (s *EvaluationService) Evaluate(ctx context.Context, environmentID string, keys []string) ([]EvaluatedFlag, error) {
	if _, err := s.environments.GetByID(ctx, environmentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationErrorf("unknown environment id %q", environmentID)
		}
		return nil, fmt.Errorf("checking environment %q: %w", environmentID, err)
	}

	flags, err := s.flags.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing feature flags: %w", err)
	}
	if keys != nil {
		wanted := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			wanted[k] = struct{}{}
		}
		filtered := flags[:0]
		for _, f := range flags {
			if _, ok := wanted[f.Key]; ok {
				filtered = append(filtered, f)
			}
		}
		flags = filtered
	}

	values, err := s.values.ListByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}
	valueByFlagID := make(map[string]*model.FeatureFlagEnvironmentValue, len(values))
	for i := range values {
		valueByFlagID[values[i].FeatureFlagID] = &values[i]
	}

	evaluated := make([]EvaluatedFlag, 0, len(flags))
	for _, f := range flags {
		v, ok := valueByFlagID[f.ID]
		if !ok {
			evaluated = append(evaluated, EvaluatedFlag{Key: f.Key, Enabled: false, Value: nil})
			continue
		}
		evaluated = append(evaluated, EvaluatedFlag{Key: f.Key, Enabled: v.Enabled, Value: v.Value})
	}

	return evaluated, nil
}
