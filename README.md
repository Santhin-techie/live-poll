# Live Polls

A live polling tool: create a poll, share the link, watch votes update in real time with no refresh.

## Stack
- **Frontend:** React (Vite) + `react-router-dom`
- **Backend:** Go + Gin
- **Database:** MongoDB (users, polls, votes — the durable record)
- **Realtime:** Redis (live vote counters + pub/sub that drives the WebSocket)

## How it fits together
```
/frontend   React app (auth, dashboard, create poll, live poll view)
/backend    Go service (REST API + WebSocket)
```
1. A signed-up user creates a poll → stored in MongoDB, counters seeded at 0 in Redis.
2. The poll's public link opens a page that connects a WebSocket to `/api/polls/:id/ws`.
3. Anyone (no login required) votes → the backend validates the option against MongoDB,
   records the vote (for history + duplicate-vote prevention), then `HINCRBY`s the count
   in Redis and `PUBLISH`es the new snapshot on a per-poll Redis channel.
4. The backend's WebSocket hub is subscribed to that channel and pushes the update to
   every connected browser instantly — that's the "no refresh needed" part.

Redis isn't decorative here: it's the read path for every poll view/vote (so reads don't
hit Mongo) and the pub/sub backbone for live updates. Mongo is the source of truth and
audit trail (who voted, when, on what).

## Running locally

### Prerequisites
- Go 1.22+
- Node 18+
- MongoDB running locally (or an Atlas URI)
- Redis running locally (or Upstash/Redis Cloud URI)

### Backend
```bash
cd backend
cp .env.example .env      # edit with your Mongo/Redis URIs and a real JWT_SECRET
go mod tidy
go run main.go
```
Runs on `http://localhost:8080`.

### Frontend
```bash
cd frontend
cp .env.example .env      # set VITE_API_URL to your backend URL
npm install
npm run dev
```
Runs on `http://localhost:5173`.

## Key decisions
- **Auth is JWT-based**, required only for creating/managing polls. Voting and viewing
  a poll stay public — the brief only asks that poll *creation* be gated.
- **Duplicate-vote prevention** uses a hash of IP + User-Agent + poll ID rather than
  requiring voters to log in, since the brief treats voting as open to "the audience."
  It's a reasonable deterrent, not a bulletproof one — a determined voter could still
  clear cookies / use another device. That trade-off is intentional given no voter
  accounts are required.
- **Server-side validation everywhere it matters**: poll options are re-validated,
  deduplicated and trimmed on the backend regardless of what the client sends; a vote's
  `option_id` is checked against the poll's actual stored options before it's counted.
- **Redis as the live-count source of truth for reads**, MongoDB as the durable log —
  avoids re-aggregating votes from Mongo on every page view/vote.

## Deploying
Any split works as long as both are publicly reachable and can talk to each other:
- **Backend:** Render / Railway / Fly.io (needs a long-running process for WebSockets — avoid platforms that only support serverless functions for this part)
- **Frontend:** Vercel / Netlify (set `VITE_API_URL` to the deployed backend URL)
- **MongoDB:** MongoDB Atlas free tier
- **Redis:** Upstash or Redis Cloud free tier

Remember to update `FRONTEND_URL` in the backend's env to the deployed frontend origin
(CORS) and `VITE_API_URL` in the frontend to the deployed backend origin.

## Submission checklist (from the brief)
- [ ] GitHub repo (public)
- [ ] Live deployed link (frontend + backend actually reachable, not `localhost`)
- [ ] 3–5 min video: your hardest challenge + honest answer on AI tool usage
- [ ] Email everything to devhiring@hclguvi.com
