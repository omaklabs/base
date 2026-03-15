-- name: CreateSession :one
INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP;
