package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/asthap08/ecommerce/internal/audit"
	"github.com/asthap08/ecommerce/internal/platform/validation"
)

const (
	maxFailedAttempts = 5
	failureWindow     = 15 * time.Minute
	lockoutDuration   = 15 * time.Minute
)

var (
	ErrInvalid       = errors.New("invalid input")
	ErrEmailTaken    = errors.New("email already exists")
	ErrBadLogin      = errors.New("invalid email or password")
	ErrLocked        = errors.New("too many failed attempts; try again later")
	ErrWrongPassword = errors.New("current password is incorrect")
)

// ValidationError carries a safe, user-facing message.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }
func (e *ValidationError) Unwrap() error { return ErrInvalid }

// dummyHash is compared against when the e-mail is unknown so that response
// time does not reveal which e-mails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

// Mailer is the e-mail sender (see internal/mailer).
type Mailer interface {
	Send(to, subject, body string) error
}

type Service struct {
	repo      *Repository
	audit     *audit.Service
	mail      Mailer
	publicURL string
	log       *slog.Logger
}

// WithMail enables password-reset e-mails; links point at publicURL/reset-password.
func (s *Service) WithMail(m Mailer, publicURL string, log *slog.Logger) *Service {
	s.mail, s.publicURL, s.log = m, publicURL, log
	return s
}

func NewService(repo *Repository, auditSvc *audit.Service) *Service {
	return &Service{repo: repo, audit: auditSvc}
}

// Register always creates a customer. Roles are never accepted from clients;
// administrators are provisioned by an operator (see README).
func (s *Service) Register(ctx context.Context, email, password, fullName string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validation.ValidateEmail(email); err != nil {
		return User{}, &ValidationError{err.Error()}
	}
	if err := validation.ValidatePassword(password); err != nil {
		return User{}, &ValidationError{err.Error()}
	}
	fullName = strings.TrimSpace(fullName)
	if utf8.RuneCountInString(fullName) > 100 {
		return User{}, &ValidationError{"name is too long"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	u, err := s.repo.Create(ctx, User{Email: email, PasswordHash: string(hash), Role: "customer", FullName: fullName})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, u.ID, "register", "user", u.ID)
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(password) > 72 || validation.ValidateEmail(email) != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte("x"))
		return User{}, ErrBadLogin
	}
	u, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, ErrBadLogin
	}
	if err != nil {
		return User{}, err
	}
	if s.repo.IsLocked(u) {
		return User{}, ErrLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		if locked, _ := s.repo.RegisterFailure(ctx, u.ID, maxFailedAttempts, failureWindow, lockoutDuration); locked {
			return User{}, ErrLocked
		}
		return User{}, ErrBadLogin
	}
	_ = s.repo.ResetFailedLogins(ctx, u.ID)
	if s.audit != nil {
		s.audit.Log(ctx, u.ID, "login", "user", u.ID)
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, id string) (User, error) { return s.repo.GetByID(ctx, id) }

func (s *Service) UpdateProfile(ctx context.Context, id, name, phone string) (User, error) {
	name, phone = strings.TrimSpace(name), strings.TrimSpace(phone)
	if utf8.RuneCountInString(name) > 100 {
		return User{}, &ValidationError{"name is too long"}
	}
	if phone != "" && (len(phone) < 7 || len(phone) > 20) {
		return User{}, &ValidationError{"enter a valid phone number"}
	}
	if err := s.repo.UpdateProfile(ctx, id, name, phone); err != nil {
		return User{}, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ChangePassword(ctx context.Context, id, current, next, keepHash string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	if err := validation.ValidatePassword(next); err != nil {
		return &ValidationError{err.Error()}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, id, "password_change", "user", id)
	}
	return s.repo.RevokeOtherSessions(ctx, id, keepHash)
}

const resetTTL = time.Hour

func hashResetToken(t string) string { h := sha256.Sum256([]byte(t)); return hex.EncodeToString(h[:]) }

// RequestReset always succeeds from the caller's point of view so the endpoint
// cannot be used to discover which e-mails have accounts.
func (s *Service) RequestReset(ctx context.Context, email string) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil || s.mail == nil {
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := s.repo.CreateReset(ctx, u.ID, hashResetToken(token), resetTTL); err != nil {
		return
	}
	link := s.publicURL + "/reset-password?token=" + token
	body := "Hi,\n\nUse this link to choose a new password (valid for 1 hour):\n\n" + link + "\n\nIf you did not ask for this, you can ignore this e-mail.\n"
	go func() {
		if err := s.mail.Send(u.Email, "Reset your password", body); err != nil && s.log != nil {
			s.log.Error("password reset e-mail failed", "err", err)
		}
	}()
}

func (s *Service) ResetPassword(ctx context.Context, token, next string) error {
	if len(token) < 20 || len(token) > 128 {
		return ErrBadResetToken
	}
	if err := validation.ValidatePassword(next); err != nil {
		return &ValidationError{err.Error()}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.ConsumeReset(ctx, hashResetToken(token), string(hash))
}
