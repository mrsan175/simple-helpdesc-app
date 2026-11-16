package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"backend/internal/domain"
)

type TicketRepository struct {
	DB *sql.DB
}

func NewTicketRepository(db *sql.DB) *TicketRepository {
	return &TicketRepository{DB: db}
}

func (r *TicketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	query := `INSERT INTO tickets (id, title, description, category, priority, status, requester_id, assignee_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.DB.ExecContext(
		ctx,
		query,
		ticket.ID,
		ticket.Title,
		ticket.Description,
		ticket.Category,
		ticket.Priority,
		ticket.Status,
		ticket.RequesterID,
		ticket.AssigneeID,
		ticket.CreatedAt,
		ticket.UpdatedAt,
	)
	return err
}

func (r *TicketRepository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	query := `
		SELECT 
			t.id, t.title, t.description, t.category, t.priority, t.status, 
			t.requester_id, t.assignee_id, t.created_at, t.updated_at,
			u1.id, u1.name, u1.email, u1.role, u1.created_at,
			u2.id, u2.name, u2.email, u2.role, u2.created_at
		FROM tickets t
		JOIN users u1 ON t.requester_id = u1.id
		LEFT JOIN users u2 ON t.assignee_id = u2.id
		WHERE t.id = $1`

	var t domain.Ticket
	var reqID, reqName, reqEmail, reqRole string
	var reqCreatedAt time.Time

	var assID, assName, assEmail, assRole sql.NullString
	var assCreatedAt sql.NullTime

	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.Category, &t.Priority, &t.Status,
		&t.RequesterID, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt,
		&reqID, &reqName, &reqEmail, &reqRole, &reqCreatedAt,
		&assID, &assName, &assEmail, &assRole, &assCreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t.Requester = &domain.User{
		ID:        reqID,
		Name:      reqName,
		Email:     reqEmail,
		Role:      domain.UserRole(reqRole),
		CreatedAt: reqCreatedAt,
	}

	if assID.Valid {
		t.Assignee = &domain.User{
			ID:        assID.String,
			Name:      assName.String,
			Email:     assEmail.String,
			Role:      domain.UserRole(assRole.String),
			CreatedAt: assCreatedAt.Time,
		}
	}

	return &t, nil
}

func (r *TicketRepository) List(ctx context.Context, filter domain.TicketFilter) ([]*domain.Ticket, int64, error) {
	var whereClauses []string
	var args []interface{}
	idx := 1

	if filter.RequesterID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.requester_id = $%d", idx))
		args = append(args, filter.RequesterID)
		idx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}

	if filter.Priority != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.priority = $%d", idx))
		args = append(args, filter.Priority)
		idx++
	}

	if filter.AssigneeID != "" {
		if filter.AssigneeID == "unassigned" {
			whereClauses = append(whereClauses, "t.assignee_id IS NULL")
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("t.assignee_id = $%d", idx))
			args = append(args, filter.AssigneeID)
			idx++
		}
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(t.title) LIKE $%d", idx))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		idx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM tickets t" + whereSQL
	var total int64
	if err := r.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortBy := "t.created_at"
	switch filter.SortBy {
	case "updated_at":
		sortBy = "t.updated_at"
	case "priority":
		sortBy = "t.priority"
	case "created_at":
		sortBy = "t.created_at"
	}

	order := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		order = "ASC"
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	query := fmt.Sprintf(`
		SELECT 
			t.id, t.title, t.description, t.category, t.priority, t.status, 
			t.requester_id, t.assignee_id, t.created_at, t.updated_at,
			u1.id, u1.name, u1.email, u1.role, u1.created_at,
			u2.id, u2.name, u2.email, u2.role, u2.created_at
		FROM tickets t
		JOIN users u1 ON t.requester_id = u1.id
		LEFT JOIN users u2 ON t.assignee_id = u2.id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereSQL, sortBy, order, idx, idx+1,
	)

	queryArgs := append(args, filter.Limit, offset)
	rows, err := r.DB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []*domain.Ticket
	for rows.Next() {
		var t domain.Ticket
		var reqID, reqName, reqEmail, reqRole string
		var reqCreatedAt time.Time
		var assID, assName, assEmail, assRole sql.NullString
		var assCreatedAt sql.NullTime

		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Category, &t.Priority, &t.Status,
			&t.RequesterID, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt,
			&reqID, &reqName, &reqEmail, &reqRole, &reqCreatedAt,
			&assID, &assName, &assEmail, &assRole, &assCreatedAt,
		); err != nil {
			return nil, 0, err
		}

		t.Requester = &domain.User{
			ID:        reqID,
			Name:      reqName,
			Email:     reqEmail,
			Role:      domain.UserRole(reqRole),
			CreatedAt: reqCreatedAt,
		}

		if assID.Valid {
			t.Assignee = &domain.User{
				ID:        assID.String,
				Name:      assName.String,
				Email:     assEmail.String,
				Role:      domain.UserRole(assRole.String),
				CreatedAt: assCreatedAt.Time,
			}
		}

		tickets = append(tickets, &t)
	}

	return tickets, total, rows.Err()
}

func (r *TicketRepository) Update(ctx context.Context, ticket *domain.Ticket) error {
	query := `
		UPDATE tickets 
		SET title = $1, description = $2, category = $3, priority = $4, status = $5, assignee_id = $6, updated_at = $7
		WHERE id = $8`
	res, err := r.DB.ExecContext(
		ctx,
		query,
		ticket.Title,
		ticket.Description,
		ticket.Category,
		ticket.Priority,
		ticket.Status,
		ticket.AssigneeID,
		ticket.UpdatedAt,
		ticket.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TicketRepository) Stats(ctx context.Context, requesterID string) (*domain.StatsResponse, error) {
	where := ""
	var args []interface{}
	if requesterID != "" {
		where = " WHERE requester_id = $1"
		args = append(args, requesterID)
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END), 0) AS open_count,
			COALESCE(SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END), 0) AS progress_count,
			COALESCE(SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END), 0) AS resolved_count,
			COALESCE(SUM(CASE WHEN priority = 'urgent' AND status != 'closed' THEN 1 ELSE 0 END), 0) AS urgent_count
		FROM tickets %s`, where)

	var stats domain.StatsResponse
	err := r.DB.QueryRowContext(ctx, query, args...).Scan(
		&stats.OpenTickets,
		&stats.InProgressTickets,
		&stats.ResolvedTickets,
		&stats.UrgentTickets,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
