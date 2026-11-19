package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"backend/internal/domain"
	"backend/internal/realtime"
	"backend/internal/service"
)

var discussionMutations = []string{"comment_create", "comment_update", "comment_delete", "file_create", "file_replace", "file_delete"}

func requestDiscussionMutation(t *testing.T, env *TestEnv, actor *domain.User, ticketID, commentID, fileID, operation string, want int) *httptest.ResponseRecorder {
	t.Helper()
	switch operation {
	case "comment_create":
		return contentRequest(t, env, actor, "POST", "/tickets/"+ticketID+"/comments", `{"body":"New comment"}`, want)
	case "comment_update":
		return contentRequest(t, env, actor, "PATCH", "/comments/"+commentID, `{"body":"Edited"}`, want)
	case "comment_delete":
		return contentRequest(t, env, actor, "DELETE", "/comments/"+commentID, "", want)
	case "file_create":
		return contentUpload(t, env, actor, "POST", "/comments/"+commentID+"/attachments", "new.txt", []byte("new"), want)
	case "file_replace":
		return contentUpload(t, env, actor, "PATCH", "/attachments/"+fileID, "new.txt", []byte("new"), want)
	case "file_delete":
		return contentRequest(t, env, actor, "DELETE", "/attachments/"+fileID, "", want)
	default:
		t.Fatalf("unknown discussion mutation: %s", operation)
		return nil
	}
}

func assertTicketClosedResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusConflict || body.Error != domain.ErrTicketClosed.Error() {
		t.Fatalf("expected distinct Closed-discussion conflict: %d %s %v", w.Code, w.Body.String(), err)
	}
}

func TestAPI_ClosedDiscussionIsReadOnlyForEveryRoleAndPrivacy(t *testing.T) {
	for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
		for _, internal := range []bool{false, true} {
			if internal && role == domain.RoleRequester {
				continue
			}
			name := string(role) + "/public"
			if internal {
				name = string(role) + "/internal"
			}
			t.Run(name, func(t *testing.T) {
				f := newContentFixture(t)
				ctx := context.Background()
				author := map[domain.UserRole]*domain.User{domain.RoleRequester: f.owner, domain.RoleAgent: f.agent, domain.RoleAdmin: f.admin}[role]
				comment := f.comment(t, author, internal)
				root := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "root.txt", []byte("root"), http.StatusCreated))
				file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, author, "POST", "/comments/"+comment.ID+"/attachments", "original.txt", []byte("original"), http.StatusCreated))
				beforeComment, err := f.CommentRepo.FindByID(ctx, comment.ID)
				if err != nil {
					t.Fatal(err)
				}
				f.ticket.Status = domain.StatusClosed
				if err := f.TicketRepo.Update(ctx, f.ticket); err != nil {
					t.Fatal(err)
				}
				events, unsubscribe := realtime.DefaultBroker.Subscribe("agent", "closed-discussion-observer")
				defer unsubscribe()
				for _, actor := range []*domain.User{nil, f.owner, f.agent, f.admin, f.otherAgent, f.outsider} {
					for _, operation := range discussionMutations {
						want := http.StatusForbidden
						if actor == nil {
							want = http.StatusUnauthorized
						} else if actor == author || (operation == "comment_create" && actor != f.outsider) {
							want = http.StatusConflict
						}
						w := requestDiscussionMutation(t, f.TestEnv, actor, f.ticket.ID, comment.ID, file.ID, operation, want)
						if want == http.StatusConflict {
							assertTicketClosedResponse(t, w)
						}
					}
				}
				// Upload preflight must win even when no multipart file was supplied.
				assertTicketClosedResponse(t, contentRequest(t, f.TestEnv, author, "POST", "/comments/"+comment.ID+"/attachments", "", http.StatusConflict))
				assertTicketClosedResponse(t, contentRequest(t, f.TestEnv, author, "PATCH", "/attachments/"+file.ID, "", http.StatusConflict))
				// Requesters still cannot create internal notes, even on Closed tickets.
				contentRequest(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/comments", `{"body":"Private","is_internal":true}`, http.StatusForbidden)
				for _, before := range []*domain.Attachment{&root, &file} {
					after, err := f.AttachRepo.FindByID(ctx, before.ID)
					if err != nil || !reflect.DeepEqual(before, after) {
						t.Fatalf("Closed mutation changed file metadata: %+v %v", after, err)
					}
				}
				afterComment, err := f.CommentRepo.FindByID(ctx, comment.ID)
				if err != nil || !reflect.DeepEqual(beforeComment, afterComment) {
					t.Fatalf("Closed mutation changed comment: %+v %v", afterComment, err)
				}
				comments, err := f.CommentRepo.ListByTicket(ctx, f.ticket.ID, true)
				if err != nil || len(comments) != 1 {
					t.Fatalf("Closed mutation added a comment: %+v %v", comments, err)
				}
				assertFileContent(t, file.FilePath, "original")
				assertFileContent(t, root.FilePath, "root")
				assertContentFiles(t, f.TestEnv, 2)
				if len(events) != 0 {
					t.Fatal("denied Closed-discussion mutations emitted events")
				}
				contentRequest(t, f.TestEnv, author, "GET", "/tickets/"+f.ticket.ID, "", http.StatusOK)
				contentRequest(t, f.TestEnv, author, "GET", "/attachments/"+file.ID+"/preview", "", http.StatusOK)
				contentRequest(t, f.TestEnv, author, "GET", "/attachments/"+file.ID+"/download", "", http.StatusOK)
				// Keep status-transition errors separate from discussion conflicts.
				contentRequest(t, f.TestEnv, f.owner, "PATCH", "/tickets/"+f.ticket.ID, `{"status":"open"}`, http.StatusUnprocessableEntity)
			})
		}
	}
}

