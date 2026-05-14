-- name: CreateCharacter :one

INSERT INTO characters (name, class, user_id)
values ($1, $2, $3)
returning *;

-- name: GetCharacterByID :one

select * from characters where id = $1 AND user_id = $2;

-- name: GetCharacterByUserID :many

select * from characters where user_id = $1;

-- name: UpdateCharacter :one

UPDATE characters
SET name = $1,
    class = $2,
    level = $3,
    strength = $4,
    dexterity = $5,
    fortitude = $6,
    willpower = $7,
    alacrity = $8,
    wisdom = $9,
    current_hp = $10,
    max_hp = $11,
    current_wp = $12,
    max_wp = $13,
    driving_fate = $14,
    binding_fate = $15,
    talents_invested = $16,
    talent_points_available = $17,
    inventory = $18,
    updated_at = NOW()
WHERE id = $19 AND user_id = $20
RETURNING *;

-- name: DeleteCharacter :exec

delete from characters where id = $1 and user_id = $2;