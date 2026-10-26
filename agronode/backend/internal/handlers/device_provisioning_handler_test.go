package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type fakeProvisioningService struct {
	registerErr error
	calls       int
}

func (fake *fakeProvisioningService) IssueToken(context.Context, uint, *uint, time.Duration, string) (*services.IssuedProvisioningToken, error) {
	return &services.IssuedProvisioningToken{Token: "agp_x", ExpiresAt: time.Now()}, nil
}

func (fake *fakeProvisioningService) Register(_ context.Context, hardwareID string, _ string) (*services.DeviceRegistration, error) {
	fake.calls++
	if fake.registerErr != nil {
		return nil, fake.registerErr
	}
	org := uint(4)
	return &services.DeviceRegistration{
		Device:       &models.Device{DeviceID: "dev-uuid", HardwareID: &hardwareID, OrganizationID: &org},
		MQTTUsername: "dev-uuid",
		MQTTPassword: "s3cret-password",
	}, nil
}

func (fake *fakeProvisioningService) RotateCredentials(context.Context, string) (*services.DeviceRegistration, error) {
	return nil, repositories.ErrDeviceNotFound
}

func (fake *fakeProvisioningService) SetLifecycleStatus(context.Context, string, string) error {
	return nil
}

func newRegisterRouter(fake *fakeProvisioningService, options ProvisioningOptions) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterPublicProvisioningRoutes(router, slog.New(slog.NewTextHandler(io.Discard, nil)), fake, options)
	return router
}

