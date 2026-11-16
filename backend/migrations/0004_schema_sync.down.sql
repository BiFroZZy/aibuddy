DROP TABLE mock_requests;
ALTER TABLE action_log DROP CONSTRAINT action_log_user_idempotency_key;
ALTER TABLE action_log ADD CONSTRAINT action_log_idempotency_key_key UNIQUE (idempotency_key);
ALTER TABLE scenario_steps DROP COLUMN confirm_by;