package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
	"backend/internal/middleware"
)

type UserHandler struct {
	userService domain.UserService
}

func NewUserHandler(userService domain.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) List(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// Admin boleh melihat semua user. Agent hanya butuh daftar agent/admin
	// untuk dropdown assignee.
	if user.Role != domain.RoleAdmin && user.Role != domain.RoleAgent {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	var users []*domain.User
	var err error
	if user.Role == domain.RoleAgent {
		agents, listErr := h.userService.List(c.Request.Context(), string(domain.RoleAgent))
		if listErr != nil {
			respondError(c, listErr)
			return
		}
		admins, listErr := h.userService.List(c.Request.Context(), string(domain.RoleAdmin))
		if listErr != nil {
			respondError(c, listErr)
			return
		}
		users = append(agents, admins...)
	} else {
		users, err = h.userService.List(c.Request.Context(), c.Query("role"))
		if err != nil {
			respondError(c, err)
			return
		}
	}
	respondData(c, http.StatusOK, users)
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if user.Role != domain.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	var req domain.UpdateUserRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	if err := h.userService.UpdateRole(c.Request.Context(), user, c.Param("id"), req.Role); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, gin.H{"message": "Role pengguna berhasil diperbarui"})
}
