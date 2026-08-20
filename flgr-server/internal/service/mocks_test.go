package service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/NicolasCunha/flgr/flgr-server/internal/model"
)

// mockUserRepository fakes repository.UserRepository for service-layer
// tests, per backend.md's "service layer tests mock the repository
// interfaces" convention.
type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) Create(ctx context.Context, u *model.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	return userOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	return userOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserRepository) List(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	args := m.Called(ctx, limit, offset)
	users, _ := args.Get(0).([]model.User)
	return users, args.Int(1), args.Error(2)
}

func (m *mockUserRepository) Update(ctx context.Context, u *model.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func userOrNil(v any) *model.User {
	if v == nil {
		return nil
	}
	return v.(*model.User)
}

// mockSessionRepository fakes repository.SessionRepository.
type mockSessionRepository struct {
	mock.Mock
}

func (m *mockSessionRepository) Create(ctx context.Context, s *model.Session) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *mockSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	args := m.Called(ctx, tokenHash)
	s, _ := args.Get(0).(*model.Session)
	return s, args.Error(1)
}

func (m *mockSessionRepository) Touch(ctx context.Context, id string, lastSeenAt, expiresAt time.Time) error {
	args := m.Called(ctx, id, lastSeenAt, expiresAt)
	return args.Error(0)
}

func (m *mockSessionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockLoginAttemptRepository fakes repository.LoginAttemptRepository.
type mockLoginAttemptRepository struct {
	mock.Mock
}

func (m *mockLoginAttemptRepository) GetByEmail(ctx context.Context, email string) (*model.LoginAttempt, error) {
	args := m.Called(ctx, email)
	a, _ := args.Get(0).(*model.LoginAttempt)
	return a, args.Error(1)
}

func (m *mockLoginAttemptRepository) Upsert(ctx context.Context, a *model.LoginAttempt) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *mockLoginAttemptRepository) Reset(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

// mockPermissionChecker fakes the PermissionChecker seam UserService uses
// for its non-self authorization branches.
type mockPermissionChecker struct {
	mock.Mock
}

func (m *mockPermissionChecker) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

// mockProfileRepository fakes repository.ProfileRepository.
type mockProfileRepository struct {
	mock.Mock
}

func (m *mockProfileRepository) Create(ctx context.Context, p *model.Profile) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockProfileRepository) GetByID(ctx context.Context, id string) (*model.Profile, error) {
	args := m.Called(ctx, id)
	return profileOrNil(args.Get(0)), args.Error(1)
}

func (m *mockProfileRepository) GetByName(ctx context.Context, name string) (*model.Profile, error) {
	args := m.Called(ctx, name)
	return profileOrNil(args.Get(0)), args.Error(1)
}

func (m *mockProfileRepository) List(ctx context.Context, limit, offset int) ([]model.Profile, int, error) {
	args := m.Called(ctx, limit, offset)
	profiles, _ := args.Get(0).([]model.Profile)
	return profiles, args.Int(1), args.Error(2)
}

func (m *mockProfileRepository) ListAll(ctx context.Context) ([]model.Profile, error) {
	args := m.Called(ctx)
	profiles, _ := args.Get(0).([]model.Profile)
	return profiles, args.Error(1)
}

func (m *mockProfileRepository) Update(ctx context.Context, p *model.Profile) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockProfileRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func profileOrNil(v any) *model.Profile {
	if v == nil {
		return nil
	}
	return v.(*model.Profile)
}

// mockProfilePermissionRepository fakes repository.ProfilePermissionRepository.
type mockProfilePermissionRepository struct {
	mock.Mock
}

func (m *mockProfilePermissionRepository) ListPermissionIDs(ctx context.Context, profileID string) ([]string, error) {
	args := m.Called(ctx, profileID)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *mockProfilePermissionRepository) ReplacePermissions(ctx context.Context, profileID string, permissionIDs []string, actorUserID string) error {
	args := m.Called(ctx, profileID, permissionIDs, actorUserID)
	return args.Error(0)
}

// mockPermissionRepository fakes repository.PermissionRepository.
type mockPermissionRepository struct {
	mock.Mock
}

func (m *mockPermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	args := m.Called(ctx)
	perms, _ := args.Get(0).([]model.Permission)
	return perms, args.Error(1)
}

// mockUserProfileRepository fakes repository.UserProfileRepository.
type mockUserProfileRepository struct {
	mock.Mock
}

func (m *mockUserProfileRepository) ListProfileIDs(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *mockUserProfileRepository) ReplaceProfiles(ctx context.Context, userID string, profileIDs []string, actorUserID string) error {
	args := m.Called(ctx, userID, profileIDs, actorUserID)
	return args.Error(0)
}

// mockUserPermissionRepository fakes repository.UserPermissionRepository.
type mockUserPermissionRepository struct {
	mock.Mock
}

