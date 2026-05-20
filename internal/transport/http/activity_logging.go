package http

import (
	"context"
	"log"
	"net/http"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
)

func logActivityBestEffort(ctx context.Context, activityService *logactivity.Service, entry activitylog.ActivityLog) {
	if activityService == nil {
		return
	}

	if err := activityService.Log(ctx, entry); err != nil {
		log.Printf("activity log failed for action %q: %v", entry.Action, err)
	}
}

func requestMetadata(r *http.Request) (ipAddress string, userAgent string) {
	return r.RemoteAddr, r.UserAgent()
}
