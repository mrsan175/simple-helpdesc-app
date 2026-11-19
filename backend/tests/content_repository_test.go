package tests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"backend/internal/database"
	"backend/internal/domain"
	"backend/internal/repository"
	"backend/internal/service"
)

// Set TEST_DATABASE_URL to use an isolated schema in an existing PostgreSQL,
// or TEST_POSTGRES_LOCAL=1 to run a disposable local cluster (initdb/postgres).
func contentPostgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("TEST_POSTGRES_LOCAL") != "1" {
			t.Skip("set TEST_DATABASE_URL or TEST_POSTGRES_LOCAL=1 for PostgreSQL integration tests")
		}
		dir, err := os.MkdirTemp("", "helpdesk-pg-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		data := filepath.Join(dir, "data")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if output, err := exec.CommandContext(ctx, "initdb", "-D", data, "-A", "trust", "-U", "postgres", "--no-locale", "--encoding=UTF8").CombinedOutput(); err != nil {
			t.Fatalf("initdb: %v: %s", err, output)
		}
		logPath := filepath.Join(dir, "postgres.log")
		log, err := os.Create(logPath)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("postgres", "-D", data, "-k", dir, "-h", "", "-p", "55439", "-F",
			"-c", "config_file="+filepath.Join(data, "postgresql.conf"),
			"-c", "hba_file="+filepath.Join(data, "pg_hba.conf"),
			"-c", "ident_file="+filepath.Join(data, "pg_ident.conf"),
			"-c", "logging_collector=off")
		cmd.Stdout, cmd.Stderr = log, log
		if err := cmd.Start(); err != nil {
			log.Close()
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait(); _ = log.Close() })
		local := &url.URL{Scheme: "postgres", User: url.User("postgres"), Path: "/postgres"}
		query := local.Query()
		query.Set("host", dir)
		query.Set("port", "55439")
		query.Set("sslmode", "disable")
		local.RawQuery = query.Encode()
		dsn = local.String()
		probe, err := sql.Open("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer probe.Close()
		deadline := time.Now().Add(10 * time.Second)
		for {
			if err = probe.PingContext(ctx); err == nil {
				break
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				output, _ := os.ReadFile(logPath)
				t.Fatalf("start postgres: %v: %s", err, output)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a postgres:// or postgresql:// URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "content_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
		admin.Close()
	})
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPostgresContentMigrationAndRepositories(t *testing.T) {
	db := contentPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, name := range []string{"0001_init.sql", "0002_attachments.sql", "0003_optional_priority.sql"} {
		migration, err := os.ReadFile(filepath.Join("..", "internal", "database", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(migration)); err != nil {
			t.Fatalf("legacy migration %s: %v", name, err)
		}
	}
	userID, ticketID, commentID, legacyID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,name,email,password_hash,role) VALUES ($1,'Author','author@test.id','hash','agent')`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO tickets (id,title,description,category,requester_id) VALUES ($1,'Ticket','Body','IT',$2)`, ticketID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO comments (id,ticket_id,author_id,body,is_internal) VALUES ($1,$2,$3,'Legacy note',TRUE)`, commentID, ticketID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO attachments (id,ticket_id,uploader_id,filename,file_path,file_size,content_type) VALUES ($1,$2,$3,'legacy.txt','uploads/legacy.txt',6,'text/plain')`, legacyID, ticketID, userID); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migration rerun: %v", err)
	}

	ar, cr := repository.NewAttachmentRepository(db), repository.NewCommentRepository(db)
	actor := &domain.User{ID: userID, Role: domain.RoleAgent}
	legacy, err := ar.FindByID(ctx, legacyID)
	if err != nil || legacy.CommentID != nil || legacy.FilePath != "uploads/legacy.txt" || legacy.Filename != "legacy.txt" || legacy.FileSize != 6 {
		t.Fatalf("legacy attachment changed: %+v %v", legacy, err)
	}
	comment, err := cr.FindByID(ctx, commentID)
	if err != nil || comment.UpdatedAt != nil || comment.Body != "Legacy note" || comment.Attachments == nil {
		t.Fatalf("legacy comment changed: %+v %v", comment, err)
	}
	var indexed bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_attachments_comment')`).Scan(&indexed); err != nil || !indexed {
		t.Fatalf("missing comment attachment index: %v", err)
	}

	newFile := func(commentID *string, path string) *domain.Attachment {
		return &domain.Attachment{ID: uuid.NewString(), TicketID: ticketID, CommentID: commentID, UploaderID: userID, Filename: "file.txt", FilePath: path, FileSize: 4, ContentType: "text/plain", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	}
	linked := newFile(&commentID, "uploads/comment.txt")
	if err := ar.Create(ctx, actor, linked); err != nil {
		t.Fatal(err)
	}
	roots, err := ar.ListByTicket(ctx, ticketID)
	if err != nil || len(roots) != 1 || roots[0].ID != legacyID || roots[0].Uploader == nil {
		t.Fatalf("root attachment query: %+v %v", roots, err)
	}
	files, err := ar.ListByComment(ctx, commentID)
	if err != nil || len(files) != 1 || files[0].ID != linked.ID || files[0].CommentID == nil {
		t.Fatalf("comment attachment query: %+v %v", files, err)
	}
	empty, err := ar.ListByComment(ctx, uuid.NewString())
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty attachments must be []: %+v %v", empty, err)
	}
	publicComments, err := cr.ListByTicket(ctx, ticketID, false)
	if err != nil || len(publicComments) != 0 {
		t.Fatalf("internal comment leaked: %+v %v", publicComments, err)
	}
	staffComments, err := cr.ListByTicket(ctx, ticketID, true)
	if err != nil || len(staffComments) != 1 || staffComments[0].UpdatedAt != nil {
		t.Fatalf("staff comments: %+v %v", staffComments, err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	comment.Body, comment.UpdatedAt, comment.IsInternal = "Edited note", &now, false
	if err := cr.Update(ctx, actor, comment); err != nil {
		t.Fatal(err)
	}
	comment, err = cr.FindByID(ctx, commentID)
	if err != nil || !comment.IsInternal || comment.UpdatedAt == nil || !comment.UpdatedAt.Equal(now) || comment.Body != "Edited note" {
		t.Fatalf("comment update toggled privacy or lost timestamp: %+v %v", comment, err)
	}

	tickets := service.NewTicketService(repository.NewTicketRepository(db), cr, repository.NewUserRepository(db), ar, 2)
	detail, err := tickets.Get(ctx, actor, ticketID)
	if err != nil || len(detail.Comments) != 1 || len(detail.Comments[0].Attachments) != 1 || len(detail.Attachments) != 1 {
		t.Fatalf("hydrated detail: %+v %v", detail, err)
	}
	requester := &domain.User{ID: userID, Role: domain.RoleRequester}
	requesterDetail, err := tickets.Get(ctx, requester, ticketID)
	if err != nil || len(requesterDetail.Comments) != 0 || len(requesterDetail.Attachments) != 1 {
		t.Fatalf("private files leaked in detail: %+v %v", requesterDetail, err)
	}
	if _, err := tickets.GetAttachment(ctx, requester, linked.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("private file download allowed: %v", err)
	}

	replacement := *linked
	replacement.FilePath, replacement.Filename, replacement.FileSize = "uploads/replaced.txt", "replaced.txt", 8
	replacement.CommentID, replacement.TicketID, replacement.CreatedAt = nil, uuid.NewString(), time.Time{}
	oldPath, err := ar.Replace(ctx, actor, &replacement)
	if err != nil || oldPath != linked.FilePath {
		t.Fatalf("replace old path: %q %v", oldPath, err)
	}
	replaced, err := ar.FindByID(ctx, linked.ID)
	if err != nil || replaced.FilePath != replacement.FilePath || replaced.CommentID == nil || *replaced.CommentID != commentID || replaced.TicketID != ticketID || !replaced.CreatedAt.Equal(linked.CreatedAt) {
		t.Fatalf("replacement changed base identity: %+v %v", replaced, err)
	}

	// Force the final DELETE to fail: attachment removals must roll back too.
	if _, err := db.ExecContext(ctx, `CREATE FUNCTION refuse_comment_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated deletion failure'; END $$; CREATE TRIGGER fail_comment_delete BEFORE DELETE ON comments FOR EACH ROW EXECUTE FUNCTION refuse_comment_delete()`); err != nil {
		t.Fatal(err)
	}
	if _, err := cr.Delete(ctx, actor, commentID); err == nil {
		t.Fatal("expected transaction failure")
	}
	if _, err := ar.FindByID(ctx, linked.ID); err != nil {
		t.Fatalf("attachment deletion was not rolled back: %v", err)
	}
	if _, err := cr.FindByID(ctx, commentID); err != nil {
		t.Fatalf("comment deletion was not rolled back: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TRIGGER fail_comment_delete ON comments; DROP FUNCTION refuse_comment_delete()`); err != nil {
		t.Fatal(err)
	}
	deletedFiles, err := cr.Delete(ctx, actor, commentID)
	if err != nil || len(deletedFiles) != 1 || deletedFiles[0].FilePath != replacement.FilePath {
		t.Fatalf("comment deletion cleanup paths: %+v %v", deletedFiles, err)
	}
	if _, err := ar.FindByID(ctx, linked.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("linked metadata remains: %v", err)
	}
	if _, err := ar.FindByID(ctx, legacyID); err != nil {
		t.Fatalf("legacy root was deleted: %v", err)
	}
	if err := ar.Create(ctx, actor, newFile(&commentID, "orphan.txt")); err == nil {
		t.Fatal("FK accepted deleted comment")
	}

	// Verify the FK itself, independently of the explicit repository deletion.
	cascadeID := uuid.NewString()
	if err := cr.Create(ctx, actor, &domain.Comment{ID: cascadeID, TicketID: ticketID, AuthorID: userID, Body: "Cascade", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	cascadeFile := newFile(&cascadeID, "cascade.txt")
	if err := ar.Create(ctx, actor, cascadeFile); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM comments WHERE id=$1`, cascadeID); err != nil {
		t.Fatal(err)
	}
	if _, err := ar.FindByID(ctx, cascadeFile.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ON DELETE CASCADE failed: %v", err)
	}

	deleted, err := ar.Delete(ctx, actor, legacyID)
	if err != nil || deleted.FilePath != legacy.FilePath {
		t.Fatalf("attachment deletion path: %+v %v", deleted, err)
	}
	if _, err := ar.Delete(ctx, actor, legacyID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete missing attachment: %v", err)
	}
	if _, err := ar.Replace(ctx, actor, legacy); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("replace missing attachment: %v", err)
	}
	if err := cr.Update(ctx, actor, comment); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("edit missing comment: %v", err)
	}
	if _, err := cr.Delete(ctx, actor, commentID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete missing comment: %v", err)
	}

	t.Run("concurrent_replacements_return_actual_old_paths", func(t *testing.T) {
		file := newFile(nil, "initial.txt")
		if err := ar.Create(ctx, actor, file); err != nil {
			t.Fatal(err)
		}
		type result struct {
			path string
			err  error
		}
		results := make(chan result, 8)
		for i := 0; i < cap(results); i++ {
			go func(i int) {
				next := *file
				next.FilePath = fmt.Sprintf("replacement-%d.txt", i)
				path, err := ar.Replace(ctx, actor, &next)
				results <- result{path, err}
			}(i)
		}
		paths := make(map[string]bool)
		for i := 0; i < cap(results); i++ {
			got := <-results
			if got.err != nil {
				t.Fatal(got.err)
			}
			if paths[got.path] {
				t.Fatalf("same old path returned twice: %s", got.path)
			}
			paths[got.path] = true
		}
		final, err := ar.FindByID(ctx, file.ID)
		if err != nil || paths[final.FilePath] || !paths[file.FilePath] {
			t.Fatalf("cleanup paths include live file or miss initial file: %+v %v", final, err)
		}
	})

	t.Run("concurrent_comment_upload_delete_captures_every_committed_file", func(t *testing.T) {
		for i := 0; i < 12; i++ {
			id := uuid.NewString()
			if err := cr.Create(ctx, actor, &domain.Comment{ID: id, TicketID: ticketID, AuthorID: userID, Body: "Race", CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
			file := newFile(&id, "race.txt")
			start := make(chan struct{})
			created := make(chan error, 1)
			type deleted struct {
				files []*domain.Attachment
				err   error
			}
			removed := make(chan deleted, 1)
			go func() { <-start; created <- ar.Create(ctx, actor, file) }()
			go func() { <-start; files, err := cr.Delete(ctx, actor, id); removed <- deleted{files, err} }()
			close(start)
			createErr, deletion := <-created, <-removed
			if deletion.err != nil {
				t.Fatal(deletion.err)
			}
			if createErr == nil && (len(deletion.files) != 1 || deletion.files[0].ID != file.ID) {
				t.Fatalf("committed upload missing from cleanup paths: %+v", deletion.files)
			}
			if createErr != nil && len(deletion.files) != 0 {
				t.Fatalf("failed upload returned cleanup metadata: %+v", deletion.files)
			}
			if _, err := ar.FindByID(ctx, file.ID); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("orphan metadata after race: %v", err)
			}
		}
	})
}
