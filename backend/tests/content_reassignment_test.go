package tests

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"backend/internal/database"
	"backend/internal/domain"
	"backend/internal/handler"
	"backend/internal/realtime"
	"backend/internal/repository"
	"backend/internal/service"
)

// These decorators commit a reassignment after the service's last access check,
// immediately before the real repository write. There are no timing assumptions.
type reassignBeforeAttachmentWrite struct {
	domain.AttachmentRepository
	before func() error
}

func (r reassignBeforeAttachmentWrite) Create(ctx context.Context, actor *domain.User, att *domain.Attachment) error {
	if err := r.before(); err != nil {
		return err
	}
	return r.AttachmentRepository.Create(ctx, actor, att)
}

func (r reassignBeforeAttachmentWrite) Replace(ctx context.Context, actor *domain.User, att *domain.Attachment) (string, error) {
	if err := r.before(); err != nil {
		return "", err
	}
	return r.AttachmentRepository.Replace(ctx, actor, att)
}

func (r reassignBeforeAttachmentWrite) Delete(ctx context.Context, actor *domain.User, id string) (*domain.Attachment, error) {
	if err := r.before(); err != nil {
		return nil, err
	}
	return r.AttachmentRepository.Delete(ctx, actor, id)
}

type reassignBeforeCommentWrite struct {
	domain.CommentRepository
	before func() error
}

func (r reassignBeforeCommentWrite) Create(ctx context.Context, actor *domain.User, comment *domain.Comment) error {
	if err := r.before(); err != nil {
		return err
	}
	return r.CommentRepository.Create(ctx, actor, comment)
}

func (r reassignBeforeCommentWrite) Update(ctx context.Context, actor *domain.User, comment *domain.Comment) error {
	if err := r.before(); err != nil {
		return err
	}
	return r.CommentRepository.Update(ctx, actor, comment)
}

func (r reassignBeforeCommentWrite) Delete(ctx context.Context, actor *domain.User, id string) ([]*domain.Attachment, error) {
	if err := r.before(); err != nil {
		return nil, err
	}
	return r.CommentRepository.Delete(ctx, actor, id)
}

type reassignmentFixture struct {
	db                      *sql.DB
	tickets                 *repository.TicketRepository
	comments                *repository.CommentRepository
	attachments             *repository.AttachmentRepository
	users                   *repository.UserRepository
	owner, nextOwner, admin *domain.User
	ticket                  *domain.Ticket
	comment                 *domain.Comment
	root, linked            *domain.Attachment
	uploadDir               string
}

