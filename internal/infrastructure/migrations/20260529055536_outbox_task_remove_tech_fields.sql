-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
DROP COLUMN request_id,
DROP COLUMN method, 
DROP COLUMN path, 
DROP COLUMN remote_address, 
DROP COLUMN user_agent, 
DROP COLUMN updated_at;

ALTER TABLE orders_statuses_outbox
ADD COLUMN timestamp TIMESTAMP NOT NULL,
ADD COLUMN description TEXT NOT NULL,
ADD COLUMN order_status order_status NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
ADD COLUMN request_id VARCHAR NOT NULL,
ADD COLUMN method VARCHAR NOT NULL, 
ADD COLUMN path VARCHAR NOT NULL, 
ADD COLUMN remote_address VARCHAR NOT NULL, 
ADD COLUMN user_agent VARCHAR NOT NULL, 
ADD COLUMN updated_at TIMESTAMP;

ALTER TABLE orders_statuses_outbox
DROP COLUMN timestamp,
DROP COLUMN description,
DROP COLUMN order_status;
-- +goose StatementEnd
