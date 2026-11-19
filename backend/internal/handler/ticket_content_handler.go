package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

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
