package repository

import (
	"context"
	"database/sql"

	"backend/internal/domain"
)

type AttachmentRepository struct {
	DB *sql.DB
}

func NewAttachmentRepository(db *sql.DB) *AttachmentRepository {
	return &AttachmentRepository{DB: db}
}

func (r *AttachmentRepository) Create(ctx context.Context, att *domain.Attachment) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO attachments (id, ticket_id, uploader_id, filename, file_path, file_size, content_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		att.ID, att.TicketID, att.UploaderID, att.Filename, att.FilePath, att.FileSize, att.ContentType, att.CreatedAt,
	)
	return err
}

func (r *AttachmentRepository) ListByTicket(ctx context.Context, ticketID string) ([]*domain.Attachment, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT a.id, a.ticket_id, a.uploader_id, a.filename, a.file_path, a.file_size, a.content_type, a.created_at,
		       u.id, u.name, u.email, u.role
		FROM attachments a
		JOIN users u ON a.uploader_id = u.id
		WHERE a.ticket_id = $1
		ORDER BY a.created_at ASC`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Attachment
	for rows.Next() {
		var a domain.Attachment
		var u domain.User
		if err := rows.Scan(&a.ID, &a.TicketID, &a.UploaderID, &a.Filename, &a.FilePath, &a.FileSize, &a.ContentType, &a.CreatedAt,
			&u.ID, &u.Name, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		a.Uploader = &u
		list = append(list, &a)
	}
	return list, rows.Err()
}

func (r *AttachmentRepository) FindByID(ctx context.Context, id string) (*domain.Attachment, error) {
	var a domain.Attachment
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, ticket_id, uploader_id, filename, file_path, file_size, content_type, created_at
		FROM attachments WHERE id = $1`, id).
		Scan(&a.ID, &a.TicketID, &a.UploaderID, &a.Filename, &a.FilePath, &a.FileSize, &a.ContentType, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
