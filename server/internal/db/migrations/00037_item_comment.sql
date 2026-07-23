-- +goose Up
-- Comments on items — the Log tab's discussion stream. Body is free text; other
-- items are referenced inline as [[<item-id>]] tokens (resolved client-side, so
-- a referenced item's title stays current and the link survives renames).
CREATE TABLE item_comment (
  id           TEXT        PRIMARY KEY,
  workspace_id TEXT        NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  item_id      TEXT        NOT NULL,
  author_id    TEXT,
  body         TEXT        NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_item_comment_item ON item_comment(workspace_id, item_id, created_at);

-- +goose Down
DROP TABLE item_comment;
