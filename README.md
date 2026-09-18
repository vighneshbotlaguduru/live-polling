# LivePoll — Real-Time Audience Polling

A full-stack live polling application where users create polls, share a link, and audiences vote with results updating in real-time — no page refresh needed.

![Stack](https://img.shields.io/badge/React-61DAFB?logo=react&logoColor=000&style=flat-square) ![Stack](https://img.shields.io/badge/Go%20(Gin)-00ADD8?logo=go&logoColor=fff&style=flat-square) ![Stack](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=fff&style=flat-square) ![Stack](https://img.shields.io/badge/Redis-DC382D?logo=redis&logoColor=fff&style=flat-square)

## Architecture

```
┌─────────────────┐     HTTP/SSE      ┌──────────────────┐
│  React Frontend │ ◄──────────────── │  Go/Gin Backend  │
│  (Vite SPA)     │ ──────────────► │  (REST + SSE)    │
└─────────────────┘                   └────────┬─────────┘
                                               │
                                    ┌──────────┴──────────┐
                                    │                     │
                              ┌─────▼─────┐        ┌─────▼─────┐
                              │  MongoDB   │        │   Redis    │
                              │ (persist)  │        │ (realtime) │
                              └───────────┘        └───────────┘
```

### How Real-Time Works

1. **Vote cast** → `POST /api/polls/:id/vote`
2. **Redis HINCRBY** → atomically increments the vote count
3. **Redis PUBLISH** → broadcasts update to Pub/Sub channel `poll:{id}`
4. **SSE listeners** → all connected clients on `/api/polls/:id/live` receive the update instantly
5. **Background sync** → a goroutine flushes Redis counts to MongoDB every 30 seconds

### What Each Technology Does

| Tech | Real Work |
|------|-----------|
| **MongoDB** | Persistent storage for users, polls, votes. Source of truth on restart. |
| **Redis** | Atomic vote counting (`HINCRBY`), duplicate detection (`SADD`), live broadcasting (`PUBLISH/SUBSCRIBE`), rate limiting (`INCR/EXPIRE`) |
| **Go/Gin** | REST API, SSE streaming, JWT auth, input validation, background sync worker |
| **React** | SPA with EventSource for SSE, animated live charts, responsive glassmorphism UI |

## Project Structure

```
├── backend/                  # Go service
│   ├── main.go               # Entrypoint, router setup, graceful shutdown
│   ├── config/config.go      # Environment-based configuration
│   ├── models/               # MongoDB document schemas
│   │   ├── user.go
│   │   ├── poll.go
│   │   └── vote.go
│   ├── handlers/             # HTTP request handlers
│   │   ├── auth.go           # Signup, login, me
│   │   ├── poll.go           # CRUD + close
│   │   ├── vote.go           # Vote submission
│   │   └── live.go           # SSE streaming
│   ├── middleware/            # Gin middleware
│   │   ├── auth.go           # JWT verification
│   │   └── ratelimit.go      # Redis-based rate limiting
│   ├── services/             # Business logic
│   │   ├── redis.go          # Redis operations
│   │   └── sync.go           # Redis → MongoDB sync worker
│   └── .env                  # Environment variables
├── frontend/                 # React SPA
│   ├── src/
│   │   ├── components/       # Reusable UI components
│   │   ├── pages/            # Route-level pages
│   │   ├── hooks/            # Custom React hooks (SSE)
│   │   ├── services/         # API client
│   │   └── context/          # Auth state management
│   └── vite.config.js        # Dev proxy configuration
├── docker-compose.yml        # MongoDB + Redis for local dev
└── README.md
```

## Quick Start

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Docker](https://docker.com/) (for MongoDB + Redis)

### 1. Start databases

```bash
docker-compose up -d
```

### 2. Start backend

```bash
cd backend
cp .env .env.local   # adjust if needed
go mod tidy
go run main.go
```

The API starts on `http://localhost:8080`.

### 3. Start frontend

```bash
cd frontend
npm install
npm run dev
```

The app opens at `http://localhost:5173`. API calls are proxied to the backend automatically.

## Key Design Decisions

- **SSE over WebSockets**: Server-Sent Events are simpler, natively supported by browsers, and one-directional (which is all we need — server pushes vote updates to clients). No socket upgrade overhead.
- **Redis as source of truth for live counts**: Vote counts live in Redis hashes (`HINCRBY` for atomicity). MongoDB is updated asynchronously every 30s. This makes voting extremely fast.
- **Voter fingerprinting**: Duplicate votes are detected via `SHA-256(IP + User-Agent)` stored in Redis SETs. Not foolproof, but prevents casual double-voting without requiring login to vote.
- **JWT with 72h expiry**: Simple, stateless auth. No refresh tokens — kept deliberately simple.
- **Vite proxy in dev**: Frontend requests to `/api/*` are proxied to Go on port 8080, so no CORS issues during development.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/auth/signup` | — | Register |
| `POST` | `/api/auth/login` | — | Login |
| `GET` | `/api/auth/me` | ✓ | Current user |
| `POST` | `/api/polls` | ✓ | Create poll |
| `GET` | `/api/polls` | ✓ | List my polls |
| `GET` | `/api/polls/:id` | — | Get poll (by ID or share code) |
| `PATCH` | `/api/polls/:id/close` | ✓ | Close poll |
| `DELETE` | `/api/polls/:id` | ✓ | Delete poll |
| `POST` | `/api/polls/:id/vote` | — | Cast vote (rate limited) |
| `GET` | `/api/polls/:id/live` | — | SSE stream |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `JWT_SECRET` | `change-me-...` | JWT signing secret |
| `PORT` | `8080` | Backend port |
| `FRONTEND_URL` | `http://localhost:5173` | Allowed CORS origin |

---

## Zero-Install Option (Cloud Databases)

If you don't have Docker or local MongoDB/Redis installed:
1. **MongoDB**: Create a free sandbox at [MongoDB Atlas](https://www.mongodb.com/atlas) and set `MONGO_URI=mongodb+srv://...` in `backend/.env`.
2. **Redis**: Create a free database at [Upstash Redis](https://upstash.com) and set `REDIS_URL=rediss://default:...@...upstash.io:6379` in `backend/.env`.

---

## Deployment Guide (Live URL)

### 1. Deploy Databases
- Create a free cluster on **MongoDB Atlas**.
- Create a free instance on **Upstash Redis**.

### 2. Deploy Backend (Render / Railway / Fly.io)
- **Root Directory**: `backend`
- **Build Command**: `go build -o server .`
- **Start Command**: `./server`
- **Environment Variables**:
  - `MONGO_URI`: Your MongoDB Atlas connection URI
  - `REDIS_URL`: Your Upstash Redis URL
  - `JWT_SECRET`: A secure random string
  - `PORT`: `8080`
  - `FRONTEND_URL`: Your deployed frontend URL (e.g. `https://your-poll-app.vercel.app`)

### 3. Deploy Frontend (Vercel / Netlify)
- **Root Directory**: `frontend`
- **Build Command**: `npm run build`
- **Output Directory**: `dist`
- **Environment Variables**:
  - `VITE_API_URL`: Your deployed backend URL (e.g. `https://livepoll-api.onrender.com`)

