-- +goose Up
-- Per-item maturity stage: 1=Concept, 2=Design, 3=Production, 4=Series (null = unset).
ALTER TABLE item ADD COLUMN maturity INT;

-- +goose Down
ALTER TABLE item DROP COLUMN maturity;
