-- name: CreateEmail :one
INSERT INTO emails (to_addr, from_addr, reply_to, subject, html_body, text_body, status)
VALUES (?, ?, ?, ?, ?, ?, 'pending') RETURNING *;

-- name: GetEmail :one
SELECT * FROM emails WHERE id = ? LIMIT 1;

-- name: ListEmails :many
SELECT * FROM emails WHERE (sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status))
ORDER BY created_at DESC LIMIT sqlc.arg(query_limit) OFFSET sqlc.arg(query_offset);

-- name: MarkEmailSent :exec
UPDATE emails SET status = 'sent', sent_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: MarkEmailFailed :exec
UPDATE emails SET status = 'failed', error = ? WHERE id = ?;

-- name: CountEmailsByStatus :many
SELECT status, COUNT(*) as count FROM emails GROUP BY status;
