package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"backend/internal/domain"
	"backend/internal/handler"
	"backend/internal/service"
)

type TestEnv struct {
	Router        http.Handler
	AuthService   *service.AuthService
	TicketService *service.TicketService
	UserService   *service.UserService
	UserRepo      *MockUserRepo
	TicketRepo    *MockTicketRepo
	CommentRepo   *MockCommentRepo
	AttachRepo    *MockAttachmentRepo
	UploadDir     string
}

func setupTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "helpdesk-upload-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	auth, tickets, users, ur, tr, cr, ar := newHarness()

	authH := handler.NewAuthHandler(auth)
	ticketH := handler.NewTicketHandler(tickets, tempDir)
	userH := handler.NewUserHandler(users)

	r := handler.SetupRouter(handler.RouterConfig{
		JWTSecret:     "test-secret",
		AuthHandler:   authH,
		TicketHandler: ticketH,
		UserHandler:   userH,
	})

	return &TestEnv{
		Router:        r,
		AuthService:   auth,
		TicketService: tickets,
		UserService:   users,
		UserRepo:      ur,
		TicketRepo:    tr,
		CommentRepo:   cr,
		AttachRepo:    ar,
		UploadDir:     tempDir,
	}
}

func (env *TestEnv) registerAndGetToken(t *testing.T, name, email, role string) (string, *domain.User) {
	t.Helper()
	reg, err := env.AuthService.Register(context.Background(), domain.RegisterReq{
		Name:     name,
		Email:    email,
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	u, _ := env.UserRepo.FindByID(context.Background(), reg.User.ID)
	if role != string(domain.RoleRequester) {
		u.Role = domain.UserRole(role)
		_ = env.UserRepo.UpdateRole(context.Background(), u.ID, u.Role)
	}

	// Re-login to get token with correct role in claims
	login, err := env.AuthService.Login(context.Background(), domain.LoginReq{
		Email:    email,
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return login.Token, u
}

// 1. Auth & Me Endpoints: POST /api/auth/register, POST /api/auth/login, GET /api/me
func TestAPI_AuthEndpoints(t *testing.T) {
	env := setupTestEnv(t)

	// Register
	regBody := `{"name":"Alice","email":"alice@example.com","password":"Password123!"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Login
	loginBody := `{"email":"alice@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var authResp struct {
		Data domain.AuthResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &authResp)
	token := authResp.Data.Token
	if token == "" {
		t.Fatal("empty token")
	}

	// GET /api/me
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get /me status expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// 2. Tickets CRUD Endpoints: POST, GET, PATCH, and filtering
func TestAPI_TicketsEndpoints(t *testing.T) {
	env := setupTestEnv(t)
	reqToken, _ := env.registerAndGetToken(t, "Requester 1", "r1@test.id", "requester")
	agentToken, _ := env.registerAndGetToken(t, "Agent 1", "a1@test.id", "agent")
	adminToken, _ := env.registerAndGetToken(t, "Admin 1", "adm1@test.id", "admin")

	// Requester creates ticket
	ticketBody := `{"title":"Network Slow","description":"Takes 5 minutes to load page","category":"Network","priority":"urgent"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tickets", bytes.NewBufferString(ticketBody))
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data domain.Ticket `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	ticketID := createResp.Data.ID

	// Agent tries to create ticket -> 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/tickets", bytes.NewBufferString(ticketBody))
	req.Header.Set("Authorization", "Bearer "+agentToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("agent should get 403 creating ticket, got %d", w.Code)
	}

	// GET /api/tickets list
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/tickets?status=open", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 listing tickets, got %d", w.Code)
	}

	// GET /api/tickets/:id
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/tickets/"+ticketID, nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 get ticket detail, got %d", w.Code)
	}

	// PATCH /api/tickets/:id - Agent picks up ticket (in_progress)
	patchBody := `{"status":"in_progress"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/tickets/"+ticketID, bytes.NewBufferString(patchBody))
	req.Header.Set("Authorization", "Bearer "+agentToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("agent update in_progress expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Admin changes priority
	patchAdmin := `{"priority":"medium"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/tickets/"+ticketID, bytes.NewBufferString(patchAdmin))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin update priority expected 200, got %d", w.Code)
	}
}

