package repository

import (
	"context"
	"database/sql"

	"backend/internal/domain"
)

// Every content write locks the ticket before checking current access and keeps
// that lock through commit. FOR UPDATE conflicts with both closure and requester
// reassignment; an earlier service check is not enough.
func beginTicketContentWrite(ctx context.Context, db *sql.DB, actor *domain.User, ticketID string) (*sql.Tx, *domain.Ticket, error) {
	if actor == nil {
		return nil, nil, domain.ErrUnauthorized
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	ticket := &domain.Ticket{ID: ticketID}
	err = tx.QueryRowContext(ctx, `SELECT requester_id, status FROM tickets WHERE id = $1 FOR UPDATE`, ticketID).Scan(&ticket.RequesterID, &ticket.Status)
	if err == sql.ErrNoRows {
		err = domain.ErrNotFound
	}
	if err == nil && actor.Role == domain.RoleRequester && ticket.RequesterID != actor.ID {
		err = domain.ErrForbidden
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	return tx, ticket, nil
}

func authorizeCommentWrite(ctx context.Context, tx *sql.Tx, actor *domain.User, ticket *domain.Ticket, commentID string) error {
	var storedTicketID, authorID string
	var internal bool
	err := tx.QueryRowContext(ctx, `SELECT ticket_id, author_id, is_internal FROM comments WHERE id = $1 FOR UPDATE`, commentID).
		Scan(&storedTicketID, &authorID, &internal)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if storedTicketID != ticket.ID || authorID != actor.ID || (internal && actor.Role != domain.RoleAgent && actor.Role != domain.RoleAdmin) {
		return domain.ErrForbidden
	}
	if ticket.Status == domain.StatusClosed {
		return domain.ErrTicketClosed
	}
	return nil
}

func beginCommentWrite(ctx context.Context, db *sql.DB, actor *domain.User, id string) (*sql.Tx, error) {
	var ticketID string
	err := db.QueryRowContext(ctx, `SELECT ticket_id FROM comments WHERE id = $1`, id).Scan(&ticketID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	tx, ticket, err := beginTicketContentWrite(ctx, db, actor, ticketID)
	if err != nil {
		return nil, err
	}
	if err := authorizeCommentWrite(ctx, tx, actor, ticket, id); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func authorizeAttachmentWrite(ctx context.Context, tx *sql.Tx, actor *domain.User, ticket *domain.Ticket, att *domain.Attachment) error {
	if att.CommentID != nil {
		return authorizeCommentWrite(ctx, tx, actor, ticket, *att.CommentID)
	}
	if ticket.RequesterID != actor.ID {
		return domain.ErrForbidden
	}
	return nil
}

func beginAttachmentWrite(ctx context.Context, db *sql.DB, actor *domain.User, id string) (*sql.Tx, *domain.Attachment, error) {
	var ticketID string
	err := db.QueryRowContext(ctx, `SELECT ticket_id FROM attachments WHERE id = $1`, id).Scan(&ticketID)
	if err == sql.ErrNoRows {
		return nil, nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	tx, ticket, err := beginTicketContentWrite(ctx, db, actor, ticketID)
	if err != nil {
		return nil, nil, err
	}
	// Ticket/linkage are immutable through the API. Read the stored attachment,
	// rather than trusting replacement fields supplied from an earlier snapshot.
	att, err := scanAttachment(tx.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = $1 FOR UPDATE`, id))
	if err == nil && att.TicketID != ticketID {
		err = domain.ErrForbidden
	}
	if err == nil {
		err = authorizeAttachmentWrite(ctx, tx, actor, ticket, att)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	return tx, att, nil
}
