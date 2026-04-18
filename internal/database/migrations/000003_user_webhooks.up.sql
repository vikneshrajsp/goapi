CREATE TABLE user_webhooks (
    username TEXT PRIMARY KEY REFERENCES users (username) ON DELETE CASCADE,
    webhook_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