// 3. Comments API: POST /api/tickets/:id/comments
func TestAPI_CommentsEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	reqToken, requester := env.registerAndGetToken(t, "Req 2", "r2@test.id", "requester")
	agentToken, _ := env.registerAndGetToken(t, "Agent 2", "a2@test.id", "agent")

	// Create ticket by requester
	ticket, _ := env.TicketService.Create(context.Background(), requester, domain.CreateTicketReq{
		Title: "Comment ticket", Description: "Need reply", Category: "IT Support", Priority: domain.PriorityLow,
	})

	// Add public reply by requester
	commentBody := `{"body":"Here is more info","is_internal":false}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tickets/"+ticket.ID+"/comments", bytes.NewBufferString(commentBody))
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 adding comment, got %d: %s", w.Code, w.Body.String())
	}

	// Requester tries to add internal note -> forbidden
	badComment := `{"body":"Hacked internal note","is_internal":true}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/tickets/"+ticket.ID+"/comments", bytes.NewBufferString(badComment))
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("requester internal note should be 403, got %d", w.Code)
	}

	// Agent adds internal note -> OK
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/tickets/"+ticket.ID+"/comments", bytes.NewBufferString(badComment))
	req.Header.Set("Authorization", "Bearer "+agentToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("agent internal note expected 201, got %d", w.Code)
	}
}

// 4. Attachments API: POST /api/tickets/:id/attachments, GET /api/attachments/:id/download
func TestAPI_AttachmentsUploadAndDownload(t *testing.T) {
	env := setupTestEnv(t)
	reqToken, requester := env.registerAndGetToken(t, "Attach User", "att@test.id", "requester")

	ticket, _ := env.TicketService.Create(context.Background(), requester, domain.CreateTicketReq{
		Title: "Attach ticket", Description: "See file", Category: "Software", Priority: domain.PriorityMedium,
	})

	// Prepare multipart form file
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, _ := mw.CreateFormFile("file", "test_report.pdf")
	part.Write([]byte("%PDF-1.4 dummy pdf content for testing"))
	mw.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/tickets/"+ticket.ID+"/attachments", &b)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("attachment upload expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var uploadResp struct {
		Data domain.Attachment `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &uploadResp)
	attID := uploadResp.Data.ID
	if attID == "" {
		t.Fatal("empty attachment id")
	}

	// Download attachment
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/attachments/"+attID+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("attachment download expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test invalid extension rejection (.exe)
	var b2 bytes.Buffer
	mw2 := multipart.NewWriter(&b2)
	part2, _ := mw2.CreateFormFile("file", "malware.exe")
	part2.Write([]byte("MZ... executable"))
	mw2.Close()

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/tickets/"+ticket.ID+"/attachments", &b2)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	req.Header.Set("Content-Type", mw2.FormDataContentType())
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 rejecting .exe, got %d", w.Code)
	}
}

// 5. Stats & Admin Stats Endpoints: GET /api/stats, GET /api/tickets/stats, GET /api/admin/stats
func TestAPI_StatsEndpoints(t *testing.T) {
	env := setupTestEnv(t)
	reqToken, _ := env.registerAndGetToken(t, "Req Stats", "rqstat@test.id", "requester")
	adminToken, _ := env.registerAndGetToken(t, "Admin Stats", "admstat@test.id", "admin")

	// GET /api/stats
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/stats, got %d", w.Code)
	}

	// GET /api/tickets/stats (alias)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/tickets/stats", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/tickets/stats, got %d", w.Code)
	}

	// Requester accessing /api/admin/stats -> 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("requester should get 403 on /api/admin/stats, got %d", w.Code)
	}

	// Admin accessing /api/admin/stats -> 200 OK
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin expected 200 on /api/admin/stats, got %d: %s", w.Code, w.Body.String())
	}
}

// 6. Users API: GET /api/users, PATCH /api/users/:id/role
func TestAPI_UsersEndpoints(t *testing.T) {
	env := setupTestEnv(t)
	reqToken, targetUser := env.registerAndGetToken(t, "User Target", "target@test.id", "requester")
	adminToken, _ := env.registerAndGetToken(t, "Admin User", "admuser@test.id", "admin")

	// Requester accessing /api/users -> 403
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+reqToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("requester on /api/users should be 403, got %d", w.Code)
	}

	// Admin accessing /api/users -> 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin on /api/users expected 200, got %d", w.Code)
	}

	// Admin updates target user role: requester -> agent
	updateRoleBody := `{"role":"agent"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/users/"+targetUser.ID+"/role", bytes.NewBufferString(updateRoleBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update role expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check updated role in repo
	u, _ := env.UserRepo.FindByID(context.Background(), targetUser.ID)
	if u.Role != domain.RoleAgent {
		t.Fatalf("expected agent role, got %s", u.Role)
	}
}

// 7. SSE Events API: GET /api/events?token=...
func TestAPI_EventsEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	token, _ := env.registerAndGetToken(t, "SSE User", "sse@test.id", "requester")

	// Request with context timeout to close stream gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "/api/events?token="+token, nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		env.Router.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("SSE handler timed out without exiting")
	}

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", w.Header().Get("Content-Type"))
	}
}
