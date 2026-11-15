-- Собственная схема имитации корпоративного сервиса отсутствий (mock-hr).
-- Намеренно отделена от основной БД: так видно, что это ВНЕШНИЙ сервис.

CREATE SCHEMA IF NOT EXISTS mockhr;

CREATE TABLE mockhr.absence_requests (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('day_off', 'vacation')),
    date_from       DATE NOT NULL,
    date_to         DATE NOT NULL CHECK (date_to >= date_from),
    comment         TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),
    idempotency_key TEXT NOT NULL UNIQUE,  -- повтор запроса вернёт ту же заявку
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);