package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"backend/internal/domain"
	"backend/internal/realtime"
)

type contentFixture struct {
	*TestEnv
	owner, agent, admin, outsider, otherAgent *domain.User
	ticket                                    *domain.Ticket
}

func newContentFixture(t *testing.T) *contentFixture {
	t.Helper()
	env := setupTestEnv(t)
	owner, agent, admin := seedUsers(t, env.UserRepo)
	f := &contentFixture{TestEnv: env, owner: owner, agent: agent, admin: admin,
		outsider:   &domain.User{ID: "outsider", Role: domain.RoleRequester},
		otherAgent: &domain.User{ID: "other-agent", Role: domain.RoleAgent}}
	var err error
	f.ticket, err = env.TicketService.Create(context.Background(), owner, domain.CreateTicketReq{Title: "Content", Description: "Files and comments", Category: "Software"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func contentToken(t *testing.T, user *domain.User) string {
	t.Helper()
	if user == nil {
		return ""
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": user.ID, "role": string(user.Role), "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func contentRequest(t *testing.T, env *TestEnv, user *domain.User, method, path, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api"+path, strings.NewReader(body))
	if user != nil {
		req.Header.Set("Authorization", "Bearer "+contentToken(t, user))
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: want %d, got %d: %s", method, path, want, w.Code, w.Body.String())
	}
	return w
}

func contentUpload(t *testing.T, env *TestEnv, user *domain.User, method, path, filename string, data []byte, want int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, "/api"+path, &body)
	if user != nil {
		req.Header.Set("Authorization", "Bearer "+contentToken(t, user))
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: want %d, got %d: %s", method, path, want, w.Code, w.Body.String())
	}
	return w
}

func contentData[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var response struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data
}

func assertContentFiles(t *testing.T, env *TestEnv, want int) {
	t.Helper()
	entries, err := os.ReadDir(env.UploadDir)
	if err != nil || len(entries) != want {
		t.Fatalf("upload files: want %d, got %d, err %v", want, len(entries), err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("file %s: want %q, got %q, err %v", path, want, data, err)
	}
}

func assertFileRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file %s still exists or unexpected error: %v", path, err)
	}
}

func (f *contentFixture) comment(t *testing.T, author *domain.User, internal bool) *domain.Comment {
	t.Helper()
	c, err := f.TicketService.AddComment(context.Background(), author, f.ticket.ID, domain.CreateCommentReq{Body: "Original comment", IsInternal: internal})
	if err != nil {
		t.Fatal(err)
	}
	if c.Attachments == nil || c.UpdatedAt != nil {
		t.Fatalf("new comment must have attachments [] and no edited timestamp: %+v", c)
	}
	return c
}

func TestAPI_RootAttachmentsOwnerOnlyAllStatuses(t *testing.T) {
	for _, status := range []domain.TicketStatus{domain.StatusOpen, domain.StatusInProgress, domain.StatusResolved, domain.StatusClosed} {
		t.Run(string(status), func(t *testing.T) {
			f := newContentFixture(t)
			f.ticket.Status = status
			if err := f.TicketRepo.Update(context.Background(), f.ticket); err != nil {
				t.Fatal(err)
			}
			legacy := &domain.Attachment{ID: "legacy", TicketID: f.ticket.ID, UploaderID: f.agent.ID, Filename: "legacy.txt", FilePath: filepath.Join(f.UploadDir, "legacy.txt"), FileSize: 3, ContentType: "text/plain", CreatedAt: time.Now().UTC()}
			if err := os.WriteFile(legacy.FilePath, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := f.AttachRepo.Create(context.Background(), f.owner, legacy); err != nil {
				t.Fatal(err)
			}
			for _, actor := range []*domain.User{nil, f.agent, f.admin, f.outsider, f.otherAgent} {
				want := http.StatusForbidden
				if actor == nil {
					want = http.StatusUnauthorized
				}
				contentUpload(t, f.TestEnv, actor, "POST", "/tickets/"+f.ticket.ID+"/attachments", "new.txt", []byte("new"), want)
				contentUpload(t, f.TestEnv, actor, "PATCH", "/attachments/legacy", "new.txt", []byte("new"), want)
				contentRequest(t, f.TestEnv, actor, "DELETE", "/attachments/legacy", "", want)
				got, err := f.AttachRepo.FindByID(context.Background(), legacy.ID)
				if err != nil || !reflect.DeepEqual(got, legacy) {
					t.Fatalf("denied request changed metadata: %+v, %v", got, err)
				}
				assertContentFiles(t, f.TestEnv, 1)
				assertFileContent(t, legacy.FilePath, "old")
			}
			updated := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "PATCH", "/attachments/legacy", "replacement.txt", []byte("new"), http.StatusOK))
			if updated.ID != legacy.ID || updated.CommentID != nil || !updated.CreatedAt.Equal(legacy.CreatedAt) {
				t.Fatalf("replacement changed identity: %+v", updated)
			}
			assertFileRemoved(t, legacy.FilePath)
			assertFileContent(t, updated.FilePath, "new")
			contentRequest(t, f.TestEnv, f.owner, "DELETE", "/attachments/legacy", "", http.StatusNoContent)
			assertFileRemoved(t, updated.FilePath)
			if _, err := f.AttachRepo.FindByID(context.Background(), legacy.ID); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("metadata not deleted: %v", err)
			}
			created := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "owner.txt", []byte("owner"), http.StatusCreated))
			if created.CommentID != nil || created.UploaderID != f.owner.ID {
				t.Fatalf("unexpected root attachment: %+v", created)
			}
			assertContentFiles(t, f.TestEnv, 1)
		})
	}
}

