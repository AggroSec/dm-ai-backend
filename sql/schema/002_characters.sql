-- +goose Up
CREATE TABLE characters (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Identity
    name            TEXT NOT NULL,
    race            TEXT NOT NULL DEFAULT '',
    class           TEXT NOT NULL,
    level           INT NOT NULL DEFAULT 1,
    experience      INT NOT NULL DEFAULT 0,

    -- Twin Fates
    driving_fate    TEXT NOT NULL DEFAULT '',
    binding_fate    TEXT NOT NULL DEFAULT '',

    -- Physical Stats
    strength        INT NOT NULL DEFAULT 10,
    dexterity       INT NOT NULL DEFAULT 10,
    fortitude       INT NOT NULL DEFAULT 10,

    -- Magical Stats
    willpower       INT NOT NULL DEFAULT 10,
    alacrity        INT NOT NULL DEFAULT 10,
    wisdom          INT NOT NULL DEFAULT 10,

    -- Resources
    max_hp          INT NOT NULL DEFAULT 100,
    current_hp      INT NOT NULL DEFAULT 100,
    max_wp          INT NOT NULL DEFAULT 100,
    current_wp      INT NOT NULL DEFAULT 100,
    action_points   INT NOT NULL DEFAULT 3,

    -- Talents & Inventory (JSONB for flexibility)
    talent_points_available INT NOT NULL DEFAULT 0,
    talents_invested        JSONB NOT NULL DEFAULT '[]',
    inventory               JSONB NOT NULL DEFAULT '[]',
    status_effects          JSONB NOT NULL DEFAULT '[]',

    -- Timestamps
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE characters;