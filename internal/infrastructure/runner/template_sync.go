package runner

import (
	"context"
	"log/slog"
	"time"
)

type Syncer interface {
	SyncGeneratedTasks(ctx context.Context) error
}

func StartTemplateSync(ctx context.Context, logger *slog.Logger, syncer Syncer, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}

	go func() {
		if err := syncer.SyncGeneratedTasks(ctx); err != nil {
			logger.Error("initial template sync failed", "error", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := syncer.SyncGeneratedTasks(ctx); err != nil {
					logger.Error("periodic template sync failed", "error", err)
				}
			}
		}
	}()
}
