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

func (r *CommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	query := `INSERT INTO comments (id, ticket_id, author_id, body, is_internal, created_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.DB.ExecContext(
		ctx,
		query,
		comment.ID,
		comment.TicketID,
		comment.AuthorID,
		comment.Body,
		comment.IsInternal,
		comment.CreatedAt,
	)
	return err
}

func (r *CommentRepository) ListByTicket(ctx context.Context, ticketID string, includeInternal bool) ([]*domain.Comment, error) {
	query := `
		SELECT 
			c.id, c.ticket_id, c.author_id, c.body, c.is_internal, c.created_at,
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
			&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.IsInternal, &c.CreatedAt,
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

		comments = append(comments, &c)
	}

	return comments, rows.Err()
}