func TestAPI_CommentAttachmentsPrivacyAndAuthorPermissions(t *testing.T) {
	f := newContentFixture(t)
	root := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "root.txt", []byte("root"), http.StatusCreated))
	public := f.comment(t, f.agent, false)
	private := f.comment(t, f.agent, true)
	publicFile := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+public.ID+"/attachments", "public.txt", []byte("public"), http.StatusCreated))
	privateFile := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+private.ID+"/attachments", "private.txt", []byte("private"), http.StatusCreated))
	if privateFile.CommentID == nil || *privateFile.CommentID != private.ID || privateFile.TicketID != f.ticket.ID {
		t.Fatalf("missing comment link: %+v", privateFile)
	}

	for _, comment := range []*domain.Comment{public, private} {
		for _, actor := range []*domain.User{nil, f.owner, f.admin, f.outsider, f.otherAgent} {
			want := http.StatusForbidden
			if actor == nil {
				want = http.StatusUnauthorized
			}
			contentRequest(t, f.TestEnv, actor, "PATCH", "/comments/"+comment.ID, `{"body":"hijacked","is_internal":false}`, want)
			contentRequest(t, f.TestEnv, actor, "DELETE", "/comments/"+comment.ID, "", want)
			contentUpload(t, f.TestEnv, actor, "POST", "/comments/"+comment.ID+"/attachments", "hijack.txt", []byte("hijack"), want)
			file := publicFile
			if comment.IsInternal {
				file = privateFile
			}
			contentUpload(t, f.TestEnv, actor, "PATCH", "/attachments/"+file.ID, "hijack.txt", []byte("hijack"), want)
			contentRequest(t, f.TestEnv, actor, "DELETE", "/attachments/"+file.ID, "", want)
			got, err := f.CommentRepo.FindByID(context.Background(), comment.ID)
			if err != nil || got.Body != comment.Body || got.IsInternal != comment.IsInternal || got.UpdatedAt != nil {
				t.Fatalf("denied action mutated comment: %+v %v", got, err)
			}
			assertContentFiles(t, f.TestEnv, 3)
			assertFileContent(t, file.FilePath, file.Filename[:len(file.Filename)-4])
		}
	}
	for _, actor := range []*domain.User{f.owner, f.agent, f.admin, f.otherAgent} {
		publicDownload := contentRequest(t, f.TestEnv, actor, "GET", "/attachments/"+publicFile.ID+"/download", "", http.StatusOK)
		if publicDownload.Body.String() != "public" {
			t.Fatal("incorrect public file download")
		}
		want := http.StatusOK
		if actor == f.owner {
			want = http.StatusForbidden
		}
		contentRequest(t, f.TestEnv, actor, "GET", "/attachments/"+privateFile.ID+"/download", "", want)
		detail := contentData[domain.Ticket](t, contentRequest(t, f.TestEnv, actor, "GET", "/tickets/"+f.ticket.ID, "", http.StatusOK))
		if len(detail.Attachments) != 1 || detail.Attachments[0].ID != root.ID {
			t.Fatalf("comment files leaked to root attachments: %+v", detail.Attachments)
		}
		wantComments := 2
		if actor == f.owner {
			wantComments = 1
		}
		if len(detail.Comments) != wantComments {
			t.Fatalf("unexpected comments: %+v", detail.Comments)
		}
		for _, c := range detail.Comments {
			if c.Attachments == nil || len(c.Attachments) != 1 {
				t.Fatalf("missing comment attachments: %+v", c)
			}
			if actor == f.owner && (c.IsInternal || c.Attachments[0].ID == privateFile.ID) {
				t.Fatal("internal file leaked to requester")
			}
		}
	}
	contentRequest(t, f.TestEnv, f.outsider, "GET", "/attachments/"+publicFile.ID+"/download", "", http.StatusForbidden)
	contentRequest(t, f.TestEnv, f.outsider, "GET", "/attachments/"+privateFile.ID+"/download", "", http.StatusForbidden)

	for _, c := range []*domain.Comment{public, private} {
		body := `{"body":"Edited","is_internal":true}`
		if c.IsInternal {
			body = `{"body":"Edited","is_internal":false}`
		}
		updated := contentData[domain.Comment](t, contentRequest(t, f.TestEnv, f.agent, "PATCH", "/comments/"+c.ID, body, http.StatusOK))
		if updated.Body != "Edited" || updated.IsInternal != c.IsInternal || updated.UpdatedAt == nil || updated.UpdatedAt.Before(c.CreatedAt) || len(updated.Attachments) != 1 {
			t.Fatalf("bad comment update: %+v", updated)
		}
	}
	replaced := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "PATCH", "/attachments/"+privateFile.ID, "edited.txt", []byte("edited"), http.StatusOK))
	if replaced.CommentID == nil || *replaced.CommentID != private.ID {
		t.Fatal("replacement lost comment link")
	}
	assertFileRemoved(t, privateFile.FilePath)
	assertFileContent(t, replaced.FilePath, "edited")
	second := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+private.ID+"/attachments", "second.txt", []byte("second"), http.StatusCreated))
	contentRequest(t, f.TestEnv, f.agent, "DELETE", "/comments/"+private.ID, "", http.StatusNoContent)
	assertFileRemoved(t, replaced.FilePath)
	assertFileRemoved(t, second.FilePath)
	assertFileContent(t, root.FilePath, "root")
	assertFileContent(t, publicFile.FilePath, "public")
	for _, id := range []string{privateFile.ID, second.ID} {
		if _, err := f.AttachRepo.FindByID(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("comment attachment metadata survived deletion: %v", err)
		}
		contentRequest(t, f.TestEnv, f.agent, "GET", "/attachments/"+id+"/download", "", http.StatusNotFound)
	}
	contentRequest(t, f.TestEnv, f.agent, "DELETE", "/comments/"+private.ID, "", http.StatusNotFound)
	contentRequest(t, f.TestEnv, f.agent, "DELETE", "/attachments/"+publicFile.ID, "", http.StatusNoContent)
	assertFileRemoved(t, publicFile.FilePath)
	assertContentFiles(t, f.TestEnv, 1)
}

