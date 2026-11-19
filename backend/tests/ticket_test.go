package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/domain"
	"backend/internal/service"
)

func newHarness() (*service.AuthService, *service.TicketService, *service.UserService, *MockUserRepo, *MockTicketRepo, *MockCommentRepo, *MockAttachmentRepo) {
	ur := NewMockUserRepo()
	tr := NewMockTicketRepo()
	cr := NewMockCommentRepo()
	ar := NewMockAttachmentRepo()
	auth := service.NewAuthService(ur, "test-secret", time.Hour)
	tickets := service.NewTicketService(tr, cr, ur, ar, 2)
	users := service.NewUserService(ur)
	return auth, tickets, users, ur, tr, cr, ar
}

func seedUsers(t *testing.T, ur *MockUserRepo) (requester *domain.User, agent *domain.User, admin *domain.User) {
	t.Helper()
	ctx := context.Background()
	requester = &domain.User{ID: "u-req", Name: "Requester", Email: "req@test.id", Role: domain.RoleRequester, CreatedAt: time.Now()}
	agent = &domain.User{ID: "u-agent", Name: "Agent", Email: "agent@test.id", Role: domain.RoleAgent, CreatedAt: time.Now()}
	admin = &domain.User{ID: "u-admin", Name: "Admin", Email: "admin@test.id", Role: domain.RoleAdmin, CreatedAt: time.Now()}
	for _, u := range []*domain.User{requester, agent, admin} {
		if err := ur.Create(ctx, u); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	return
}

func TestAuthRegisterAndLogin(t *testing.T) {
	auth, _, _, _, _, _, _ := newHarness()
	ctx := context.Background()

	reg, err := auth.Register(ctx, domain.RegisterReq{
		Name:     "New Requester",
		Email:    "new@test.id",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if reg.Token == "" {
		t.Fatal("expected token from register")
	}
	if reg.User.Role != domain.RoleRequester {
		t.Fatalf("expected requester role, got %s", reg.User.Role)
	}

	login, err := auth.Login(ctx, domain.LoginReq{Email: "new@test.id", Password: "Password123!"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if login.Token == "" {
		t.Fatal("expected token from login")
	}

	if _, err := auth.Login(ctx, domain.LoginReq{Email: "new@test.id", Password: "wrongpass"}); err == nil {
		t.Fatal("expected login failure for wrong password")
	}
}

func TestTicketIsolation(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, _, _ := seedUsers(t, ur)

	other := &domain.User{ID: "u-other", Name: "Other", Email: "other@test.id", Role: domain.RoleRequester, CreatedAt: time.Now()}
	if err := ur.Create(ctx, other); err != nil {
		t.Fatalf("seed other requester: %v", err)
	}

	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Isolated ticket",
		Description: "Only owner may read",
		Category:    "IT Support",
		Priority:    domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := tickets.Get(ctx, other, ticket.ID); err == nil {
		t.Fatal("expected isolation failure when other requester reads ticket")
	}

	if _, err := tickets.Get(ctx, requester, ticket.ID); err != nil {
		t.Fatalf("owner should read own ticket: %v", err)
	}
}

func TestValidStatusTransitions(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, agent, _ := seedUsers(t, ur)

	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Valid flow",
		Description: "open to resolved",
		Category:    "Network",
		Priority:    domain.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	inProgress := domain.StatusInProgress
	updated, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &inProgress})
	if err != nil {
		t.Fatalf("open -> in_progress should succeed: %v", err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", updated.Status)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != agent.ID {
		t.Fatalf("expected auto-assign to agent %s, got %v", agent.ID, updated.AssigneeID)
	}

	resolved := domain.StatusResolved
	updated, err = tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &resolved})
	if err != nil {
		t.Fatalf("in_progress -> resolved should succeed: %v", err)
	}
	if updated.Status != domain.StatusResolved {
		t.Fatalf("expected resolved, got %s", updated.Status)
	}

	closed := domain.StatusClosed
	updated, err = tickets.Update(ctx, requester, ticket.ID, domain.UpdateTicketReq{Status: &closed})
	if err != nil {
		t.Fatalf("resolved -> closed by requester should succeed: %v", err)
	}
	if updated.Status != domain.StatusClosed {
		t.Fatalf("expected closed, got %s", updated.Status)
	}
}

func TestInvalidStatusTransitions(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, agent, _ := seedUsers(t, ur)

	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Invalid flow",
		Description: "jump status should fail",
		Category:    "Software",
		Priority:    domain.PriorityLow,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	resolved := domain.StatusResolved
	if _, err := tickets.Update(ctx, requester, ticket.ID, domain.UpdateTicketReq{Status: &resolved}); err == nil {
		t.Fatal("requester open -> resolved should be rejected")
	}

	inProgress := domain.StatusInProgress
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &inProgress}); err != nil {
		t.Fatalf("agent open -> in_progress should succeed: %v", err)
	}
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &resolved}); err != nil {
		t.Fatalf("agent in_progress -> resolved should succeed: %v", err)
	}

	open := domain.StatusOpen
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &open}); err == nil {
		t.Fatal("agent reopen resolved -> open should be rejected")
	}
}

