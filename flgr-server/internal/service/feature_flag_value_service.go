package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

// FeatureFlagValueService implements the per-environment value half of
// docs/business/requirements/0005-feature-flag-management.md, gated by
// FeatureFlagValue: Read/Write — see FeatureFlagService's doc comment for
// why this is a separate service rather than folded into it.
type FeatureFlagValueService struct {
	flags        repository.FeatureFlagRepository
	environments repository.EnvironmentRepository
	values       repository.FeatureFlagEnvironmentValueRepository
}

// NewFeatureFlagValueService returns a FeatureFlagValueService backed by
// the given repositories.
func NewFeatureFlagValueService(flags repository.FeatureFlagRepository, environments repository.EnvironmentRepository, values repository.FeatureFlagEnvironmentValueRepository) *FeatureFlagValueService {
	return &FeatureFlagValueService{flags: flags, environments: environments, values: values}
}

func (s *FeatureFlagValueService) getFlag(ctx context.Context, id string) (*model.FeatureFlag, error) {
	f, err := s.flags.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFeatureFlagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetching feature flag: %w", err)
	}
	return f, nil
}

// List returns every configured per-environment value for featureFlagID —
// "configured" meaning an explicit row exists. An environment with no row
// here is implicitly disabled, per 0005 ("a flag with no configured value
// for a given environment evaluates as disabled"); callers render that as
// off, not as an error or a missing entry needing special handling.
func (s *FeatureFlagValueService) List(ctx context.Context, featureFlagID string) ([]model.FeatureFlagEnvironmentValue, error) {
	if _, err := s.getFlag(ctx, featureFlagID); err != nil {
		return nil, err
	}
	values, err := s.values.ListByFlag(ctx, featureFlagID)
	if err != nil {
		return nil, fmt.Errorf("listing feature flag environment values: %w", err)
	}
	return values, nil
}

// UpsertValueInput is the payload for FeatureFlagValueService.Upsert.
// Value is optional — nil means "leave whatever value was previously
// configured unchanged" (e.g. flipping Enabled without resupplying it
// every time); resolveFeatureFlagValue requires it to be present (either
// just-supplied or already on record) whenever a non-Boolean flag is being
// enabled, since that's what gets served once it's on.
type UpsertValueInput struct {
	Enabled bool
	Value   *string
}

// Upsert sets, for featureFlagID and environmentID, whether the flag is
// enabled and (for non-Boolean types) its value — creating the row if none
// exists yet, updating it otherwise.
func (s *FeatureFlagValueService) Upsert(ctx context.Context, actor Actor, featureFlagID, environmentID string, input UpsertValueInput) (*model.FeatureFlagEnvironmentValue, error) {
	if actor.IsZero() {
		return nil, ErrForbidden
	}

	flag, err := s.getFlag(ctx, featureFlagID)
	if err != nil {
		return nil, err
	}
	if _, err := s.environments.GetByID(ctx, environmentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationErrorf("unknown environment id %q", environmentID)
		}
		return nil, fmt.Errorf("checking environment %q: %w", environmentID, err)
	}

	existing, err := s.values.GetByFlagAndEnvironment(ctx, featureFlagID, environmentID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("fetching feature flag environment value: %w", err)
		}
		existing = nil
	}

	value, err := resolveFeatureFlagValue(flag.Type, input, existing)
	if err != nil {
		return nil, err
	}

	ts := now()
	if existing != nil {
		existing.Enabled = input.Enabled
		existing.Value = value
		existing.ModifiedByUserID = actor.UserID
		existing.ModifiedByServiceKeyID = actor.ServiceKeyID
		existing.ModifiedOn = ts
		if err := s.values.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("updating feature flag environment value: %w", err)
		}
		return existing, nil
	}

	v := &model.FeatureFlagEnvironmentValue{
		ID:            newID(),
		FeatureFlagID: featureFlagID,
		EnvironmentID: environmentID,
		Enabled:       input.Enabled,
		Value:         value,
		AuditFields: model.AuditFields{
			CreatedOn:              ts,
			ModifiedOn:             ts,
			CreatedByUserID:        actor.UserID,
			CreatedByServiceKeyID:  actor.ServiceKeyID,
			ModifiedByUserID:       actor.UserID,
			ModifiedByServiceKeyID: actor.ServiceKeyID,
		},
	}
	if err := s.values.Create(ctx, v); err != nil {
		if !errors.Is(err, repository.ErrFeatureFlagEnvironmentValueExists) {
			return nil, fmt.Errorf("creating feature flag environment value: %w", err)
		}

		// Lost a race with a concurrent first-write for the same (flag,
		// environment) pair — refetch the winner's row and update it
		// instead, re-merging Value against its now-known prior state (same
		// race-handling shape as EnvironmentService.resolveCategory).
		// Re-validating here isn't needed and can't fail: the merge only
		// differs from the one above when input.Value is nil, and if it
		// were nil with Enabled true, resolveFeatureFlagValue would already
		// have rejected the request before Create was ever attempted — so
		// reaching this point guarantees either a concrete input.Value (the
		// merge is unchanged) or Enabled is false (the requirement doesn't
		// apply).
		winner, getErr := s.values.GetByFlagAndEnvironment(ctx, featureFlagID, environmentID)
		if getErr != nil {
			return nil, fmt.Errorf("re-fetching feature flag environment value after a create race: %w", getErr)
		}
		value := mergeFeatureFlagValue(flag.Type, input, winner)
		winner.Enabled = input.Enabled
		winner.Value = value
		winner.ModifiedByUserID = actor.UserID
		winner.ModifiedByServiceKeyID = actor.ServiceKeyID
		winner.ModifiedOn = ts
		if err := s.values.Update(ctx, winner); err != nil {
			return nil, fmt.Errorf("updating feature flag environment value: %w", err)
		}
		return winner, nil
	}

	return v, nil
}

// mergeFeatureFlagValue resolves the value to persist: input's Value if
// supplied, else existing's (nil if there is no prior row) — "leave it
// unchanged" when the caller didn't resupply one — and always nil for
// Boolean flags, which never store one (0005: "unused for Boolean flags").
func mergeFeatureFlagValue(flagType string, input UpsertValueInput, existing *model.FeatureFlagEnvironmentValue) *string {
	if flagType == model.FeatureFlagTypeBoolean {
		return nil
	}
	value := input.Value
	if value == nil && existing != nil {
		value = existing.Value
	}
	return value
}

// resolveFeatureFlagValue is mergeFeatureFlagValue plus the one rule that
// can actually reject a write: enabling a non-Boolean flag always requires
// a resolvable value, since that's what gets served once it's on.
func resolveFeatureFlagValue(flagType string, input UpsertValueInput, existing *model.FeatureFlagEnvironmentValue) (*string, error) {
	value := mergeFeatureFlagValue(flagType, input, existing)
	if flagType != model.FeatureFlagTypeBoolean && input.Enabled && (value == nil || strings.TrimSpace(*value) == "") {
		return nil, validationErrorf("value is required to enable a %s flag in this environment", flagType)
	}
	return value, nil
}
