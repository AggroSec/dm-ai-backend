-- name: InsertMessage :one
INSERT INTO messages (campaign_id, role, content, tool_calls, tool_call_id, sequence)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetMessagesByCampaign :many
SELECT * FROM messages
WHERE campaign_id = $1
ORDER BY sequence ASC;

-- name: GetRecentMessages :many
SELECT * FROM messages
WHERE campaign_id = $1
ORDER BY sequence DESC
LIMIT $2;

-- name: GetMessagesAfterSequence :many
SELECT * FROM messages
WHERE campaign_id = $1
AND sequence > $2
ORDER BY sequence ASC;

-- name: GetNextSequence :one
SELECT COALESCE(MAX(sequence), 0) + 1 AS next_sequence
FROM messages
WHERE campaign_id = $1;

-- name: GetMessageCount :one
SELECT COUNT(*) FROM messages
WHERE campaign_id = $1;

-- name: DeleteMessagesByCampaign :exec
DELETE FROM messages
WHERE campaign_id = $1;

-- name: CountMessagesAfterSequence :one
SELECT COUNT(*) FROM messages
WHERE campaign_id = $1
AND sequence > $2;