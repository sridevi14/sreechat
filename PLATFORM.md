# SreeChat — Platform Overview (Frontend + Backend)

SreeChat is a real-time chat application: users authenticate with phone + password, search for other users by phone, open or reuse **direct** chats, exchange messages over **WebSocket**, and load history via **REST**. The backend uses **MongoDB** for persistence, **Redis** for per-room sequence numbers and Pub/Sub fan-out across API instances, and **JWT** for API and WebSocket authentication.

---

## Architecture at a Glance

```
Browser (React)
    │  REST  /api/*     JWT Bearer
    │  WS    /ws?token=&room_id=
    ▼
Go API (Gin) ──► MongoDB (users, rooms, messages)
    │
    └── Redis: INCR room:{id}:seq  →  total order per room
        Redis: PUBLISH room:{id}   →  all API replicas broadcast to local WS clients
```

**Scaling idea:** Multiple API servers share one MongoDB and one Redis. Sequence assignment (`INCR`) and Pub/Sub are global, so ordering and cross-instance delivery stay consistent.

---

## Tech Stack

| Area | Technology |
|------|------------|
| **Frontend** | React 18, TypeScript, Vite 5, Tailwind CSS, Zustand, Axios |
| **Backend** | Go 1.22, Gin, gorilla/websocket, JWT (golang-jwt/jwt/v5), bcrypt |
| **Data** | MongoDB (documents), Redis (seq + pub/sub + optional presence keys) |
| **Containers** | Docker Compose (API, MongoDB, Redis, Nginx-served SPA) |

---

## Repository Layout

```
sreechat/
├── backend/
│   ├── cmd/server/main.go          # Entry: DB/Redis, routes, hub, pubsub
│   └── internal/
│       ├── config/                 # Env-based configuration
│       ├── handlers/               # HTTP + WebSocket handlers
│       ├── hub/                    # In-memory room → WebSocket clients
│       ├── middleware/             # JWT for REST
│       ├── models/                 # User, Room, Message, WS envelopes
│       ├── pubsub/                 # Redis Publish/Subscribe + NextSeq + presence helpers
│       └── repository/             # MongoDB access (users, rooms, messages)
├── frontend/
│   ├── src/
│   │   ├── api/client.ts           # Axios base URL /api, auth + rooms API
│   │   ├── components/             # AuthPage, Sidebar, ChatWindow, MessageInput
│   │   ├── hooks/useWebSocket.ts   # WS connect, reconnect, send message/typing
│   │   ├── store/                  # authStore, chatStore (Zustand)
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── vite.config.ts              # Dev proxy: /api and /ws → localhost:8080
│   ├── Dockerfile                  # Build SPA → nginx
│   └── nginx.conf                  # Prod: proxy /api and /ws to api:8080
└── docker-compose.yml
```

---

## Backend

### Configuration (environment)

| Variable | Role | Typical local default |
|----------|------|------------------------|
| `PORT` | HTTP listen port | `8080` |
| `MONGO_URI` / `MONGO_DB` | MongoDB connection and database name | `mongodb://localhost:27017`, `sreechat` |
| `REDIS_ADDR` / `REDIS_PASSWORD` | Redis | `localhost:6379` |
| `JWT_SECRET` | Sign and verify JWTs | must be set in production |
| `CORS_ORIGIN` | Allowed browser origin for REST | e.g. `http://localhost:5173` |

### HTTP API (`/api`)

