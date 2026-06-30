-- +goose Up
CREATE TABLE IF NOT EXISTS hosts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip TEXT NOT NULL UNIQUE,
    port INTEGER NOT NULL,
    isp TEXT,
    asn TEXT,
    country TEXT,
    city TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    scanned_at DATETIME
);

CREATE TABLE IF NOT EXISTS models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    size INTEGER,
    family TEXT,
    parameter_size TEXT,
    digest TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(host_id) REFERENCES hosts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT,
    total_duration_ms INTEGER,
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    verdict TEXT CHECK(verdict IN ('success', 'failed', 'pending')) DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    http_status_code INTEGER, 
    notes TEXT,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_inferences_model_id ON inferences(model_id);

-- +goose Down
DROP INDEX IF EXISTS idx_inferences_model_id;
DROP TABLE IF EXISTS inferences;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS hosts;
