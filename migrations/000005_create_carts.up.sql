BEGIN;

CREATE TABLE carts (
    id         VARCHAR(50)      PRIMARY KEY,
    user_id    INTEGER     NOT NULL
);

CREATE INDEX idx_carts_user_id    ON carts (user_id);

COMMIT;
