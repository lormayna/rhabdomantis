-- +goose Up
ALTER TABLE hosts ADD COLUMN scan_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE hosts ADD COLUMN failed_scan_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support dropping columns easily in older versions, 
-- but we can leave it or handle it with a table recreation if needed.
-- For now, we'll keep it simple as goose migrations for SQLite often 
-- don't bother with complex downs for column additions.
