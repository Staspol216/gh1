-- +goose Up
-- +goose StatementBegin
CREATE TYPE outbox_task_status AS ENUM ('created', 'processing', 'failed');

ALTER TABLE audit_logs RENAME TO orders_statuses_outbox;

ALTER TABLE orders_statuses_outbox RENAME COLUMN "timestamp" TO created_at;

ALTER TABLE orders_statuses_outbox RENAME COLUMN details TO order_status_details;

ALTER TABLE orders_statuses_outbox
ADD COLUMN status outbox_task_status DEFAULT 'created' NOT NULL,
ADD COLUMN processing_at TIMESTAMP,
ADD COLUMN failed_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders_statuses_outbox
DROP COLUMN status,
DROP COLUMN processing_at,
DROP COLUMN failed_at;

ALTER TABLE orders_statuses_outbox RENAME COLUMN order_status_details TO details;

ALTER TABLE orders_statuses_outbox RENAME COLUMN created_at TO "timestamp";

ALTER TABLE orders_statuses_outbox RENAME TO audit_logs;

DROP TYPE outbox_task_status;
-- +goose StatementEnd
