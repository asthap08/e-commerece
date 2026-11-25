package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegisterRejectsInvalidInput(t *testing.T) {
	service := NewService(nil, nil)
	for _, tc := range []struct{ name, email, password string }{
		{"bad email", "not-an-email", "Passw0rd!x"},
		{"short", "person@example.com", "Aa1!"},
		{"display-name email", "Display Name <person@example.com>", "Passw0rd!x"},
		{"no special", "person@example.com", "Password123"},
		{"too long for bcrypt", "person@example.com", "Aa1!" + strings.Repeat("x", 80)},
	} {
		_, err := service.Register(context.Background(), tc.email, tc.password, "")
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: expected a ValidationError, got %v", tc.name, err)
		}
	}
}
