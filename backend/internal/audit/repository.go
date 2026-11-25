package audit

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Log(ctx context.Context, actorID, action, entity, entityID string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO audit_logs(actor_id,action,entity,entity_id) VALUES($1,$2,$3,$4)`, actorID, action, entity, entityID)
	return err
}
