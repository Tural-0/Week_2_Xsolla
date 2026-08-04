BEGIN;

CREATE TABLE carts (
    id         SERIAL      PRIMARY KEY,
    user_id    INTEGER     NOT NULL
);

CREATE INDEX idx_carts_user_id    ON orders (user_id);

COMMIT;
