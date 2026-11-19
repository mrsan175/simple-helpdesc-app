package repository

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/domain"
)

type CommentRepository struct {
	DB *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

func (r *CommentRepository) Create(ctx context.Context, actor *domain.User, comment *domain.Comment) error {
	tx, _, err := beginTicketContentWrite(ctx, r.DB, actor, comment.TicketID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if comment.AuthorID != actor.ID || (comment.IsInternal && actor.Role != domain.RoleAgent && actor.Role != domain.RoleAdmin) {
		return domain.ErrForbidden
	}
	query := `INSERT INTO comments (id, ticket_id, author_id, body, is_internal, created_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(
		ctx,
		query,
		comment.ID,
		comment.TicketID,
		comment.AuthorID,
		comment.Body,
		comment.IsInternal,
		comment.CreatedAt,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *CommentRepository) ListByTicket(ctx context.Context, ticketID string, includeInternal bool) ([]*domain.Comment, error) {
	query := `
		SELECT
			c.id, c.ticket_id, c.author_id, c.body, c.is_internal, c.created_at, c.updated_at,
			u.id, u.name, u.email, u.role, u.created_at
		FROM comments c
		JOIN users u ON c.author_id = u.id
		WHERE c.ticket_id = $1`

	var args []interface{}
	args = append(args, ticketID)

	if !includeInternal {
		query += " AND c.is_internal = FALSE"
	}

	query += " ORDER BY c.created_at ASC"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments = make([]*domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		var authorID, authorName, authorEmail, authorRole string
		var authorCreatedAt time.Time

		if err := rows.Scan(
			&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.IsInternal, &c.CreatedAt, &c.UpdatedAt,
			&authorID, &authorName, &authorEmail, &authorRole, &authorCreatedAt,
		); err != nil {
			return nil, err
		}

		c.Author = &domain.User{
			ID:        authorID,
			Name:      authorName,
			Email:     authorEmail,
			Role:      domain.UserRole(authorRole),
			CreatedAt: authorCreatedAt,
		}

		c.Attachments = make([]*domain.Attachment, 0)
		comments = append(comments, &c)
	}

	return comments, rows.Err()
}

func (r *CommentRepository) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	var c domain.Comment
	var author domain.User
	err := r.DB.QueryRowContext(ctx, `
		SELECT c.id, c.ticket_id, c.author_id, c.body, c.is_internal, c.created_at, c.updated_at,
		       u.id, u.name, u.email, u.role, u.created_at
		FROM comments c JOIN users u ON c.author_id = u.id WHERE c.id = $1`, id).
		Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.IsInternal, &c.CreatedAt, &c.UpdatedAt,
			&author.ID, &author.Name, &author.Email, &author.Role, &author.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Author = &author
	c.Attachments = make([]*domain.Attachment, 0)
	return &c, nil
}

func (r *CommentRepository) Update(ctx context.Context, actor *domain.User, c *domain.Comment) error {
	tx, err := beginCommentWrite(ctx, r.DB, actor, c.ID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE comments SET body = $2, updated_at = $3 WHERE id = $1`, c.ID, c.Body, c.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// Delete checks access and captures file paths in the deletion transaction.
// Ticket/comment locks also serialize uploads so files cannot be orphaned.
func (r *CommentRepository) Delete(ctx context.Context, actor *domain.User, id string) ([]*domain.Attachment, error) {
	tx, err := beginCommentWrite(ctx, r.DB, actor, id)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `DELETE FROM attachments WHERE comment_id = $1 RETURNING `+attachmentColumns, id)
	if err != nil {
		return nil, err
	}
	files := make([]*domain.Attachment, 0)
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		files = append(files, a)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return files, nil
}
