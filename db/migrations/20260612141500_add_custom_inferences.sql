-- +goose Up
CREATE TABLE IF NOT EXISTS custom_inferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip TEXT NOT NULL,
    model_id INTEGER NOT NULL,
    prompt TEXT NOT NULL,
    reply TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(ip) REFERENCES hosts(ip) ON DELETE CASCADE,
    FOREIGN KEY(model_id) REFERENCES models(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_custom_inferences_ip ON custom_inferences(ip);
CREATE INDEX IF NOT EXISTS idx_custom_inferences_model_id ON custom_inferences(model_id);

-- +goose Down
DROP INDEX IF EXISTS idx_custom_inferences_model_id;
DROP INDEX IF EXISTS idx_custom_inferences_ip;
DROP TABLE IF EXISTS custom_inferences;
