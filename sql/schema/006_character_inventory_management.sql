-- +goose Up
ALTER TABLE characters
    ADD COLUMN equipped_slots JSONB DEFAULT NULL;

-- +goose Down
ALTER TABLE characters
    DROP COLUMN equipped_slots;