func (m *mockUserPermissionRepository) ListPermissionIDs(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *mockUserPermissionRepository) ReplacePermissions(ctx context.Context, userID string, permissionIDs []string, actorUserID string) error {
	args := m.Called(ctx, userID, permissionIDs, actorUserID)
	return args.Error(0)
}

// mockEnvironmentRepository fakes repository.EnvironmentRepository.
type mockEnvironmentRepository struct {
	mock.Mock
}

func (m *mockEnvironmentRepository) Create(ctx context.Context, e *model.Environment) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *mockEnvironmentRepository) GetByID(ctx context.Context, id string) (*model.Environment, error) {
	args := m.Called(ctx, id)
	return environmentOrNil(args.Get(0)), args.Error(1)
}

func (m *mockEnvironmentRepository) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	args := m.Called(ctx, name)
	return environmentOrNil(args.Get(0)), args.Error(1)
}

func (m *mockEnvironmentRepository) List(ctx context.Context, limit, offset int) ([]model.Environment, int, error) {
	args := m.Called(ctx, limit, offset)
	environments, _ := args.Get(0).([]model.Environment)
	return environments, args.Int(1), args.Error(2)
}

func (m *mockEnvironmentRepository) Update(ctx context.Context, e *model.Environment) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *mockEnvironmentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func environmentOrNil(v any) *model.Environment {
	if v == nil {
		return nil
	}
	return v.(*model.Environment)
}

// mockEnvironmentCategoryRepository fakes repository.EnvironmentCategoryRepository.
type mockEnvironmentCategoryRepository struct {
	mock.Mock
}

func (m *mockEnvironmentCategoryRepository) Create(ctx context.Context, c *model.EnvironmentCategory) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *mockEnvironmentCategoryRepository) GetByID(ctx context.Context, id string) (*model.EnvironmentCategory, error) {
	args := m.Called(ctx, id)
	return environmentCategoryOrNil(args.Get(0)), args.Error(1)
}

func (m *mockEnvironmentCategoryRepository) GetByName(ctx context.Context, name string) (*model.EnvironmentCategory, error) {
	args := m.Called(ctx, name)
	return environmentCategoryOrNil(args.Get(0)), args.Error(1)
}

func (m *mockEnvironmentCategoryRepository) ListAll(ctx context.Context) ([]model.EnvironmentCategory, error) {
	args := m.Called(ctx)
	categories, _ := args.Get(0).([]model.EnvironmentCategory)
	return categories, args.Error(1)
}

func environmentCategoryOrNil(v any) *model.EnvironmentCategory {
	if v == nil {
		return nil
	}
	return v.(*model.EnvironmentCategory)
}

// mockServiceKeyRepository fakes repository.ServiceKeyRepository.
type mockServiceKeyRepository struct {
	mock.Mock
}

func (m *mockServiceKeyRepository) Create(ctx context.Context, k *model.ServiceKey) error {
	args := m.Called(ctx, k)
	return args.Error(0)
}

func (m *mockServiceKeyRepository) GetByID(ctx context.Context, id string) (*model.ServiceKey, error) {
	args := m.Called(ctx, id)
	return serviceKeyOrNil(args.Get(0)), args.Error(1)
}

func (m *mockServiceKeyRepository) List(ctx context.Context, limit, offset int) ([]model.ServiceKey, int, error) {
	args := m.Called(ctx, limit, offset)
	keys, _ := args.Get(0).([]model.ServiceKey)
	return keys, args.Int(1), args.Error(2)
}

func (m *mockServiceKeyRepository) Update(ctx context.Context, k *model.ServiceKey) error {
	args := m.Called(ctx, k)
	return args.Error(0)
}

func (m *mockServiceKeyRepository) GetBySecretHash(ctx context.Context, hash string) (*model.ServiceKey, error) {
	args := m.Called(ctx, hash)
	return serviceKeyOrNil(args.Get(0)), args.Error(1)
}

func serviceKeyOrNil(v any) *model.ServiceKey {
	if v == nil {
		return nil
	}
	return v.(*model.ServiceKey)
}

// mockServiceKeyEnvironmentRepository fakes repository.ServiceKeyEnvironmentRepository.
type mockServiceKeyEnvironmentRepository struct {
	mock.Mock
}

func (m *mockServiceKeyEnvironmentRepository) ListEnvironmentIDs(ctx context.Context, serviceKeyID string) ([]string, error) {
	args := m.Called(ctx, serviceKeyID)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *mockServiceKeyEnvironmentRepository) ReplaceEnvironments(ctx context.Context, serviceKeyID string, environmentIDs []string, actorUserID string) error {
	args := m.Called(ctx, serviceKeyID, environmentIDs, actorUserID)
	return args.Error(0)
}

// mockFeatureFlagRepository fakes repository.FeatureFlagRepository.
type mockFeatureFlagRepository struct {
	mock.Mock
}

