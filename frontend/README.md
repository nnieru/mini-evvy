# mini-evvy frontend

Vue 3 organizer app for the mini-evvy API.

## Stack

- Vue 3 + Vite + TypeScript
- TanStack Query (server state)
- Pinia (auth + UI state)
- Tailwind CSS
- `html5-qrcode` for camera barcode check-in

## Setup

```bash
cp .env.example .env
npm install
npm run dev
```

API must run at `http://localhost:8080`. Requests go through Vite proxy at `/api`.

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Dev server |
| `npm run build` | Production build |
| `npm run preview` | Preview production build |
| `npm run gen:api` | Regenerate types from `../backend/docs/openapi.yaml` |

## Architecture

Feature-based layout under `src/features/`:

- `auth` — login, register, session
- `organizations` — orgs and members
- `events` — event CRUD, finalize seating
- `categories`, `seats`, `guests` — event setup
- `bookings`, `payments` — reservations and payments
- `attendance` — camera/manual check-in
- `jobs` — background job status

Shared layers: `src/shared/api`, `src/shared/ui`, `src/shared/lib`.

## Camera check-in

On `/events/:eventId/attendance`:

1. **Camera** — scan QR/barcode via device camera (phone or laptop)
2. **Manual** — type or USB wedge scanner
3. **Guest + seat** — fallback when barcode unavailable

Requires secure context (`localhost` or HTTPS).
