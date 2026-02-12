package psql_audit_log_repo

import (
	"context"
	"log"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/audit_log"
	db "github.com/Staspol216/gh1/internal/infrastructure/postgres"
)

type AuditLogRepo struct {
	Db      db.DB
	Context context.Context
}

func (r *AuditLogRepo) AddAuditLog(audit_log *pvz_domain.AuditLog) (int64, error) {
	query := `INSERT INTO audit_logs (
		request_id,
		timestamp,
		method,
		path,
		remote_address,
        user_agent,
        status_response,
        duration_ms,
        details      
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;`

	row := r.Db.ExecQueryRow(r.Context, query,
		audit_log.RequestID,
		audit_log.Timestamp,
		audit_log.Method,
		audit_log.Path,
		audit_log.RemoteAddress,
		audit_log.UserAgent,
		audit_log.StatusResponse,
		audit_log.DurationMs,
		audit_log.Details,
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		log.Println(err)
	}
	return id, err
}
