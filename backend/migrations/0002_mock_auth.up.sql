CREATE TABLE mock_auth_tokens (
    token   TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);
