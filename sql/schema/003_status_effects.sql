-- +goose Up
ALTER TABLE characters DROP COLUMN status_effects;

CREATE TABLE status_effects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    effect TEXT NOT NULL,
    duration INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    persists BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE status_effects;

ALTER TABLE characters ADD COLUMN status_effects JSONB NOT NULL DEFAULT '[]';