-- name: AddStatusEffect :one

INSERT INTO status_effects (character_id, effect, duration, persists, instruction)
values ($1, $2, $3, $4, $5)
returning *;

-- name: GetStatusEffectsByID :many

SELECT * FROM status_effects WHERE character_id = $1 and is_active = TRUE;

-- name: UpdateStatusDuration :one

UPDATE status_effects
SET duration = $3,
    updated_at = NOW(),
    is_active = $4
where id = $1 and character_id = $2
returning *;

-- name: PurgeInactiveEffects :exec

DELETE FROM status_effects
WHERE character_id = $1 AND is_active = FALSE;

-- name: PurgeNonPersistingEffects :exec

DELETE FROM status_effects
WHERE character_id = $1 AND persists = FALSE;

-- name: RemoveStatusEffect :exec
DELETE FROM status_effects
WHERE id = $1 AND character_id = $2;