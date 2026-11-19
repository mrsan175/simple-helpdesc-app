package domain

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrForbidden         = errors.New("forbidden")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrTicketClosed      = errors.New("Diskusi pada tiket yang sudah ditutup tidak dapat diubah")
	ErrValidation        = errors.New("validation error")
)