func postRegister(router *gin.Engine, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

const validRegisterBody = `{"hardwareId":"AA:BB:CC:DD:EE:FF","provisioningToken":"agp_token"}`

func TestRegisterEndpointReturnsCredentialsOnce(t *testing.T) {
	router := newRegisterRouter(&fakeProvisioningService{}, ProvisioningOptions{})
	recorder := postRegister(router, validRegisterBody, nil)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"password":"s3cret-password"`) || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected credentials with no-store, got %s / %v", recorder.Body.String(), recorder.Header())
	}
}

func TestRegisterEndpointErrorMapping(t *testing.T) {
	cases := map[string]struct {
		err  error
		body string
		want int
	}{
		"invalid token":      {repositories.ErrProvisioningTokenInvalid, validRegisterBody, http.StatusUnauthorized},
		"duplicate hardware": {repositories.ErrDuplicateHardwareID, validRegisterBody, http.StatusConflict},
		"validation":         {errors.Join(services.ErrDeviceValidation, errors.New("bad")), validRegisterBody, http.StatusBadRequest},
		"internal":           {errors.New("db down"), validRegisterBody, http.StatusInternalServerError},
		"missing fields":     {nil, `{"hardwareId":"AA"}`, http.StatusBadRequest},
	}
	for name, testCase := range cases {
		recorder := postRegister(newRegisterRouter(&fakeProvisioningService{registerErr: testCase.err}, ProvisioningOptions{}), testCase.body, nil)
		if recorder.Code != testCase.want {
			t.Errorf("%s: expected %d, got %d (%s)", name, testCase.want, recorder.Code, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "password") {
			t.Errorf("%s: error responses must never carry credentials", name)
		}
	}
}

func TestRegisterEndpointRequiresHTTPS(t *testing.T) {
	fake := &fakeProvisioningService{}
	router := newRegisterRouter(fake, ProvisioningOptions{RequireHTTPS: true})

	if recorder := postRegister(router, validRegisterBody, nil); recorder.Code != http.StatusForbidden || fake.calls != 0 {
		t.Fatalf("plain http must be refused before reaching the service, got %d", recorder.Code)
	}
	if recorder := postRegister(router, validRegisterBody, map[string]string{"X-Forwarded-Proto": "https"}); recorder.Code != http.StatusCreated {
		t.Fatalf("https via proxy must be accepted, got %d", recorder.Code)
	}
}

func TestRegisterEndpointRateLimit(t *testing.T) {
	router := newRegisterRouter(&fakeProvisioningService{registerErr: repositories.ErrProvisioningTokenInvalid}, ProvisioningOptions{RegisterRateLimit: 2})
	codes := []int{}
	for i := 0; i < 3; i++ {
		codes = append(codes, postRegister(router, validRegisterBody, nil).Code)
	}
	if codes[0] != 401 || codes[1] != 401 || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("unexpected codes %v", codes)
	}
}

type fakeAuthorizer struct {
	authErr    error
	publishErr error
	subErr     error
	lastTopics []string
}

func (fake *fakeAuthorizer) Authenticate(context.Context, string, string, string) error {
	return fake.authErr
}

func (fake *fakeAuthorizer) AuthorizePublish(context.Context, string, string) error {
	return fake.publishErr
}

func (fake *fakeAuthorizer) AuthorizeSubscribe(_ context.Context, _ string, topics []string) error {
	fake.lastTopics = topics
	return fake.subErr
}

func callWebhook(fake *fakeAuthorizer, hook string, headerHook string, body string) string {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterVerneMQWebhookRoutes(router, slog.New(slog.NewTextHandler(io.Discard, nil)), fake)

	request := httptest.NewRequest(http.MethodPost, "/vernemq/"+hook, bytes.NewBufferString(body))
	request.Header.Set("vernemq-hook", headerHook)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return strings.TrimSpace(recorder.Body.String())
}

func TestWebhookContract(t *testing.T) {
	const ok = `{"result":"ok"}`
	const denied = `{"result":{"error":"not_authorized"}}`
	register := `{"peer_addr":"1.2.3.4","peer_port":1,"username":"u","password":"p","mountpoint":"","client_id":"u","clean_session":true}`
	publish := `{"username":"u","client_id":"u","mountpoint":"","qos":1,"topic":"agronode/u/telemetry","payload":"e30=","retain":false}`
	subscribe := `{"username":"u","client_id":"u","mountpoint":"","topics":[{"topic":"agronode/u/activation","qos":1},{"topic":"x","qos":0}]}`

	allow := &fakeAuthorizer{}
	if got := callWebhook(allow, "auth_on_register", "auth_on_register", register); got != ok {
		t.Errorf("register allow: %s", got)
	}
	if got := callWebhook(allow, "auth_on_publish", "auth_on_publish", publish); got != ok {
		t.Errorf("publish allow: %s", got)
	}
	if got := callWebhook(allow, "auth_on_subscribe", "auth_on_subscribe", subscribe); got != ok || len(allow.lastTopics) != 2 {
		t.Errorf("subscribe allow: %s %v", got, allow.lastTopics)
	}

	deny := &fakeAuthorizer{authErr: services.ErrMQTTNotAuthorized, publishErr: services.ErrMQTTNotAuthorized, subErr: services.ErrMQTTNotAuthorized}
	if got := callWebhook(deny, "auth_on_register", "auth_on_register", register); got != denied {
		t.Errorf("register deny: %s", got)
	}
	if got := callWebhook(deny, "auth_on_publish", "auth_on_publish", publish); got != denied {
		t.Errorf("publish deny: %s", got)
	}
	if got := callWebhook(deny, "auth_on_subscribe", "auth_on_subscribe", subscribe); got != denied {
		t.Errorf("subscribe deny: %s", got)
	}

	broken := &fakeAuthorizer{authErr: errors.New("db down")}
	if got := callWebhook(broken, "auth_on_register", "auth_on_register", register); got != `{"result":{"error":"internal_error"}}` {
		t.Errorf("backend errors must fail closed: %s", got)
	}
	if got := callWebhook(allow, "auth_on_register", "auth_on_register", `{"username":"u","mountpoint":"other"}`); got != denied {
		t.Errorf("non-default mountpoint must be denied: %s", got)
	}
	if got := callWebhook(allow, "auth_on_register", "auth_on_publish", register); got == ok {
		t.Errorf("mismatched hook header must not be allowed")
	}
	if got := callWebhook(allow, "auth_on_register", "auth_on_register", `not json`); got != denied {
		t.Errorf("malformed payload must be denied: %s", got)
	}
}
