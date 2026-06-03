-- +goose Up
ALTER TABLE status_effects ADD COLUMN instruction TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE status_effects DROP COLUMN instruction;