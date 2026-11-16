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
		switch t.Status {
		case domain.StatusOpen:
			stats.OpenTickets++
		case domain.StatusInProgress:
			stats.InProgressTickets++
		case domain.StatusResolved:
			stats.ResolvedTickets++
		}
		if t.Priority == domain.PriorityUrgent && t.Status != domain.StatusClosed {
			stats.UrgentTickets++
		}
	}
	return &stats, nil
}

type MockCommentRepo struct {
	mu       sync.Mutex
	comments []*domain.Comment
}

func NewMockCommentRepo() *MockCommentRepo {
	return &MockCommentRepo{}
}

func (m *MockCommentRepo) Create(ctx context.Context, c *domain.Comment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.comments = append(m.comments, c)
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
		res = append(res, c)
	}
	return res, nil
}
