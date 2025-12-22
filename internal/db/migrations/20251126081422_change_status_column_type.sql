-- +goose Up
-- +goose StatementBegin
ALTER TABLE order_records
ALTER COLUMN status TYPE order_status USING status::order_status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE order_records
ALTER COLUMN status TYPE status USING status::text;
-- +goose StatementEnd
