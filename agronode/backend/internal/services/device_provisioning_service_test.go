package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/tenancy"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newProvisioningTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared&_busy_timeout=5000", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	statements := []string{
		`CREATE TABLE devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL UNIQUE,
			organization_id INTEGER,
			asset_unit_id INTEGER,
			device_type TEXT NOT NULL DEFAULT 'publisher',
			hardware_id VARCHAR(64),
			status TEXT,
			firmware_version TEXT,
			metadata TEXT, tags TEXT, desired_state TEXT, reported_state TEXT, discovered_sensors TEXT,
			api_key_hash TEXT, provisioning_token_hash TEXT,
			mqtt_username TEXT, mqtt_password_hash TEXT,
			lifecycle_status TEXT NOT NULL DEFAULT 'pending',
			registered_at DATETIME, credentials_rotated_at DATETIME, last_seen DATETIME,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_devices_hardware_id ON devices (hardware_id) WHERE hardware_id IS NOT NULL`,
		`CREATE UNIQUE INDEX idx_devices_mqtt_username ON devices (mqtt_username) WHERE mqtt_username IS NOT NULL`,
		`CREATE TABLE provisioning_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT NOT NULL UNIQUE,
			organization_id INTEGER NOT NULL,
			asset_unit_id INTEGER,
			created_by TEXT NOT NULL DEFAULT '',
			expires_at DATETIME NOT NULL,
			consumed_at DATETIME,
			consumed_device_id TEXT,
			revoked_at DATETIME,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE asset_units (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			asset_section_id INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL DEFAULT 'z',
			created_at DATETIME, updated_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}

	return db
}

type provisioningFixture struct {
	db      *gorm.DB
	repo    *repositories.GormProvisioningRepository
	service *DeviceProvisioningService
	clock   *atomic.Int64
}

func newProvisioningFixture(t *testing.T) *provisioningFixture {
	t.Helper()

	db := newProvisioningTestDB(t)
	repo := repositories.NewGormProvisioningRepository(db)
	service := NewDeviceProvisioningService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	clock := &atomic.Int64{}
	clock.Store(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano())
	service.now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }

	return &provisioningFixture{db: db, repo: repo, service: service, clock: clock}
}

func (fixture *provisioningFixture) advance(duration time.Duration) {
	fixture.clock.Add(int64(duration))
}

func (fixture *provisioningFixture) issue(t *testing.T, organizationID uint) string {
	t.Helper()
	issued, err := fixture.service.IssueToken(context.Background(), organizationID, nil, time.Hour, "admin@test")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return issued.Token
}

func (fixture *provisioningFixture) deviceCount(t *testing.T) int64 {
	t.Helper()
	var count int64
	if err := fixture.db.Model(&models.Device{}).Count(&count).Error; err != nil {
		t.Fatalf("count devices: %v", err)
	}
	return count
}

func TestRegisterSuccessStoresOnlyHashesAndBindsTokenScope(t *testing.T) {
	fixture := newProvisioningFixture(t)
	fixture.db.Create(&models.Zone{ID: 7, OrganizationID: 3, FieldID: 1, Name: "z", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assetUnit := uint(7)
	issued, err := fixture.service.IssueToken(context.Background(), 3, &assetUnit, time.Hour, "admin@test")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	result, err := fixture.service.Register(context.Background(), "aa:bb:cc:dd:ee:ff", issued.Token)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if result.MQTTPassword == "" || result.MQTTUsername != result.Device.DeviceID {
		t.Fatalf("unexpected credentials: %+v", result)
	}
	device := result.Device
	if device.OrganizationID == nil || *device.OrganizationID != 3 || device.ZoneID == nil || *device.ZoneID != 7 {
		t.Fatalf("device not bound to token scope: %+v", device)
	}
	if device.LifecycleStatus != models.LifecycleActive || *device.HardwareID != "AABBCCDDEEFF" {
		t.Fatalf("unexpected device state: %+v", device)
	}
	if device.MQTTPasswordHash == "" || device.MQTTPasswordHash == result.MQTTPassword {
		t.Fatal("password must be stored hashed")
	}
	if err := auth.ComparePassword(device.MQTTPasswordHash, result.MQTTPassword); err != nil {
		t.Fatalf("stored hash must verify password: %v", err)
	}

	var stored models.ProvisioningToken
	fixture.db.First(&stored)
	if stored.TokenHash == issued.Token || stored.TokenHash != HashProvisioningToken(issued.Token) || stored.ConsumedAt == nil {
		t.Fatalf("token must be stored hashed and consumed: %+v", stored)
	}
}

func TestRegisterRejectsInvalidExpiredAndReusedTokens(t *testing.T) {
	fixture := newProvisioningFixture(t)
	ctx := context.Background()

	if _, err := fixture.service.Register(ctx, "AABBCCDDEEFF", "agp_unknown"); !errors.Is(err, repositories.ErrProvisioningTokenInvalid) {
		t.Fatalf("unknown token: got %v", err)
	}

	expired := fixture.issue(t, 1)
	fixture.advance(2 * time.Hour)
	if _, err := fixture.service.Register(ctx, "AABBCCDDEEFF", expired); !errors.Is(err, repositories.ErrProvisioningTokenInvalid) {
		t.Fatalf("expired token: got %v", err)
	}

	revoked := fixture.issue(t, 1)
	fixture.db.Model(&models.ProvisioningToken{}).Where("token_hash = ?", HashProvisioningToken(revoked)).Update("revoked_at", time.Now())
	if _, err := fixture.service.Register(ctx, "AABBCCDDEEFF", revoked); !errors.Is(err, repositories.ErrProvisioningTokenInvalid) {
		t.Fatalf("revoked token: got %v", err)
	}

	valid := fixture.issue(t, 1)
	if _, err := fixture.service.Register(ctx, "AABBCCDDEEFF", valid); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if _, err := fixture.service.Register(ctx, "112233445566", valid); !errors.Is(err, repositories.ErrProvisioningTokenInvalid) {
		t.Fatalf("reused token: got %v", err)
	}
	if fixture.deviceCount(t) != 1 {
		t.Fatalf("only one device expected, got %d", fixture.deviceCount(t))
	}
}

func TestRegisterValidatesHardwareID(t *testing.T) {
	fixture := newProvisioningFixture(t)
	token := fixture.issue(t, 1)

	if _, err := fixture.service.Register(context.Background(), "not-a-mac", token); !errors.Is(err, ErrDeviceValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, err := fixture.service.Register(context.Background(), "AABBCCDDEEFF", token); err != nil {
		t.Fatalf("token must survive a rejected request: %v", err)
	}
}

func TestRegisterDuplicateHardwareRollsBackTokenConsumption(t *testing.T) {
	fixture := newProvisioningFixture(t)
	ctx := context.Background()

	if _, err := fixture.service.Register(ctx, "AABBCCDDEEFF", fixture.issue(t, 1)); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	second := fixture.issue(t, 1)
	_, err := fixture.service.Register(ctx, "aa-bb-cc-dd-ee-ff", second)
	if !errors.Is(err, repositories.ErrDuplicateHardwareID) {
		t.Fatalf("expected duplicate hardware error, got %v", err)
	}

	var token models.ProvisioningToken
	fixture.db.Where("token_hash = ?", HashProvisioningToken(second)).First(&token)
	if token.ConsumedAt != nil {
		t.Fatal("failed registration must roll back token consumption")
	}
	if fixture.deviceCount(t) != 1 {
		t.Fatalf("expected a single device, got %d", fixture.deviceCount(t))
	}
	if _, err := fixture.service.Register(ctx, "112233445566", second); err != nil {
		t.Fatalf("rolled-back token must remain usable: %v", err)
	}
}

func TestRegisterConcurrentSameTokenCreatesOneDevice(t *testing.T) {
	fixture := newProvisioningFixture(t)
	token := fixture.issue(t, 1)

	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			hardwareID := fmt.Sprintf("AABBCCDDEE%02X", index)
			if _, err := fixture.service.Register(context.Background(), hardwareID, token); err == nil {
				successes.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if successes.Load() != 1 || fixture.deviceCount(t) != 1 {
		t.Fatalf("expected exactly one registration, successes=%d devices=%d", successes.Load(), fixture.deviceCount(t))
	}
}

func TestRegisterConcurrentSameHardwareCreatesOneDevice(t *testing.T) {
	fixture := newProvisioningFixture(t)
	tokens := make([]string, 8)
	for i := range tokens {
		tokens[i] = fixture.issue(t, 1)
	}

	var wg sync.WaitGroup
	var successes atomic.Int32
	for _, token := range tokens {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			if _, err := fixture.service.Register(context.Background(), "AABBCCDDEEFF", token); err == nil {
				successes.Add(1)
			}
		}(token)
	}
	wg.Wait()

	var consumed int64
	fixture.db.Model(&models.ProvisioningToken{}).Where("consumed_at IS NOT NULL").Count(&consumed)
	if successes.Load() != 1 || fixture.deviceCount(t) != 1 || consumed != 1 {
		t.Fatalf("successes=%d devices=%d consumedTokens=%d", successes.Load(), fixture.deviceCount(t), consumed)
	}
}

func TestIssueTokenScopeAndTTLRules(t *testing.T) {
	fixture := newProvisioningFixture(t)

	scoped := tenancy.WithOrganizationID(context.Background(), 1)
	if _, err := fixture.service.IssueToken(scoped, 2, nil, time.Hour, "org-user"); !errors.Is(err, ErrProvisioningForbidden) {
		t.Fatalf("cross-organization issuance must be forbidden, got %v", err)
	}
	if _, err := fixture.service.IssueToken(context.Background(), 1, nil, MaxProvisioningTTL+time.Hour, "admin"); !errors.Is(err, ErrDeviceValidation) {
		t.Fatalf("excessive ttl must be rejected, got %v", err)
	}
	unit := uint(99)
	if _, err := fixture.service.IssueToken(context.Background(), 1, &unit, time.Hour, "admin"); !errors.Is(err, ErrDeviceValidation) {
		t.Fatalf("foreign asset unit must be rejected, got %v", err)
	}
}

func registerDevice(t *testing.T, fixture *provisioningFixture, organizationID uint, hardwareID string) *DeviceRegistration {
	t.Helper()
	result, err := fixture.service.Register(context.Background(), hardwareID, fixture.issue(t, organizationID))
	if err != nil {
		t.Fatalf("register %s: %v", hardwareID, err)
	}
	return result
}

func TestMQTTAuthenticate(t *testing.T) {
	fixture := newProvisioningFixture(t)
	auth := NewMQTTAuthService(fixture.repo, "backend-svc", "svc-secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	device := registerDevice(t, fixture, 1, "AABBCCDDEEFF")
	id, password := device.MQTTUsername, device.MQTTPassword

	if err := auth.Authenticate(ctx, id, id, password); err != nil {
		t.Fatalf("active device must authenticate: %v", err)
	}
	for name, attempt := range map[string]error{
		"wrong password":   auth.Authenticate(ctx, id, id, password+"x"),
		"empty password":   auth.Authenticate(ctx, id, id, ""),
		"foreign clientId": auth.Authenticate(ctx, id, "other", password),
		"unknown user":     auth.Authenticate(ctx, "nobody", "nobody", password),
		"service wrong pw": auth.Authenticate(ctx, "backend-svc", "x", "bad"),
	} {
		if !errors.Is(attempt, ErrMQTTNotAuthorized) {
			t.Errorf("%s: expected rejection, got %v", name, attempt)
		}
	}
	if err := auth.Authenticate(ctx, "backend-svc", "agronode-backend-1", "svc-secret"); err != nil {
		t.Fatalf("service account must authenticate: %v", err)
	}

	for _, status := range []string{models.LifecycleDisabled, models.LifecycleRevoked} {
		if err := fixture.service.SetLifecycleStatus(ctx, id, status); err != nil {
			t.Fatalf("set %s: %v", status, err)
		}
		if err := auth.Authenticate(ctx, id, id, password); !errors.Is(err, ErrMQTTNotAuthorized) {
			t.Fatalf("%s device must be rejected, got %v", status, err)
		}
	}
	if err := fixture.service.SetLifecycleStatus(ctx, id, models.LifecycleActive); !errors.Is(err, repositories.ErrDeviceNotFound) {
		t.Fatalf("revoked device must not be reactivated, got %v", err)
	}
}

func TestMQTTAuthFailsClosedOnStoreError(t *testing.T) {
	fixture := newProvisioningFixture(t)
	auth := NewMQTTAuthService(fixture.repo, "", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	fixture.db.Exec("DROP TABLE devices")

	err := auth.Authenticate(context.Background(), "some-id", "some-id", "pw")
	if err == nil || errors.Is(err, ErrMQTTNotAuthorized) {
		t.Fatalf("store errors must surface as errors (webhook then denies), got %v", err)
	}
}

func TestLegacyDeviceWithoutCredentialsCannotAuthenticate(t *testing.T) {
	fixture := newProvisioningFixture(t)
	auth := NewMQTTAuthService(fixture.repo, "", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	username := "legacy-1"
	fixture.db.Create(&models.Device{DeviceID: "legacy-1", MQTTUsername: &username, LifecycleStatus: models.LifecycleActive, MQTTPasswordHash: ""})

	if err := auth.Authenticate(context.Background(), username, username, "anything"); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Fatalf("device without a password hash must be rejected, got %v", err)
	}
}

func TestRotateCredentialsInvalidatesOldPasswordAndHonorsScope(t *testing.T) {
	fixture := newProvisioningFixture(t)
	auth := NewMQTTAuthService(fixture.repo, "", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	device := registerDevice(t, fixture, 1, "AABBCCDDEEFF")
	id := device.MQTTUsername

	otherOrg := tenancy.WithOrganizationID(ctx, 2)
	if _, err := fixture.service.RotateCredentials(otherOrg, id); !errors.Is(err, repositories.ErrDeviceNotFound) {
		t.Fatalf("other organization must not rotate, got %v", err)
	}

	rotated, err := fixture.service.RotateCredentials(tenancy.WithOrganizationID(ctx, 1), id)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := auth.Authenticate(ctx, id, id, device.MQTTPassword); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Fatalf("old password must stop working, got %v", err)
	}
	if err := auth.Authenticate(ctx, id, id, rotated.MQTTPassword); err != nil {
		t.Fatalf("new password must work: %v", err)
	}

	fixture.service.SetLifecycleStatus(ctx, id, models.LifecycleRevoked)
	if _, err := fixture.service.RotateCredentials(ctx, id); !errors.Is(err, repositories.ErrDeviceNotFound) {
		t.Fatalf("revoked device must not receive new credentials, got %v", err)
	}
}

func TestMQTTTopicAuthorization(t *testing.T) {
	fixture := newProvisioningFixture(t)
	auth := NewMQTTAuthService(fixture.repo, "backend-svc", "svc-secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	deviceA := registerDevice(t, fixture, 1, "AABBCCDDEE01").MQTTUsername
	deviceB := registerDevice(t, fixture, 2, "AABBCCDDEE02").MQTTUsername

	allowedPublish := []string{"agronode/" + deviceA + "/telemetry", "agronode/" + deviceA + "/status"}
	for _, topic := range allowedPublish {
		if err := auth.AuthorizePublish(ctx, deviceA, topic); err != nil {
			t.Errorf("publish %s should be allowed: %v", topic, err)
		}
	}

	deniedPublish := []string{
		"agronode/" + deviceB + "/telemetry",
		"agronode/" + deviceB + "/status",
		"agronode/" + deviceA + "/activation",
		"agronode/" + deviceA + "/register",
		"agronode/" + deviceA + "/telemetry/extra",
		"agronode/+/telemetry",
		"agronode/#",
		"$SYS/broker/load",
		"other/" + deviceA + "/telemetry",
		"",
	}
	for _, topic := range deniedPublish {
		if err := auth.AuthorizePublish(ctx, deviceA, topic); !errors.Is(err, ErrMQTTNotAuthorized) {
			t.Errorf("publish %q must be denied, got %v", topic, err)
		}
	}

	if err := auth.AuthorizeSubscribe(ctx, deviceA, []string{"agronode/" + deviceA + "/activation"}); err != nil {
		t.Errorf("own activation subscription should be allowed: %v", err)
	}
	deniedSubscribe := [][]string{
		{"agronode/" + deviceB + "/activation"},
		{"agronode/" + deviceA + "/telemetry"},
		{"agronode/#"},
		{"agronode/+/activation"},
		{"#"},
		{"agronode/" + deviceA + "/activation", "agronode/" + deviceB + "/activation"},
		{},
	}
	for _, topics := range deniedSubscribe {
		if err := auth.AuthorizeSubscribe(ctx, deviceA, topics); !errors.Is(err, ErrMQTTNotAuthorized) {
			t.Errorf("subscribe %v must be denied, got %v", topics, err)
		}
	}

	if err := auth.AuthorizePublish(ctx, "unknown-user", "agronode/unknown-user/telemetry"); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Errorf("unknown principal must be denied, got %v", err)
	}

	fixture.service.SetLifecycleStatus(ctx, deviceA, models.LifecycleDisabled)
	if err := auth.AuthorizePublish(ctx, deviceA, allowedPublish[0]); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Errorf("disabled device must lose publish rights immediately, got %v", err)
	}
	if err := auth.AuthorizeSubscribe(ctx, deviceA, []string{"agronode/" + deviceA + "/activation"}); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Errorf("disabled device must lose subscribe rights immediately, got %v", err)
	}

	if err := auth.AuthorizeSubscribe(ctx, "backend-svc", []string{"agronode/#"}); err != nil {
		t.Errorf("backend must subscribe to agronode/#: %v", err)
	}
	if err := auth.AuthorizePublish(ctx, "backend-svc", "agronode/"+deviceB+"/activation"); err != nil {
		t.Errorf("backend must publish activations: %v", err)
	}
	if err := auth.AuthorizePublish(ctx, "backend-svc", "$SYS/x"); !errors.Is(err, ErrMQTTNotAuthorized) {
		t.Errorf("backend must stay inside agronode/: %v", err)
	}
}
