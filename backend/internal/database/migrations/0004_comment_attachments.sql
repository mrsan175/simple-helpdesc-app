ALTER TABLE attachments
    ADD COLUMN comment_id UUID REFERENCES comments(id) ON DELETE CASCADE;

ALTER TABLE comments
    ADD COLUMN updated_at TIMESTAMPTZ;

CREATE INDEX idx_attachments_comment ON attachments(comment_id);
