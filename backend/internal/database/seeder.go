package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"backend/internal/domain"
)

const seedPassword = "Password123!"

func Seed(db *sql.DB) error {
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	users := []struct {
		name  string
		email string
		role  domain.UserRole
	}{
		{"Admin Simple Journey", "admin@simplejourney.id", domain.RoleAdmin},
		{"Agent Simple Journey", "agent@simplejourney.id", domain.RoleAgent},
		{"Requester A (Budi)", "requester1@simplejourney.id", domain.RoleRequester},
		{"Requester B (Siti)", "requester2@simplejourney.id", domain.RoleRequester},
	}

	var ids = make(map[string]string)
	for _, u := range users {
		var id string
		err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, u.email).Scan(&id)
		if err == sql.ErrNoRows {
			id = uuid.NewString()
			_, err = db.ExecContext(ctx,
				`INSERT INTO users (id, name, email, password_hash, role, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
				id, u.name, u.email, string(hash), u.role, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("seed user %s: %w", u.email, err)
			}
			log.Printf("seeded user: %s (%s)", u.email, u.role)
		} else if err != nil {
			return fmt.Errorf("lookup seed user %s: %w", u.email, err)
		}
		ids[u.email] = id
	}

	samples := []struct {
		title          string
		description    string
		category       string
		priority       domain.TicketPriority
		status         domain.TicketStatus
		requesterEmail string
		assignee       interface{}
	}{
		{"Laptop Budi tidak bisa terhubung ke Wi-Fi kantor", "Sejak pagi laptop saya tidak menemukan SSID kantor. Sudah restart tapi tetap tidak muncul.", "Network", domain.PriorityHigh, domain.StatusOpen, "requester1@simplejourney.id", nil},
		{"Akses VPN Budi habis masa berlaku", "VPN token saya kadaluarsa dan tidak bisa login dari rumah. Mohon bantuan reset.", "IT Support", domain.PriorityUrgent, domain.StatusOpen, "requester1@simplejourney.id", nil},
		{"Printer lantai 3 macet (dilaporkan Budi)", "Printer di lantai 3 menampilkan error paper jam padahal tidak ada kertas tersangkut.", "Hardware", domain.PriorityMedium, domain.StatusInProgress, "requester1@simplejourney.id", ids["agent@simplejourney.id"]},

		{"Permintaan lisensi Figma dari Siti", "Tim desain butuh 1 lisensi Figma Professional tambahan untuk bulan ini.", "Software", domain.PriorityLow, domain.StatusResolved, "requester2@simplejourney.id", ids["agent@simplejourney.id"]},
		{"Kursi kerja Siti rusak", "Kursi di meja saya hidroliknya tidak bisa naik turun. Mohon diganti.", "HR/Facility", domain.PriorityMedium, domain.StatusClosed, "requester2@simplejourney.id", ids["agent@simplejourney.id"]},
		{"Monitor Siti berkedip saat digunakan", "Monitor eksternal berkedip setiap beberapa menit setelah docking station terpasang.", "Hardware", domain.PriorityMedium, domain.StatusOpen, "requester2@simplejourney.id", nil},
	}

	for _, s := range samples {
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tickets WHERE title = $1)`, s.title).Scan(&exists); err != nil {
			return fmt.Errorf("check seed ticket %q: %w", s.title, err)
		}
		if exists {
			continue
		}

		now := time.Now().UTC()
		if _, err := db.ExecContext(ctx,
			`INSERT INTO tickets (id, title, description, category, priority, status, requester_id, assignee_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			uuid.NewString(), s.title, s.description, s.category, s.priority, s.status,
			ids[s.requesterEmail], s.assignee, now, now); err != nil {
			return fmt.Errorf("seed ticket %q: %w", s.title, err)
		}
	}

	return nil
}
