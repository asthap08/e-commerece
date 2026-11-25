package redis

import "testing"

func TestNewParsesRedisURL(t *testing.T) {
	client, err := New("redis://localhost:6379/0")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsInvalidURL(t *testing.T) {
	if _, err := New("not-a-redis-url"); err == nil {
		t.Fatal("expected invalid Redis URL to be rejected")
	}
}