func TestInternalNoteSecurity(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, agent, _ := seedUsers(t, ur)

	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Secret notes",
		Description: "internal comment should be hidden",
		Category:    "Hardware",
		Priority:    domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := tickets.AddComment(ctx, requester, ticket.ID, domain.CreateCommentReq{
		Body:       "requester cannot create internal note",
		IsInternal: true,
	}); err == nil {
		t.Fatal("requester internal comment should be rejected")
	}

	if _, err := tickets.AddComment(ctx, agent, ticket.ID, domain.CreateCommentReq{
		Body:       "agent internal note",
		IsInternal: true,
	}); err != nil {
		t.Fatalf("agent internal comment should succeed: %v", err)
	}

	detail, err := tickets.Get(ctx, requester, ticket.ID)
	if err != nil {
		t.Fatalf("get ticket as requester: %v", err)
	}
	for _, c := range detail.Comments {
		if c.IsInternal {
			t.Fatal("internal comment leaked to requester")
		}
	}

	agentDetail, err := tickets.Get(ctx, agent, ticket.ID)
	if err != nil {
		t.Fatalf("get ticket as agent: %v", err)
	}
	if len(agentDetail.Comments) == 0 {
		t.Fatal("agent should see internal comments")
	}
}

func TestAgentCannotCreateTicket(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	_, agent, _ := seedUsers(t, ur)

	_, err := tickets.Create(ctx, agent, domain.CreateTicketReq{
		Title:       "Agent ticket",
		Description: "Should be forbidden",
		Category:    "Hardware",
		Priority:    domain.PriorityLow,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAgentCannotChangePriorityOrAssignee(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, agent, _ := seedUsers(t, ur)

	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Locked fields",
		Description: "agent cannot change assignee",
		Category:    "Hardware",
		Priority:    domain.PriorityLow,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Agent tidak di-assign: prioritas dilarang
	urgent := domain.PriorityUrgent
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Priority: &urgent}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("agent priority change on unassigned ticket should be forbidden, got %v", err)
	}

	// Agent memproses tiket: open -> in_progress (auto-assign ke agent)
	inProgress := domain.StatusInProgress
	updated, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Status: &inProgress})
	if err != nil {
		t.Fatalf("agent open -> in_progress should succeed: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != agent.ID {
		t.Fatalf("expected auto-assigned to agent, got %v", updated.AssigneeID)
	}

	// Agent tidak boleh mengubah assignee (misal ke user lain atau unassigned)
	unassigned := "unassigned"
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{AssigneeID: &unassigned}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("agent assignee change should be forbidden, got %v", err)
	}

	// Agent yang sedang mengerjakan tiket BOLEH ubah prioritas
	updated, err = tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{Priority: &urgent})
	if err != nil {
		t.Fatalf("assigned agent should be allowed to change priority: %v", err)
	}
	if updated.Priority != domain.PriorityUrgent {
		t.Fatalf("expected priority urgent, got %s", updated.Priority)
	}
}

