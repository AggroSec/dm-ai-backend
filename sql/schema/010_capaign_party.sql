-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN party JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN party;