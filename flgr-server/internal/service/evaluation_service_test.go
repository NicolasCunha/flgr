package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
	"github.com/NicolasCunha/flgr/flgr-server/internal/repository"
)

func newEvaluationServiceMocks() (*mockFeatureFlagRepository, *mockEnvironmentRepository, *mockFeatureFlagEnvironmentValueRepository) {
	return new(mockFeatureFlagRepository), new(mockEnvironmentRepository), new(mockFeatureFlagEnvironmentValueRepository)
}

// TestEvaluationService_NoAuditLogDependency is structural proof (not just
// a doc comment) that Evaluate can never write to the audit log, per
// docs/business/requirements/0007-feature-flag-evaluation-api.md:
// NewEvaluationService doesn't even accept an AuditLogRepository to write
// through.
func TestEvaluationService_NoAuditLogDependency(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	svc := NewEvaluationService(flags, environments, values)
	if svc == nil {
		t.Fatal("NewEvaluationService() returned nil")
	}
}

func TestEvaluationService_Evaluate_UnknownEnvironment(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "missing").Return(nil, repository.ErrNotFound)

	svc := NewEvaluationService(flags, environments, values)
	_, err := svc.Evaluate(context.Background(), "missing", nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Evaluate() error = %v, want ErrValidation", err)
	}
	flags.AssertNotCalled(t, "ListAll")
}

func TestEvaluationService_Evaluate_EnvironmentLookupError(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(nil, errors.New("db down"))

	svc := NewEvaluationService(flags, environments, values)
	_, err := svc.Evaluate(context.Background(), "env-1", nil)
	if err == nil || errors.Is(err, ErrValidation) {
		t.Fatalf("Evaluate() error = %v, want a generic wrapped error", err)
	}
}

func TestEvaluationService_Evaluate_ListAllError(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	flags.On("ListAll", mock.Anything).Return(nil, errors.New("db down"))

	svc := NewEvaluationService(flags, environments, values)
	_, err := svc.Evaluate(context.Background(), "env-1", nil)
	if err == nil {
		t.Fatal("Evaluate() expected an error, got nil")
	}
}

func TestEvaluationService_Evaluate_ListByEnvironmentError(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	flags.On("ListAll", mock.Anything).Return([]model.FeatureFlag{{ID: "f1", Key: "checkout"}}, nil)
	values.On("ListByEnvironment", mock.Anything, "env-1").Return(nil, errors.New("db down"))

	svc := NewEvaluationService(flags, environments, values)
	_, err := svc.Evaluate(context.Background(), "env-1", nil)
	if err == nil {
		t.Fatal("Evaluate() expected an error, got nil")
	}
}

// TestEvaluationService_Evaluate_ConfiguredAndUnconfigured confirms a flag
// with a configured row returns that row's enabled/value, and a flag with
// no configured row for the environment returns Enabled=false, Value=nil,
// per 0005/0007 — not omitted from the result.
func TestEvaluationService_Evaluate_ConfiguredAndUnconfigured(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	flags.On("ListAll", mock.Anything).Return([]model.FeatureFlag{
		{ID: "f1", Key: "checkout"},
		{ID: "f2", Key: "welcome-message"},
	}, nil)
	welcomeValue := "hello"
	values.On("ListByEnvironment", mock.Anything, "env-1").Return([]model.FeatureFlagEnvironmentValue{
		{FeatureFlagID: "f2", Enabled: true, Value: &welcomeValue},
	}, nil)

	svc := NewEvaluationService(flags, environments, values)
	got, err := svc.Evaluate(context.Background(), "env-1", nil)
	if err != nil {
		t.Fatalf("Evaluate() returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Evaluate()) = %d, want 2", len(got))
	}

	byKey := map[string]EvaluatedFlag{}
	for _, f := range got {
		byKey[f.Key] = f
	}

	checkout, ok := byKey["checkout"]
	if !ok {
		t.Fatal(`Evaluate() missing "checkout"`)
	}
	if checkout.Enabled || checkout.Value != nil {
		t.Errorf("checkout = %+v, want Enabled=false Value=nil (no configured row)", checkout)
	}

	welcome, ok := byKey["welcome-message"]
	if !ok {
		t.Fatal(`Evaluate() missing "welcome-message"`)
	}
	if !welcome.Enabled || welcome.Value == nil || *welcome.Value != "hello" {
		t.Errorf("welcome-message = %+v, want Enabled=true Value=\"hello\"", welcome)
	}
}

// TestEvaluationService_Evaluate_KeysFilter confirms the keys filter narrows
// the result to only the requested (known) keys, silently dropping unknown
// ones rather than erroring.
func TestEvaluationService_Evaluate_KeysFilter(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	flags.On("ListAll", mock.Anything).Return([]model.FeatureFlag{
		{ID: "f1", Key: "checkout"},
		{ID: "f2", Key: "welcome-message"},
		{ID: "f3", Key: "beta-banner"},
	}, nil)
	values.On("ListByEnvironment", mock.Anything, "env-1").Return(nil, nil)

	svc := NewEvaluationService(flags, environments, values)
	got, err := svc.Evaluate(context.Background(), "env-1", []string{"checkout", "does-not-exist"})
	if err != nil {
		t.Fatalf("Evaluate() returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Key != "checkout" {
		t.Errorf("Evaluate() = %+v, want exactly [checkout] (unknown key silently dropped)", got)
	}
}

// TestEvaluationService_Evaluate_EmptyKeysReturnsEverything confirms a
// non-nil-but-empty keys slice is treated the same as any other non-nil
// filter — narrowing to nothing matched — while a nil keys slice (the
// "no filter" case, exercised by the other tests above) returns everything.
func TestEvaluationService_Evaluate_EmptyKeysReturnsEverything(t *testing.T) {
	flags, environments, values := newEvaluationServiceMocks()
	environments.On("GetByID", mock.Anything, "env-1").Return(&model.Environment{ID: "env-1"}, nil)
	flags.On("ListAll", mock.Anything).Return([]model.FeatureFlag{
		{ID: "f1", Key: "checkout"},
	}, nil)
	values.On("ListByEnvironment", mock.Anything, "env-1").Return(nil, nil)

	svc := NewEvaluationService(flags, environments, values)
	got, err := svc.Evaluate(context.Background(), "env-1", nil)
	if err != nil {
		t.Fatalf("Evaluate() returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Key != "checkout" {
		t.Errorf("Evaluate(nil keys) = %+v, want every flag returned", got)
	}
}
