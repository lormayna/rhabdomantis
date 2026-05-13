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

-- name: GetIPs :many
SELECT ip, port FROM hosts where active = 1;

-- name: UpdateHostInactive :exec
UPDATE hosts SET active = 0, scanned_at=CURRENT_TIMESTAMP WHERE ip = ?;

-- name: UpdateHostActive :exec
UPDATE hosts SET active = 1, scanned_at=CURRENT_TIMESTAMP WHERE ip = ?;

-- name: DeleteModelsByHost :exec
DELETE FROM models 
WHERE host_id = (SELECT id FROM hosts WHERE ip = ?);

-- name: SaveModel :exec
INSERT INTO models (
    host_id, 
    name, 
    size, 
    family, 
    parameter_size, 
    digest
) VALUES (
    (SELECT id FROM hosts WHERE ip = ? LIMIT 1), 
    ?, ?, ?, ?, ?
);

-- name: GetRandomModelByIP :one
SELECT 
    m.id, 
    m.name, 
    h.ip, 
    h.port
FROM models m
JOIN hosts h ON m.host_id = h.id
WHERE h.ip = ?
ORDER BY RANDOM()
LIMIT 1;

-- name: SaveInference :exec
INSERT INTO inferences (
    model_id, 
    prompt, 
    response, 
    total_duration_ms, 
    prompt_tokens, 
    completion_tokens,
    verdict
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);