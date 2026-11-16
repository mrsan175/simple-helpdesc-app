package domain

type RegisterReq struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Role      UserRole `json:"role"`
	CreatedAt string   `json:"created_at,omitempty"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type CreateTicketReq struct {
	Title       string         `json:"title" binding:"required"`
	Description string         `json:"description" binding:"required"`
	Category    string         `json:"category" binding:"required"`
	Priority    TicketPriority `json:"priority" binding:"required"`
}

type UpdateTicketReq struct {
	Status     *TicketStatus   `json:"status,omitempty"`
	Priority   *TicketPriority `json:"priority,omitempty"`
	AssigneeID *string         `json:"assignee_id,omitempty"`
}

type CreateCommentReq struct {
	Body       string `json:"body" binding:"required"`
	IsInternal bool   `json:"is_internal"`
}

type UpdateUserRoleReq struct {
	Role UserRole `json:"role" binding:"required"`
}

type TicketFilter struct {
	Page        int
	Limit       int
	Status      string
	Priority    string
	AssigneeID  string
	RequesterID string
	Search      string
	SortBy      string
	Order       string
}

type PaginationMeta struct {
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
	CurrentPage int   `json:"current_page"`
	Limit       int   `json:"limit"`
}

type TicketListResponse struct {
	Data []*Ticket      `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type StatsResponse struct {
	OpenTickets       int64 `json:"open_tickets"`
	InProgressTickets int64 `json:"in_progress_tickets"`
	ResolvedTickets   int64 `json:"resolved_tickets"`
	UrgentTickets     int64 `json:"urgent_tickets"`
}
