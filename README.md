# Errolan

> Comments, with character.

A self-hostable Disqus alternative in Go, with a drop-in JavaScript SDK that supports both a flat conversation widget and **paragraph-anchored marginalia** — readers leave notes next to specific paragraphs, library-annotation style.

The design language is editorial: Newsreader / Instrument Serif / JetBrains Mono on paper, with a vermilion accent. Per-site custom emoji packs replace the up/down vote model with named reactions (`:ship-it:`, `:hot-take:`, `:this:`…). Live design exploration lives in [`design/`](./design).

## Features

- **Marginalia** mode: comments anchor to specific paragraphs on the host page via `data-errolan-anchor`; clicking a margin stamp opens a side panel with the focused thread
- **Cadence** mode: flat conversation river, sort by best / newest / oldest, pinned-first
- **Per-site emoji packs** with reaction toggling (replaces plain ±1 votes); reactions and counts denormalized for fast reads
- Sites → threads → nested comments, with flagging, pinning, ban, edit history, soft delete
- JWT auth + per-account login lockout; per-site option to allow anonymous posting
- Per-site allowed origins (CORS); cached site lookups
- Moderation: lock threads, ban users, flag queue, audit log
- gzip compression, ETag-based 304s, denormalized counters, pagination
- Real-time updates via Server-Sent Events (polling fallback)
- Hardening: rate limiting (global + auth + write), body-size cap, security headers, honeypot anti-spam
- Optional outbound webhook for moderation events
- Single Go binary, embedded SQLite (pure Go — no CGO required)
- Vanilla JS SDK — drop in a `<script>` tag and a `<div>`, done

## Quick start

```bash
# build
go build -o errolan ./cmd/errolan

# run (creates ./errolan.db and bootstraps an admin)
ERROLAN_ADMIN_EMAIL=admin@example.com \
ERROLAN_ADMIN_PASSWORD=changeme1234 \
ERROLAN_JWT_SECRET=$(openssl rand -hex 32) \
./errolan
```

The server listens on `:8080` by default. Set `ERROLAN_ADDR=:9000` to change it.

### Live reload during development

