package returns

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("return not found")
	ErrNotEligible   = errors.New("only delivered orders that you placed can be returned")
	ErrAlreadyOpen   = errors.New("a return request for this order already exists")
	ErrBadItems      = errors.New("choose at least one item from this order")
	ErrBadTransition = errors.New("that status change is not allowed")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const cols = `r.id,r.order_id,COALESCE(r.user_id::text,''),r.status,COALESCE(r.reason,''),COALESCE(r.refund_amount_minor,0),COALESCE(r.refund_method,'original'),
	COALESCE(r.return_items,'[]'::jsonb),COALESCE(r.return_label_url,''),COALESCE(r.return_authorization,''),r.admin_note,r.processed_at,r.created_at,COALESCE(o.customer_email,'')`

func scan(row pgx.Row) (ReturnRequest, error) {
	var rr ReturnRequest
	var items []byte
	err := row.Scan(&rr.ID, &rr.OrderID, &rr.UserID, &rr.Status, &rr.Reason, &rr.RefundAmountMinor, &rr.RefundMethod, &items,
		&rr.ReturnLabelURL, &rr.ReturnAuthorization, &rr.AdminNote, &rr.ProcessedAt, &rr.CreatedAt, &rr.CustomerEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return rr, ErrNotFound
	}
	_ = json.Unmarshal(items, &rr.ReturnItems)
	if rr.ReturnItems == nil {
		rr.ReturnItems = []Item{}
	}
	return rr, err
}

// CreateForCustomer validates ownership/eligibility and prices the refund from
// the stored order lines (never from client-supplied amounts).
func (r *Repository) CreateForCustomer(ctx context.Context, userID, orderID string, itemIDs []string, reason string) (ReturnRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReturnRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM orders WHERE id::text=$1 AND user_id=$2 FOR UPDATE`, orderID, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && status != "delivered") {
		return ReturnRequest{}, ErrNotEligible
	}
	if err != nil {
		return ReturnRequest{}, err
	}
	var open bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM return_requests WHERE order_id::text=$1 AND status NOT IN ('rejected'))`, orderID).Scan(&open); err != nil {
		return ReturnRequest{}, err
	}
	if open {
		return ReturnRequest{}, ErrAlreadyOpen
	}
	rows, err := tx.Query(ctx, `SELECT id::text,name,quantity,price_minor FROM order_items WHERE order_id::text=$1 AND id::text = ANY($2)`, orderID, itemIDs)
	if err != nil {
		return ReturnRequest{}, err
	}
	var items []Item
	var refund int64
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Quantity, &it.PriceMinor); err != nil {
			rows.Close()
			return ReturnRequest{}, err
		}
		refund += it.PriceMinor * int64(it.Quantity)
		items = append(items, it)
	}
	rows.Close()
	if len(items) == 0 {
		return ReturnRequest{}, ErrBadItems
	}
	itemsJSON, _ := json.Marshal(items)
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO return_requests(order_id,user_id,status,reason,return_reason,refund_amount_minor,refund_method,return_items)
		VALUES($1,$2,'requested',$3,$3,$4,'original',$5::jsonb) RETURNING id`, orderID, userID, reason, refund, itemsJSON).Scan(&id); err != nil {
		return ReturnRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReturnRequest{}, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Get(ctx context.Context, id string) (ReturnRequest, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM return_requests r JOIN orders o ON o.id=r.order_id WHERE r.id::text=$1`, id))
}

func (r *Repository) list(ctx context.Context, where string, args ...any) ([]ReturnRequest, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM return_requests r JOIN orders o ON o.id=r.order_id `+where+` ORDER BY r.created_at DESC LIMIT 200`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReturnRequest, 0)
	for rows.Next() {
		rr, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]ReturnRequest, error) {
	return r.list(ctx, `WHERE r.user_id=$1`, userID)
}
func (r *Repository) ListByOrder(ctx context.Context, orderID string) ([]ReturnRequest, error) {
	return r.list(ctx, `WHERE r.order_id::text=$1`, orderID)
}
func (r *Repository) ListAll(ctx context.Context, status string) ([]ReturnRequest, error) {
	if status == "" {
		return r.list(ctx, ``)
	}
	return r.list(ctx, `WHERE r.status=$1`, status)
}

var transitions = map[string][]string{
	"requested": {"approved", "rejected"},
	"approved":  {"received", "rejected"},
	"received":  {"refunded"},
}

// SetStatus advances a return through its lifecycle, recording label/auth data and a note.
func (r *Repository) SetStatus(ctx context.Context, id, to, note, labelURL, authCode string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var cur string
	if err := tx.QueryRow(ctx, `SELECT status FROM return_requests WHERE id::text=$1 FOR UPDATE`, id).Scan(&cur); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	ok := false
	for _, s := range transitions[cur] {
		ok = ok || s == to
	}
	if !ok {
		return ErrBadTransition
	}
	var processed *time.Time
	if to == "refunded" || to == "rejected" {
		now := time.Now()
		processed = &now
	}
	if _, err := tx.Exec(ctx, `UPDATE return_requests SET status=$2, admin_note=CASE WHEN $3<>'' THEN $3 ELSE admin_note END,
		return_label_url=CASE WHEN $4<>'' THEN $4 ELSE return_label_url END, return_authorization=CASE WHEN $5<>'' THEN $5 ELSE return_authorization END,
		processed_at=COALESCE($6,processed_at), updated_at=now() WHERE id::text=$1`, id, to, note, labelURL, authCode, processed); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
