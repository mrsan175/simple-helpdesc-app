-- Ticket attachments: metadata only; bytes stored on disk.

CREATE TABLE IF NOT EXISTS attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id    UUID         NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    uploader_id  UUID         NOT NULL REFERENCES users(id),
    filename     VARCHAR(255) NOT NULL,
    file_path    VARCHAR(500) NOT NULL,
    file_size    BIGINT       NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attachments_ticket ON attachments(ticket_id);