The repo ships with a [`.air.toml`](./.air.toml) config for [air](https://github.com/air-verse/air), which rebuilds and restarts the binary on every `.go` or `.sql` save. Migrations are embedded via `go:embed`, so SQL changes need a rebuild — air handles both.

```bash
# install once (drops the binary in $(go env GOPATH)/bin — make sure that's on PATH)
go install github.com/air-verse/air@latest

# put dev secrets in .env (not committed)
cat > .env <<'EOF'
ERROLAN_ADMIN_EMAIL=admin@example.com
ERROLAN_ADMIN_PASSWORD=changeme1234
ERROLAN_JWT_SECRET=dev-only-secret
EOF

# load env and start the reloader
export $(grep -v '^#' .env | xargs) && air
```

Air watches `cmd/` and `internal/`; it ignores `sdk/`, `design/`, and `tmp/`. SDK changes are plain static files — just refresh the browser.

### Configuration (env vars)

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `ERROLAN_ADDR` | `:8080` | HTTP listen address |
| `ERROLAN_DB` | `errolan.db` | SQLite database path |
| `ERROLAN_JWT_SECRET` | *(random per start)* | HMAC secret for JWTs — set this in production |
| `ERROLAN_ADMIN_EMAIL` | *(unset)* | If the DB is empty, bootstrap an admin with this email |
| `ERROLAN_ADMIN_PASSWORD` | *(unset)* | Password for the bootstrap admin |
| `ERROLAN_ADMIN_CORS` | `*` | Comma-separated origins allowed for admin endpoints |
| `ERROLAN_SDK_DIR` | `sdk` | Directory served at `/sdk/`; leave default to serve the bundled SDK |
| `ERROLAN_TRUST_FORWARDED` | `false` | Set to `true` only behind a proxy that strips spoofed `X-Forwarded-For` headers |
| `ERROLAN_WEBHOOK_URL` | *(unset)* | If set, receives POST JSON for `comment.created`, `comment.delete`, `comment.flag`, `user.ban`, `thread.lock` events |

## Create a site

After the admin is bootstrapped, log in to get a token and create a site:

```bash
TOKEN=$(curl -s http://localhost:8080/api/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@example.com","password":"changeme1234"}' | jq -r .token)

curl -s http://localhost:8080/api/sites \
  -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{"slug":"my-blog","name":"My Blog","allowed_origins":"https://my-blog.com,http://localhost:5173"}'
```

The response contains the site's `api_key` (starts with `erl_`). That key is **public** — it identifies your site to the SDK, similar to a Disqus shortname.

## Embed the SDK

The SDK ships in two modes — pick one per page.

### Cadence — flat conversation (default)

```html
<link rel="stylesheet" href="https://comments.example.com/sdk/errolan.css">

<div data-errolan-thread="post-2026-blog-launch"
     data-errolan-site="erl_yourpublickey"
     data-errolan-api="https://comments.example.com"
     data-errolan-title="My blog launch"
     data-errolan-url="https://my-blog.com/posts/launch"></div>

<script src="https://comments.example.com/sdk/errolan.js"></script>
```

### Marginalia — paragraph-anchored comments in the right margin

The killer feature: mark each paragraph on your article with `data-errolan-anchor="<stable-id>"`, then mount the widget in `marginalia` mode pointed at the article element. Errolan injects a rail of stamps next to each anchored paragraph; clicking one opens a side panel with the focused thread and a composer that posts with the anchor set.

```html
<article id="post">
  <p data-errolan-anchor="p1">First paragraph…</p>
  <p data-errolan-anchor="p2">Second paragraph…</p>
  <pre data-errolan-anchor="p6">// snippets work too</pre>
</article>

<div data-errolan-thread="post-2026-blog-launch"
     data-errolan-site="erl_yourpublickey"
     data-errolan-api="https://comments.example.com"
     data-errolan-mode="marginalia"
     data-errolan-article="#post"></div>

<link rel="stylesheet" href="https://comments.example.com/sdk/errolan.css">
<script src="https://comments.example.com/sdk/errolan.js"></script>
```

The anchor id should be **stable** — derive it from a paragraph slug, not a random number on each render. Replies to anchored comments automatically inherit the parent's anchor (so a side conversation stays pinned to the same paragraph). On screens narrower than 1100px the rail hides; the SDK still works in cadence-style as a fallback. Both directions live at `/sdk/demo.html`.

### Programmatic mount

```js
import "https://comments.example.com/sdk/errolan.js"; // or <script>

Errolan.mount(document.getElementById("comments"), {
  api: "https://comments.example.com",
  site: "erl_yourpublickey",
  thread: "post-2026-blog-launch",
  mode: "marginalia",        // or omit for cadence
  article: "#post",          // marginalia only — CSS selector for the article
  title: document.title,
  url: location.href,
});
```

## HTTP API

All site-scoped endpoints require the `X-Errolan-Site: erl_...` header. Authenticated endpoints accept `Authorization: Bearer <jwt>`.

### Auth

| Method | Path | Notes |
| -- | -- | -- |
| `POST` | `/api/auth/register` | `{email, name, password}` → `{token, user}` |
| `POST` | `/api/auth/login` | `{email, password}` → `{token, user}` |
| `GET` | `/api/auth/me` | Requires bearer token |

### Sites (admin)

| Method | Path | Notes |
| -- | -- | -- |
| `GET` | `/api/sites` | List sites and API keys |
| `POST` | `/api/sites` | `{slug, name, allowed_origins?, require_auth?}` |

### Threads & comments

| Method | Path | Notes |
| -- | -- | -- |
| `GET` | `/api/threads/{slug}?title=…&url=…&sort=best\|newest\|oldest&limit=50&before_id=N` | Auto-creates the thread; supports `If-None-Match` for 304s. Response includes the site's emoji pack. |
| `GET` | `/api/threads/{slug}/events` | Server-Sent Events stream for live updates (`?site=…` accepted for `EventSource`) |
| `POST` | `/api/threads/{slug}/comments` | `{body, parent_id?, author_name?, website?, anchor?}` — `website` is the honeypot; `anchor` is the paragraph id for marginalia mode |
| `POST` | `/api/threads/{slug}/lock` | Admin only; `{locked: bool}` |
| `PATCH` | `/api/comments/{id}` | Edit own (or admin) |
| `DELETE` | `/api/comments/{id}` | Soft delete own (or admin) |
| `POST` | `/api/comments/{id}/reactions` | `{code: "ship-it"}` — toggle one reaction. Code must be in the site's pack |
| `POST` | `/api/comments/{id}/flag` | `{reason?}` — one flag per authenticated user per comment |
| `POST` | `/api/comments/{id}/pin` | Admin only; `{pinned: bool}` |
| `POST` | `/api/comments/{id}/vote` | `{value: -1\|0\|1}` — legacy; reactions are the new model |

### Emoji pack

A site's emoji pack is a per-site whitelist of reaction codes. Users can only react with codes that exist in the pack. SVG can be inline markup or an `https://…` URL.

| Method | Path | Notes |
| -- | -- | -- |
| `GET` | `/api/emojis` | Read the pack for the site identified by `X-Errolan-Site` |
| `POST` | `/api/emojis` | Admin only; `{code, label?, svg, sort?}` — upsert by code |
| `DELETE` | `/api/emojis/{code}` | Admin only |

### Moderation & admin

| Method | Path | Notes |
| -- | -- | -- |
| `GET` | `/api/mod/flagged` | Admin only |
| `GET` | `/api/mod/audit?limit=100` | Admin only — moderation audit log |
| `GET` | `/api/admin/users?limit=50&offset=0` | Admin only |
| `POST` | `/api/admin/users/{id}/ban` | Admin only; `{banned: bool}` — cannot ban yourself or another admin |

### Security headers & rate limits

The server applies `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` (except `/sdk/`), `Referrer-Policy: no-referrer-when-downgrade`, and a conservative `Permissions-Policy`. Rate limits are token-bucket per IP:

- **Global**: 20 req/s, burst 60
- **Auth** (`/api/auth/*`): 1 every 5s, burst 5
- **Writes** (comment posts, flags): 1 every 2s, burst 10
- **Login lockout**: 5 failures within 15 min locks the account for 15 min

Request bodies are capped at 64 KB.

## Architecture

```
cmd/errolan/       — entrypoint, config, graceful shutdown
internal/db/        — SQLite open + embedded migrations
internal/models/    — domain types (Site, User, Thread, Comment, AuditEntry)
internal/store/     — repository layer
internal/auth/      — JWT + bcrypt
internal/cache/     — small TTL+LRU for site lookups
internal/ratelimit/ — in-memory token bucket
internal/lockout/   — per-account login lockout
internal/hub/       — SSE fan-out for thread events
internal/api/       — HTTP handlers, CORS, middleware (gzip, body limit, security headers, rate limit)
sdk/                — vanilla JS SDK + CSS + demo page
```

## Production notes

- Run behind a TLS-terminating proxy (nginx, Caddy). The Go binary speaks plain HTTP.
- Set `ERROLAN_JWT_SECRET` to a stable value (≥32 random bytes). Otherwise tokens are invalidated on every restart.
- Set `ERROLAN_TRUST_FORWARDED=true` **only** when the proxy strips spoofed `X-Forwarded-For` / `X-Real-IP` headers — otherwise IP-based rate limiting becomes trivially bypassable.
- Use a persistent volume for `ERROLAN_DB`. SQLite is the default for simplicity — for higher write throughput, swap in Postgres (the store layer is the only place that needs to change).
- Restrict `allowed_origins` per site to your real frontends.
- The site `api_key` is intended to be public (like a Disqus shortname). It does **not** grant moderation rights on its own.
- The moderation audit log (`audit_log` table, also exposed at `/api/mod/audit`) records site creation, thread locks, comment pins/edits-by-admins/deletes-by-admins, and user bans.

## License

MIT — go ahead and self-host it.
