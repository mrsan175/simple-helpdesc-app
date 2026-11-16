package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
	"backend/internal/middleware"
)

type TicketHandler struct {
	ticketService domain.TicketService
}

func NewTicketHandler(ticketService domain.TicketService) *TicketHandler {
	return &TicketHandler{ticketService: ticketService}
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
