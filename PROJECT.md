# Watchtower — Project Documentation

> Uptime monitoring, incident intelligence, and SLA tracking for teams that ship fast.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
  - [System Diagram](#system-diagram)
  - [Request Lifecycle](#request-lifecycle)
  - [Middleware Pipeline](#middleware-pipeline)
- [Backend](#backend)
  - [Server Configuration](#server-configuration)
  - [API Endpoints](#api-endpoints)
  - [Authentication & Authorization](#authentication--authorization)
  - [Database Schema](#database-schema)
  - [Middleware Details](#middleware-details)
  - [Error Handling](#error-handling)
- [Frontend](#frontend)
  - [Pages](#pages)
  - [SPA Routing](#spa-routing)
  - [Design System](#design-system)
  - [Component Inventory](#component-inventory)
  - [State Management](#state-management)
  - [Demo Data](#demo-data)
- [Features](#features)
- [Dependencies](#dependencies)
- [Running the Project](#running-the-project)
- [Environment Variables](#environment-variables)
- [Design Principles](#design-principles)
- [API Reference](#api-reference)

---

## Overview

Watchtower is a full-stack SaaS uptime monitoring platform built with Go (backend) and vanilla HTML/CSS/JS (frontend). It provides HTTP/TCP/DNS monitoring, incident analysis with root-cause summaries, degradation detection, multi-step API chain monitoring, SSL certificate tracking, and SLA compliance reporting.

The application follows a monolithic architecture with a single Go binary serving both the REST API and the frontend SPA. Data is persisted in SQLite with WAL mode for concurrent access. Authentication uses JWT (HS256) with bcrypt password hashing, and supports both Bearer tokens and API keys.

**Key numbers:**
- ~178 lines server entrypoint
- ~106 lines auth service
- ~288 lines database layer
- ~288 lines HTTP handlers
- ~182 lines middleware
- ~519 lines frontend JavaScript
- ~739 lines CSS design system
- ~791 lines HTML (7 pages)

---

## Tech Stack

| Layer | Technology | Version | Purpose |
|---|---|---|---|
| Runtime | Go | 1.24.0 | HTTP server, business logic |
| Database | SQLite3 | WAL mode | User data, API keys |
| Auth | JWT (HS256) | golang-jwt/jwt/v5 | Token-based authentication |
| Crypto | bcrypt | golang.org/x/crypto | Password hashing |
| Frontend | Vanilla HTML/CSS/JS | ES6+ | Single-page application |
| Typography | Inter | Google Fonts | UI typeface |
| Build | `go build` | — | Single binary output |

---

## Project Structure

```
saas/
├── main.go                          Server entrypoint, routing, config
├── go.mod                           Go module definition
├── go.sum                           Dependency checksums
├── saas-server                      Compiled binary
├── saas.db                          SQLite database
├── saas.db-wal                      SQLite write-ahead log
├── saas.db-shm                      SQLite shared memory
│
├── auth/
│   └── auth.go                      JWT service, bcrypt, API key generation
│
├── store/
│   └── store.go                     SQLite layer, schema, CRUD operations
│
├── handlers/
│   ├── auth.go                      Signup & Login HTTP handlers
│   ├── api.go                       Profile, Plan, API Keys, Dashboard handlers
│   └── health.go                    Health check endpoint, writeJSON helper
│
├── middleware/
│   └── middleware.go                Logging, CORS, Auth, RateLimit, RequirePlan, Chain
│
├── frontend/
│   ├── index.html                   SPA — all pages (landing, about, status, login, signup, dashboard, profile)
│   ├── style.css                    Design system — dark theme, all components
│   └── app.js                       Routing, API client, auth, dashboard logic, rendering
│
└── .github/
    └── skills/
        └── frontend/
            └── SKILL.md             Design enforcement rules (no emojis, flat cards, etc.)
```

---

## Architecture

### System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client (Browser)                        │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Landing  │  │  Auth    │  │Dashboard │  │   Profile     │  │
│  │  About   │  │  Login   │  │ Monitors │  │   Settings    │  │
│  │  Status  │  │  Signup  │  │ Incidents│  │   API Keys    │  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘  │
│                         │                                       │
│                    SPA Routing (JS)                              │
│                    localStorage (JWT)                            │
└─────────────────────────┬───────────────────────────────────────┘
                          │ HTTP (JSON)
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go HTTP Server (:8080)                       │
│                                                                 │
│  ┌────────────────── Middleware Pipeline ──────────────────────┐│
│  │  CORS → RateLimit (120/min) → Logging → [Auth] → [Plan]   ││
│  └────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐   │
│  │  handlers/  │  │    auth/    │  │    middleware/        │   │
│  │  auth.go    │  │  auth.go    │  │  Logging             │   │
│  │  api.go     │  │  JWT+bcrypt │  │  CORS                │   │
│  │  health.go  │  │  API keys   │  │  Auth (JWT+API Key)  │   │
│  └──────┬──────┘  └─────────────┘  │  RateLimit           │   │
│         │                           │  RequirePlan         │   │
│         ▼                           │  Chain               │   │
│  ┌─────────────┐                   └──────────────────────┘   │
│  │   store/    │                                               │
│  │  store.go   │                                               │
│  │  SQLite WAL │                                               │
│  └──────┬──────┘                                               │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────┐  ┌──────────────┐                             │
│  │   saas.db   │  │  frontend/   │                             │
│  │   (SQLite)  │  │  index.html  │ ← Served via GET /{$}      │
│  │             │  │  style.css   │ ← Served via GET /static/*  │
│  │             │  │  app.js      │                             │
│  └─────────────┘  └──────────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

### Request Lifecycle

```
1. Browser sends HTTP request
2. CORS middleware adds cross-origin headers
3. RateLimit middleware checks IP bucket (120 req/min)
4. Logging middleware records method, path, starts timer
5. [Protected routes] Auth middleware extracts user:
   a. Check X-API-Key header → lookup user by key in DB
   b. Check Authorization: Bearer <token> → validate JWT, fetch user
   c. Inject user into request context
6. [Gated routes] RequirePlan checks user.Plan against allowed tiers
7. Handler executes business logic
8. Response written as JSON
9. Logging middleware records status code + duration
```

### Middleware Pipeline

Middleware is composed using the `Chain` function which applies middlewares in reverse order:

```go
// Chain(handler, A, B, C) executes as: C(B(A(handler)))
// So C runs first, A runs last (closest to handler)

// Global stack (all routes):
handler := middleware.Chain(mux,
    middleware.CORS(allowedOrigins),    // 1st: CORS headers
    middleware.RateLimit(120),          // 2nd: Rate limiting
    middleware.Logging,                 // 3rd: Request logging
)

// Per-route auth:
mux.Handle("GET /api/v1/me", middleware.Chain(
    http.HandlerFunc(apiHandler.GetProfile),
    requireAuth,                       // JWT/API key validation
))

// Per-route plan gating:
mux.Handle("GET /api/v1/pro/analytics", middleware.Chain(
    http.HandlerFunc(handler),
    requireAuth,                       // Must be authenticated
    requirePro,                        // Must be pro or enterprise
))
```

---

## Backend

### Server Configuration

| Setting | Default | Env Var | Description |
|---|---|---|---|
| Port | `8080` | `PORT` | HTTP listen port |
| JWT Secret | `change-me-to-a-real-secret-in-production` | `JWT_SECRET` | HS256 signing key |
| Database Path | `saas.db` | `DB_PATH` | SQLite file location |
| Allowed Origins | `*` | `ALLOWED_ORIGINS` | CORS origin |
| Read Timeout | 10s | — | Max time to read request |
| Write Timeout | 30s | — | Max time to write response |
| Idle Timeout | 60s | — | Keep-alive idle timeout |
| Rate Limit | 120 req/min | — | Per-IP request limit |
| Token TTL | 24 hours | — | JWT expiration |

### API Endpoints

#### Public Routes

| Method | Path | Handler | Description |
|---|---|---|---|
| `GET` | `/health` | `handlers.HealthCheck` | Liveness probe — returns uptime, Go version, goroutines, memory |
| `POST` | `/api/v1/auth/signup` | `authHandler.Signup` | Register new user — returns JWT + user |
| `POST` | `/api/v1/auth/login` | `authHandler.Login` | Authenticate — returns JWT + user |
| `GET` | `/{$}` | — | Serve `frontend/index.html` (exact match, Go 1.22+ pattern) |
| `GET` | `/static/*` | `http.FileServer` | Serve frontend CSS/JS assets |

#### Protected Routes (require authentication)

| Method | Path | Handler | Description |
|---|---|---|---|
| `GET` | `/api/v1/me` | `apiHandler.GetProfile` | Return authenticated user object |
| `PUT` | `/api/v1/me` | `apiHandler.UpdateProfile` | Update name and/or email |
| `DELETE` | `/api/v1/me` | `apiHandler.DeleteAccount` | Permanently delete account and all data |
| `PUT` | `/api/v1/me/plan` | `apiHandler.UpdatePlan` | Change plan (free/pro/enterprise) |
| `GET` | `/api/v1/dashboard` | `apiHandler.Dashboard` | Dashboard stats — user, plan, monitors count, monitor limit |
| `POST` | `/api/v1/keys` | `apiHandler.CreateAPIKey` | Generate new `sk_*` API key |
| `GET` | `/api/v1/keys` | `apiHandler.ListAPIKeys` | List all user's API keys |
| `DELETE` | `/api/v1/keys?id=N` | `apiHandler.DeleteAPIKey` | Revoke an API key (returns 404 if not found) |
| `POST` | `/api/v1/monitors` | `apiHandler.CreateMonitor` | Create a new monitor (enforces plan limits) |
| `GET` | `/api/v1/monitors` | `apiHandler.ListMonitors` | List all user's monitors |
| `DELETE` | `/api/v1/monitors?id=N` | `apiHandler.DeleteMonitor` | Delete a monitor (returns 404 if not found) |

#### Plan-Gated Routes (require pro or enterprise plan)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/pro/analytics` | Premium analytics data |

### Authentication & Authorization

#### Authentication Methods

The system supports two authentication methods, checked in order:

1. **API Key** — `X-API-Key` header with `sk_*` key
   - Key is looked up in `api_keys` table
   - User is fetched via JOIN query
2. **Bearer Token** — `Authorization: Bearer <jwt>` header
   - JWT is validated (signature + expiration)
   - Claims contain UserID, Email, Plan
   - User is fetched from DB by UserID

#### JWT Token Structure

```json
{
  "user_id": 1,
  "email": "user@example.com",
  "plan": "pro",
  "exp": 1741824000,
  "iat": 1741737600
}
```

- **Algorithm:** HS256 (HMAC-SHA256)
- **TTL:** 24 hours
- **Library:** `github.com/golang-jwt/jwt/v5`

#### Password Security

- **Algorithm:** bcrypt with `DefaultCost` (10 rounds)
- **Library:** `golang.org/x/crypto/bcrypt`
- **Validation:** Minimum 8 characters
- **Email normalization:** trimmed + lowercased before storage

#### API Key Format

```
sk_ + 64 hex characters (32 random bytes from crypto/rand)
Example: sk_a1b2c3d4e5f6...
```

#### Authorization Levels

| Level | Routes | Check |
|---|---|---|
| Public | `/health`, `/auth/*`, `/static/*`, `/` | None |
| Authenticated | `/api/v1/me`, `/api/v1/keys`, `/api/v1/dashboard`, `/api/v1/monitors` | Valid JWT or API key |
| Plan-gated | `/api/v1/pro/*` | Authenticated + plan is `pro` or `enterprise` |

### Database Schema

SQLite with WAL mode enabled, busy timeout of 5 seconds, foreign keys enforced.

#### `users` Table

```sql
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT    UNIQUE NOT NULL,
    password_hash TEXT    NOT NULL,
    name          TEXT    NOT NULL DEFAULT '',
    plan          TEXT    NOT NULL DEFAULT 'free',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### `api_keys` Table

```sql
CREATE TABLE IF NOT EXISTS api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key        TEXT    UNIQUE NOT NULL,
    name       TEXT    NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key  ON api_keys(key);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
```

#### `monitors` Table

```sql
CREATE TABLE IF NOT EXISTS monitors (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,
    url           TEXT    NOT NULL,
    type          TEXT    NOT NULL DEFAULT 'http',
    interval_sec  INTEGER NOT NULL DEFAULT 60,
    status        TEXT    NOT NULL DEFAULT 'pending',
    uptime        REAL    NOT NULL DEFAULT 100.0,
    response_time INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitors_user ON monitors(user_id);
```

#### Go Struct Mapping

```go
type User struct {
    ID           int64  `json:"id"`
    Email        string `json:"email"`
    PasswordHash string `json:"-"`           // Never serialized to JSON
    Name         string `json:"name"`
    Plan         string `json:"plan"`
    CreatedAt    string `json:"created_at"`
}

type APIKey struct {
    ID        int64  `json:"id"`
    UserID    int64  `json:"user_id"`
    Key       string `json:"key"`
    Name      string `json:"name"`
    CreatedAt string `json:"created_at"`
}

type Monitor struct {
    ID           int64   `json:"id"`
    UserID       int64   `json:"user_id"`
    Name         string  `json:"name"`
    URL          string  `json:"url"`
    Type         string  `json:"type"`
    Interval     int     `json:"interval"`
    Status       string  `json:"status"`
    Uptime       float64 `json:"uptime"`
    ResponseTime int     `json:"response_time"`
    CreatedAt    string  `json:"created_at"`
}
```

#### Plan Limits (Hardcoded)

| Plan | API Calls/Day | Monitors | Check Interval |
|---|---|---|---|
| Free | 100 | 5 | 5 minutes |
| Pro | 10,000 | 50 | 30 seconds |
| Enterprise | 1,000,000 | Unlimited | 10 seconds |

### Security Features

| Feature | Implementation |
|---|---|
| Request body size | All body-reading endpoints enforce 1 MB limit via `http.MaxBytesReader` |
| Startup warnings | Log warnings if JWT secret is default or CORS allows all origins |
| Foreign keys | SQLite `_foreign_keys=ON` — cascading deletes for user data |
| Input validation | Names, emails, URLs trimmed; monitor types normalized to lowercase |
| Plan enforcement | Monitor creation checks count vs plan limit before INSERT |
| Delete verification | `DeleteAPIKey` and `DeleteMonitor` verify `RowsAffected > 0`, return 404 otherwise |

### Middleware Details

#### 1. CORS

- Allows configurable origin (default `*`)
- Methods: GET, POST, PUT, DELETE, OPTIONS
- Headers: Content-Type, Authorization, X-API-Key
- Preflight cache: 86,400 seconds (24 hours)
- OPTIONS requests short-circuit with 204

#### 2. Rate Limiting

- In-memory per-IP bucket (not distributed)
- 120 requests per minute (default)
- Per-minute sliding window
- Respects `X-Forwarded-For` for proxied clients
- Returns `429 Too Many Requests` with `Retry-After: 60` header

#### 3. Logging

- Logs: `METHOD PATH STATUS_CODE DURATION`
- Captures status code via custom `statusWriter` wrapper
- Microsecond precision timing

#### 4. Auth

- Dual-mode: API Key first, then Bearer Token
- Injects user into `context.Context` via `UserContextKey`
- Returns 401 on failure with descriptive error

#### 5. RequirePlan

- Variadic plan list: `RequirePlan("pro", "enterprise")`
- Returns 403 with `upgrade required` message

#### 6. Chain (Composition Helper)

```go
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler
```

Applies middlewares right-to-left so the leftmost runs first.

### Error Handling

All errors are returned as JSON:

```json
{
    "error": "descriptive error message"
}
```

| Status | Meaning |
|---|---|
| 400 | Bad request body, missing fields, validation failure |
| 401 | Missing or invalid authentication |
| 403 | Insufficient plan / upgrade required |
| 404 | Resource not found |
| 409 | Conflict (email already registered) |
| 429 | Rate limit exceeded |
| 500 | Internal server error |

---

## Frontend

### Pages

The frontend is a single-page application with 7 distinct views:

| Page | ID | Access | Description |
|---|---|---|---|
| Landing | `page-landing` | Public | Hero, features (6 cards), pricing (3 tiers), detailed footer |
| About | `page-about` | Public | Mission, how it works, security, team story, platform stats, CTA |
| Status | `page-status` | Public | 6 system services, recent incidents, 90-day uptime bar |
| Login | `page-login` | Public | Email + password form |
| Signup | `page-signup` | Public | Name + email + password form |
| Dashboard | `page-dashboard` | Protected | 5 tabs: Overview, Monitors, Incidents, SSL & Domains, SLA |
| Profile | `page-profile` | Protected | Account settings, plan, API keys, notifications, danger zone |

### SPA Routing

```javascript
function showPage(page) {
    // Hide all pages
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    // Show target page
    document.getElementById('page-' + page).classList.add('active');
    window.scrollTo(0, 0);
    // Load data for protected pages
    if (page === 'dashboard') loadDashboard();
    if (page === 'profile') loadProfile();
}
```

Pages are toggled via CSS `.page { display: none }` / `.page.active { display: block }`. No URL changes or history API.

**Initialization:** On load, checks `localStorage` for a JWT token. If found, shows dashboard; otherwise, shows landing.

### Design System

#### Color Tokens

| Token | Value | Usage |
|---|---|---|
| `--bg` | `#0a0a0f` | Page background |
| `--surface` | `#111117` | Card backgrounds |
| `--surface-2` | `#1a1a24` | Nested containers, icon backgrounds |
| `--border` | `#1f1f2e` | Default borders |
| `--border-h` | `#2a2a3d` | Hover borders |
| `--text` | `#e4e4ec` | Primary text |
| `--text-2` | `#9898b0` | Secondary text |
| `--text-3` | `#5a5a72` | Tertiary text, labels |
| `--accent` | `#6366f1` | Primary action (indigo) |
| `--accent-h` | `#818cf8` | Accent hover state |
| `--green` | `#10b981` | Up / success / compliant |
| `--yellow` | `#f59e0b` | Warning / degraded |
| `--red` | `#ef4444` | Down / error / danger |

#### Typography

- **Font:** Inter (Google Fonts), fallback: -apple-system, BlinkMacSystemFont, sans-serif
- **Weights:** 400 (body), 500 (labels/navigation), 600 (card titles), 700 (headings/stats)
- **Line height:** 1.6 (body), 1.15 (hero headings)
- **Letter spacing:** -0.025em (large headings), 0.04-0.06em (uppercase labels)

#### Button Variants

| Class | Style |
|---|---|
| `.btn--primary` | Indigo background, white text |
| `.btn--ghost` | Transparent, muted text |
| `.btn--outline` | Transparent, border, text inherits |
| `.btn--danger` | Transparent, red text |
| `.btn--lg` | Larger padding |
| `.btn--sm` | Smaller padding |
| `.btn--block` | Full width |

#### Status Indicators

8px colored dots only. No emojis, no glow, no pulsing:

- `.status-dot--up` → green (`#10b981`)
- `.status-dot--degraded` → yellow (`#f59e0b`)
- `.status-dot--down` → red (`#ef4444`)
- `.status-dot--pending` → gray (`#5a5a72`)

#### Responsive Breakpoints

| Breakpoint | Changes |
|---|---|
| `max-width: 768px` | Single-column grids, hide nav text links, hide monitor metrics/bars, stack footer |
| `max-width: 480px` | Single-column footer, single-column stats, vertical status services |

### Component Inventory

#### Landing Page
- **Nav:** Fixed, backdrop-blur, logo + links + buttons
- **Hero:** Centered headline + subtitle + CTA button + note
- **Features Grid:** 3-column, 1px gap borders, SVG icons (20x20), flat background
- **Pricing Grid:** 3 cards (Free $0, Pro $29/mo, Enterprise $99/mo), featured card has accent border
- **Footer:** 4-column (Brand + description, Product links, Company links, Legal links) + bottom bar (copyright + social links)

#### Dashboard
- **Dash Nav:** Sticky, user name + plan badge + profile button + logout
- **Stats Row:** 6 stat boxes (Total, Up, Degraded, Down, Avg Response, SLA)
- **Tabs:** Overview | Monitors | Incidents | SSL & Domains | SLA
- **Monitor Row:** Status dot, name, type badge, URL, response time, uptime %, history bars (30 days)
- **Uptime Bars:** 3px wide, green/yellow/red per-day
- **Alert Row:** Yellow background, dot + title + description + analysis box
- **Incident Row:** Red/yellow background variant, dot + title + analysis + timestamp
- **SSL Row:** Status dot, domain, issuer, expiry date, days remaining countdown
- **SLA Section:** Ring gauge (140px circle), compliance badge, 4 KV cards, per-monitor bar chart
- **Modals:** Add Monitor, Create API Chain, Add Domain

#### Profile
- **Avatar:** 48px circle, accent color, first letter of name
- **Profile Card:** Avatar + name + email + join date
- **Plan Card:** Plan name + description + upgrade button
- **API Keys List:** Key value (monospace) + created date + revoke button
- **Notifications:** 2-column checkbox grid
- **Danger Zone:** Red-tinted card with delete account button

### State Management

Global variables (no framework):

```javascript
const API = '/api/v1';
let currentUser = null;    // User object from /me endpoint
let monitors = [];         // Monitor array (from API or demo)
let sslDomains = [];       // SSL domain objects
let incidents = [];        // Incident objects
```

- **Token persistence:** `localStorage.getItem('token')` / `localStorage.setItem('token', ...)`
- **Auth state:** `currentUser !== null` means logged in
- **Logout:** Clear localStorage, null all state, show landing

### Demo Data

The frontend ships with demo data that activates when the API endpoints aren't available (e.g., no `/api/v1/monitors` route yet):

**5 Demo Monitors:**

| Name | Type | Status | Uptime | Response |
|---|---|---|---|---|
| Production API | HTTP | Up | 99.98% | 142ms |
| User Login Flow | Chain (3 steps) | Up | 99.85% | 312ms |
| Checkout Service | HTTP | Degraded | 99.74% | 890ms |
| Payment Service | HTTP | Down | 99.12% | — |
| CDN Origin | HTTP | Up | 99.95% | 34ms |

**4 Demo SSL Domains:**

| Domain | Issuer | Status | Days Left |
|---|---|---|---|
| myapp.com | Let's Encrypt | OK | 89 |
| api.myapp.com | DigiCert | Warning | 12 |
| payments.myapp.com | Let's Encrypt | Expired | -2 |
| cdn.myapp.com | Cloudflare | OK | 320 |

**3 Demo Incidents:**

| Monitor | Status | Started | Analysis |
|---|---|---|---|
| Payment Service | Active | 12m ago | SSL certificate expired |
| Checkout Service | Degraded | 47m ago | DB connection pool saturation |
| Production API | Resolved | 6h ago | Brief outage during deployment |

---

## Features

### Implemented (Full Stack)
1. **User Authentication** — Signup, login, JWT tokens, bcrypt passwords
2. **API Key Management** — Create, list, revoke `sk_*` keys
3. **Multi-Tier Plans** — Free, Pro ($29/mo), Enterprise ($99/mo)
4. **Plan Gating** — Middleware restricts features by plan
5. **Health Monitoring** — Server health endpoint with runtime metrics
6. **Rate Limiting** — Per-IP, 120 req/min
7. **CORS Support** — Configurable origin policy
8. **Profile Management** — View/edit name, email, plan, API keys

### Implemented (Frontend Only, Demo Data)
1. **Uptime Monitoring** — HTTP + TCP checks with 30-day history bars
2. **Multi-Step API Chains** — Monitor sequential API workflows
3. **Degradation Detection** — Alerts on response time drift above baseline
4. **Incident Intelligence** — Root cause analysis summaries
5. **SSL & Domain Tracking** — Certificate expiry countdown
6. **SLA Compliance** — Uptime targets, downtime budget tracking
7. **System Status Page** — Public service status + incident history
8. **About Page** — Company mission, platform stats

### Not Yet Implemented (Backend)
- `POST/GET/DELETE /api/v1/monitors` — CRUD for monitors
- Actual monitoring engine (HTTP checks, TCP probes)
- Real incident detection and alerting pipeline
- SSL certificate scanning
- Webhook/Slack/Email notification delivery
- SLA calculation engine

---

## Dependencies

```
module saas
go 1.24.0

require (
    github.com/golang-jwt/jwt/v5 v5.3.1      // JWT token creation and validation
    github.com/mattn/go-sqlite3  v1.14.34     // SQLite3 driver (CGO required)
    golang.org/x/crypto          v0.48.0      // bcrypt password hashing
)
```

**Note:** `go-sqlite3` requires CGO. Build with `CGO_ENABLED=1` (default on Linux/macOS).

---

## Running the Project

### Build and Run

```bash
cd /home/mishu/mayank/goprojects/saas

# Build
go build -o saas-server .

# Run
./saas-server
```

Server starts at `http://localhost:8080`.

### Quick Test

```bash
# Health check
curl http://localhost:8080/health

# Sign up
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@example.com","password":"password123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Get profile (with token from login response)
curl http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <token>"
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server listen port |
| `JWT_SECRET` | `change-me-to-a-real-secret-in-production` | HS256 signing secret |
| `DB_PATH` | `saas.db` | SQLite database file path |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |

---

## Design Principles

Enforced via `.github/skills/frontend/SKILL.md`:

| Rule | Description |
|---|---|
| No emojis | Use SVG icons (20x20, stroke-only) or CSS indicators |
| Flat cards | 1px border, no box-shadow, no glow effects |
| Status dots only | 8px colored circles for up/degraded/down |
| No gradient text | Solid colors only |
| Minimal transitions | Color and opacity only, max 200ms |
| No hover transforms | No `translateY`, no scale effects |
| Dark-first | `#0a0a0f` base, surface layers for depth |
| Data-first layout | Numbers prominent, tables clean, labels uppercase |
| Inter typeface | 400-700 weights, tight letter-spacing on headings |

---

## API Reference

### POST /api/v1/auth/signup

**Request:**
```json
{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securepassword"
}
```

**Response (201):**
```json
{
    "token": "eyJhbGciOi...",
    "user": {
        "id": 1,
        "email": "john@example.com",
        "name": "John Doe",
        "plan": "free",
        "created_at": "2026-03-11T12:00:00Z"
    }
}
```

### POST /api/v1/auth/login

**Request:**
```json
{
    "email": "john@example.com",
    "password": "securepassword"
}
```

**Response (200):** Same structure as signup.

### GET /api/v1/me

**Headers:** `Authorization: Bearer <token>` or `X-API-Key: sk_...`

**Response (200):**
```json
{
    "id": 1,
    "email": "john@example.com",
    "name": "John Doe",
    "plan": "free",
    "created_at": "2026-03-11T12:00:00Z"
}
```

### PUT /api/v1/me/plan

**Request:**
```json
{
    "plan": "pro"
}
```

**Response (200):**
```json
{
    "message": "plan updated",
    "plan": "pro"
}
```

### GET /api/v1/dashboard

**Response (200):**
```json
{
    "user": { ... },
    "plan": "pro",
    "api_calls_today": 42,
    "api_limit": 10000
}
```

### POST /api/v1/keys

**Request:**
```json
{
    "name": "production-key"
}
```

**Response (201):**
```json
{
    "id": 1,
    "user_id": 1,
    "key": "sk_a1b2c3d4...",
    "name": "production-key",
    "created_at": "2026-03-11T12:00:00Z"
}
```

### GET /api/v1/keys

**Response (200):**
```json
[
    {
        "id": 1,
        "user_id": 1,
        "key": "sk_a1b2c3d4...",
        "name": "production-key",
        "created_at": "2026-03-11T12:00:00Z"
    }
]
```

### DELETE /api/v1/keys?id=1

**Response (200):**
```json
{
    "message": "key deleted"
}
```

### GET /health

**Response (200):**
```json
{
    "status": "healthy",
    "uptime": "4h28m35s",
    "go_version": "go1.24.0",
    "goroutines": 12,
    "memory_mb": 24
}
```
