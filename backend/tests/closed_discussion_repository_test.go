package tests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"backend/internal/database"
	"backend/internal/domain"
	"backend/internal/realtime"
)

func snapshotDiscussion(t *testing.T, f *reassignmentFixture) func() {
	t.Helper()
	ctx := context.Background()
	comment, err := f.comments.FindByID(ctx, f.comment.ID)
	if err != nil {
		t.Fatal(err)
	}
	files := make([]*domain.Attachment, 0, 2)
	for _, file := range []*domain.Attachment{f.root, f.linked} {
		stored, err := f.attachments.FindByID(ctx, file.ID)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, stored)
	}
	return func() {
		t.Helper()
		afterComment, err := f.comments.FindByID(ctx, comment.ID)
		if err != nil || !reflect.DeepEqual(comment, afterComment) {
			t.Fatalf("denied discussion write changed comment: %+v %v", afterComment, err)
		}
		for _, before := range files {
			after, err := f.attachments.FindByID(ctx, before.ID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("denied discussion write changed attachment: %+v %v", after, err)
			}
			assertFileContent(t, before.FilePath, "original")
		}
		var comments, attachments int
		if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM comments WHERE ticket_id=$1`, f.ticket.ID).Scan(&comments); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM attachments WHERE ticket_id=$1`, f.ticket.ID).Scan(&attachments); err != nil {
			t.Fatal(err)
		}
		if comments != 1 || attachments != 2 {
			t.Fatalf("denied create changed counts: %d comments, %d files", comments, attachments)
		}
		assertContentFiles(t, &TestEnv{UploadDir: f.uploadDir}, 2)
	}
}

func repositoryDiscussionMutation(ctx context.Context, f *reassignmentFixture, operation string) error {
	now := time.Now().UTC().Truncate(time.Microsecond)
	switch operation {
	case "comment_create":
		return f.comments.Create(ctx, f.owner, &domain.Comment{ID: uuid.NewString(), TicketID: f.ticket.ID, AuthorID: f.owner.ID, Body: "New", IsInternal: f.comment.IsInternal, CreatedAt: now})
	case "comment_update":
		updated := *f.comment
		updated.Body, updated.UpdatedAt = "Edited", &now
		return f.comments.Update(ctx, f.owner, &updated)
	case "comment_delete":
		files, err := f.comments.Delete(ctx, f.owner, f.comment.ID)
		if err != nil && files != nil {
			return fmt.Errorf("failed comment deletion exposed cleanup paths: %v", files)
		}
		return err
	case "file_create":
		file := *f.linked
		file.ID, file.FilePath = uuid.NewString(), filepath.Join(f.uploadDir, "new.txt")
		return f.attachments.Create(ctx, f.owner, &file)
	case "file_replace":
		file := *f.linked
		file.Filename, file.FilePath, file.ContentType, file.FileSize = "new.csv", filepath.Join(f.uploadDir, "new.csv"), "text/csv", 3
		// A stale/forged snapshot must not turn a comment file into a root file.
		file.CommentID, file.TicketID = nil, uuid.NewString()
		oldPath, err := f.attachments.Replace(ctx, f.owner, &file)
		if err != nil && oldPath != "" {
			return fmt.Errorf("failed replacement exposed cleanup path: %s", oldPath)
		}
		return err
	case "file_delete":
		file, err := f.attachments.Delete(ctx, f.owner, f.linked.ID)
		if err != nil && file != nil {
			return fmt.Errorf("failed file deletion exposed cleanup metadata: %+v", file)
		}
		return err
	default:
		return fmt.Errorf("unknown discussion mutation: %s", operation)
	}
}

func setFixtureInternal(t *testing.T, f *reassignmentFixture, internal bool) {
	t.Helper()
	f.comment.IsInternal = internal
	if _, err := f.db.Exec(`UPDATE comments SET is_internal=$1 WHERE id=$2`, internal, f.comment.ID); err != nil {
		t.Fatal(err)
	}
}

