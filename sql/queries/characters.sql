-- name: CreateCharacter :one

INSERT INTO characters (name, class, user_id)
values ($1, $2, $3)
returning *;

-- name: GetCharacterByID :one

select * from characters where id = $1;

-- name: GetCharacterByUserID :many

select * from characters where user_id = $1;

-- name: UpdateCharacter :one

UPDATE characters
SET name = $1,
    race = $2,
    class = $3,
    level = $4,
    strength = $5,
    dexterity = $6,
    fortitude = $7,
    willpower = $8,
    alacrity = $9,
    wisdom = $10,
    current_hp = $11,
    max_hp = $12,
    current_wp = $13,
    max_wp = $14,
    driving_fate = $15,
    binding_fate = $16,
    talents_invested = $17,
    talent_points_available = $18,
    inventory = $19,
    updated_at = NOW(),
    max_ap = $20,
    overcap_ap = $21,
    action_points = $22

WHERE id = $23
RETURNING *;

-- name: DeleteCharacter :exec

delete from characters where id = $1 and user_id = $2;

-- name: UpdateCharacterHP :exec
UPDATE characters SET current_hp = $2 WHERE id = $1;

-- name: UpdateCharacterWP :exec  
UPDATE characters SET current_wp = $2 WHERE id = $1;

-- name: ValidateCharacterOwnership :one
SELECT id FROM characters WHERE id = $1 AND user_id = $2;