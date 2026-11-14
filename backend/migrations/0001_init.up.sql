CREATE TABLE teams (
    id   UUID PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id         UUID PRIMARY KEY,
    login      TEXT NOT NULL UNIQUE,
    full_name  TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('intern', 'mentor', 'manager', 'hr_bp')),
    team_id    UUID REFERENCES teams(id),
    mentor_id  UUID REFERENCES users(id),
    manager_id UUID REFERENCES users(id)
);

CREATE TABLE scenarios (
    id    UUID PRIMARY KEY,
    code  TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

CREATE TABLE scenario_steps (
    id          UUID PRIMARY KEY,
    scenario_id TEXT NOT NULL REFERENCES scenarios(id),
    position    INT  NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    UNIQUE (scenario_id, position)
);

CREATE TABLE intern_steps (
    user_id    UUID NOT NULL REFERENCES users(id),
    step_id    UUID NOT NULL REFERENCES scenario_steps(id),
    status     TEXT NOT NULL DEFAULT 'not_started'
               CHECK (status IN ('not_started', 'in_progress', 'waiting', 'done')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, step_id)
);

CREATE TABLE conversations (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX messages_conv_idx ON messages (conversation_id, id);

CREATE TABLE action_log (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    tool            TEXT NOT NULL,
    args            JSONB NOT NULL DEFAULT '{}',
    result          JSONB,
    status          TEXT NOT NULL
                    CHECK (status IN ('pending_confirmation', 'running', 'done',
                                      'failed', 'cancelled', 'denied')),
    idempotency_key TEXT NOT NULL UNIQUE,
    external_id     UUID,
    error           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX action_log_user_idx ON action_log (user_id, created_at DESC);

CREATE TABLE kb_documents (
    id            UUID PRIMARY KEY,
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,
    allowed_roles TEXT[] NOT NULL DEFAULT '{intern,mentor,manager,hr_bp}',
    team_id       UUID REFERENCES teams(id)
);

CREATE TABLE kb_chunks (
    id          UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES kb_documents(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    tsv         TSVECTOR GENERATED ALWAYS AS (to_tsvector('russian', content)) STORED
);
CREATE INDEX kb_chunks_tsv_idx ON kb_chunks USING GIN (tsv);

CREATE TABLE events (
    id         UUID PRIMARY KEY,
    user_id    UUID,
    name       TEXT NOT NULL,
    payload    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX events_name_idx ON events (name, created_at);
