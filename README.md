# mini-evvy

Event seating and guest management API — organizations, events, seat maps, bookings, payments, check-in, and background jobs (finalize seating, invitation emails).

## Repository layout

| Path | Description |
|------|-------------|
| `backend/` | Go API (`cmd/api`) and worker (`cmd/worker`) |
| `frontend/` | Vue 3 organizer app (Vite, TanStack Query, Pinia, Tailwind) |
| `backend/docs/openapi.yaml` | OpenAPI 3.0 spec for frontend / client generation |
| `backend/migrations/` | PostgreSQL schema (`golang-migrate`) |

Bruno API collection (local testing): `/Users/niell/Documents/bruno/mini-evvy` (not in this repo).

## Quick start

### Prerequisites

- Go 1.22+
- PostgreSQL (e.g. Neon)
- [`golang-migrate`](https://github.com/golang-migrate/migrate)

### Environment

Copy `backend/.env.example` to `backend/.env` and set:

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | API port (default `8080`) |
| `DATABASE_URL` | Yes | Postgres connection string for app + worker |
| `MIGRATE_DATABASE_URL` | Yes | Postgres URL for migrations (can differ from pooler URL) |
| `JWT_SECRET` | Yes | HS256 secret for JWT |
| `RESEND_API_KEY` | Worker | Resend API key for invitation emails |
| `EMAIL_FROM` | Worker | Sender address (e.g. `onboarding@resend.dev`) |
| `S3_ENDPOINT` | API | Supabase Storage S3 endpoint (`…/storage/v1/s3`) |
| `S3_REGION` | API | Supabase storage region |
| `S3_ACCESS_KEY_ID` | API | Storage S3 access key |
| `S3_SECRET_ACCESS_KEY` | API | Storage S3 secret |
| `S3_BUCKET` | API | Public bucket for email banners (e.g. `email-banners`) |
| `S3_PUBLIC_BASE_URL` | API | Public object URL prefix (`…/object/public/<bucket>`) |

Quote URLs that contain `&` in `.env`.

### Database

```bash
cd backend
migrate -path migrations -database "$MIGRATE_DATABASE_URL" up
```

### Run API

```bash
cd backend
go run ./cmd/api/
```

Health: `GET http://localhost:8080/health`

### Run worker

Required for `finalize_seating` and `send_invitation` jobs:

```bash
cd backend
go run ./cmd/worker/
```

### Build

```bash
cd backend
go build ./cmd/api/ ./cmd/worker/
```

### Run frontend

Requires the API running on `http://localhost:8080`.

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api` to the backend.

**Camera check-in:** use HTTPS or `localhost` so the browser can access phone/laptop cameras. On the event Check-in page, pick a camera from the dropdown (rear camera on phones, built-in webcam on laptops).

Regenerate API types after OpenAPI changes:

```bash
cd frontend
npm run gen:api
```

## API overview (frontend)

**Base URL:** `http://localhost:8080` (or your deployed host)

**OpenAPI:** [`backend/docs/openapi.yaml`](backend/docs/openapi.yaml) — use with OpenAPI Generator, `openapi-typescript`, Swagger UI, etc.

### Response envelope

Most endpoints return JSON:

```json
{
  "success": true,
  "data": { ... }
}
```

Errors:

```json
{
  "success": false,
  "error": {
    "code": "BOOKING_NOT_FOUND",
    "message": "booking not found"
  }
}
```

`GET /health` is **not** wrapped — plain `{"status":"ok","timestamp":"..."}`.

### Authentication

1. `POST /auth/register` or `POST /auth/login` → `data.token` (JWT)
2. Send on all protected routes: `Authorization: Bearer <token>`

JWT subject is the user UUID. Token uses HS256.

### Authorization (org-scoped)

Access is tied to organization membership on the event/org resource:

| Role | Typical access |
|------|----------------|
| **owner** / **admin** | Create/update/delete resources, payments, check-in, finalize, resend |
| **member** | List/get most resources |

Exact rules per route are documented in OpenAPI `description` fields.

### Core domain flow

1. **Org** → create org, add members (`owner`, `admin`, `member`)
2. **Event** → categories → seats → guests
3. **Bookings** — manual `POST /events/{eventId}/bookings` or **finalize** `POST /events/{eventId}/finalize-seating` (worker assigns seats by `paid_date`, sends invitations)
4. **Payments** — `POST /bookings/{bookingId}/payments` with `status: "success"` marks booking paid, seat occupied, sets `guest.paid_date` if null
5. **Attendance** — check-in by `guest_id` + `seat_id` or `barcode` (paid booking required)
6. **Jobs** — poll `GET /events/{eventId}/jobs` or `GET /jobs/{jobId}` after finalize / resend

### Status enums (summary)

| Entity | Values |
|--------|--------|
| Booking | `pending`, `not_paid`, `paid`, `cancelled` |
| Seat | `available`, `reserved`, `occupied`, `blocked` |
| Payment | `pending`, `success`, `failed`, `refunded` |
| Attendance | `checked_in`, `not_checked_in` |
| Job | `pending`, `in_process`, `done`, `failed` |
| Event | `draft`, `active`, `cancelled`, `completed` |

### Soft delete

`DELETE /bookings/{id}` and `DELETE /attendance/{id}` set `deleted_at` — rows disappear from lists but remain in DB. Booking delete also cancels and releases the seat.

### Frontend client tips

- Store JWT securely (memory or httpOnly cookie if you add a BFF; API expects Bearer header today).
- Use UUID strings for all IDs in paths and bodies.
- Dates: event `start_date` / `end_date` as `YYYY-MM-DD`; guest `paid_date` as RFC3339.
- Payment `amount` is a decimal **string** (e.g. `"150000.00"`).
- After async actions (`finalize-seating`, `resend-invitation`), poll job status until `done` or `failed`.
- Check-in UI: support barcode scan → `POST .../attendance` with `barcode` only.

Generate TypeScript types:

```bash
npx openapi-typescript backend/docs/openapi.yaml -o src/api/schema.d.ts
```

## Testing with Bruno

1. Open collection at your Bruno `mini-evvy` folder
2. Environment **local** → `BASE_URL=http://localhost:8080`
3. Run `auth/login` → saves `AUTH_TOKEN`
4. Run org → event → category → seat → guest → booking flow
5. See prior smoke checklist in chat / run folder sequentially

## Migrations

| Version | Description |
|---------|-------------|
| `000001` | Initial schema |
| `000002` | `jobs.type`, `seat_bookings.barcode` |

## Deployment (Docker + GitHub Actions)

Production runs on a Linux VPS (recommended: 2 vCPU / 4 GB RAM) with **managed Postgres** (Neon, etc.) — do not run Postgres on the same small VPS.

### Architecture

- **frontend** container: Caddy serves the Vue SPA and proxies `/api/*` to the API
- **api** and **worker** containers: pull from GHCR, share `/opt/mini-evvy/.env`
- Push to `main` runs tests, builds three images, pushes to GHCR, SSH-deploys via [`docker-compose.yml`](docker-compose.yml)

### One-time VPS setup

1. Install Docker and Docker Compose on Ubuntu LTS.
2. Create app directory and production env:

```bash
sudo mkdir -p /opt/mini-evvy
sudo chown $USER:$USER /opt/mini-evvy
cp backend/.env.example /opt/mini-evvy/.env
# Edit /opt/mini-evvy/.env — set DATABASE_URL, JWT_SECRET, RESEND_*, S3_*, etc.
```

3. Add `GHCR_OWNER` to `/opt/mini-evvy/.env` (your GitHub username or org — matches `github.repository_owner` in CI):

```bash
GHCR_OWNER=your-github-username
```

4. Log in to GHCR on the VPS (required if packages are private):

```bash
echo <PAT_with_read:packages> | docker login ghcr.io -u your-github-username --password-stdin
```

Alternatively, make the three `mini-evvy-*` GHCR packages **public** to skip `docker login` on the server.

5. Open firewall: **80** (and **443** when you add TLS), **22** for SSH. Do not expose API port 8080 publicly.

### GitHub Actions secrets

Add in repo **Settings → Secrets and variables → Actions**:

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | VPS IP or hostname |
| `VPS_USERNAME` | SSH user (e.g. `ubuntu`) |
| `VPS_SSH_KEY` | Private key (PEM) for that user |
| `VPS_PORT` | Optional SSH port (default **22** if not set in workflow; add `port:` to deploy job if not 22) |

### Deploy

Push to `main`. Workflow [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml):

1. `go test ./...` in `backend/`
2. Build and push `mini-evvy-api`, `mini-evvy-worker`, `mini-evvy-frontend` to `ghcr.io/<owner>/...`
3. SCP `docker-compose.yml` to `/opt/mini-evvy/`
4. `docker compose pull && docker compose up -d --remove-orphans`

Manual deploy on the VPS:

```bash
cd /opt/mini-evvy
docker compose pull
docker compose up -d --remove-orphans
```

### TLS / custom domain

[`frontend/Caddyfile`](frontend/Caddyfile) listens on `:80` by default. When you have a domain, replace `:80` with your hostname so Caddy issues Let's Encrypt certificates, and add `443:443` to the `frontend` service in `docker-compose.yml`.

### Local Docker smoke test

```bash
# From repo root, with backend/.env copied to .env and GHCR_OWNER set
docker compose up -d
# Open http://localhost (frontend proxies /api to api service)
```
