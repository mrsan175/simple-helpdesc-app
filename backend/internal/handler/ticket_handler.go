package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"backend/internal/domain"
	"backend/internal/middleware"
	"backend/internal/realtime"
)

type TicketHandler struct {
	ticketService domain.TicketService
	uploadDir     string
}

func NewTicketHandler(ticketService domain.TicketService, uploadDir string) *TicketHandler {
	return &TicketHandler{ticketService: ticketService, uploadDir: uploadDir}
}

func (h *TicketHandler) List(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := domain.TicketFilter{
		Page:       page,
		Limit:      limit,
		Status:     c.Query("status"),
		Priority:   c.Query("priority"),
		AssigneeID: c.Query("assignee_id"),
		Search:     c.Query("search"),
		SortBy:     c.DefaultQuery("sort_by", "created_at"),
		Order:      c.DefaultQuery("order", "desc"),
	}

	if filter.AssigneeID == "me" {
		filter.AssigneeID = user.ID
	}

	res, err := h.ticketService.List(c.Request.Context(), user, filter)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res.Data, "meta": res.Meta})
}

func (h *TicketHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ticket, err := h.ticketService.Get(c.Request.Context(), user, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, ticket)
}

func (h *TicketHandler) Create(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req domain.CreateTicketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Judul, deskripsi, kategori, dan prioritas wajib diisi"})
		return
	}

	ticket, err := h.ticketService.Create(c.Request.Context(), user, req)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, ticket)
}

func (h *TicketHandler) Update(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req domain.UpdateTicketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
		return
	}

	ticket, err := h.ticketService.Update(c.Request.Context(), user, c.Param("id"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, ticket)
}

func (h *TicketHandler) AddComment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req domain.CreateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Isi komentar wajib diisi"})
		return
	}

	comment, err := h.ticketService.AddComment(c.Request.Context(), user, c.Param("id"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, comment)
}

func (h *TicketHandler) Stats(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := h.ticketService.Stats(c.Request.Context(), user)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, stats)
}

func (h *TicketHandler) AdminStats(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if user.Role != domain.RoleAdmin {
		respondError(c, domain.ErrForbidden)
		return
	}

	stats, err := h.ticketService.AdminStats(c.Request.Context(), user)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, stats)
}

func (h *TicketHandler) UploadAttachment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Max 5 MB
	const maxSize = 5 << 20
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	if fileHeader.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file melebihi batas maksimal 5MB"})
		return
	}

	allowedExt := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".zip":  "application/zip",
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	expectedMime, ok := allowedExt[ext]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe file tidak didukung. Format didukung: jpg, png, webp, gif, pdf, txt, csv, doc, docx, xls, xlsx, zip"})
		return
	}

	uploadDir := h.uploadDir
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan direktori berkas"})
		return
	}

	fileID := uuid.NewString()
	sanitizedName := filepath.Base(fileHeader.Filename)
	storedFilename := fmt.Sprintf("%s_%s", fileID, sanitizedName)
	storedPath := filepath.Join(uploadDir, storedFilename)

	if err := c.SaveUploadedFile(fileHeader, storedPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan berkas"})
		return
	}

	att := &domain.Attachment{
		Filename:    sanitizedName,
		FilePath:    storedPath,
		FileSize:    fileHeader.Size,
		ContentType: expectedMime,
	}

	created, err := h.ticketService.AddAttachment(c.Request.Context(), user, c.Param("id"), att)
	if err != nil {
		_ = os.Remove(storedPath)
		respondError(c, err)
		return
	}

	respondData(c, http.StatusCreated, created)
}

func (h *TicketHandler) DownloadAttachment(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	att, err := h.ticketService.GetAttachment(c.Request.Context(), user, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, url.PathEscape(att.Filename)))
	c.Header("Content-Type", att.ContentType)
	c.File(att.FilePath)
}

func (h *TicketHandler) StreamEvents(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteString(": connected\n\n")
	c.Writer.Flush()

	ch, unsubscribe := realtime.DefaultBroker.Subscribe()
	defer unsubscribe()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := c.Writer.Write(msg); err != nil {
				return
			}
			c.Writer.Flush()
		case <-ticker.C:
			if _, err := io.WriteString(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
