-- +goose Up
-- +goose StatementBegin
CREATE TYPE order_status AS ENUM ('pending', 'delivered', 'refunded', 'returned', 'canceled');

CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    recipient_id BIGINT NOT NULL,
    expiration_date TIMESTAMP NOT NULL,
    delivered_date TIMESTAMP NULL,
    refunded_date TIMESTAMP NULL,
    returned_date TIMESTAMP NULL,
    status order_status NOT NULL,
    weight DOUBLE PRECISION NOT NULL,
    worth DOUBLE PRECISION NOT NULL
);

CREATE TABLE order_records (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    status TEXT NOT NULL,
    description TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE orders;
DROP TABLE order_records;
-- +goose StatementEnd
