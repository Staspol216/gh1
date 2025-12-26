-- +goose Up
-- +goose StatementBegin
ALTER TABLE audit_logs ADD COLUMN timestamp TIMESTAMP
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE audit_logs DROP COLUMN timestamp TIMESTAMP
-- +goose StatementEnd
