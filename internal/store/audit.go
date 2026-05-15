package store

import (
	"time"

	"github.com/enekos/errolan/internal/models"
)

// AddAudit appends a moderation/admin event to the audit log. Failures are
// swallowed: an audit write should never block the user-facing operation.
func (s *Store) AddAudit(actorID *int64, actorName, action, kind string, targetID int64, metadata string) {
	_, _ = s.DB.Exec(
		`INSERT INTO audit_log (actor_id, actor_name, action, target_kind, target_id, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		actorID, actorName, action, kind, targetID, metadata, time.Now().Unix(),
	)
}

func (s *Store) ListAudit(limit int) ([]*models.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.Query(
		`SELECT id, actor_id, actor_name, action, target_kind, target_id, metadata, created_at FROM audit_log ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.AuditEntry
	for rows.Next() {
		e := &models.AuditEntry{}
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.Action, &e.TargetKind, &e.TargetID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
