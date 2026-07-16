-- +goose Up
-- Generalize the What/Why/How "W-fields" into ordinary schema fields (item.data),
-- retiring the dedicated columns. "who" free-text is dropped (superseded by
-- assignee_id). Runs once, so the injected fields stay removable afterwards.

-- 1) Backfill: move the column prose into the item's data bag. Existing data
--    keys win (backfill first, data second) so nothing already stored is lost.
UPDATE item
SET data = jsonb_strip_nulls(jsonb_build_object(
             'what', NULLIF(what, ''),
             'why',  NULLIF(why, ''),
             'how',  NULLIF(how, '')
           )) || data;

-- 2) Inject the standard What/Why/Where fields (Long text) into every stored
--    item_type that lacks them, prepended, so existing workspaces show them too.
UPDATE app_setting a
SET value = (
  SELECT jsonb_agg(
    jsonb_set(t, '{fields}',
      (
        (SELECT COALESCE(jsonb_agg(bf), '[]'::jsonb)
         FROM jsonb_array_elements('[{"key":"what","label":"What","type":"textarea"},{"key":"why","label":"Why","type":"textarea"},{"key":"how","label":"Where","type":"textarea"}]'::jsonb) bf
         WHERE NOT EXISTS (
           SELECT 1 FROM jsonb_array_elements(COALESCE(t->'fields', '[]'::jsonb)) ef
           WHERE ef->>'key' = bf->>'key'
         ))
        || COALESCE(t->'fields', '[]'::jsonb)
      )
    )
  )::text
  FROM jsonb_array_elements(a.value::jsonb) t
)
WHERE a.key = 'item_types' AND a.value IS NOT NULL AND a.value <> '';

-- 3) Drop the retired columns.
ALTER TABLE item DROP COLUMN what;
ALTER TABLE item DROP COLUMN why;
ALTER TABLE item DROP COLUMN how;
ALTER TABLE item DROP COLUMN who;

-- +goose Down
ALTER TABLE item ADD COLUMN what TEXT NOT NULL DEFAULT '';
ALTER TABLE item ADD COLUMN why  TEXT NOT NULL DEFAULT '';
ALTER TABLE item ADD COLUMN how  TEXT NOT NULL DEFAULT '';
ALTER TABLE item ADD COLUMN who  TEXT NOT NULL DEFAULT '';
UPDATE item SET
  what = COALESCE(data->>'what', ''),
  why  = COALESCE(data->>'why', ''),
  how  = COALESCE(data->>'how', '');
