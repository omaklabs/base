-- name: ListNotes :many
SELECT * FROM notes WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountNotesByUser :one
SELECT COUNT(*) as count FROM notes WHERE user_id = ?;

-- name: GetNote :one
SELECT * FROM notes WHERE id = ? LIMIT 1;

-- name: CreateNote :one
INSERT INTO notes (user_id, title, body) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateNote :one
UPDATE notes SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING *;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = ?;
