package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"backend/internal/domain"
	"backend/internal/realtime"
)

func (s *TicketService) accessibleTicket(ctx context.Context, actor *domain.User, id string) (*domain.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleRequester && ticket.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}
	return ticket, nil
}

func (s *TicketService) accessibleComment(ctx context.Context, actor *domain.User, id string) (*domain.Comment, *domain.Ticket, error) {
	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	ticket, err := s.accessibleTicket(ctx, actor, comment.TicketID)
	if err != nil {
		return nil, nil, err
	}
	if comment.IsInternal && actor.Role != domain.RoleAgent && actor.Role != domain.RoleAdmin {
		return nil, nil, domain.ErrForbidden
	}
	return comment, ticket, nil
}

func (s *TicketService) ownComment(ctx context.Context, actor *domain.User, id string) (*domain.Comment, *domain.Ticket, error) {
	comment, ticket, err := s.accessibleComment(ctx, actor, id)
	if err != nil {
		return nil, nil, err
	}
	if comment.AuthorID != actor.ID {
		return nil, nil, domain.ErrForbidden
	}
	if ticket.Status == domain.StatusClosed {
		return nil, nil, domain.ErrTicketClosed
	}
	return comment, ticket, nil
}

func (s *TicketService) commentAttachments(ctx context.Context, id string) ([]*domain.Attachment, error) {
	attachments, err := s.attachmentRepo.ListByComment(ctx, id)
	if err != nil {
		return nil, err
	}
	if attachments == nil {
		attachments = make([]*domain.Attachment, 0)
	}
	return attachments, nil
}

func broadcastContent(ticket *domain.Ticket, event string) {
	realtime.DefaultBroker.BroadcastTo(
		realtime.Audience{
			Roles:   []string{string(domain.RoleAgent), string(domain.RoleAdmin)},
			UserIDs: []string{ticket.RequesterID},
		}, event, map[string]string{"ticket_id": ticket.ID},
	)
}

func (s *TicketService) UpdateComment(ctx context.Context, actor *domain.User, id string, req domain.UpdateCommentReq) (*domain.Comment, error) {
	comment, ticket, err := s.ownComment(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Body) == "" {
		return nil, domain.ErrValidation
	}
	attachments, err := s.commentAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	// Copy before changing fields so failed persistence cannot mutate cached records.
	updated := *comment
	now := time.Now().UTC()
	updated.Body = req.Body
	updated.UpdatedAt = &now
	updated.Attachments = attachments
	if err := s.commentRepo.Update(ctx, actor, &updated); err != nil {
		return nil, err
	}
	broadcastContent(ticket, "comment_updated")
	return &updated, nil
}

func (s *TicketService) DeleteComment(ctx context.Context, actor *domain.User, id string) ([]*domain.Attachment, error) {
	_, ticket, err := s.ownComment(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	files, err := s.commentRepo.Delete(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	broadcastContent(ticket, "comment_deleted")
	return files, nil
}

func (s *TicketService) CheckAttachmentUpload(ctx context.Context, actor *domain.User, ticketID string) error {
	ticket, err := s.accessibleTicket(ctx, actor, ticketID)
	if err != nil {
		return err
	}
	if ticket.RequesterID != actor.ID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *TicketService) CheckCommentAttachmentUpload(ctx context.Context, actor *domain.User, commentID string) error {
	_, _, err := s.ownComment(ctx, actor, commentID)
	return err
}

func (s *TicketService) AddAttachment(ctx context.Context, actor *domain.User, ticketID string, file *domain.Attachment) (*domain.Attachment, error) {
	ticket, err := s.accessibleTicket(ctx, actor, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}
	return s.createAttachment(ctx, actor, ticket, nil, file)
}

func (s *TicketService) AddCommentAttachment(ctx context.Context, actor *domain.User, commentID string, file *domain.Attachment) (*domain.Attachment, error) {
	comment, ticket, err := s.ownComment(ctx, actor, commentID)
	if err != nil {
		return nil, err
	}
	return s.createAttachment(ctx, actor, ticket, &comment.ID, file)
}

func (s *TicketService) createAttachment(ctx context.Context, actor *domain.User, ticket *domain.Ticket, commentID *string, file *domain.Attachment) (*domain.Attachment, error) {
	att := *file
	att.ID = uuid.NewString()
	att.TicketID = ticket.ID
	att.CommentID = commentID
	att.UploaderID = actor.ID
	att.CreatedAt = time.Now().UTC()
	att.Uploader = actor
	if err := s.attachmentRepo.Create(ctx, actor, &att); err != nil {
		return nil, err
	}
	broadcastContent(ticket, "attachment_added")
	return &att, nil
}

func (s *TicketService) GetAttachment(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, error) {
	att, _, err := s.accessibleAttachment(ctx, actor, id)
	return att, err
}

func (s *TicketService) accessibleAttachment(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, *domain.Ticket, error) {
	att, err := s.attachmentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	ticket, err := s.accessibleTicket(ctx, actor, att.TicketID)
	if err != nil {
		return nil, nil, err
	}
	if att.CommentID != nil {
		comment, _, err := s.accessibleComment(ctx, actor, *att.CommentID)
		if err != nil {
			return nil, nil, err
		}
		if comment.TicketID != att.TicketID {
			return nil, nil, domain.ErrForbidden
		}
	}
	return att, ticket, nil
}

func (s *TicketService) managedAttachment(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, *domain.Ticket, error) {
	att, ticket, err := s.accessibleAttachment(ctx, actor, id)
	if err != nil {
		return nil, nil, err
	}
	ownerID := ticket.RequesterID
	if att.CommentID != nil {
		comment, err := s.commentRepo.FindByID(ctx, *att.CommentID)
		if err != nil {
			return nil, nil, err
		}
		ownerID = comment.AuthorID
	}
	if ownerID != actor.ID {
		return nil, nil, domain.ErrForbidden
	}
	if att.CommentID != nil && ticket.Status == domain.StatusClosed {
		return nil, nil, domain.ErrTicketClosed
	}
	return att, ticket, nil
}

func (s *TicketService) CheckAttachmentUpdate(ctx context.Context, actor *domain.User, id string) error {
	_, _, err := s.managedAttachment(ctx, actor, id)
	return err
}

func (s *TicketService) ReplaceAttachment(ctx context.Context, actor *domain.User, id string, file *domain.Attachment) (*domain.Attachment, string, error) {
	att, ticket, err := s.managedAttachment(ctx, actor, id)
	if err != nil {
		return nil, "", err
	}
	updated := *att
	updated.Filename = file.Filename
	updated.FilePath = file.FilePath
	updated.FileSize = file.FileSize
	updated.ContentType = file.ContentType
	updated.UploaderID = actor.ID
	updated.Uploader = actor
	oldPath, err := s.attachmentRepo.Replace(ctx, actor, &updated)
	if err != nil {
		return nil, "", err
	}
	broadcastContent(ticket, "attachment_updated")
	return &updated, oldPath, nil
}

func (s *TicketService) DeleteAttachment(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, error) {
	_, ticket, err := s.managedAttachment(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	deleted, err := s.attachmentRepo.Delete(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	broadcastContent(ticket, "attachment_deleted")
	return deleted, nil
}
