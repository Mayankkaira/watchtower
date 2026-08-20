# Watchtower

Uptime monitoring SaaS built with Go and vanilla JS. Single binary, no frameworks, no build step.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-WAL-003B57?logo=sqlite&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)

## What it does

- HTTP/TCP/DNS endpoint monitoring
- Dashboard with monitor status, uptime %, response times
- API key management (`sk_*` keys)
- Plan tiers (free / pro / enterprise) with monitor limits
- Account management (signup, login, profile edit, delete)
- Dark theme SPA frontend — no React, no Vue, just HTML/CSS/JS

## Stack

| What | How |
|---|---|
| Backend | Go stdlib `net/http` (Go 1.22+ routing) |
| Database | SQLite3 with WAL mode |
| Auth | JWT (HS256) + bcrypt passwords + API keys |
| Frontend | Vanilla HTML/CSS/JS, single `index.html` SPA |
| Build | `go build` → one binary |

No ORM. No router library. No CSS framework. ~3200 lines total.

## Quick start

```bash
git clone https://github.com/Mayankkaira/watchtower.git
cd watchtower
go build -o watchtower .
./watchtower
```

Open [http://localhost:8080](http://localhost:8080)

**Access from phone (same WiFi):** [http://192.168.1.96:8080](http://192.168.1.96:8080)

## Environment variables

| Variable | Default | What it does |
|---|---|---|
| `PORT` | `8080` | Server port |
| `JWT_SECRET` | `change-me-...` | JWT signing key (change this!) |
| `DB_PATH` | `saas.db` | SQLite file path |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |

For production:

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
export ALLOWED_ORIGINS="https://yourdomain.com"
./watchtower
```

## API routes

**Public:**

```
POST   /api/v1/auth/signup    — create account
POST   /api/v1/auth/login     — get JWT token
GET    /health                 — server health
```

**Authenticated** (Bearer token or `X-API-Key` header):

```
GET    /api/v1/me              — your profile
PUT    /api/v1/me              — update name/email
DELETE /api/v1/me              — delete account
PUT    /api/v1/me/plan         — change plan
GET    /api/v1/dashboard       — dashboard data

POST   /api/v1/monitors        — add monitor
GET    /api/v1/monitors        — list monitors
DELETE /api/v1/monitors?id=N   — remove monitor

POST   /api/v1/keys            — create API key
GET    /api/v1/keys            — list API keys
DELETE /api/v1/keys?id=N       — revoke API key
```

**Pro/Enterprise only:**

```
GET    /api/v1/pro/analytics   — analytics data
```

## Project structure

```
├── main.go              — entry point, routing, config
├── auth/auth.go         — JWT + bcrypt + API key gen
├── store/store.go       — SQLite schema + all queries
├── handlers/
│   ├── auth.go          — signup, login
│   ├── api.go           — everything else
│   └── health.go        — health check
├── middleware/
│   └── middleware.go     — CORS, rate limit, auth, logging
└── frontend/
    ├── index.html       — all 7 pages in one file
    ├── style.css        — dark theme design system
    └── app.js           — SPA routing + API client
```

## Monitor limits by plan

| Plan | Monitors | Check interval |
|---|---|---|
| Free | 5 | 5 min |
| Pro | 50 | 30 sec |
| Enterprise | Unlimited | 10 sec |

## Security stuff

- Passwords hashed with bcrypt (cost 10)
- JWT tokens expire in 24h
- All request bodies capped at 1 MB
- Rate limited to 120 req/min per IP
- SQLite foreign keys ON — deleting a user cascades to their monitors and API keys
- Server warns on startup if you're using default JWT secret or wildcard CORS

## License

MIT
