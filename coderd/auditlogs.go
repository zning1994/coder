package coderd

import (
	"net/http"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/codersdk"
)

func (api *api) getAuditLogs(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
	)

	paginationParams, ok := parsePagination(rw, r)
	if !ok {
		return
	}

	alogs, err := api.Database.GetAuditLogsBefore(ctx, database.GetAuditLogsBeforeParams{
		ID:       paginationParams.AfterID,
		RowLimit: int32(paginationParams.Limit),
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: err.Error(),
		})
		return
	}

	httpapi.Write(rw, http.StatusOK, codersdk.AuditLogResponse{
		AuditLogs: auditLogsFromDB(alogs),
	})
}

func auditLogsFromDB(dblogs []database.AuditLog) []codersdk.AuditLog {
	apilogs := make([]codersdk.AuditLog, 0, len(dblogs))

	for _, alog := range dblogs {
		apilogs = append(apilogs, codersdk.AuditLog{
			ID:             alog.ID,
			Time:           alog.Time,
			UserID:         alog.UserID,
			OrganizationID: alog.OrganizationID,
			Ip:             alog.Ip.IPNet.IP,
			UserAgent:      alog.UserAgent,
			ResourceType:   alog.ResourceType,
			ResourceID:     alog.ResourceID,
			ResourceTarget: alog.ResourceTarget,
			Action:         alog.Action,
			Diff:           alog.Diff,
			StatusCode:     alog.StatusCode,
		})
	}

	return apilogs
}
