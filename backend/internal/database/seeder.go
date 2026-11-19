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
		{"Admin Helpdesk", "admin@helpdesk.local", domain.RoleAdmin},
		{"Agent Helpdesk", "agent@helpdesk.local", domain.RoleAgent},
		{"Requester A (Budi)", "requester1@helpdesk.local", domain.RoleRequester},
		{"Requester B (Siti)", "requester2@helpdesk.local", domain.RoleRequester},
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
		ageHours       int
	}{
		{"Laptop Budi tidak bisa terhubung ke Wi-Fi kantor", "Sejak pagi laptop saya tidak menemukan SSID kantor. Sudah restart tapi tetap tidak muncul.", "Network", domain.PriorityHigh, domain.StatusOpen, "requester1@helpdesk.local", nil, 5},
		{"Akses VPN Budi habis masa berlaku", "VPN token saya kadaluarsa dan tidak bisa login dari rumah. Mohon bantuan reset.", "IT Support", domain.PriorityUrgent, domain.StatusOpen, "requester1@helpdesk.local", nil, 30},
		{"Printer lantai 3 macet (dilaporkan Budi)", "Printer di lantai 3 menampilkan error paper jam padahal tidak ada kertas tersangkut.", "Hardware", domain.PriorityMedium, domain.StatusInProgress, "requester1@helpdesk.local", ids["agent@helpdesk.local"], 26},
		{"Email Budi tidak bisa mengirim lampiran besar", "Gagal mengirim email dengan lampiran di atas 10 MB, pesan error 'message too large'.", "IT Support", domain.PriorityHigh, domain.StatusOpen, "requester1@helpdesk.local", nil, 8},
		{"Keyboard laptop Budi beberapa tombol tidak berfungsi", "Tombol spasi dan enter sering tidak merespons, sudah dibersihkan tapi tetap bermasalah.", "Hardware", domain.PriorityLow, domain.StatusResolved, "requester1@helpdesk.local", ids["agent@helpdesk.local"], 50},
		{"Akses Google Drive Budi terblokir", "Tidak bisa membuka drive.google.com, muncul halaman blocked by administrator.", "Network", domain.PriorityMedium, domain.StatusClosed, "requester1@helpdesk.local", ids["agent@helpdesk.local"], 72},

		{"Permintaan lisensi Figma dari Siti", "Tim desain butuh 1 lisensi Figma Professional tambahan untuk bulan ini.", "Software", domain.PriorityLow, domain.StatusResolved, "requester2@helpdesk.local", ids["agent@helpdesk.local"], 48},
		{"Kursi kerja Siti rusak", "Kursi di meja saya hidroliknya tidak bisa naik turun. Mohon diganti.", "HR/Facility", domain.PriorityMedium, domain.StatusClosed, "requester2@helpdesk.local", ids["agent@helpdesk.local"], 96},
		{"Monitor Siti berkedip saat digunakan", "Monitor eksternal berkedip setiap beberapa menit setelah docking station terpasang.", "Hardware", domain.PriorityMedium, domain.StatusOpen, "requester2@helpdesk.local", nil, 12},
		{"Siti lupa password akun HRIS", "Password HRIS saya tidak diterima setelah reset via email, tolong dibukakan kembali.", "IT Support", domain.PriorityHigh, domain.StatusInProgress, "requester2@helpdesk.local", ids["agent@helpdesk.local"], 20},
		{"VPN Siti sering terputus saat video call", "Koneksi VPN drop setiap 5-10 menit saat meeting, sinyal Wi-Fi stabil saja.", "Network", domain.PriorityMedium, domain.StatusOpen, "requester2@helpdesk.local", nil, 3},
		{"Permintaan instalasi antivirus untuk laptop Siti", "Laptop baru belum terpasang antivirus standar perusahaan, mohon dibantu instal.", "Software", domain.PriorityLow, domain.StatusResolved, "requester2@helpdesk.local", ids["agent@helpdesk.local"], 80},
		{"Meja meeting lantai 2 rusak", "Salah satu kaki meja meeting lantai 2 patah, tidak aman digunakan rapat.", "HR/Facility", domain.PriorityMedium, domain.StatusOpen, "requester2@helpdesk.local", nil, 15},
	}

	for _, s := range samples {
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tickets WHERE title = $1)`, s.title).Scan(&exists); err != nil {
			return fmt.Errorf("check seed ticket %q: %w", s.title, err)
		}
		if exists {
			continue
		}

		createdAt := time.Now().UTC().Add(-time.Duration(s.ageHours) * time.Hour)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO tickets (id, title, description, category, priority, status, requester_id, assignee_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			uuid.NewString(), s.title, s.description, s.category, s.priority, s.status,
			ids[s.requesterEmail], s.assignee, createdAt, createdAt); err != nil {
			return fmt.Errorf("seed ticket %q: %w", s.title, err)
		}
	}

	return nil
}
