-- +goose Up
-- Versioning & attribution for items. Every artifact now carries a monotonic
-- version plus who/when created & last edited it, and every change is recorded
-- as an immutable revision (a full snapshot). Baselines stop duplicating item
-- data and instead reference a specific (item_id, version).

ALTER TABLE item
  ADD COLUMN version    INT         NOT NULL DEFAULT 1,
  ADD COLUMN created_by TEXT,
  ADD COLUMN updated_by TEXT,
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE item_revision (
  workspace_id TEXT        NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  item_id      TEXT        NOT NULL,
  version      INT         NOT NULL,
  snapshot     JSONB       NOT NULL,
  edited_by    TEXT,
  edited_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, item_id, version)
);
CREATE INDEX idx_item_revision_item ON item_revision(workspace_id, item_id);

-- Seed a v1 revision for every existing item so history and version-based
-- baselines have a point to reference.
INSERT INTO item_revision (workspace_id, item_id, version, snapshot, edited_by)
SELECT workspace_id, id, 1,
       jsonb_build_object(
         'id', id, 'swimlaneId', swimlane_id, 'subLaneId', sub_lane_id,
         'year', year, 'month', month, 'title', title,
         'what', what, 'why', why, 'how', how, 'who', who,
         'when', to_char(when_date, 'YYYY-MM-DD'),
         'kind', kind, 'typeKey', type_key, 'marker', marker,
         'startDate', to_char(start_date, 'YYYY-MM-DD'),
         'endDate', to_char(end_date, 'YYYY-MM-DD'),
         'color', color, 'maturity', maturity, 'progress', progress,
         'scmUrl', scm_url, 'assigneeId', assignee_id,
         'sourceSystem', source_system, 'data', data, 'version', 1
       ),
       created_by
FROM item;

-- Baselines reference an item version instead of copying its fields. The old
-- per-row snapshot columns stay (older baselines still resolve through them) but
-- become optional for new version-pointer rows.
ALTER TABLE baseline_item ADD COLUMN version INT;
ALTER TABLE baseline_item
  ALTER COLUMN swimlane_id DROP NOT NULL,
  ALTER COLUMN title       DROP NOT NULL,
  ALTER COLUMN year        DROP NOT NULL,
  ALTER COLUMN month       DROP NOT NULL;

-- +goose Down
ALTER TABLE baseline_item DROP COLUMN IF EXISTS version;
DROP TABLE IF EXISTS item_revision;
ALTER TABLE item
  DROP COLUMN IF EXISTS version,
  DROP COLUMN IF EXISTS created_by,
  DROP COLUMN IF EXISTS updated_by,
  DROP COLUMN IF EXISTS created_at,
  DROP COLUMN IF EXISTS updated_at;
