package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	if len(email) > 254 {
		return fmt.Errorf("email too long")
	}
	return nil
}

// ValidatePassword requires 8-72 bytes (bcrypt silently/loudly breaks beyond
// 72) with upper, lower, digit and a symbol. Any printable symbol is accepted
// (the old regex rejected e.g. "-", "_" and spaces).
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 bytes")
	}
	var lower, upper, digit, special bool
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			lower = true
		case c >= 'A' && c <= 'Z':
			upper = true
		case c >= '0' && c <= '9':
			digit = true
		case c > ' ' && c != 0x7f:
			special = true
		}
	}
	if !lower || !upper || !digit || !special {
		return fmt.Errorf("password must contain uppercase, lowercase, number, and special character")
	}
	return nil
}

func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("invalid slug format")
	}
	return nil
}

func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Limit length
	if r := []rune(s); len(r) > 10000 {
		s = string(r[:10000])
	}
	return s
}

func ValidatePositiveInt(v int) error {
	if v <= 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

func ValidateNonNegativeInt(v int) error {
	if v < 0 {
		return fmt.Errorf("value cannot be negative")
	}
	return nil
}
