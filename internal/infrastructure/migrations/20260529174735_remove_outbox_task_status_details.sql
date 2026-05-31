-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
DROP COLUMN order_status_details;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
ADD COLUMN order_status_details JSON;
-- +goose StatementEnd
