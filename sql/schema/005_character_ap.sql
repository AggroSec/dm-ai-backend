-- +goose Up
ALTER TABLE characters
    ADD COLUMN max_ap INTEGER NOT NULL DEFAULT 3,
    ADD COLUMN overcap_ap INTEGER NOT NULL DEFAULT 5;

-- +goose Down
ALTER TABLE characters
    DROP COLUMN max_ap,
    DROP COLUMN overcap_ap;