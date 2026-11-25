package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const userCols = `id,email,password_hash,role,full_name,phone,created_at,COALESCE(failed_login_attempts,0),locked_until,last_failed_login`

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.Phone, &u.CreatedAt, &u.FailedLoginAttempts, &u.LockedUntil, &u.LastFailedLogin)
	return u, err
}

func (r *Repository) Create(ctx context.Context, u User) (User, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO users(email,password_hash,role,full_name) VALUES($1,$2,$3,$4) RETURNING id,created_at`,
		u.Email, u.PasswordHash, u.Role, u.FullName).Scan(&u.ID, &u.CreatedAt)
	return u, err
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email)=lower($1)`, email))
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

func (r *Repository) UpdateProfile(ctx context.Context, id, name, phone string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET full_name=$2, phone=$3 WHERE id=$1`, id, name, phone)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, id, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

// RegisterFailure atomically records a failed login and locks the account once
// the threshold is reached. Failures older than the window do not count.
func (r *Repository) RegisterFailure(ctx context.Context, id string, max int, window, lock time.Duration) (locked bool, err error) {
	err = r.pool.QueryRow(ctx, `
		UPDATE users SET
		  failed_login_attempts = CASE WHEN last_failed_login IS NULL OR last_failed_login < now() - make_interval(secs => $3) THEN 1 ELSE COALESCE(failed_login_attempts,0)+1 END,
		  last_failed_login = now(),
		  locked_until = CASE WHEN (CASE WHEN last_failed_login IS NULL OR last_failed_login < now() - make_interval(secs => $3) THEN 1 ELSE COALESCE(failed_login_attempts,0)+1 END) >= $2
		                      THEN now() + make_interval(secs => $4) ELSE locked_until END
		WHERE id=$1
		RETURNING locked_until IS NOT NULL AND locked_until > now()`, id, max, window.Seconds(), lock.Seconds()).Scan(&locked)
	return
}

func (r *Repository) ResetFailedLogins(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET failed_login_attempts=0, locked_until=NULL, last_failed_login=NULL WHERE id=$1`, userID)
	return err
}

func (r *Repository) RevokeOtherSessions(ctx context.Context, userID, keepHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE user_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL AND token_hash<>$2`, userID, keepHash)
	return err
}

func (r *Repository) IsLocked(u User) bool {
	return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil)
}

func (r *Repository) CreateReset(ctx context.Context, userID, tokenHash string, ttl time.Duration) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_resets SET used_at=now() WHERE user_id=$1 AND used_at IS NULL`, userID)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO password_resets(user_id,token_hash,expires_at) VALUES($1,$2,now()+make_interval(secs=>$3))`, userID, tokenHash, ttl.Seconds())
	return err
}

var ErrBadResetToken = errors.New("this reset link is invalid or has expired")

// ConsumeReset atomically validates a one-time token, sets the new password,
// signs the user out everywhere and clears any lockout.
func (r *Repository) ConsumeReset(ctx context.Context, tokenHash, passwordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID string
	err = tx.QueryRow(ctx, `UPDATE password_resets SET used_at=now() WHERE token_hash=$1 AND used_at IS NULL AND expires_at>now() RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBadResetToken
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, failed_login_attempts=0, locked_until=NULL, last_failed_login=NULL WHERE id=$1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE user_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
