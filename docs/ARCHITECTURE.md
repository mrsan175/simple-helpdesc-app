# Architecture Notes

## Backend (Go)

Clean/layered architecture with strict dependency inversion:

- **Domain** (`internal/domain`): pure structs, errors, interfaces. No external dependencies.
- **Repository** (`internal/repository`): raw SQL operations. No business logic.
- **Service** (`internal/service`): business logic, state machine validation, RBAC.
- **Handler** (`internal/handler`): HTTP parsing, response mapping, error → HTTP status.
- **Middleware** (`internal/middleware`): JWT auth, role guards.

### Key Decisions

- **JWT** via `golang-jwt/jwt/v5` with HMAC-SHA256.
- **Password hashing** via `bcrypt`.
- **UUID** for all primary keys (prevents IDOR).
- **State machine** isolated in `service.ValidateTransition` — easy to unit test without DB.
- **Idempotent migrations** via `schema_migrations` table.

## Frontend (Vue 3)

Feature-driven clean architecture:

- **Core** (`src/services/api.js`): single Axios instance with interceptors.
- **Stores** (`src/stores/`): Pinia state (auth, etc.).
- **Composables** (`src/composables/`): reusable UI logic.
- **Components** (`src/components/`): dumb UI components.
- **Views** (`src/views/`): page-level orchestration.

### Key Decisions

- **Vue Router** with `requiresAuth`, `guestOnly`, `roles` meta.
- **Axios interceptor** auto-logs out on 401.
- **Client-side transition map** (`src/constants/ticket.js`) for rendering only valid action buttons.
- **Tailwind CSS** with custom `.card`, `.btn`, `.badge` utility classes.

## Database

- PostgreSQL 16 with `gen_random_uuid()` for UUID generation.
- Idempotent migrations tracked in `schema_migrations`.
- Seeded with 3 default users + 5 sample tickets on first run.