// Observe the actual lock dependency instead of assuming a goroutine has started.
func waitForContentBlock(t *testing.T, ctx context.Context, db *sql.DB, blocker int, queryPattern string, result <-chan error) int {
	t.Helper()
	for {
		var pid int
		err := db.QueryRowContext(ctx, `SELECT pid FROM pg_stat_activity
			WHERE $1::int = ANY(pg_blocking_pids(pid)) AND query LIKE $2 LIMIT 1`, blocker, queryPattern).Scan(&pid)
		if err == nil {
			return pid
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
		select {
		case err := <-result:
			t.Fatalf("operation bypassed expected lock: %v", err)
		case <-ctx.Done():
			t.Fatalf("operation never waited for expected lock: %v", ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestPostgresClosedDiscussion(t *testing.T) {
	db := contentPostgres(t)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
		for _, internal := range []bool{false, true} {
			if internal && role == domain.RoleRequester {
				continue
			}
			for _, operation := range discussionMutations {
				name := fmt.Sprintf("%s/internal=%v/%s", role, internal, operation)
				t.Run("close_after_service_check/"+name, func(t *testing.T) {
					f := newReassignmentFixture(t, db, role)
					setFixtureInternal(t, f, internal)
					unchanged := snapshotDiscussion(t, f)
					closed := false
					closeTicket := func() error {
						ticket := *f.ticket
						ticket.Status = domain.StatusClosed
						err := f.tickets.Update(context.Background(), &ticket)
						closed = err == nil
						return err
					}
					env := f.api(t, reassignBeforeAttachmentWrite{f.attachments, closeTicket}, reassignBeforeCommentWrite{f.comments, closeTicket})
					events, unsubscribe := realtime.DefaultBroker.Subscribe("agent", uuid.NewString())
					defer unsubscribe()
					w := requestDiscussionMutation(t, env, f.owner, f.ticket.ID, f.comment.ID, f.linked.ID, operation, http.StatusConflict)
					assertTicketClosedResponse(t, w)
					if !closed {
						t.Fatal("did not close between service preflight and repository write")
					}
					unchanged()
					if len(events) != 0 {
						t.Fatal("failed transaction emitted a content event")
					}
				})
				t.Run("closer_locks_first/"+name, func(t *testing.T) {
					f := newReassignmentFixture(t, db, role)
					setFixtureInternal(t, f, internal)
					unchanged := snapshotDiscussion(t, f)
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					tx, err := db.BeginTx(ctx, nil)
					if err != nil {
						t.Fatal(err)
					}
					defer tx.Rollback()
					var closerPID int
					if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&closerPID); err != nil {
						t.Fatal(err)
					}
					if _, err := tx.ExecContext(ctx, `UPDATE tickets SET status='closed' WHERE id=$1`, f.ticket.ID); err != nil {
						t.Fatal(err)
					}
					result := make(chan error, 1)
					go func() { result <- repositoryDiscussionMutation(ctx, f, operation) }()
					waitForContentBlock(t, ctx, db, closerPID, "SELECT requester_id, status FROM tickets%FOR UPDATE%", result)
					if err := tx.Commit(); err != nil {
						t.Fatal(err)
					}
					if err := <-result; !errors.Is(err, domain.ErrTicketClosed) {
						t.Fatalf("write after close committed: %v", err)
					}
					unchanged()
				})
			}
		}
	}

	t.Run("root_owner_CRUD_and_authorization_unchanged", func(t *testing.T) {
		for _, role := range []domain.UserRole{domain.RoleRequester, domain.RoleAgent, domain.RoleAdmin} {
			t.Run(string(role), func(t *testing.T) {
				f := newReassignmentFixture(t, db, role)
				ctx := context.Background()
				if _, err := db.ExecContext(ctx, `UPDATE tickets SET status='closed' WHERE id=$1`, f.ticket.ID); err != nil {
					t.Fatal(err)
				}
				// Non-authors/owners remain forbidden, not privileged by role or Closed.
				if _, err := f.attachments.Delete(ctx, f.admin, f.root.ID); !errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("non-owner root deletion: %v", err)
				}
				if _, err := f.comments.Delete(ctx, f.admin, f.comment.ID); !errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("non-author comment deletion: %v", err)
				}
				if _, err := f.attachments.Delete(ctx, f.admin, f.linked.ID); !errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("non-author comment-file deletion: %v", err)
				}
				env := f.api(t, f.attachments, f.comments)
				replaced := contentData[domain.Attachment](t, contentUpload(t, env, f.owner, "PATCH", "/attachments/"+f.root.ID, "new.txt", []byte("new"), http.StatusOK))
				assertFileRemoved(t, f.root.FilePath)
				assertFileContent(t, replaced.FilePath, "new")
				contentRequest(t, env, f.owner, "DELETE", "/attachments/"+f.root.ID, "", http.StatusNoContent)
				assertFileRemoved(t, replaced.FilePath)
				contentUpload(t, env, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "new.txt", []byte("new"), http.StatusCreated)
				assertFileContent(t, f.linked.FilePath, "original")
			})
		}
	})

	t.Run("writer_locks_first", func(t *testing.T) {
		// A test-only trigger pauses each writer after it holds the ticket lock.
		// Closure must wait for its commit, not slip between validation and write.
		_, err := db.Exec(`CREATE FUNCTION pause_discussion_write() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				PERFORM pg_advisory_xact_lock(732987221);
				IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
				RETURN NEW;
			END $$;
			CREATE TRIGGER pause_comment_write BEFORE INSERT OR UPDATE OR DELETE ON comments FOR EACH ROW EXECUTE FUNCTION pause_discussion_write();
			CREATE TRIGGER pause_file_write BEFORE INSERT OR UPDATE OR DELETE ON attachments FOR EACH ROW EXECUTE FUNCTION pause_discussion_write()`)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := db.Exec(`DROP TRIGGER pause_comment_write ON comments; DROP TRIGGER pause_file_write ON attachments; DROP FUNCTION pause_discussion_write()`); err != nil {
				t.Error(err)
			}
		}()
		for _, operation := range discussionMutations {
			t.Run(operation, func(t *testing.T) {
				f := newReassignmentFixture(t, db, domain.RoleAgent)
				setFixtureInternal(t, f, true)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				holder, err := db.Conn(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer holder.Close()
				var holderPID int
				if err := holder.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
					t.Fatal(err)
				}
				if _, err := holder.ExecContext(ctx, `SELECT pg_advisory_lock(732987221)`); err != nil {
					t.Fatal(err)
				}
				defer holder.ExecContext(context.Background(), `SELECT pg_advisory_unlock_all()`)
				written := make(chan error, 1)
				go func() { written <- repositoryDiscussionMutation(ctx, f, operation) }()
				writerPID := waitForContentBlock(t, ctx, db, holderPID, "%", written)
				closed := make(chan error, 1)
				go func() {
					ticket := *f.ticket
					ticket.Status = domain.StatusClosed
					closed <- f.tickets.Update(ctx, &ticket)
				}()
				waitForContentBlock(t, ctx, db, writerPID, "%UPDATE tickets%", closed)
				if _, err := holder.ExecContext(ctx, `SELECT pg_advisory_unlock(732987221)`); err != nil {
					t.Fatal(err)
				}
				if err := <-written; err != nil {
					t.Fatalf("write preceding close should commit: %v", err)
				}
				if err := <-closed; err != nil {
					t.Fatalf("close after writer: %v", err)
				}
				ticket, err := f.tickets.FindByID(ctx, f.ticket.ID)
				if err != nil || ticket.Status != domain.StatusClosed {
					t.Fatalf("close did not commit after write: %+v %v", ticket, err)
				}
				if operation == "comment_create" || operation == "file_create" {
					if err := repositoryDiscussionMutation(ctx, f, operation); !errors.Is(err, domain.ErrTicketClosed) {
						t.Fatalf("subsequent write on Closed accepted: %v", err)
					}
				}
				// Repository-only tests never touch actual disk contents.
				assertFileContent(t, f.root.FilePath, "original")
				assertFileContent(t, f.linked.FilePath, "original")
			})
		}
	})

	t.Run("internal_privacy_precedes_closed_for_demoted_author", func(t *testing.T) {
		f := newReassignmentFixture(t, db, domain.RoleAgent)
		setFixtureInternal(t, f, true)
		unchanged := snapshotDiscussion(t, f)
		if _, err := db.Exec(`UPDATE tickets SET status='closed' WHERE id=$1`, f.ticket.ID); err != nil {
			t.Fatal(err)
		}
		f.owner.Role = domain.RoleRequester
		for _, operation := range discussionMutations {
			if operation == "comment_create" {
				continue
			}
			if err := repositoryDiscussionMutation(context.Background(), f, operation); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("internal restriction bypassed by Closed: %s: %v", operation, err)
			}
		}
		env := f.api(t, f.attachments, f.comments)
		for _, suffix := range []string{"/preview", "/download"} {
			w := contentRequest(t, env, f.owner, "GET", "/attachments/"+f.linked.ID+suffix, "", http.StatusForbidden)
			if strings.Contains(w.Body.String(), f.linked.FilePath) {
				t.Fatal("private file path leaked")
			}
		}
		unchanged()
	})
}