func TestAPI_RequesterCommentAuthorAndRequiredBody(t *testing.T) {
	f := newContentFixture(t)
	comment := f.comment(t, f.owner, false)
	for _, body := range []string{`{}`, `{"body":""}`, `{"body":" \n\t"}`, `{"body":null}`, `{"body":42}`} {
		contentRequest(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/comments", body, http.StatusBadRequest)
		contentRequest(t, f.TestEnv, f.owner, "PATCH", "/comments/"+comment.ID, body, http.StatusBadRequest)
	}
	file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/comments/"+comment.ID+"/attachments", "author.txt", []byte("author"), http.StatusCreated))
	updated := contentData[domain.Comment](t, contentRequest(t, f.TestEnv, f.owner, "PATCH", "/comments/"+comment.ID, `{"body":"My update","is_internal":true,"author_id":"u-admin"}`, http.StatusOK))
	if updated.IsInternal || updated.AuthorID != f.owner.ID || len(updated.Attachments) != 1 || updated.Attachments[0].ID != file.ID {
		t.Fatalf("comment edit changed immutable data: %+v", updated)
	}
	contentRequest(t, f.TestEnv, f.admin, "DELETE", "/comments/"+comment.ID, "", http.StatusForbidden)
	contentRequest(t, f.TestEnv, f.owner, "DELETE", "/comments/"+comment.ID, "", http.StatusNoContent)
	assertFileRemoved(t, file.FilePath)
}

func TestAPI_ContentPersistenceFailuresKeepOldFilesAndMetadata(t *testing.T) {
	f := newContentFixture(t)
	ctx := context.Background()
	failed := errors.New("database unavailable")
	f.AttachRepo.CreateErr = failed
	contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "failed.txt", []byte("failed"), http.StatusInternalServerError)
	comment := f.comment(t, f.agent, true)
	contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+comment.ID+"/attachments", "failed.txt", []byte("failed"), http.StatusInternalServerError)
	assertContentFiles(t, f.TestEnv, 0)
	f.AttachRepo.CreateErr = nil
	file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+comment.ID+"/attachments", "original.txt", []byte("original"), http.StatusCreated))
	f.AttachRepo.ReplaceErr = failed
	contentUpload(t, f.TestEnv, f.agent, "PATCH", "/attachments/"+file.ID, "failed.txt", []byte("failed"), http.StatusInternalServerError)
	got, _ := f.AttachRepo.FindByID(ctx, file.ID)
	if !reflect.DeepEqual(got, &file) {
		t.Fatalf("failed replace changed metadata: %+v", got)
	}
	f.AttachRepo.DeleteErr = failed
	contentRequest(t, f.TestEnv, f.agent, "DELETE", "/attachments/"+file.ID, "", http.StatusInternalServerError)
	f.CommentRepo.UpdateErr = failed
	contentRequest(t, f.TestEnv, f.agent, "PATCH", "/comments/"+comment.ID, `{"body":"failed"}`, http.StatusInternalServerError)
	c, _ := f.CommentRepo.FindByID(ctx, comment.ID)
	if c.Body != comment.Body || c.UpdatedAt != nil {
		t.Fatalf("failed edit changed comment: %+v", c)
	}
	f.CommentRepo.DeleteErr = failed
	contentRequest(t, f.TestEnv, f.agent, "DELETE", "/comments/"+comment.ID, "", http.StatusInternalServerError)
	assertFileContent(t, file.FilePath, "original")
	assertContentFiles(t, f.TestEnv, 1)
	if _, err := f.AttachRepo.FindByID(ctx, file.ID); err != nil {
		t.Fatal("failed delete removed metadata")
	}
}

