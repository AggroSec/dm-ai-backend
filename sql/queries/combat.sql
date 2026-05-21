-- name: CreateCombatSession :one
INSERT INTO combat_sessions (campaign_id, current_turn, turn_order, combatants)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCombatSession :one
SELECT * FROM combat_sessions
WHERE id = $1;

-- name: UpdateCombatState :one
UPDATE combat_sessions
SET current_turn = $2,
    turn_order = $3,
    combatants = $4,
    round = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: EndCombatSession :one
UPDATE combat_sessions
SET status = 'ended',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetActiveCombatBySession :one
SELECT * FROM combat_sessions
WHERE id = $1 AND status = 'active';