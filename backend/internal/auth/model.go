package auth

import "time"

type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	Role                string     `json:"role"`
	FullName            string     `json:"full_name"`
	Phone               string     `json:"phone"`
	CreatedAt           time.Time  `json:"created_at"`
	FailedLoginAttempts int        `json:"-"`
	LockedUntil         *time.Time `json:"-"`
	LastFailedLogin     *time.Time `json:"-"`
}
