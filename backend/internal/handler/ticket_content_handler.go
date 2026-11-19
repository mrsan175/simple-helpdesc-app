package handler

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
	"backend/internal/middleware"
)

func (h *TicketHandler) UpdateComment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req domain.UpdateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Isi komentar wajib diisi"})
		return
	}
	comment, err := h.ticketService.UpdateComment(c.Request.Context(), user, c.Param("id"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, comment)
}

func (h *TicketHandler) DeleteComment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	files, err := h.ticketService.DeleteComment(c.Request.Context(), user, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	for _, file := range files {
		h.removeAttachmentFile(file.FilePath)
	}
	c.Status(http.StatusNoContent)
}

func (h *TicketHandler) DeleteAttachment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	deleted, err := h.ticketService.DeleteAttachment(c.Request.Context(), user, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	h.removeAttachmentFile(deleted.FilePath)
	c.Status(http.StatusNoContent)
}

func (h *TicketHandler) PreviewAttachment(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	user := middleware.CurrentUser(c)
	if user == nil {
		respondError(c, domain.ErrUnauthorized)
		return
	}
	att, err := h.ticketService.GetAttachment(c.Request.Context(), user, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	// Open before writing file headers so a missing disk file remains a safe JSON
	// error, not a successful preview or a filesystem error containing its path.
	file, err := os.Open(att.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(c, domain.ErrNotFound)
		} else {
			respondError(c, err)
		}
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		respondError(c, err)
		return
	}
	if !info.Mode().IsRegular() {
		respondError(c, domain.ErrNotFound)
		return
	}

	name := filepath.Base(strings.ReplaceAll(att.Filename, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": name}))
	c.Header("Content-Type", previewContentType(att.ContentType))
	// Never grant uploaded documents same-origin privileges or script execution.
	c.Header("Content-Security-Policy", "sandbox; default-src 'none'; base-uri 'none'; form-action 'none'")
	// Do not expose modification-time or ETag cache validators for private files.
	http.ServeContent(c.Writer, c.Request, name, time.Time{}, file)
}

func previewContentType(stored string) string {
	mediaType, _, err := mime.ParseMediaType(stored)
	if err != nil {
		return "application/octet-stream"
	}
	// Do not echo arbitrary stored types/parameters (especially HTML/SVG/XML).
	switch mediaType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf",
		"application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip":
		return mediaType
	case "text/plain", "text/csv":
		return mediaType + "; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func (h *TicketHandler) attachmentDir() string {
	if h.uploadDir == "" {
		return "./uploads"
	}
	return h.uploadDir
}

// Metadata is already deleted/replaced. Never follow a stored path outside the
// upload directory or recursively remove a directory; cleanup failures are logged.
func (h *TicketHandler) removeAttachmentFile(path string) {
	base, err := filepath.Abs(h.attachmentDir())
	if err != nil {
		log.Printf("attachment cleanup: resolve upload directory: %v", err)
		return
	}
	full, err := filepath.Abs(path)
	if err != nil || filepath.Dir(full) != base {
		log.Printf("attachment cleanup: refusing path outside upload directory: %q", path)
		return
	}
	info, err := os.Lstat(full)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Printf("attachment cleanup: stat %q: %v", full, err)
		return
	}
	if info.IsDir() {
		log.Printf("attachment cleanup: refusing directory: %q", full)
		return
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		log.Printf("attachment cleanup: remove %q: %v", full, err)
	}
}
