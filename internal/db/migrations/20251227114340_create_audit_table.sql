-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    request_id VARCHAR NOT NULL,
    method VARCHAR NOT NULL,
    path VARCHAR NOT NULL,
    remote_address VARCHAR NOT NULL,
    user_agent VARCHAR NOT NULL,
    status_response BIGINT NOT NULL,
    duration_ms BIGINT NOT NULL,
    details JSON
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_logs;
-- +goose StatementEnd
