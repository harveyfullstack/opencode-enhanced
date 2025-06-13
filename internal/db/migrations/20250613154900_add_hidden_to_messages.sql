-- +goose Up
ALTER TABLE messages ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE messages DROP COLUMN hidden;