func TestAPI_CommentUploadAndReplacementValidation(t *testing.T) {
	f := newContentFixture(t)
	comment := f.comment(t, f.agent, false)
	file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+comment.ID+"/attachments", "first.txt", []byte("first"), http.StatusCreated))
	for _, method := range []string{"POST", "PATCH"} {
		path := "/comments/" + comment.ID + "/attachments"
		if method == "PATCH" {
			path = "/attachments/" + file.ID
		}
		for _, tc := range []struct {
			name string
			size int
			want int
		}{
			{"blocked.exe", 5, http.StatusBadRequest},
			{"too_big.txt", (5 << 20) + 1, http.StatusBadRequest},
			{"exact_5mib.TXT", 5 << 20, http.StatusCreated},
		} {
			want := tc.want
			if method == "PATCH" && want == http.StatusCreated {
				want = http.StatusOK
			}
			contentUpload(t, f.TestEnv, f.agent, method, path, tc.name, bytes.Repeat([]byte("a"), tc.size), want)
			if tc.want == http.StatusBadRequest {
				assertFileContent(t, file.FilePath, "first")
			}
		}
	}
	assertFileRemoved(t, file.FilePath)
	assertContentFiles(t, f.TestEnv, 2)
}

func TestAPI_CleanupNeverRemovesOutsideFilesOrDirectories(t *testing.T) {
	f := newContentFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{outside, f.UploadDir, filepath.Join(f.UploadDir, "missing.txt")} {
		a := &domain.Attachment{ID: "unsafe", TicketID: f.ticket.ID, FilePath: path}
		if err := f.AttachRepo.Create(context.Background(), f.owner, a); err != nil {
			t.Fatal(err)
		}
		contentRequest(t, f.TestEnv, f.owner, "DELETE", "/attachments/unsafe", "", http.StatusNoContent)
		if _, err := f.AttachRepo.FindByID(context.Background(), a.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("DB deletion failed: %v", err)
		}
	}
	assertFileContent(t, outside, "safe")
	assertContentFiles(t, f.TestEnv, 0)
}

