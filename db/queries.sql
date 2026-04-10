-- db/queries.sql

-- name: ListHosts :many
SELECT * FROM hosts;

-- name: InsertHost :exec
INSERT INTO hosts (ip, port, isp, asn, country, city, scanned_at)
VALUES (?, ?, ?, ?, ?, ?, ?);
