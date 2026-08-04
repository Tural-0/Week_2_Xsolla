BEGIN;

CREATE TABLE cart_items (
    id       SERIAL  PRIMARY KEY,
    cart_id  VARCHAR(50) NOT NULL REFERENCES carts (id),
    item_id  INTEGER NOT NULL REFERENCES items  (id),
    price    INTEGER NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_cart_items_cart_id ON cart_items (cart_id);
CREATE INDEX idx_cart_items_item_id  ON cart_items (item_id);

COMMIT;
