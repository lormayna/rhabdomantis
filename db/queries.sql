-- db/queries.sql

-- name: ListHosts :many
SELECT * FROM hosts;

-- name: InsertHost :exec
INSERT INTO hosts (ip, port, isp, asn, country, city, scanned_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertModel :exec
INSERT INTO models (host_id, name)
VALUES (?, ?);

-- name: ListModelsByHost :many
SELECT id, host_id, name, created_at FROM models
WHERE host_id = ?
ORDER BY created_at DESC;
