-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
DROP COLUMN status_response,
DROP COLUMN duration_ms;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
ADD COLUMN status_response BIGINT NOT NULL,
ADD COLUMN duration_ms BIGINT NOT NULL;
-- +goose StatementEnd