func TestContentPermissionsFollowCurrentRequesterAndRequireAccess(t *testing.T) {
	for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			f := newContentFixture(t)
			ctx := context.Background()
			newOwner := &domain.User{ID: "new-owner", Role: role}
			if err := f.UserRepo.Create(ctx, newOwner); err != nil {
				t.Fatal(err)
			}
			comment := f.comment(t, f.owner, false)
			legacy, err := f.TicketService.AddAttachment(ctx, f.owner, f.ticket.ID, &domain.Attachment{Filename: "legacy.txt"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.TicketService.Update(ctx, f.admin, f.ticket.ID, domain.UpdateTicketReq{RequesterID: &newOwner.ID}); err != nil {
				t.Fatal(err)
			}
			if _, err := f.TicketService.AddAttachment(ctx, newOwner, f.ticket.ID, &domain.Attachment{}); err != nil {
				t.Fatalf("actual owner cannot upload: %v", err)
			}
			if _, _, err := f.TicketService.ReplaceAttachment(ctx, newOwner, legacy.ID, &domain.Attachment{}); err != nil {
				t.Fatalf("actual owner cannot replace legacy file: %v", err)
			}
			if _, err := f.TicketService.DeleteAttachment(ctx, newOwner, legacy.ID); err != nil {
				t.Fatalf("actual owner cannot delete legacy file: %v", err)
			}
			if _, err := f.TicketService.AddAttachment(ctx, f.owner, f.ticket.ID, &domain.Attachment{}); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("former owner upload: %v", err)
			}
			if _, err := f.TicketService.UpdateComment(ctx, f.owner, comment.ID, domain.UpdateCommentReq{Body: "lost access"}); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("author without ticket access edited comment: %v", err)
			}
			if _, err := f.TicketService.AddCommentAttachment(ctx, f.owner, comment.ID, &domain.Attachment{}); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("author without ticket access uploaded: %v", err)
			}
			if _, err := f.TicketService.DeleteComment(ctx, f.owner, comment.ID); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("author without ticket access deleted: %v", err)
			}
		})
	}
	f := newContentFixture(t)
	private := f.comment(t, f.agent, true)
	f.agent.Role = domain.RoleRequester
	f.ticket.RequesterID = f.agent.ID
	if err := f.TicketRepo.Update(context.Background(), f.ticket); err != nil {
		t.Fatal(err)
	}
	if err := f.TicketService.CheckCommentAttachmentUpload(context.Background(), f.agent, private.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("demoted author accessed internal comment: %v", err)
	}
}

func TestContentMutationEvents(t *testing.T) {
	f := newContentFixture(t)
	comment := f.comment(t, f.agent, false)
	file, err := f.TicketService.AddCommentAttachment(context.Background(), f.agent, comment.ID, &domain.Attachment{})
	if err != nil {
		t.Fatal(err)
	}
	channels := make([]chan []byte, 0)
	for _, user := range []*domain.User{f.owner, f.agent, f.admin} {
		ch, unsubscribe := realtime.DefaultBroker.Subscribe(string(user.Role), user.ID)
		defer unsubscribe()
		channels = append(channels, ch)
	}
	outsider, unsubscribe := realtime.DefaultBroker.Subscribe(string(f.outsider.Role), f.outsider.ID)
	defer unsubscribe()
	for _, step := range []struct {
		event  string
		mutate func() error
	}{
		{"comment_updated", func() error {
			_, err := f.TicketService.UpdateComment(context.Background(), f.agent, comment.ID, domain.UpdateCommentReq{Body: "edited"})
			return err
		}},
		{"attachment_updated", func() error {
			_, _, err := f.TicketService.ReplaceAttachment(context.Background(), f.agent, file.ID, &domain.Attachment{})
			return err
		}},
		{"attachment_deleted", func() error {
			_, err := f.TicketService.DeleteAttachment(context.Background(), f.agent, file.ID)
			return err
		}},
		{"comment_deleted", func() error {
			_, err := f.TicketService.DeleteComment(context.Background(), f.agent, comment.ID)
			return err
		}},
	} {
		if err := step.mutate(); err != nil {
			t.Fatal(err)
		}
		for _, ch := range channels {
			select {
			case msg := <-ch:
				if !strings.Contains(string(msg), "event: "+step.event+"\n") || !strings.Contains(string(msg), `"ticket_id":"`+f.ticket.ID+`"`) {
					t.Fatalf("unexpected event: %s", msg)
				}
			default:
				t.Fatalf("missing %s event", step.event)
			}
		}
		select {
		case msg := <-outsider:
			t.Fatalf("event leaked to unrelated requester: %s", msg)
		default:
		}
	}
}
