package session

// HashToken returns the stored (SHA-256) form of a session token.
func HashToken(token string) string { return hashToken(token) }
