package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newFeatureFlagValueServiceMocks() (*mockFeatureFlagRepository, *mockEnvironmentRepository, *mockFeatureFlagEnvironmentValueRepository) {
	return new(mockFeatureFlagRepository), new(mockEnvironmentRepository), new(mockFeatureFlagEnvironmentValueRepository)
}

func TestFeatureFlagValueService_List_Success(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	values.On("ListByFlag", mock.Anything, "f1").Return([]model.FeatureFlagEnvironmentValue{{ID: "v1"}}, nil)

	svc := NewFeatureFlagValueService(flags, environments, values)
	got, err := svc.List(context.Background(), "f1")
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("List() = %v, want 1 value", got)
	}
}

func TestFeatureFlagValueService_List_FlagNotFound(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagValueService(flags, environments, values)
	_, err := svc.List(context.Background(), "missing")
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("List() error = %v, want ErrFeatureFlagNotFound", err)
	}
	values.AssertNotCalled(t, "ListByFlag")
}

func TestFeatureFlagValueService_List_FlagLookupError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	_, err := svc.List(context.Background(), "f1")
	if err == nil || errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("List() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagValueService_List_RepositoryError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1"}, nil)
	values.On("ListByFlag", mock.Anything, "f1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	_, err := svc.List(context.Background(), "f1")
	if err == nil {
		t.Fatal("List() expected an error, got nil")
	}
}

func TestFeatureFlagValueService_Upsert_NilActor(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	svc := NewFeatureFlagValueService(flags, environments, values)

	_, err := svc.Upsert(context.Background(), Actor{}, "f1", "env-1", UpsertValueInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Upsert() error = %v, want ErrForbidden", err)
	}
}

func TestFeatureFlagValueService_Upsert_FlagNotFound(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "missing", "env-1", UpsertValueInput{Enabled: true})
	if !errors.Is(err, ErrFeatureFlagNotFound) {
		t.Fatalf("Upsert() error = %v, want ErrFeatureFlagNotFound", err)
	}
}

func TestFeatureFlagValueService_Upsert_UnknownEnvironment(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "missing", UpsertValueInput{Enabled: true})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Upsert() error = %v, want ErrValidation", err)
	}
}

func TestFeatureFlagValueService_Upsert_EnvironmentLookupError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err == nil || errors.Is(err, ErrValidation) {
		t.Fatalf("Upsert() error = %v, want a generic wrapped error", err)
	}
}

func TestFeatureFlagValueService_Upsert_ExistingLookupError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, errors.New("db down"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err == nil {
		t.Fatal("Upsert() expected an error, got nil")
	}
}

func TestFeatureFlagValueService_Upsert_BooleanIgnoresValue(t *testing.T) {
	fixedNow(t)
	fixedID(t, "value-1")
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.Enabled && v.Value == nil
	})).Return(nil)

	ignoredValue := "should be dropped"
	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	got, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true, Value: &ignoredValue})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
	if got.Value != nil {
		t.Errorf("Value = %v, want nil for a Boolean flag", got.Value)
	}
}

func TestFeatureFlagValueService_Upsert_NonBooleanEnabledRequiresValue(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeString}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Upsert() error = %v, want ErrValidation", err)
	}
	values.AssertNotCalled(t, "Create")
}

func TestFeatureFlagValueService_Upsert_NonBooleanDisabledAllowsNilValue(t *testing.T) {
	fixedNow(t)
	fixedID(t, "value-1")
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeString}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return !v.Enabled && v.Value == nil
	})).Return(nil)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: false})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
}

func TestFeatureFlagValueService_Upsert_CreateSuccess(t *testing.T) {
	fixedNow(t)
	fixedID(t, "value-1")
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeString}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.ID == "value-1" && v.Enabled && v.Value != nil && *v.Value == "on"
	})).Return(nil)

	newValue := "on"
	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	got, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true, Value: &newValue})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
	if got.Value == nil || *got.Value != "on" {
		t.Errorf("Value = %v, want \"on\"", got.Value)
	}
}

func TestFeatureFlagValueService_Upsert_CreateRepositoryError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err == nil {
		t.Fatal("Upsert() expected an error, got nil")
	}
}

