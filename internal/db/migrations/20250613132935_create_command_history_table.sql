-- +goose Up
-- +goose StatementBegin
CREATE TABLE command_history (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    command_text TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE command_history;
-- +goose StatementEnd
