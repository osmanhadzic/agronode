package workers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type offlineDetectionServiceStub struct {
	markInactiveCallCount int
	markInactiveInput     time.Duration
	markInactiveResult    int
	markInactiveError     error
}

func (stub *offlineDetectionServiceStub) MarkInactiveDevicesOffline(_ context.Context, inactivityThreshold time.Duration) (int, error) {
	stub.markInactiveCallCount++
	stub.markInactiveInput = inactivityThreshold
	return stub.markInactiveResult, stub.markInactiveError
}

func TestOfflineDetectionWorker_CheckAndMarkOffline(t *testing.T) {
	t.Run("calls MarkInactiveDevicesOffline with correct threshold", func(t *testing.T) {
		service := &offlineDetectionServiceStub{markInactiveResult: 5}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		worker := NewOfflineDetectionWorker(service, logger, 15*time.Minute, 1*time.Minute)

		worker.checkAndMarkOffline(context.Background())

		if service.markInactiveCallCount != 1 {
			t.Fatalf("expected 1 call to MarkInactiveDevicesOffline, got %d", service.markInactiveCallCount)
		}

		if service.markInactiveInput != 15*time.Minute {
			t.Fatalf("expected threshold 15m, got %v", service.markInactiveInput)
		}
	})

	t.Run("handles service errors gracefully", func(t *testing.T) {
		testErr := errors.New("test error")
		service := &offlineDetectionServiceStub{markInactiveError: testErr}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		worker := NewOfflineDetectionWorker(service, logger, 15*time.Minute, 1*time.Minute)

		// Should not panic
		worker.checkAndMarkOffline(context.Background())

		if service.markInactiveCallCount != 1 {
			t.Fatalf("expected service to be called despite error")
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		service := &offlineDetectionServiceStub{
			markInactiveError: context.DeadlineExceeded,
		}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		worker := NewOfflineDetectionWorker(service, logger, 15*time.Minute, 1*time.Minute)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should not panic even with cancelled context
		worker.checkAndMarkOffline(ctx)
	})
}