func newReassignmentFixture(t *testing.T, db *sql.DB, ownerRole domain.UserRole) *reassignmentFixture {
	t.Helper()
	ctx := context.Background()
	f := &reassignmentFixture{db: db, tickets: repository.NewTicketRepository(db), comments: repository.NewCommentRepository(db), attachments: repository.NewAttachmentRepository(db), users: repository.NewUserRepository(db), uploadDir: t.TempDir()}
	f.owner = &domain.User{ID: uuid.NewString(), Role: ownerRole}
	f.nextOwner = &domain.User{ID: uuid.NewString(), Role: domain.RoleRequester}
	f.admin = &domain.User{ID: uuid.NewString(), Role: domain.RoleAdmin}
	for _, user := range []*domain.User{f.owner, f.nextOwner, f.admin} {
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id,name,email,password_hash,role) VALUES ($1,'Test',$2,'hash',$3)`, user.ID, user.ID+"@test.id", user.Role); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.ticket = &domain.Ticket{ID: uuid.NewString(), RequesterID: f.owner.ID, Title: "Reassignment", Description: "Access must be current", Category: "IT", Status: domain.StatusOpen, CreatedAt: now, UpdatedAt: now}
	if err := f.tickets.Create(ctx, f.ticket); err != nil {
		t.Fatal(err)
	}
	f.comment = &domain.Comment{ID: uuid.NewString(), TicketID: f.ticket.ID, AuthorID: f.owner.ID, Body: "Original", CreatedAt: now}
	if err := f.comments.Create(ctx, f.owner, f.comment); err != nil {
		t.Fatal(err)
	}
	for _, commentID := range []*string{nil, &f.comment.ID} {
		att := &domain.Attachment{ID: uuid.NewString(), TicketID: f.ticket.ID, CommentID: commentID, UploaderID: f.owner.ID, Filename: "original.txt", FileSize: 8, ContentType: "text/plain", CreatedAt: now}
		att.FilePath = filepath.Join(f.uploadDir, att.ID+".txt")
		if err := os.WriteFile(att.FilePath, []byte("original"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := f.attachments.Create(ctx, f.owner, att); err != nil {
			t.Fatal(err)
		}
		if commentID == nil {
			f.root = att
		} else {
			f.linked = att
		}
	}
	return f
}

func (f *reassignmentFixture) api(t *testing.T, attachments domain.AttachmentRepository, comments domain.CommentRepository) *TestEnv {
	t.Helper()
	tickets := service.NewTicketService(f.tickets, comments, f.users, attachments, 2)
	auth, _, users, _, _, _, _ := newHarness()
	return &TestEnv{TicketService: tickets, UploadDir: f.uploadDir, Router: handler.SetupRouter(handler.RouterConfig{
		JWTSecret: "test-secret", AuthHandler: handler.NewAuthHandler(auth), UserHandler: handler.NewUserHandler(users), TicketHandler: handler.NewTicketHandler(tickets, f.uploadDir),
	})}
}

func TestPostgresContentWritesRecheckCommittedReassignment(t *testing.T) {
	db := contentPostgres(t)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
		operations := []string{"root_create", "root_replace", "root_delete"}
		if role == domain.RoleRequester {
			operations = append(operations, "comment_create", "comment_update", "comment_delete", "comment_attachment_create", "comment_attachment_replace", "comment_attachment_delete")
		}
		for _, operation := range operations {
			t.Run(string(role)+"/"+operation, func(t *testing.T) {
				f := newReassignmentFixture(t, db, role)
				beforeRoot, err := f.attachments.FindByID(context.Background(), f.root.ID)
				if err != nil {
					t.Fatal(err)
				}
				beforeLinked, err := f.attachments.FindByID(context.Background(), f.linked.ID)
				if err != nil {
					t.Fatal(err)
				}
				beforeComment, err := f.comments.FindByID(context.Background(), f.comment.ID)
				if err != nil {
					t.Fatal(err)
				}
				reassigned := false
				reassign := func() error {
					base := service.NewTicketService(f.tickets, f.comments, f.users, f.attachments, 2)
					_, err := base.Update(context.Background(), f.admin, f.ticket.ID, domain.UpdateTicketReq{RequesterID: &f.nextOwner.ID})
					if err == nil {
						reassigned = true
					}
					return err
				}
				env := f.api(t, reassignBeforeAttachmentWrite{f.attachments, reassign}, reassignBeforeCommentWrite{f.comments, reassign})
				events, unsubscribe := realtime.DefaultBroker.Subscribe("agent", uuid.NewString())
				defer unsubscribe()
				switch operation {
				case "root_create":
					contentUpload(t, env, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "new.txt", []byte("new"), http.StatusForbidden)
				case "root_replace":
					contentUpload(t, env, f.owner, "PATCH", "/attachments/"+f.root.ID, "new.txt", []byte("new"), http.StatusForbidden)
				case "root_delete":
					contentRequest(t, env, f.owner, "DELETE", "/attachments/"+f.root.ID, "", http.StatusForbidden)
				case "comment_create":
					contentRequest(t, env, f.owner, "POST", "/tickets/"+f.ticket.ID+"/comments", `{"body":"New comment"}`, http.StatusForbidden)
				case "comment_update":
					contentRequest(t, env, f.owner, "PATCH", "/comments/"+f.comment.ID, `{"body":"Edited"}`, http.StatusForbidden)
				case "comment_delete":
					contentRequest(t, env, f.owner, "DELETE", "/comments/"+f.comment.ID, "", http.StatusForbidden)
				case "comment_attachment_create":
					contentUpload(t, env, f.owner, "POST", "/comments/"+f.comment.ID+"/attachments", "new.txt", []byte("new"), http.StatusForbidden)
				case "comment_attachment_replace":
					contentUpload(t, env, f.owner, "PATCH", "/attachments/"+f.linked.ID, "new.txt", []byte("new"), http.StatusForbidden)
				case "comment_attachment_delete":
					contentRequest(t, env, f.owner, "DELETE", "/attachments/"+f.linked.ID, "", http.StatusForbidden)
				}
				if !reassigned {
					t.Fatal("did not reach the gap after the service's last authorization check")
				}
				for _, before := range []*domain.Attachment{beforeRoot, beforeLinked} {
					after, err := f.attachments.FindByID(context.Background(), before.ID)
					if err != nil || !reflect.DeepEqual(after, before) {
						t.Fatalf("denied mutation changed attachment: %+v %v", after, err)
					}
					assertFileContent(t, before.FilePath, "original")
				}
				afterComment, err := f.comments.FindByID(context.Background(), f.comment.ID)
				if err != nil || !reflect.DeepEqual(afterComment, beforeComment) {
					t.Fatalf("denied mutation changed comment: %+v %v", afterComment, err)
				}
				var commentCount, attachmentCount int
				if err := db.QueryRow(`SELECT count(*) FROM comments WHERE ticket_id=$1`, f.ticket.ID).Scan(&commentCount); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow(`SELECT count(*) FROM attachments WHERE ticket_id=$1`, f.ticket.ID).Scan(&attachmentCount); err != nil {
					t.Fatal(err)
				}
				if commentCount != 1 || attachmentCount != 2 {
					t.Fatalf("denied create changed counts: %d comments, %d files", commentCount, attachmentCount)
				}
				assertContentFiles(t, env, 2)
				for len(events) > 0 {
					if event := <-events; !strings.Contains(string(event), "event: ticket_updated\n") {
						t.Fatalf("denied content mutation emitted an event: %s", event)
					}
				}
				// The current owner can manage a legacy file regardless of its uploader.
				if _, err := f.attachments.Delete(context.Background(), f.nextOwner, f.root.ID); err != nil {
					t.Fatalf("current owner denied: %v", err)
				}
			})
		}
	}
}

func TestPostgresStaffCommentAuthorsKeepAccessAfterReassignment(t *testing.T) {
	db := contentPostgres(t)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	for _, role := range []domain.UserRole{domain.RoleAgent, domain.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			f := newReassignmentFixture(t, db, role)
			env := f.api(t, f.attachments, f.comments)
			if _, err := env.TicketService.Update(context.Background(), f.admin, f.ticket.ID, domain.UpdateTicketReq{RequesterID: &f.nextOwner.ID}); err != nil {
				t.Fatal(err)
			}
			// Staff authors still have ticket access; reassignment revokes only
			// their root-attachment ownership, not their comment authorship.
			comment := contentData[domain.Comment](t, contentRequest(t, env, f.owner, "POST", "/tickets/"+f.ticket.ID+"/comments", `{"body":"Private note","is_internal":true}`, http.StatusCreated))
			contentRequest(t, env, f.owner, "PATCH", "/comments/"+comment.ID, `{"body":"Edited"}`, http.StatusOK)
			file := contentData[domain.Attachment](t, contentUpload(t, env, f.owner, "POST", "/comments/"+comment.ID+"/attachments", "private.txt", []byte("private"), http.StatusCreated))
			replaced := contentData[domain.Attachment](t, contentUpload(t, env, f.owner, "PATCH", "/attachments/"+file.ID, "updated.txt", []byte("updated"), http.StatusOK))
			assertFileRemoved(t, file.FilePath)
			// Repositories must enforce authorship too, with no admin or owner override.
			if err := f.comments.Update(context.Background(), f.admin, &comment); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("non-author admin edited a comment: %v", err)
			}
			if _, err := f.attachments.Delete(context.Background(), f.nextOwner, file.ID); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("ticket owner deleted an internal comment file: %v", err)
			}
			contentRequest(t, env, f.owner, "DELETE", "/attachments/"+file.ID, "", http.StatusNoContent)
			assertFileRemoved(t, replaced.FilePath)
			contentRequest(t, env, f.owner, "DELETE", "/comments/"+comment.ID, "", http.StatusNoContent)
			contentRequest(t, env, f.owner, "PATCH", "/comments/"+f.comment.ID, `{"body":"Still the author"}`, http.StatusOK)
			contentRequest(t, env, f.owner, "DELETE", "/comments/"+f.comment.ID, "", http.StatusNoContent)
			assertFileRemoved(t, f.linked.FilePath)
			assertFileContent(t, f.root.FilePath, "original")
		})
	}
}

func TestPostgresContentWriteWaitsForReassignmentCommit(t *testing.T) {
	db := contentPostgres(t)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	f := newReassignmentFixture(t, db, domain.RoleRequester)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var reassignPID int
	if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&reassignPID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tickets SET requester_id=$1 WHERE id=$2`, f.nextOwner.ID, f.ticket.ID); err != nil {
		t.Fatal(err)
	}
	// Reads still see the old owner while the admin transaction is uncommitted.
	// The content transaction must wait, then authorize against the committed owner.
	result := make(chan error, 1)
	go func() { _, err := f.attachments.Delete(ctx, f.owner, f.root.ID); result <- err }()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE $1::int = ANY(pg_blocking_pids(pid))
			AND query LIKE 'SELECT requester_id FROM tickets%FOR UPDATE%'
		)`, reassignPID).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err := <-result:
			t.Fatalf("mutation bypassed the ticket lock: %v", err)
		case <-ctx.Done():
			t.Fatal("content write never waited for requester reassignment")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("former owner deletion after reassignment: %v", err)
	}
	if _, err := f.attachments.FindByID(ctx, f.root.ID); err != nil {
		t.Fatalf("former owner deleted metadata: %v", err)
	}
	assertFileContent(t, f.root.FilePath, "original")
}
