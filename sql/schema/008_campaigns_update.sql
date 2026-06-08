-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN theme TEXT NOT NULL DEFAULT '',
    ADD COLUMN narrative_summary TEXT NOT NULL DEFAULT '',
    ADD COLUMN dm_notes TEXT NOT NULL DEFAULT '',
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN summarized_through INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE campaigns
    DROP COLUMN theme,
    DROP COLUMN narrative_summary,
    DROP COLUMN dm_notes,
    DROP COLUMN status,
    DROP COLUMN summarized_through;