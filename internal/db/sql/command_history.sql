-- name: CreateCommandHistory :one
INSERT INTO command_history (
    id,
    session_id,
    command_text,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, strftime('%s', 'now'), strftime('%s', 'now')
)
RETURNING *;

-- name: ListCommandHistoryBySession :many
SELECT *
FROM command_history
WHERE session_id = ?
ORDER BY created_at ASC;

-- name: DeleteSessionCommandHistory :exec
DELETE FROM command_history
WHERE session_id = ?;
