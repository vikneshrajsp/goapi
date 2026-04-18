CREATE TABLE users (
    username TEXT PRIMARY KEY,
    auth_token TEXT NOT NULL
);

CREATE TABLE coin_balances (
    username TEXT PRIMARY KEY REFERENCES users (username) ON DELETE CASCADE,
    balance BIGINT NOT NULL CHECK (balance >= 0)
);
