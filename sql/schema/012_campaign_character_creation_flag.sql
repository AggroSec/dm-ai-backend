-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN character_creation_complete BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN character_creation_complete;