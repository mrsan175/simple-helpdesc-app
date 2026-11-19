package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"backend/internal/domain"
)

func TestAPI_AttachmentPreviewRawBytesAndAuthorization(t *testing.T) {
	f := newContentFixture(t)
	public, internal := f.comment(t, f.agent, false), f.comment(t, f.agent, true)
	payload := []byte{'P', 'K', 3, 4, 0, 0xff, '\n', '{', '}'}
	root := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "root.zip", payload, http.StatusCreated))
	publicFile := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+public.ID+"/attachments", "public.zip", payload, http.StatusCreated))
	internalFile := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.agent, "POST", "/comments/"+internal.ID+"/attachments", "private.zip", payload, http.StatusCreated))
	for _, status := range []domain.TicketStatus{domain.StatusOpen, domain.StatusClosed} {
		f.ticket.Status = status
		if err := f.TicketRepo.Update(context.Background(), f.ticket); err != nil {
			t.Fatal(err)
		}
		for _, file := range []*domain.Attachment{&root, &publicFile, &internalFile} {
			for _, actor := range []*domain.User{nil, f.owner, f.agent, f.admin, f.otherAgent, f.outsider} {
				want := http.StatusOK
				if actor == nil {
					want = http.StatusUnauthorized
				} else if actor == f.outsider || (file.ID == internalFile.ID && actor == f.owner) {
					want = http.StatusForbidden
				}
				preview := contentRequest(t, f.TestEnv, actor, "GET", "/attachments/"+file.ID+"/preview", "", want)
				download := contentRequest(t, f.TestEnv, actor, "GET", "/attachments/"+file.ID+"/download", "", want)
				if want != http.StatusOK {
					if bytes.Contains(preview.Body.Bytes(), payload) || !strings.HasPrefix(preview.Header().Get("Content-Type"), "application/json") {
						t.Fatal("denied preview returned bytes instead of a JSON error")
					}
					continue
				}
				if !bytes.Equal(preview.Body.Bytes(), payload) || !bytes.Equal(download.Body.Bytes(), payload) {
					t.Fatal("preview/download changed raw bytes or returned a JSON envelope")
				}
				if preview.Header().Get("Content-Type") != "application/zip" || preview.Header().Get("Cache-Control") != "private, no-store" || preview.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Fatalf("unsafe preview headers: %v", preview.Header())
				}
				if csp := preview.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox;") || !strings.Contains(csp, "default-src 'none'") {
					t.Fatalf("missing preview sandbox: %q", csp)
				}
				if preview.Header().Get("ETag") != "" || preview.Header().Get("Last-Modified") != "" {
					t.Fatal("preview exposed cache validators")
				}
				kind, params, err := mime.ParseMediaType(preview.Header().Get("Content-Disposition"))
				if err != nil || kind != "inline" || params["filename"] != file.Filename {
					t.Fatalf("bad inline disposition: %v %v", preview.Header(), err)
				}
				if !strings.HasPrefix(download.Header().Get("Content-Disposition"), "attachment;") {
					t.Fatal("download disposition changed")
				}
			}
			stored, err := f.AttachRepo.FindByID(context.Background(), file.ID)
			if err != nil || !reflect.DeepEqual(stored, file) {
				t.Fatalf("preview changed metadata: %+v %v", stored, err)
			}
			got, err := os.ReadFile(file.FilePath)
			if err != nil || !bytes.Equal(got, payload) {
				t.Fatalf("preview changed disk file: %v", err)
			}
		}
	}
}

