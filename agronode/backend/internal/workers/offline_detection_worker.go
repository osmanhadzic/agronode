package workers

import (
	"context"
	"log/slog"
	"time"
)

type OfflineDetectionWorker struct {
	deviceService     OfflineDetectionService
	logger            *slog.Logger
	inactivityTimeout time.Duration
	checkInterval     time.Duration
}

type OfflineDetectionService interface {
	MarkInactiveDevicesOffline(ctx context.Context, inactivityThreshold time.Duration) (int, error)
}

func NewOfflineDetectionWorker(
	deviceService OfflineDetectionService,
	logger *slog.Logger,
	inactivityTimeout time.Duration,
	checkInterval time.Duration,
) *OfflineDetectionWorker {
	return &OfflineDetectionWorker{
		deviceService:     deviceService,
		logger:            logger,
		inactivityTimeout: inactivityTimeout,
		checkInterval:     checkInterval,
	}
}

// Run starts the periodic offline detection worker
func (worker *OfflineDetectionWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(worker.checkInterval)
	defer ticker.Stop()

	if worker.logger != nil {
		worker.logger.Info(
			"offline detection worker starting",
			"inactivityTimeout", worker.inactivityTimeout,
			"checkInterval", worker.checkInterval,
		)
	}

	for {
		select {
		case <-ctx.Done():
			if worker.logger != nil {
				worker.logger.Info("offline detection worker stopped")
			}
			return
		case <-ticker.C:
			worker.checkAndMarkOffline(ctx)
		}
	}
}

func (worker *OfflineDetectionWorker) checkAndMarkOffline(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	marked, err := worker.deviceService.MarkInactiveDevicesOffline(checkCtx, worker.inactivityTimeout)
	if err != nil {
		if worker.logger != nil {
			worker.logger.Error("offline detection failed", "error", err)
		}
		return
	}

	if marked > 0 {
		if worker.logger != nil {
			worker.logger.Info("offline detection completed", "markedOffline", marked)
		}
	}
}
