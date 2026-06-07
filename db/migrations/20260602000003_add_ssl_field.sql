-- +goose Up
ALTER TABLE hosts ADD COLUMN ssl_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE hosts DROP COLUMN ssl_enabled;