func TestService_DiscussionStatusPreflightBeforeRepositoryWrites(t *testing.T) {
	for _, status := range []domain.TicketStatus{domain.StatusOpen, domain.StatusInProgress, domain.StatusResolved, domain.StatusClosed} {
		for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
			for _, internal := range []bool{false, true} {
				if internal && role == domain.RoleRequester {
					continue
				}
				f := newContentFixture(t)
				ctx := context.Background()
				author := map[domain.UserRole]*domain.User{domain.RoleRequester: f.owner, domain.RoleAgent: f.agent, domain.RoleAdmin: f.admin}[role]
				comment := f.comment(t, author, internal)
				file, err := f.TicketService.AddCommentAttachment(ctx, author, comment.ID, &domain.Attachment{Filename: "original.txt"})
				if err != nil {
					t.Fatal(err)
				}
				f.ticket.Status = status
				if err := f.TicketRepo.Update(ctx, f.ticket); err != nil {
					t.Fatal(err)
				}
				// Stop at the repository boundary, independently of mock status checks.
				reached := errors.New("repository write reached")
				writes := 0
				before := func() error { writes++; return reached }
				s := service.NewTicketService(f.TicketRepo, reassignBeforeCommentWrite{f.CommentRepo, before}, f.UserRepo, reassignBeforeAttachmentWrite{f.AttachRepo, before}, 2)
				operations := map[string]func() error{
					"comment_create": func() error {
						_, err := s.AddComment(ctx, author, f.ticket.ID, domain.CreateCommentReq{Body: "New", IsInternal: internal})
						return err
					},
					"comment_update": func() error {
						_, err := s.UpdateComment(ctx, author, comment.ID, domain.UpdateCommentReq{Body: "Edit"})
						return err
					},
					"comment_delete": func() error { _, err := s.DeleteComment(ctx, author, comment.ID); return err },
					"file_create": func() error {
						_, err := s.AddCommentAttachment(ctx, author, comment.ID, &domain.Attachment{})
						return err
					},
					"file_replace": func() error { _, _, err := s.ReplaceAttachment(ctx, author, file.ID, &domain.Attachment{}); return err },
					"file_delete":  func() error { _, err := s.DeleteAttachment(ctx, author, file.ID); return err },
				}
				for name, operation := range operations {
					want := reached
					if status == domain.StatusClosed {
						want = domain.ErrTicketClosed
					}
					if err := operation(); !errors.Is(err, want) {
						t.Fatalf("%s/%s/internal=%v/%s: got %v, want %v", status, role, internal, name, err, want)
					}
				}
				wantWrites := len(operations)
				if status == domain.StatusClosed {
					wantWrites = 0
				}
				if writes != wantWrites {
					t.Fatalf("%s: repository writes %d, want %d", status, writes, wantWrites)
				}
				for _, err := range []error{s.CheckCommentAttachmentUpload(ctx, author, comment.ID), s.CheckAttachmentUpdate(ctx, author, file.ID)} {
					if status == domain.StatusClosed && !errors.Is(err, domain.ErrTicketClosed) || status != domain.StatusClosed && err != nil {
						t.Fatalf("%s upload preflight: %v", status, err)
					}
				}
			}
		}
	}
}