func TestAPI_AttachmentPreviewSafeMIMEAndFilename(t *testing.T) {
	f := newContentFixture(t)
	for _, tc := range []struct{ stored, want string }{
		{"image/jpeg", "image/jpeg"}, {"image/png", "image/png"}, {"image/gif", "image/gif"}, {"image/webp", "image/webp"},
		{"application/pdf", "application/pdf"}, {"text/plain", "text/plain; charset=utf-8"}, {"text/csv; charset=iso-8859-1", "text/csv; charset=utf-8"},
		{"application/msword", "application/msword"}, {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"application/vnd.ms-excel", "application/vnd.ms-excel"}, {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"application/zip", "application/zip"}, {"text/html", "application/octet-stream"}, {"image/svg+xml", "application/octet-stream"}, {"application/xml", "application/octet-stream"},
		{"", "application/octet-stream"}, {"text/plain\r\nX-Injected: yes", "application/octet-stream"},
	} {
		t.Run(tc.stored, func(t *testing.T) {
			file := &domain.Attachment{ID: uuid.NewString(), TicketID: f.ticket.ID, UploaderID: f.owner.ID,
				Filename: "../private\\folder/berkas \"日本\"\r\n.txt", ContentType: tc.stored, FileSize: 8, CreatedAt: time.Now().UTC()}
			file.FilePath = filepath.Join(f.UploadDir, file.ID)
			if err := os.WriteFile(file.FilePath, []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := f.AttachRepo.Create(context.Background(), f.owner, file); err != nil {
				t.Fatal(err)
			}
			w := contentRequest(t, f.TestEnv, f.owner, "GET", "/attachments/"+file.ID+"/preview", "", http.StatusOK)
			kind, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
			if err != nil || kind != "inline" || params["filename"] != "berkas \"日本\".txt" {
				t.Fatalf("unsafe filename: %q %v", w.Header().Get("Content-Disposition"), err)
			}
			if w.Header().Get("Content-Type") != tc.want || w.Header().Get("X-Injected") != "" || w.Body.String() != "original" {
				t.Fatalf("unsafe MIME or changed bytes: %v %q", w.Header(), w.Body.String())
			}
		})
	}
}

func TestAPI_AttachmentPreviewMissingFilesAreSafeJSON(t *testing.T) {
	f := newContentFixture(t)
	file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "missing.txt", []byte("secret"), http.StatusCreated))
	if err := os.Remove(file.FilePath); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{file.ID, uuid.NewString()} {
		w := contentRequest(t, f.TestEnv, f.owner, "GET", "/attachments/"+id+"/preview", "", http.StatusNotFound)
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response["error"] != "Data tidak ditemukan" {
			t.Fatalf("missing file error: %s %v", w.Body.String(), err)
		}
		if strings.Contains(w.Body.String(), f.UploadDir) || strings.Contains(w.Body.String(), file.Filename) || w.Header().Get("Content-Disposition") != "" || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("missing preview leaked file details: %v %s", w.Header(), w.Body.String())
		}
	}
	contentRequest(t, f.TestEnv, f.outsider, "GET", "/attachments/"+file.ID+"/preview", "", http.StatusForbidden)
	if _, err := f.AttachRepo.FindByID(context.Background(), file.ID); err != nil {
		t.Fatalf("missing disk file deleted metadata: %v", err)
	}
}

func TestAPI_AttachmentPreviewRangeAndConditionalRequests(t *testing.T) {
	f := newContentFixture(t)
	file := contentData[domain.Attachment](t, contentUpload(t, f.TestEnv, f.owner, "POST", "/tickets/"+f.ticket.ID+"/attachments", "range.pdf", []byte("%PDF-original"), http.StatusCreated))
	for _, partial := range []bool{false, true} {
		req := httptest.NewRequest("GET", "/api/attachments/"+file.ID+"/preview", nil)
		req.Header.Set("Authorization", "Bearer "+contentToken(t, f.owner))
		req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		if partial {
			req.Header.Set("Range", "bytes=0-3")
		}
		w := httptest.NewRecorder()
		f.Router.ServeHTTP(w, req)
		wantStatus, wantBody := http.StatusOK, "%PDF-original"
		if partial {
			wantStatus, wantBody = http.StatusPartialContent, "%PDF"
		}
		if w.Code != wantStatus || w.Body.String() != wantBody || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("range/conditional preview: %d %s %v", w.Code, w.Body.String(), w.Header())
		}
	}
}
