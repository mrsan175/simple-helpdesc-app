package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, role string) ([]*User, error)
	UpdateRole(ctx context.Context, id string, role UserRole) error
	CountByRole(ctx context.Context, role UserRole) (int64, error)
}

type TicketRepository interface {
	Create(ctx context.Context, ticket *Ticket) error
	FindByID(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context, filter TicketFilter) ([]*Ticket, int64, error)
	Update(ctx context.Context, ticket *Ticket) error
	Stats(ctx context.Context, requesterID string) (*StatsResponse, error)
	AdminStats(ctx context.Context, slaHours float64) (*AdminDetailedStats, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, actor *User, att *Attachment) error
	ListByTicket(ctx context.Context, ticketID string) ([]*Attachment, error)
	ListByComment(ctx context.Context, commentID string) ([]*Attachment, error)
	FindByID(ctx context.Context, id string) (*Attachment, error)
	Replace(ctx context.Context, actor *User, att *Attachment) (oldPath string, err error)
	Delete(ctx context.Context, actor *User, id string) (*Attachment, error)
}

type CommentRepository interface {
	Create(ctx context.Context, actor *User, comment *Comment) error
	ListByTicket(ctx context.Context, ticketID string, includeInternal bool) ([]*Comment, error)
	FindByID(ctx context.Context, id string) (*Comment, error)
	Update(ctx context.Context, actor *User, comment *Comment) error
	Delete(ctx context.Context, actor *User, id string) ([]*Attachment, error)
}

type AuthService interface {
	Register(ctx context.Context, req RegisterReq) (*AuthResponse, error)
	Login(ctx context.Context, req LoginReq) (*AuthResponse, error)
}

type TicketService interface {
	List(ctx context.Context, actor *User, filter TicketFilter) (*TicketListResponse, error)
	Get(ctx context.Context, actor *User, id string) (*Ticket, error)
	Create(ctx context.Context, actor *User, req CreateTicketReq) (*Ticket, error)
	Update(ctx context.Context, actor *User, id string, req UpdateTicketReq) (*Ticket, error)
	AddComment(ctx context.Context, actor *User, ticketID string, req CreateCommentReq) (*Comment, error)
	UpdateComment(ctx context.Context, actor *User, id string, req UpdateCommentReq) (*Comment, error)
	DeleteComment(ctx context.Context, actor *User, id string) ([]*Attachment, error)
	Stats(ctx context.Context, actor *User) (*StatsResponse, error)
	AdminStats(ctx context.Context, actor *User) (*AdminDetailedStats, error)
	AddAttachment(ctx context.Context, actor *User, ticketID string, att *Attachment) (*Attachment, error)
	GetAttachment(ctx context.Context, actor *User, attachmentID string) (*Attachment, error)
	CheckAttachmentUpload(ctx context.Context, actor *User, ticketID string) error
	CheckCommentAttachmentUpload(ctx context.Context, actor *User, commentID string) error
	CheckAttachmentUpdate(ctx context.Context, actor *User, id string) error
	AddCommentAttachment(ctx context.Context, actor *User, commentID string, att *Attachment) (*Attachment, error)
	ReplaceAttachment(ctx context.Context, actor *User, id string, file *Attachment) (*Attachment, string, error)
	DeleteAttachment(ctx context.Context, actor *User, id string) (*Attachment, error)
}

type UserService interface {
	List(ctx context.Context, role string) ([]*User, error)
	UpdateRole(ctx context.Context, actor *User, id string, role UserRole) error
}
