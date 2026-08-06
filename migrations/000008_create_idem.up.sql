BEGIN;

CREATE TABLE idempotency_keys (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    order_id        INTEGER NOT NULL REFERENCES orders(id),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMIT;