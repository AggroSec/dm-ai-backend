-- name: CreateCampaign :one
INSERT INTO campaigns (name, owner_id, theme)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCampaign :one
SELECT * FROM campaigns
WHERE id = $1;

-- name: GetCampaignsByOwner :many
SELECT * FROM campaigns
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: UpdateCampaignTheme :one
UPDATE campaigns
SET theme = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE campaigns
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCampaignDMNotes :one
UPDATE campaigns
SET dm_notes = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateNarrativeSummary :one
UPDATE campaigns
SET narrative_summary = $2,
    summarized_through = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;