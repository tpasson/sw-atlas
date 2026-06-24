-- +goose Up
-- The default milestone marker moved from the legacy geometric 'diamond' to the
-- explicit Lucide marker 'l:Diamond' (in the library, shown in the legend, filled).
UPDATE item SET marker = 'l:Diamond' WHERE marker = 'diamond';
ALTER TABLE item ALTER COLUMN marker SET DEFAULT 'l:Diamond';

-- +goose Down
ALTER TABLE item ALTER COLUMN marker SET DEFAULT 'diamond';
UPDATE item SET marker = 'diamond' WHERE marker = 'l:Diamond';