func TestAdminCanChangeRequester(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	reqA, agent, admin := seedUsers(t, ur)

	// Create second requester
	reqB := &domain.User{
		ID:        "user-req-b",
		Name:      "Requester B",
		Email:     "reqb@test.id",
		Role:      domain.RoleRequester,
		CreatedAt: time.Now().UTC(),
	}
	_ = ur.Create(ctx, reqB)

	// Requester A creates ticket
	ticket, err := tickets.Create(ctx, reqA, domain.CreateTicketReq{
		Title:       "Issue by A",
		Description: "Details",
		Category:    "Software",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if ticket.RequesterID != reqA.ID {
		t.Fatalf("expected requester A, got %s", ticket.RequesterID)
	}

	// Requester B tries to change requester -> forbidden
	if _, err := tickets.Update(ctx, reqB, ticket.ID, domain.UpdateTicketReq{RequesterID: &reqB.ID}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("requester changing requester should be forbidden, got %v", err)
	}

	// Agent tries to change requester -> forbidden
	if _, err := tickets.Update(ctx, agent, ticket.ID, domain.UpdateTicketReq{RequesterID: &reqB.ID}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("agent changing requester should be forbidden, got %v", err)
	}

	// Admin changes requester to Requester B -> allowed
	updated, err := tickets.Update(ctx, admin, ticket.ID, domain.UpdateTicketReq{RequesterID: &reqB.ID})
	if err != nil {
		t.Fatalf("admin changing requester failed: %v", err)
	}
	if updated.RequesterID != reqB.ID {
		t.Fatalf("expected updated requester B, got %s", updated.RequesterID)
	}
}

func TestSLAOverdueCalculation(t *testing.T) {
	_, tickets, _, ur, tr, _, _ := newHarness()
	ctx := context.Background()
	requester, _, _ := seedUsers(t, ur)

	// Create urgent ticket
	ticket, err := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Server down",
		Description: "Production outage",
		Category:    "Network",
		Priority:    domain.PriorityUrgent,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Backdate ticket to 3 hours ago (SLA is 2 hours)
	tr.mu.Lock()
	tr.tickets[ticket.ID].CreatedAt = time.Now().Add(-3 * time.Hour)
	tr.mu.Unlock()

	detail, err := tickets.Get(ctx, requester, ticket.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if !detail.IsOverdue {
		t.Fatal("expected ticket to be marked overdue")
	}
}

func TestAttachmentUploadAndAccess(t *testing.T) {
	_, tickets, _, ur, _, _, _ := newHarness()
	ctx := context.Background()
	requester, _, _ := seedUsers(t, ur)

	other := &domain.User{ID: "u-other", Name: "Other", Email: "other@test.id", Role: domain.RoleRequester, CreatedAt: time.Now()}
	_ = ur.Create(ctx, other)

	ticket, _ := tickets.Create(ctx, requester, domain.CreateTicketReq{
		Title:       "Attachment test",
		Description: "Has screenshot",
		Category:    "Software",
		Priority:    domain.PriorityLow,
	})

	att, err := tickets.AddAttachment(ctx, requester, ticket.ID, &domain.Attachment{
		Filename:    "screenshot.png",
		FilePath:    "uploads/dummy.png",
		FileSize:    1024,
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("add attachment: %v", err)
	}

	// Other requester cannot download attachment
	if _, err := tickets.GetAttachment(ctx, other, att.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for other requester, got %v", err)
	}

	// Owner can get attachment
	if got, err := tickets.GetAttachment(ctx, requester, att.ID); err != nil || got.ID != att.ID {
		t.Fatalf("owner should get attachment: %v", err)
	}
}
