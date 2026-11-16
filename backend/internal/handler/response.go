package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
)

type errorBody struct {
	Error string `json:"error"`
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, errorBody{Error: "Email atau password salah"})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, errorBody{Error: "Anda tidak memiliki akses untuk aksi ini"})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, errorBody{Error: "Data tidak ditemukan"})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, errorBody{Error: "Email sudah terdaftar"})
	case errors.Is(err, domain.ErrInvalidTransition):
		c.JSON(http.StatusUnprocessableEntity, errorBody{Error: "Transisi status tidak valid"})
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, errorBody{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, errorBody{Error: "Terjadi kesalahan pada server"})
	}
}

func respondData(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{"data": data})
}
