package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CookieName = "omniretail_session"
const Lifetime = 7 * 24 * time.Hour

var ErrInvalidSession = errors.New("invalid or expired session")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type Identity struct {
	UserID string
	Role   string
}

func (s *Store) Create(ctx context.Context, userID string, now time.Time) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := now.Add(Lifetime)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO user_sessions(user_id, token_hash, expires_at)
		VALUES($1, $2, $3)
	`, userID, hashToken(token), expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *Store) Resolve(ctx context.Context, token string, now time.Time) (Identity, error) {
	if len(token) != 43 {
		return Identity{}, ErrInvalidSession
	}
	var identity Identity
	err := s.pool.QueryRow(ctx, `
		SELECT us.user_id::text, u.role
		FROM user_sessions us
		JOIN users u ON u.id=us.user_id
		WHERE us.token_hash=$1
		  AND us.expires_at > $2
		  AND us.revoked_at IS NULL
	`, hashToken(token), now).Scan(&identity.UserID, &identity.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, ErrInvalidSession
	}
	return identity, err
}

func (s *Store) Revoke(ctx context.Context, token string, now time.Time) error {
	if len(token) != 43 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at=$2
		WHERE token_hash=$1 AND revoked_at IS NULL
	`, hashToken(token), now)
	return err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
