ALTER TABLE scenario_steps
    ADD COLUMN confirm_by TEXT NOT NULL DEFAULT 'self'
        CHECK (confirm_by IN ('self', 'mentor'));

ALTER TABLE action_log DROP CONSTRAINT action_log_idempotency_key_key;
ALTER TABLE action_log ADD CONSTRAINT action_log_user_idempotency_key UNIQUE (user_id, idempotency_key);

CREATE TABLE mock_requests (
    id              UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    kind            TEXT NOT NULL CHECK (kind IN ('absence', 'access')),
    user_id         UUID NOT NULL REFERENCES users(id),
    payload         JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'pending_approval'
                    CHECK (status IN ('pending_approval', 'approved', 'rejected')),
    approver_role   TEXT NOT NULL,
    decided_by      UUID REFERENCES users(id),
    decision_comment TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at      TIMESTAMPTZ
);
CREATE INDEX mock_requests_user_idx ON mock_requests (user_id, created_at DESC);