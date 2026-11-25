CREATE TABLE IF NOT EXISTS checkout_idempotency (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, idempotency_key),
    CHECK (length(idempotency_key) BETWEEN 16 AND 128),
    CHECK (length(request_hash) = 64)
);
