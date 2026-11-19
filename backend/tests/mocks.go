package tests

import (
	"context"
	"sync"

	"backend/internal/domain"
)

type MockUserRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users: make(map[string]*domain.User),
	}
}

func (m *MockUserRepo) Create(ctx context.Context, u *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	return nil
}

func (m *MockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *MockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockUserRepo) List(ctx context.Context, role string) ([]*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.User
	for _, u := range m.users {
		if role == "" || string(u.Role) == role {
			res = append(res, u)
		}
	}
	return res, nil
}

func (m *MockUserRepo) CountByRole(ctx context.Context, role domain.UserRole) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, u := range m.users {
		if u.Role == role {
			count++
		}
	}
	return count, nil
}

func (m *MockUserRepo) UpdateRole(ctx context.Context, id string, role domain.UserRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Role = role
	return nil
}

type MockTicketRepo struct {
	mu      sync.Mutex
	tickets map[string]*domain.Ticket
}

func NewMockTicketRepo() *MockTicketRepo {
	return &MockTicketRepo{
		tickets: make(map[string]*domain.Ticket),
	}
}

func (m *MockTicketRepo) Create(ctx context.Context, t *domain.Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[t.ID] = t
	return nil
}

func (m *MockTicketRepo) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	// return copy
	cp := *t
	return &cp, nil
}

func (m *MockTicketRepo) List(ctx context.Context, filter domain.TicketFilter) ([]*domain.Ticket, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.Ticket
	for _, t := range m.tickets {
		if filter.RequesterID != "" && t.RequesterID != filter.RequesterID {
			continue
		}
		if filter.Status != "" && string(t.Status) != filter.Status {
			continue
		}
		cp := *t
		res = append(res, &cp)
	}
	return res, int64(len(res)), nil
}

func (m *MockTicketRepo) Update(ctx context.Context, t *domain.Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tickets[t.ID]; !ok {
		return domain.ErrNotFound
	}
	m.tickets[t.ID] = t
	return nil
}

func (m *MockTicketRepo) Stats(ctx context.Context, requesterID string) (*domain.StatsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var stats domain.StatsResponse
	for _, t := range m.tickets {
		if requesterID != "" && t.RequesterID != requesterID {
			continue
		}
		stats.TotalTickets++
		switch t.Status {
		case domain.StatusOpen:
			stats.OpenTickets++
		case domain.StatusInProgress:
			stats.InProgressTickets++
		case domain.StatusResolved:
			stats.ResolvedTickets++
		case domain.StatusClosed:
			stats.ClosedTickets++
		}
		if t.Priority == domain.PriorityUrgent && t.Status != domain.StatusClosed {
			stats.UrgentTickets++
		}
	}
	return &stats, nil
}

func (m *MockTicketRepo) AdminStats(ctx context.Context, slaHours float64) (*domain.AdminDetailedStats, error) {
	stats, err := m.Stats(ctx, "")
	if err != nil {
		return nil, err
	}
	return &domain.AdminDetailedStats{
		StatsResponse: *stats,
		Categories:    []domain.CategoryCount{},
		Priorities:    []domain.PriorityCount{},
		Agents:        []domain.AgentWorkload{},
	}, nil
}

// lockDiscussion mirrors the repository ticket-row lock for discussion writes.
func (m *MockTicketRepo) lockDiscussion(ticketID string) (func(), error) {
	m.mu.Lock()
	ticket, ok := m.tickets[ticketID]
	if !ok {
		m.mu.Unlock()
		return nil, domain.ErrNotFound
	}
	if ticket.Status == domain.StatusClosed {
		m.mu.Unlock()
		return nil, domain.ErrTicketClosed
	}
	return m.mu.Unlock, nil
}

type MockCommentRepo struct {
	mu          sync.Mutex
	tickets     *MockTicketRepo
	comments    []*domain.Comment
	attachments *MockAttachmentRepo
	CreateErr   error
	UpdateErr   error
	DeleteErr   error
}

func NewMockCommentRepo() *MockCommentRepo {
	return &MockCommentRepo{}
}

func (m *MockCommentRepo) Create(ctx context.Context, actor *domain.User, c *domain.Comment) error {
	unlock, err := m.tickets.lockDiscussion(c.TicketID)
	if err != nil {
		return err
	}
	defer unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateErr != nil {
		return m.CreateErr
	}
	cp := *c
	m.comments = append(m.comments, &cp)
	return nil
}

func (m *MockCommentRepo) ListByTicket(ctx context.Context, ticketID string, includeInternal bool) ([]*domain.Comment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.Comment
	for _, c := range m.comments {
		if c.TicketID != ticketID {
			continue
		}
		if !includeInternal && c.IsInternal {
			continue
		}
		cp := *c
		res = append(res, &cp)
	}
	return res, nil
}

