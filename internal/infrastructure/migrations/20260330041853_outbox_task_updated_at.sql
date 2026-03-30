-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
DROP COLUMN processing_at,
DROP COLUMN failed_at;

ALTER TABLE orders_statuses_outbox
ADD COLUMN updated_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
ADD COLUMN processing_at TIMESTAMP,
ADD COLUMN failed_at TIMESTAMP;

ALTER TABLE orders_statuses_outbox
DROP COLUMN updated_at;
-- +goose StatementEnd
