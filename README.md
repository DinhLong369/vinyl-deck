# Vinyl Deck

Turntable-style focus music player. Records are YouTube mixes; audio plays through a hidden YouTube IFrame player and the turntable is the UI.

- `backend/` Go 1.22+ (stdlib only) JSON API, records stored in a JSON file
- `frontend/` Nuxt 3 SPA

## Run locally

```bash
# API on :8080 with live reload (data in backend/data/records.json)
go install github.com/air-verse/air@latest   # once
cd backend && air

# Web on :3000
cd frontend && yarn install && yarn dev
```

Or `docker compose up --build`.

## API

| Method | Path | Body | Notes |
|---|---|---|---|
| GET | /api/records | | shelf, ordered |
| POST | /api/records | `{"url": "<youtube link>"}` | validates via YouTube oEmbed; 400 bad link, 409 duplicate, 422 unavailable |
| DELETE | /api/records/{id} | | |
| PUT | /api/records/order | `{"ids": [...]}` | reorder shelf |
| GET | /api/health | | |

Env: `ADDR` (`:8080`), `DATA_FILE` (`data/records.json`), `CORS_ORIGIN` (`*`). Frontend: `NUXT_PUBLIC_API_BASE`.

Note: some videos (most music-label uploads) block embedding; the plate shows a message when that happens.

## Free deployment: GitHub Pages (no backend)

Without `NUXT_PUBLIC_API_BASE` the frontend runs on its own: each visitor's shelf is stored in their browser,
starting from `frontend/public/records.json` (edit that file to change the default shelf). Link validation and
titles come straight from YouTube oEmbed in the browser.

1. Create a GitHub repo and push this project to the `main` branch.
2. Repo **Settings → Pages → Build and deployment → Source: GitHub Actions**.
3. Every push to `main` runs `.github/workflows/deploy-pages.yml` and publishes to
   `https://<user>.github.io/<repo>/`.

Local development still uses the Go API through `frontend/.env` (`NUXT_PUBLIC_API_BASE=http://localhost:8080`).
Delete or empty that line to try browser-only mode locally.
