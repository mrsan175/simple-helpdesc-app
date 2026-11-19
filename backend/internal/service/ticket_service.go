package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"backend/internal/domain"
	"backend/internal/realtime"
)

type TicketService struct {
	ticketRepo     domain.TicketRepository
	commentRepo    domain.CommentRepository
	userRepo       domain.UserRepository
	attachmentRepo domain.AttachmentRepository
	slaDuration    time.Duration
}

func NewTicketService(
	ticketRepo domain.TicketRepository,
	commentRepo domain.CommentRepository,
	userRepo domain.UserRepository,
	attachmentRepo domain.AttachmentRepository,
	slaHours int,
) *TicketService {
	return &TicketService{
		ticketRepo:     ticketRepo,
		commentRepo:    commentRepo,
		userRepo:       userRepo,
		attachmentRepo: attachmentRepo,
		slaDuration:    time.Duration(slaHours) * time.Hour,
	}
}

func (s *TicketService) List(ctx context.Context, actor *domain.User, filter domain.TicketFilter) (*domain.TicketListResponse, error) {
	if actor.Role == domain.RoleRequester {
		filter.RequesterID = actor.ID
	}

	tickets, total, err := s.ticketRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	for _, t := range tickets {
		if t.Status == domain.StatusOpen && t.Priority == domain.PriorityUrgent {
			if now.Sub(t.CreatedAt) > s.slaDuration {
				t.IsOverdue = true
			}
		}
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	return &domain.TicketListResponse{
		Data: tickets,
		Meta: domain.PaginationMeta{
			TotalItems:  total,
			TotalPages:  totalPages,
			CurrentPage: filter.Page,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *TicketService) Get(ctx context.Context, actor *domain.User, id string) (*domain.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if actor.Role == domain.RoleRequester && ticket.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}

	includeInternal := actor.Role == domain.RoleAgent || actor.Role == domain.RoleAdmin
	comments, err := s.commentRepo.ListByTicket(ctx, id, includeInternal)
	if err != nil {
		return nil, err
	}
	ticket.Comments = comments

	attachments, err := s.attachmentRepo.ListByTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	ticket.Attachments = attachments

	if ticket.Status == domain.StatusOpen && ticket.Priority == domain.PriorityUrgent {
		if time.Since(ticket.CreatedAt) > s.slaDuration {
			ticket.IsOverdue = true
		}
	}

	return ticket, nil
}

func (s *TicketService) Create(ctx context.Context, actor *domain.User, req domain.CreateTicketReq) (*domain.Ticket, error) {
	if actor.Role == domain.RoleAgent {
		return nil, domain.ErrForbidden
	}
	priority := req.Priority

	requesterID := actor.ID
	requesterUser := actor
	if req.RequesterID != nil && *req.RequesterID != "" && actor.Role == domain.RoleAdmin {
		target, err := s.userRepo.FindByID(ctx, *req.RequesterID)
		if err != nil {
			return nil, domain.ErrNotFound
		}
		requesterID = target.ID
		requesterUser = target
	}

	now := time.Now().UTC()
	ticket := &domain.Ticket{
		ID:          uuid.NewString(),
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Priority:    priority,
		Status:      domain.StatusOpen,
		RequesterID: requesterID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}

	ticket.Requester = requesterUser
	realtime.DefaultBroker.BroadcastTo(
		realtime.Audience{Roles: []string{string(domain.RoleAgent), string(domain.RoleAdmin)}},
		"ticket_created",
		map[string]string{"id": ticket.ID, "title": ticket.Title},
	)
	return ticket, nil
}

func (s *TicketService) Update(ctx context.Context, actor *domain.User, id string, req domain.UpdateTicketReq) (*domain.Ticket, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if actor.Role == domain.RoleRequester {
		if ticket.RequesterID != actor.ID {
			return nil, domain.ErrForbidden
		}
		if req.Priority != nil || req.AssigneeID != nil || req.RequesterID != nil {
			return nil, domain.ErrForbidden
		}
	}

	if actor.Role == domain.RoleAgent {
		// Agent tidak boleh mengubah assignee maupun pelapor sama sekali.
		if req.AssigneeID != nil || req.RequesterID != nil {
			return nil, domain.ErrForbidden
		}
		// Agent & Admin dapat mengatur prioritas tiket.
	}

	if req.Status != nil && *req.Status != ticket.Status {
		if err := ValidateTransition(ticket.Status, *req.Status, *actor); err != nil {
			return nil, domain.ErrInvalidTransition
		}
		ticket.Status = *req.Status
		if *req.Status == domain.StatusInProgress && actor.Role == domain.RoleAgent {
			ticket.AssigneeID = &actor.ID
			ticket.Assignee = actor
		}
	}

	if req.Priority != nil && (actor.Role == domain.RoleAdmin || actor.Role == domain.RoleAgent) {
		ticket.Priority = *req.Priority
	}

	if req.AssigneeID != nil && actor.Role == domain.RoleAdmin {
		if *req.AssigneeID == "" || *req.AssigneeID == "unassigned" {
			ticket.AssigneeID = nil
			ticket.Assignee = nil
		} else {
			assigneeUser, err := s.userRepo.FindByID(ctx, *req.AssigneeID)
			if err != nil {
				return nil, domain.ErrNotFound
			}
			ticket.AssigneeID = &assigneeUser.ID
			ticket.Assignee = assigneeUser
		}
	}

	// Admin dapat mengubah pelapor (requester) tiket.
	if req.RequesterID != nil && actor.Role == domain.RoleAdmin {
		if *req.RequesterID == "" {
			return nil, domain.ErrNotFound
		}
		requesterUser, err := s.userRepo.FindByID(ctx, *req.RequesterID)
		if err != nil {
			return nil, domain.ErrNotFound
		}
		ticket.RequesterID = requesterUser.ID
		ticket.Requester = requesterUser
	}

	ticket.UpdatedAt = time.Now().UTC()
	if err := s.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, err
	}

	realtime.DefaultBroker.BroadcastTo(
		realtime.Audience{
			Roles:   []string{string(domain.RoleAgent), string(domain.RoleAdmin)},
			UserIDs: []string{ticket.RequesterID},
		},
		"ticket_updated",
		map[string]string{"id": ticket.ID},
	)
	return s.Get(ctx, actor, ticket.ID)
}

func (s *TicketService) AddComment(ctx context.Context, actor *domain.User, ticketID string, req domain.CreateCommentReq) (*domain.Comment, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if actor.Role == domain.RoleRequester {
		if ticket.RequesterID != actor.ID {
			return nil, domain.ErrForbidden
		}
		if req.IsInternal {
			return nil, domain.ErrForbidden
		}
	}

	comment := &domain.Comment{
		ID:         uuid.NewString(),
		TicketID:   ticketID,
		AuthorID:   actor.ID,
		Body:       req.Body,
		IsInternal: req.IsInternal,
		CreatedAt:  time.Now().UTC(),
		Author:     actor,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	realtime.DefaultBroker.BroadcastTo(
		realtime.Audience{
			Roles:   []string{string(domain.RoleAgent), string(domain.RoleAdmin)},
			UserIDs: []string{ticket.RequesterID},
		},
		"comment_added",
		map[string]string{"ticket_id": ticketID},
	)
	return comment, nil
}

func (s *TicketService) AddAttachment(ctx context.Context, actor *domain.User, ticketID string, att *domain.Attachment) (*domain.Attachment, error) {
	ticket, err := s.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if actor.Role == domain.RoleRequester && ticket.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}

	att.ID = uuid.NewString()
	att.TicketID = ticketID
	att.UploaderID = actor.ID
	att.CreatedAt = time.Now().UTC()
	att.Uploader = actor

	if err := s.attachmentRepo.Create(ctx, att); err != nil {
		return nil, err
	}

	realtime.DefaultBroker.BroadcastTo(
		realtime.Audience{
			Roles:   []string{string(domain.RoleAgent), string(domain.RoleAdmin)},
			UserIDs: []string{ticket.RequesterID},
		},
		"attachment_added",
		map[string]string{"ticket_id": ticketID},
	)
	return att, nil
}

func (s *TicketService) GetAttachment(ctx context.Context, actor *domain.User, attachmentID string) (*domain.Attachment, error) {
	att, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return nil, err
	}

	// Verify actor can see the ticket
	ticket, err := s.ticketRepo.FindByID(ctx, att.TicketID)
	if err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleRequester && ticket.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}

	return att, nil
}

func (s *TicketService) Stats(ctx context.Context, actor *domain.User) (*domain.StatsResponse, error) {
	requesterID := ""
	if actor.Role == domain.RoleRequester {
		requesterID = actor.ID
	}
	return s.ticketRepo.Stats(ctx, requesterID)
}

func (s *TicketService) AdminStats(ctx context.Context, actor *domain.User) (*domain.AdminDetailedStats, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}
	return s.ticketRepo.AdminStats(ctx, s.slaDuration.Hours())
}