func (m *MockCommentRepo) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.comments {
		if c.ID == id {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockCommentRepo) Update(ctx context.Context, actor *domain.User, c *domain.Comment) error {
	stored, err := m.FindByID(ctx, c.ID)
	if err != nil {
		return err
	}
	unlock, err := m.tickets.lockDiscussion(stored.TicketID)
	if err != nil {
		return err
	}
	defer unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	for i, existing := range m.comments {
		if existing.ID == c.ID {
			cp := *existing
			cp.Body, cp.UpdatedAt = c.Body, c.UpdatedAt
			m.comments[i] = &cp
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *MockCommentRepo) Delete(ctx context.Context, actor *domain.User, id string) ([]*domain.Attachment, error) {
	stored, err := m.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	unlock, err := m.tickets.lockDiscussion(stored.TicketID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return nil, m.DeleteErr
	}
	for i, c := range m.comments {
		if c.ID != id {
			continue
		}
		var files []*domain.Attachment
		if m.attachments != nil {
			m.attachments.mu.Lock()
			kept := make([]*domain.Attachment, 0)
			for _, a := range m.attachments.attachments {
				if a.CommentID != nil && *a.CommentID == id {
					cp := *a
					files = append(files, &cp)
				} else {
					kept = append(kept, a)
				}
			}
			m.attachments.attachments = kept
			m.attachments.mu.Unlock()
		}
		m.comments = append(m.comments[:i], m.comments[i+1:]...)
		return files, nil
	}
	return nil, domain.ErrNotFound
}

type MockAttachmentRepo struct {
	mu          sync.Mutex
	tickets     *MockTicketRepo
	attachments []*domain.Attachment
	comments    *MockCommentRepo
	CreateErr   error
	ReplaceErr  error
	DeleteErr   error
}

func NewMockAttachmentRepo() *MockAttachmentRepo {
	return &MockAttachmentRepo{}
}

func (m *MockAttachmentRepo) lockWrite(att *domain.Attachment) (func(), error) {
	if att.CommentID == nil {
		return func() {}, nil
	}
	return m.tickets.lockDiscussion(att.TicketID)
}

func (m *MockAttachmentRepo) Create(ctx context.Context, actor *domain.User, att *domain.Attachment) error {
	unlock, err := m.lockWrite(att)
	if err != nil {
		return err
	}
	defer unlock()
	if att.CommentID != nil && m.comments != nil {
		m.comments.mu.Lock()
		defer m.comments.mu.Unlock()
		found := false
		for _, c := range m.comments.comments {
			if c.ID == *att.CommentID {
				found = true
			}
		}
		if !found {
			return domain.ErrNotFound
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateErr != nil {
		return m.CreateErr
	}
	cp := *att
	m.attachments = append(m.attachments, &cp)
	return nil
}

func (m *MockAttachmentRepo) ListByTicket(ctx context.Context, ticketID string) ([]*domain.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.Attachment
	for _, a := range m.attachments {
		if a.TicketID == ticketID && a.CommentID == nil {
			cp := *a
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (m *MockAttachmentRepo) ListByComment(ctx context.Context, id string) ([]*domain.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]*domain.Attachment, 0)
	for _, a := range m.attachments {
		if a.CommentID != nil && *a.CommentID == id {
			cp := *a
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (m *MockAttachmentRepo) Replace(ctx context.Context, actor *domain.User, att *domain.Attachment) (string, error) {
	stored, err := m.FindByID(ctx, att.ID)
	if err != nil {
		return "", err
	}
	unlock, err := m.lockWrite(stored)
	if err != nil {
		return "", err
	}
	defer unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ReplaceErr != nil {
		return "", m.ReplaceErr
	}
	for i, a := range m.attachments {
		if a.ID == att.ID {
			cp := *att
			m.attachments[i] = &cp
			return a.FilePath, nil
		}
	}
	return "", domain.ErrNotFound
}

func (m *MockAttachmentRepo) Delete(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, error) {
	stored, err := m.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	unlock, err := m.lockWrite(stored)
	if err != nil {
		return nil, err
	}
	defer unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return nil, m.DeleteErr
	}
	for i, a := range m.attachments {
		if a.ID == id {
			cp := *a
			m.attachments = append(m.attachments[:i], m.attachments[i+1:]...)
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockAttachmentRepo) FindByID(ctx context.Context, id string) (*domain.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.attachments {
		if a.ID == id {
			cp := *a
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}
