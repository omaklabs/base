-- name: CreateJob :one
INSERT INTO jobs (type, payload, status, run_at) VALUES (?, ?, 'pending', ?) RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ? LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs WHERE (sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status))
ORDER BY created_at DESC LIMIT sqlc.arg(query_limit) OFFSET sqlc.arg(query_offset);

-- name: ClaimNextJob :one
UPDATE jobs
SET status = 'running', attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= CURRENT_TIMESTAMP
    ORDER BY run_at ASC
    LIMIT 1
)
RETURNING *;

-- name: CompleteJob :exec
UPDATE jobs SET status = 'done' WHERE id = ?;

-- name: FailJob :exec
UPDATE jobs SET status = 'failed', error = ? WHERE id = ?;

-- name: RetryJob :exec
UPDATE jobs SET status = 'pending', error = '' WHERE id = ? AND status = 'failed';

-- name: CancelJob :exec
UPDATE jobs SET status = 'cancelled' WHERE id = ? AND status = 'pending';

-- name: CountJobsByStatus :many
SELECT status, COUNT(*) as count FROM jobs GROUP BY status;
