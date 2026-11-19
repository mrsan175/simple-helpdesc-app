package domain

import (
	"time"
)

type UserRole string

const (
	RoleRequester UserRole = "requester"
	RoleAgent     UserRole = "agent"
	RoleAdmin     UserRole = "admin"
)

type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityMedium TicketPriority = "medium"
	PriorityHigh   TicketPriority = "high"
	PriorityUrgent TicketPriority = "urgent"
)

type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Ticket struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Priority    TicketPriority `json:"priority"`
	Status      TicketStatus   `json:"status"`
	RequesterID string         `json:"requester_id"`
	AssigneeID  *string        `json:"assignee_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Optional relations loaded
	Requester *User      `json:"requester,omitempty"`
	Assignee  *User      `json:"assignee,omitempty"`
	Comments  []*Comment `json:"comments"`
	Attachments []*Attachment `json:"attachments"`
	IsOverdue bool       `json:"is_overdue"`
}

type Attachment struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	UploaderID  string    `json:"uploader_id"`
	Filename    string    `json:"filename"`
	FilePath    string    `json:"file_path,omitempty"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`

	Uploader *User `json:"uploader,omitempty"`
}

type Comment struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	AuthorID   string    `json:"author_id"`
	Body       string    `json:"body"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`

	Author *User `json:"author,omitempty"`
}
