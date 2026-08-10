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
| `RESEND_API_KEY` | API + Worker | Resend API key for invitation emails |
| `EMAIL_FROM` | API + Worker | Sender address — see [Resend email](#resend-email) below |
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

### Resend email

Invitation emails use [Resend](https://resend.com). Test sends (API) go to the logged-in user; guest invitations (worker) go to the guest address.

**Sandbox / testing address:** If `EMAIL_FROM` is `onboarding@resend.dev` (or any unverified domain), Resend only allows sending to the **email on your Resend account**. Other recipients fail with a Resend API error (often HTTP 403).

**Production (evvy.fun):** In Resend → Domains, add and verify `evvy.fun` (DNS SPF/DKIM). Then set on the VPS:

```bash
EMAIL_FROM=Mini Evvy <noreply@evvy.fun>
```

Restart api and worker after changing env:

```bash
cd /opt/mini-evvy
sudo docker compose up -d api worker
```

**Troubleshooting failed sends:** Resend returns the error body in app logs:

```bash
sudo docker compose logs worker --tail 100 | grep -i resend
sudo docker compose logs api --tail 100 | grep -i resend
```

Look for `resend send failed` / `resend test send failed` with the full `resend api error: status …` message.

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

4. **GHCR pull access** — pick one:

   **A. Public packages (simplest)** — GitHub → **Packages** → each of `mini-evvy-api`, `mini-evvy-worker`, `mini-evvy-frontend` → **Package settings** → **Change visibility** → Public. No login on VPS.

   **B. Private packages** — create a GitHub PAT with `read:packages`, add repo secret `GHCR_PAT`. CI logs into GHCR on the VPS before `docker compose pull`. For manual pulls on the VPS:

```bash
echo <PAT_with_read:packages> | docker login ghcr.io -u your-github-username --password-stdin
```

5. Open firewall: **80**, **443**, **22** (SSH). Do not expose API port 8080 publicly. Uptime Kuma: use `https://kuma.evvy.fun` after DNS, or port **3001** (restrict to your IP if exposed).

### GitHub Actions secrets

Add in repo **Settings → Secrets and variables → Actions**:

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | VPS IP or hostname |
| `VPS_USERNAME` | SSH user (e.g. `ubuntu`) |
| `VPS_SSH_KEY` | Private key (PEM) for that user |
| `GHCR_PAT` | GitHub PAT with `read:packages` (required if GHCR images are **private**; omit if packages are public) |
| `VPS_PORT` | Optional SSH port (default **22** if not set in workflow; add `port:` to deploy job if not 22) |

### Deploy

Push to `main`. Workflow [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml):

1. `go test ./...` in `backend/`
2. Build and push `mini-evvy-api`, `mini-evvy-worker`, `mini-evvy-frontend` to `ghcr.io/<owner>/...`
3. SCP `docker-compose.yml` to `/opt/mini-evvy/`
4. `docker login ghcr.io` (if private) + `docker compose pull && docker compose up -d --remove-orphans`

Manual deploy on the VPS (after `docker login ghcr.io` if packages are private):

```bash
cd /opt/mini-evvy
docker compose pull
docker compose up -d --remove-orphans
```

### Subpath deploy (`/mini-evvy`)

The app is configured for **`https://evvy.fun/mini-evvy/`** (subpath, not site root), so you can host other apps on the same VPS later.

| File | Setting |
|------|---------|
| [`frontend/Caddyfile`](frontend/Caddyfile) | `evvy.fun` + `/mini-evvy/api/*` → API, `/mini-evvy/*` → SPA; `kuma.evvy.fun` → Uptime Kuma |
| [`frontend/.env.example`](frontend/.env.example) | `VITE_BASE_PATH=/mini-evvy`, `VITE_API_BASE_URL=/mini-evvy/api` |
| [`frontend/Dockerfile`](frontend/Dockerfile) | Same values as build `ARG`s |

**URLs after deploy:**

- App: `https://evvy.fun/mini-evvy/`
- Health: `https://evvy.fun/mini-evvy/api/health`
- Uptime Kuma: `https://kuma.evvy.fun` (or `http://YOUR_VPS_IP:3001`)

Plain IP still works on HTTP: `http://YOUR_VPS_IP/mini-evvy/`

Local dev: copy `frontend/.env.example` to `frontend/.env`, then `npm run dev` → `http://localhost:5173/mini-evvy/`

**Adding another app later:** keep mini-evvy on `/mini-evvy`, add a root Caddy (or nginx) on the host that routes `/mini-evvy` → this stack and `/other-app` → another container. Do not publish two apps both on host `:80` without a path-based router.

### DNS for evvy.fun

At your domain registrar, point records to your VPS IP (e.g. `43.133.134.234`):

| Type | Name | Value |
|------|------|--------|
| A | `@` | VPS IP |
| A | `www` | VPS IP |
| A | `kuma` | VPS IP |

Wait for DNS to propagate (`dig +short evvy.fun`). Open Lighthouse firewall **443** and **80** (443 is required for `https://` and for many browsers that default to HTTPS).

**If the site “does not load”:** port **443** is often still closed in the Lighthouse console while **80** is open. Caddy may redirect to HTTPS; if 443 is blocked, the browser hangs. After deploying the Caddyfile with `auto_https disable_redirects`, `http://evvy.fun/mini-evvy/` works on port 80 even before 443 is open. Verify ports from your laptop:

```bash
nc -zv evvy.fun 80
nc -zv evvy.fun 443   # must succeed for https://
```

### TLS

[`frontend/Caddyfile`](frontend/Caddyfile) uses `evvy.fun`, `www.evvy.fun`, and `kuma.evvy.fun`; Caddy requests Let's Encrypt certificates automatically. [`docker-compose.yml`](docker-compose.yml) maps **443** on the `frontend` service and mounts named volumes `caddy_data` / `caddy_config` so certs survive recreate/redeploy.

After DNS works, rebuild and redeploy the **frontend** image, then:

```bash
cd /opt/mini-evvy
sudo docker compose pull frontend
sudo docker compose up -d frontend
```

Test: `curl -I https://evvy.fun/mini-evvy/api/health`

**Rate limit (`HTTP 429` / too many certificates):** Let's Encrypt allows only **5 new certs per exact hostname set per 168h**. If the frontend container was recreated without persistent `/data`, Caddy keeps minting new certs and gets blocked. Fix: keep the `caddy_data` / `caddy_config` volumes (above), then wait until the `retry after` time in the logs (do not keep restarting frontend hoping it helps). HTTP still works via `auto_https disable_redirects` while waiting.

### Local Docker smoke test

```bash
# From repo root, with backend/.env copied to .env and GHCR_OWNER set
docker compose up -d
# Open http://localhost/mini-evvy/ (frontend proxies /mini-evvy/api to api service)
```

### Monitoring (Uptime Kuma)

[`docker-compose.yml`](docker-compose.yml) includes **Uptime Kuma** on port **3001** (Docker Hub image, not GHCR).

**Start / update on VPS:**

```bash
cd /opt/mini-evvy
sudo docker compose pull uptime-kuma   # optional; pulls latest Kuma image
sudo docker compose up -d uptime-kuma
```

**UI:** `https://kuma.evvy.fun` (after DNS + frontend redeploy) or `http://YOUR_VPS_IP:3001`

**Firewall (recommended):** allow `3001` only from your IP, or use SSH tunnel:

```bash
ssh -L 3001:127.0.0.1:3001 ubuntu@YOUR_VPS_IP
# then open http://localhost:3001
```

**Monitors to add** (Monitor Type → HTTP(s)):

| Name | URL | Notes |
|------|-----|--------|
| mini-evvy API health | `https://evvy.fun/mini-evvy/api/health` | Keyword: `ok` (optional) |
| mini-evvy SPA | `https://evvy.fun/mini-evvy/` | Expect 200 |
| mini-evvy login | `https://evvy.fun/mini-evvy/login` | Expect 200 |

Use your public IP or domain. Interval **60s** is fine; enable email/Telegram/Discord notification in **Settings → Notifications**.

**Internal check (optional):** from the same Docker network Kuma can hit `http://frontend/mini-evvy/api/health` — useful to detect Caddy-only issues vs public routing; public URLs are what matter for real availability.

Data persists in Docker volume `uptime-kuma` across restarts.