All JSON. Protected routes expect `Authorization: Bearer <token>`.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/register` | Register (username, phone, password); returns JWT + user |
| `POST` | `/api/auth/login` | Login with phone + password; returns JWT + user |
| `GET` | `/api/users/search?phone=...` | Search users by phone substring (excludes self), max 20 |
| `GET` | `/api/rooms` | List rooms where the user is a member; direct rooms get the other user’s username as display name |
| `POST` | `/api/rooms` | Create a room (name, type `direct` \| `group`, members) |
| `POST` | `/api/rooms/direct` | Body: `peer_id` — find existing direct room or create one |
| `GET` | `/api/rooms/:id/messages` | Query: `after` (seq cursor), `limit` (capped at 100). Returns messages in **descending** seq order (newest first) |

### WebSocket (`GET /ws`)

- **Query:** `token=<JWT>` and `room_id=<ObjectID hex>`.
- Server validates JWT, loads the room, ensures the user is in `members`, then upgrades to WebSocket.
- **Inbound frames (JSON):**
  - `{ "type": "message", "room_id": "...", "payload": { "content": "..." } }`
  - `{ "type": "typing", "room_id": "...", "payload": { "is_typing": true|false } }`
- **Outbound:** Same envelope; `message` payloads include `sender_id`, `username`, `seq`, `content`. `typing` includes `sender_id`, `username`, `is_typing`. After a chat message, the server also publishes a typing-off event for the sender.

### Message path (ordering and fan-out)

1. Client sends a `message` over WS.
2. Server **`INCR`** `room:{roomID}:seq` in Redis → monotonic `seq` per room (works across replicas).
3. Message is stored in MongoDB with that `seq` (unique index on `(room_id, seq)`).
4. Server **publishes** the full `WSMessage` JSON to Redis channel `room:{roomID}`.
5. Every API process that has subscribed to that channel receives the payload and **broadcasts** to all **local** WebSocket clients in that room via the in-memory **hub**.
6. Redis subscriber goroutines are started per room when the first client connects (`Subscribe(roomID)` is idempotent per room on that instance).

### Data model (MongoDB)

- **users:** unique `username`, unique `phone`, password hash, optional `avatar`, `created_at`.
- **rooms:** `name`, `type` (`direct` / `group`), `members` (ObjectID array), `created_at`. Direct chats between two users are deduplicated via `FindDirectRoom`.
- **messages:** `room_id`, `sender_id`, `content`, `seq`, `created_at`.

### Cross-cutting behavior

- **JWT (REST):** Middleware reads `Bearer` token, sets `user_id` and `username` on the Gin context.
- **JWT (WS):** Parsed from query string; same secret and claims (`sub`, `username`).
- **Presence:** On WS connect, `SetOnline` sets `online:{userID}` in Redis with TTL (~35s). Not currently surfaced in the React UI beyond a static “Online” label in the chat header.

---

## Frontend

### Role in the system

- **Auth:** `AuthPage` toggles login vs register; `authStore` persists `token` and `user` in `localStorage` and attaches the token to Axios via an interceptor.
- **Room list + discovery:** `Sidebar` loads rooms on mount, supports “New Chat” → phone search → `startDirect(peer.id)` → refresh rooms and select the room.
- **Chat:** `ChatWindow` loads message history when `activeRoom` changes (`GET /rooms/:id/messages`), connects **one WebSocket per selected room** (`useWebSocket`), renders messages and typing indicators, and sends outbound messages/typing through the hook.

### State (Zustand)

- **`authStore`:** `user`, `token`, `isAuthenticated`, `login`, `register`, `logout`, `hydrate` (restore from `localStorage` on app load).
- **`chatStore`:** `rooms`, `activeRoom`, `messages` keyed by room id, `typingUsers` keyed by room id; `fetchRooms`, `fetchMessages`, `addMessage`, `setTyping`. **`addMessage`** merges by `seq` and, if it detects a **sequence gap** (`message.seq > lastSeq + 1`), calls the messages API to backfill (gap healing).

### Networking

- **Development:** `vite.config.ts` proxies `/api` and `/ws` to `http://localhost:8080`, so the browser uses same-origin requests to `localhost:5173` while the Go server runs separately.
- **Docker Compose:** The SPA is served by nginx on port **3000**; `nginx.conf` proxies `/api/` and `/ws` to the `api` service at `8080`, so the browser still uses relative `/api` and `/ws` URLs.

### WebSocket client (`useWebSocket`)

- Builds `ws://` or `wss://` from `window.location.host` with query `token` and `room_id`.
- On each incoming `message`, normalizes to the shared `Message` shape and calls `addMessage`; also `fetchRooms()` to refresh the sidebar.
- Reconnects after **2s** on close.
- Typing: debounced sends of `is_typing: true`, then `false` after idle; server echoes to other clients.

### UI stack

Tailwind-driven layout: dark slate/indigo theme, sidebar + main chat area, bubbles for own vs peer messages.

---

## How Frontend and Backend Fit Together

1. User registers or logs in → JWT stored → all REST calls include `Authorization`.
2. User picks or creates a room → client loads history with cursor `after` (seq); store reverses/sorts to show chronological order.
3. User selects a room → WebSocket opens for that `room_id` → real-time messages and typing; ordering relies on `seq` from the server.
4. If the client misses messages (gap in `seq`), `chatStore.addMessage` triggers a REST fetch to fill the range.

---

## Running Locally

**Docker Compose (full stack):** from the repo root, `docker-compose up --build` — frontend at [http://localhost:3000](http://localhost:3000), API at [http://localhost:8080](http://localhost:8080), MongoDB and Redis exposed on default ports.

**Split dev:** start MongoDB and Redis, run `go run ./cmd/server` in `backend/`, run `npm install && npm run dev` in `frontend/` (Vite on port 5173 with proxy to 8080).

---

## Operational Notes

- Change **`JWT_SECRET`** and restrict **`CORS_ORIGIN`** in production; the Compose file uses a placeholder secret.
- WebSocket **`CheckOrigin`** is currently permissive in code; tighten if exposing beyond a known front-end origin.
- Horizontal scaling requires shared MongoDB and Redis; each API instance runs its own hub + Redis subscribers for rooms that have active clients on that instance.

This document reflects the application as implemented in the repository and is meant as a single reference for both the React client and the Go API.
