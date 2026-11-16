# Simple Journey Indonesia – Internal Helpdesk System

Aplikasi web helpdesk internal untuk menggantikan pelacakan tiket dukungan karyawan berbasis email dan spreadsheet.

---

## 1. Tech Stack

- **Backend:** Go 1.22+ (Clean/layered architecture: handler → service → repository → domain)
- **Frontend:** Vue 3 + Vite + Pinia + Vue Router + Tailwind CSS
- **Database:** PostgreSQL 16
- **Containerization:** Docker + docker-compose

---

## 2. Quick Start

### Prerequisites
- Docker & docker-compose
- Go 1.22+ (optional, for local backend development)

### With Docker (Recommended)

```bash
docker compose up --build
```

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:3000`

### With Go (Local)

```bash
# Backend
cd backend
go mod tidy
go build -o server ./cmd/api
./server

# Frontend
cd frontend
npm install
npm run dev
```

---

## 3. Seed Data (Default Accounts)

| Role   | Email                        | Password     |
|--------|------------------------------|--------------|
| Admin  | admin@simplejourney.id       | Password123! |
| Agent  | agent@simplejourney.id       | Password123! |
| Requester A (Budi) | requester1@simplejourney.id | Password123! |
| Requester B (Siti) | requester2@simplejourney.id | Password123! |
| Requester (legacy) | requester@simplejourney.id | Password123! |

---

## 4. API Endpoints

### Auth
- `POST /api/auth/register` – Register new user (role auto: requester)
- `POST /api/auth/login` – Login with email & password
- `GET /api/me` – Get current user profile

### Tickets
- `GET /api/tickets` – List tickets (with pagination, filters, search)
- `POST /api/tickets` – Create new ticket
- `GET /api/tickets/{id}` – Get ticket detail (with comments)
- `PATCH /api/tickets/{id}` – Update ticket (status, priority, assignee)
- `POST /api/tickets/{id}/comments` – Add comment
- `GET /api/stats` – Stats summary (agent/admin only)

### Users (Admin only)
- `GET /api/users` – List users (filter by role)
- `PATCH /api/users/{id}/role` – Update user role

---

## 5. State Machine Rules

| Current      | Target       | Allowed Roles        |
|--------------|--------------|----------------------|
| open         | in_progress  | agent, admin         |
| open         | closed       | requester, admin     |
| in_progress  | resolved     | agent, admin         |
| resolved     | closed       | requester, agent, admin |
| resolved     | open         | requester            |

---

## 6. Tests

```bash
cd backend
go test ./...
```

Tests cover:
- Auth registration & login
- Ticket isolation (requester cannot read others' tickets)
- Valid & invalid status transitions
- Internal note security

---

## 7. Project Structure

See `docs/ARCHITECTURE.md` for detailed architecture notes.