func TestFeatureFlagValueService_Upsert_CreateRace_RefetchAndUpdate(t *testing.T) {
	fixedNow(t)
	fixedID(t, "value-1")
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.Anything).Return(repository.ErrFeatureFlagEnvironmentValueExists)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(&model.FeatureFlagEnvironmentValue{ID: "winner-id", Enabled: false}, nil)
	values.On("Update", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.ID == "winner-id" && v.Enabled
	})).Return(nil)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	got, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
	if got.ID != "winner-id" {
		t.Errorf("ID = %q, want %q", got.ID, "winner-id")
	}
}

func TestFeatureFlagValueService_Upsert_CreateRace_RefetchError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.Anything).Return(repository.ErrFeatureFlagEnvironmentValueExists)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(nil, errors.New("db down"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err == nil {
		t.Fatal("Upsert() expected an error, got nil")
	}
}

func TestFeatureFlagValueService_Upsert_CreateRace_PreservesWinnerValueWhenDisabling(t *testing.T) {
	fixedNow(t)
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeString}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.Anything).Return(repository.ErrFeatureFlagEnvironmentValueExists)
	// The winner's row (created by the concurrent request we lost the race
	// to) already has a value on record — disabling here, with no Value
	// resupplied, must preserve it rather than wiping it to nil.
	winnerValue := "winner's value"
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(&model.FeatureFlagEnvironmentValue{ID: "winner-id", Enabled: true, Value: &winnerValue}, nil)
	values.On("Update", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.ID == "winner-id" && !v.Enabled && v.Value != nil && *v.Value == "winner's value"
	})).Return(nil)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	got, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: false})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
	if got.Value == nil || *got.Value != "winner's value" {
		t.Errorf("Value = %v, want \"winner's value\" (preserved from the winner's row)", got.Value)
	}
}

func TestFeatureFlagValueService_Upsert_CreateRace_UpdateError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(nil, repository.ErrNotFound)
	values.On("Create", mock.Anything, mock.Anything).Return(repository.ErrFeatureFlagEnvironmentValueExists)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Once().Return(&model.FeatureFlagEnvironmentValue{ID: "winner-id"}, nil)
	values.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: true})
	if err == nil {
		t.Fatal("Upsert() expected an error, got nil")
	}
}

func TestFeatureFlagValueService_Upsert_UpdateExisting(t *testing.T) {
	fixedNow(t)
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeString}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	existingValue := "previous"
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "existing-id", Enabled: true, Value: &existingValue}, nil)
	values.On("Update", mock.Anything, mock.MatchedBy(func(v *model.FeatureFlagEnvironmentValue) bool {
		return v.ID == "existing-id" && !v.Enabled && v.Value != nil && *v.Value == "previous"
	})).Return(nil)

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	// Enabled flips to false; Value omitted (nil) — should fall back to the
	// existing "previous" value rather than being cleared.
	got, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: false})
	if err != nil {
		t.Fatalf("Upsert() returned unexpected error: %v", err)
	}
	if got.Enabled {
		t.Error("Enabled = true, want false")
	}
	if got.Value == nil || *got.Value != "previous" {
		t.Errorf("Value = %v, want \"previous\" (kept from existing row)", got.Value)
	}
	values.AssertNotCalled(t, "Create")
}

func TestFeatureFlagValueService_Upsert_UpdateExistingError(t *testing.T) {
	flags, environments, values := newFeatureFlagValueServiceMocks()
	flags.On("GetByID", mock.Anything, "f1").Return(&model.FeatureFlag{ID: "f1", Type: model.FeatureFlagTypeBoolean}, nil)
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	values.On("GetByFlagAndEnvironment", mock.Anything, "f1", "env-1").Return(&model.FeatureFlagEnvironmentValue{ID: "existing-id"}, nil)
	values.On("Update", mock.Anything, mock.Anything).Return(errors.New("disk full"))

	svc := NewFeatureFlagValueService(flags, environments, values)
	actor := &model.User{ID: "actor-id"}
	_, err := svc.Upsert(context.Background(), ActorFromUser(actor), "f1", "env-1", UpsertValueInput{Enabled: false})
	if err == nil {
		t.Fatal("Upsert() expected an error, got nil")
	}
}
