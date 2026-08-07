BEGIN;

DROP TABLE idempotency_keys;

CREATE TABLE idempotency_keys (
    user_id INTEGER NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    order_id INTEGER NOT NULL REFERENCES orders(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, idempotency_key)
);

COMMIT;