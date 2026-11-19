# Architecture Notes

## Backend (Go)

Clean/layered architecture with strict dependency inversion:

- **Domain** (`internal/domain`): pure structs, errors, interfaces. No external dependencies.
- **Repository** (`internal/repository`): raw SQL operations. No business logic.
- **Service** (`internal/service`): business logic, state machine validation, RBAC, SLA check, optional ticket priority (assigned later by agent/admin).
- **Handler** (`internal/handler`): HTTP parsing, multipart upload validation, SSE event streaming, error → HTTP status.
- **Middleware** (`internal/middleware`): JWT auth (header and query token for EventSource), role guards.
- **Realtime** (`internal/realtime`): in-memory pub/sub broker for SSE fan-out with role/user audience scoping (ticket_created scoped to agent/admin; updates/comments scoped to staff + ticket requester).

### Key Decisions

- **JWT** via `golang-jwt/jwt/v5` with HMAC-SHA256 (supports header & query param for EventSource).
- **Password hashing** via `bcrypt`.
- **UUID** for all primary keys (prevents IDOR).
- **State machine** isolated in `service.ValidateTransition` — easy to unit test without DB.
- **Idempotent migrations** via `schema_migrations` table.
- **Real-time SSE** via standard HTTP streaming (`c.Stream`), zero third-party dependencies.
- **File Uploads**: stored in configured upload directory with UUID prefix, extension allowlist, 5 MiB limit. MIME type comes from the extension allowlist, not from content sniffing.
- **Attachments**: main ticket files have `comment_id = NULL`; comment files reference the comment and inherit its visibility. Both support download and inline preview (`GET /api/attachments/{id}/preview`) with `nosniff` and a sandbox CSP.
- **Closed discussions**: comments and comment attachments become read-only when a ticket is Closed; main ticket attachments remain manageable by the ticket owner.
- **Optional priority**: `tickets.priority` is nullable (migration `0003`); new tickets start without priority and agent/admin set it later. Empty values are stored as `NULL` via `NULLIF(..., '')`; reads use `COALESCE(priority, '')`.

## Frontend (Vue 3)

Feature-driven clean architecture:

- **Core** (`src/services/api.js`): single Axios instance with interceptors.
- **Stores** (`src/stores/`): Pinia state (auth, etc.).
- **Composables** (`src/composables/`): reusable UI logic.
- **Components** (`src/components/`): dumb UI components.
- **Views** (`src/views/`): page-level orchestration.

### Key Decisions

- **Vue Router** with `requiresAuth`, `guestOnly`, `roles` meta; global route guards also drive the top loading bar.
- **Axios interceptor** auto-logs out on 401.
- **Client-side transition map** (`src/constants/ticket.js`) for rendering only valid action buttons.
- **Tailwind CSS** with custom `.card`, `.btn`, `.badge` utility classes.
- **Priority optional**: UI renders an "Unset" badge for empty priority; agents/admins can set it from the ticket detail page.
- **Attachment preview**: ticket, comment, and draft files can be previewed in-app without downloading; every expanded ticket row is marked in the desktop table.

## Database

- PostgreSQL 16 with `gen_random_uuid()` for UUID generation.
- Idempotent migrations tracked in `schema_migrations` (init, attachments, optional priority, comment attachments).
- Seeded with 4 default users + 13 sample tickets on first run.