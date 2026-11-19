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

const attachmentColumns = `id, ticket_id, comment_id, uploader_id, filename, file_path, file_size, content_type, created_at`

func scanAttachment(row interface{ Scan(...any) error }) (*domain.Attachment, error) {
	var a domain.Attachment
	err := row.Scan(&a.ID, &a.TicketID, &a.CommentID, &a.UploaderID, &a.Filename, &a.FilePath, &a.FileSize, &a.ContentType, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepository) Create(ctx context.Context, actor *domain.User, att *domain.Attachment) error {
	tx, ticket, err := beginTicketContentWrite(ctx, r.DB, actor, att.TicketID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := authorizeAttachmentWrite(ctx, tx, actor, ticket, att); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO attachments (`+attachmentColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		att.ID, att.TicketID, att.CommentID, att.UploaderID, att.Filename, att.FilePath, att.FileSize, att.ContentType, att.CreatedAt,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ListByTicket returns only main-ticket attachments, including unlinked legacy files.
func (r *AttachmentRepository) ListByTicket(ctx context.Context, ticketID string) ([]*domain.Attachment, error) {
	return r.list(ctx, "a.ticket_id = $1 AND a.comment_id IS NULL", ticketID)
}

func (r *AttachmentRepository) ListByComment(ctx context.Context, commentID string) ([]*domain.Attachment, error) {
	return r.list(ctx, "a.comment_id = $1", commentID)
}

func (r *AttachmentRepository) list(ctx context.Context, predicate, id string) ([]*domain.Attachment, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT a.id, a.ticket_id, a.comment_id, a.uploader_id, a.filename, a.file_path, a.file_size, a.content_type, a.created_at,
		       u.id, u.name, u.email, u.role
		FROM attachments a
		JOIN users u ON a.uploader_id = u.id
		WHERE `+predicate+`
		ORDER BY a.created_at ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*domain.Attachment, 0)
	for rows.Next() {
		var a domain.Attachment
		var u domain.User
		if err := rows.Scan(&a.ID, &a.TicketID, &a.CommentID, &a.UploaderID, &a.Filename, &a.FilePath, &a.FileSize, &a.ContentType, &a.CreatedAt,
			&u.ID, &u.Name, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		a.Uploader = &u
		list = append(list, &a)
	}
	return list, rows.Err()
}

func (r *AttachmentRepository) FindByID(ctx context.Context, id string) (*domain.Attachment, error) {
	return scanAttachment(r.DB.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = $1`, id))
}

// Replace returns the path actually replaced, even when replacements race.
func (r *AttachmentRepository) Replace(ctx context.Context, actor *domain.User, att *domain.Attachment) (string, error) {
	tx, previous, err := beginAttachmentWrite(ctx, r.DB, actor, att.ID)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE attachments SET uploader_id = $2, filename = $3, file_path = $4, file_size = $5, content_type = $6 WHERE id = $1`,
		att.ID, att.UploaderID, att.Filename, att.FilePath, att.FileSize, att.ContentType)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return previous.FilePath, nil
}

func (r *AttachmentRepository) Delete(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, error) {
	tx, _, err := beginAttachmentWrite(ctx, r.DB, actor, id)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	deleted, err := scanAttachment(tx.QueryRowContext(ctx, `DELETE FROM attachments WHERE id = $1 RETURNING `+attachmentColumns, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return deleted, nil
}
