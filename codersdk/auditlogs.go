package codersdk

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/coder/coder/coderd/database"
)

type AuditLogResponse struct {
	AuditLogs []AuditLog `json:"audit_logs"`
}

type AuditLog struct {
	ID             uuid.UUID             `json:"id"`
	Time           time.Time             `json:"time"`
	UserID         uuid.UUID             `json:"user_id"`
	OrganizationID uuid.UUID             `json:"organization_id"`
	Ip             net.IP                `json:"ip"`
	UserAgent      string                `json:"user_agent"`
	ResourceType   database.ResourceType `json:"resource_type"`
	ResourceID     uuid.UUID             `json:"resource_id"`
	ResourceTarget string                `json:"resource_target"`
	Action         database.AuditAction  `json:"action"`
	Diff           json.RawMessage       `json:"diff"`
	StatusCode     int32                 `json:"status_code"`
}