func (m *mockFeatureFlagRepository) Create(ctx context.Context, f *model.FeatureFlag) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *mockFeatureFlagRepository) GetByID(ctx context.Context, id string) (*model.FeatureFlag, error) {
	args := m.Called(ctx, id)
	return featureFlagOrNil(args.Get(0)), args.Error(1)
}

func (m *mockFeatureFlagRepository) GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error) {
	args := m.Called(ctx, key)
	return featureFlagOrNil(args.Get(0)), args.Error(1)
}

func (m *mockFeatureFlagRepository) List(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int, error) {
	args := m.Called(ctx, limit, offset)
	flags, _ := args.Get(0).([]model.FeatureFlag)
	return flags, args.Int(1), args.Error(2)
}

func (m *mockFeatureFlagRepository) Update(ctx context.Context, f *model.FeatureFlag) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *mockFeatureFlagRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockFeatureFlagRepository) ListAll(ctx context.Context) ([]model.FeatureFlag, error) {
	args := m.Called(ctx)
	flags, _ := args.Get(0).([]model.FeatureFlag)
	return flags, args.Error(1)
}

func featureFlagOrNil(v any) *model.FeatureFlag {
	if v == nil {
		return nil
	}
	return v.(*model.FeatureFlag)
}

// mockFeatureFlagEnvironmentValueRepository fakes repository.FeatureFlagEnvironmentValueRepository.
type mockFeatureFlagEnvironmentValueRepository struct {
	mock.Mock
}

func (m *mockFeatureFlagEnvironmentValueRepository) Create(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *mockFeatureFlagEnvironmentValueRepository) GetByFlagAndEnvironment(ctx context.Context, featureFlagID, environmentID string) (*model.FeatureFlagEnvironmentValue, error) {
	args := m.Called(ctx, featureFlagID, environmentID)
	return featureFlagEnvironmentValueOrNil(args.Get(0)), args.Error(1)
}

func (m *mockFeatureFlagEnvironmentValueRepository) ListByFlag(ctx context.Context, featureFlagID string) ([]model.FeatureFlagEnvironmentValue, error) {
	args := m.Called(ctx, featureFlagID)
	values, _ := args.Get(0).([]model.FeatureFlagEnvironmentValue)
	return values, args.Error(1)
}

func (m *mockFeatureFlagEnvironmentValueRepository) Update(ctx context.Context, v *model.FeatureFlagEnvironmentValue) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *mockFeatureFlagEnvironmentValueRepository) ListByEnvironment(ctx context.Context, environmentID string) ([]model.FeatureFlagEnvironmentValue, error) {
	args := m.Called(ctx, environmentID)
	values, _ := args.Get(0).([]model.FeatureFlagEnvironmentValue)
	return values, args.Error(1)
}

func featureFlagEnvironmentValueOrNil(v any) *model.FeatureFlagEnvironmentValue {
	if v == nil {
		return nil
	}
	return v.(*model.FeatureFlagEnvironmentValue)
}

// mockAuthzRepository fakes repository.AuthzRepository.
type mockAuthzRepository struct {
	mock.Mock
}

func (m *mockAuthzRepository) EffectivePermissions(ctx context.Context, userID string) ([]model.Permission, error) {
	args := m.Called(ctx, userID)
	perms, _ := args.Get(0).([]model.Permission)
	return perms, args.Error(1)
}

// mockNotificationChannelRepository fakes repository.NotificationChannelRepository.
type mockNotificationChannelRepository struct {
	mock.Mock
}

func (m *mockNotificationChannelRepository) Create(ctx context.Context, c *model.NotificationChannel) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *mockNotificationChannelRepository) GetByID(ctx context.Context, id string) (*model.NotificationChannel, error) {
	args := m.Called(ctx, id)
	return notificationChannelOrNil(args.Get(0)), args.Error(1)
}

func (m *mockNotificationChannelRepository) ListByFeatureFlagID(ctx context.Context, featureFlagID string) ([]model.NotificationChannel, error) {
	args := m.Called(ctx, featureFlagID)
	channels, _ := args.Get(0).([]model.NotificationChannel)
	return channels, args.Error(1)
}

func (m *mockNotificationChannelRepository) Update(ctx context.Context, c *model.NotificationChannel) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *mockNotificationChannelRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func notificationChannelOrNil(v any) *model.NotificationChannel {
	if v == nil {
		return nil
	}
	return v.(*model.NotificationChannel)
}

// mockAuditLogRepository fakes repository.AuditLogRepository.
type mockAuditLogRepository struct {
	mock.Mock
}

func (m *mockAuditLogRepository) Create(ctx context.Context, log *model.AuditLog, link *model.AuditLogFeatureFlag) error {
	args := m.Called(ctx, log, link)
	return args.Error(0)
}

// mockNotificationOutboxRepository fakes repository.NotificationOutboxRepository.
type mockNotificationOutboxRepository struct {
	mock.Mock
}

func (m *mockNotificationOutboxRepository) Create(ctx context.Context, entry *model.NotificationOutboxEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}
