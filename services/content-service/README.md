# Content Service (MS-10)

Standalone article CMS for DUT.PBL6. Go 1.22, Chi v5, pgx/v5, PostgreSQL 16,
Zerolog and Testify. No Redis, Kafka, ORM or paid service is required.

**Deployment status:** the image builds (`Dockerfile`, `Dockerfile.coolify`) but this
service is **not yet wired into Coolify CD** (`.github/workflows/cd-deploy.yml`,
`docs/06_operations/cd-coolify.md`, Kong route). Those are tracked as a follow-up.

## Run locally

From this directory:

```sh
docker compose up --build -d
curl http://localhost:8010/livez
curl http://localhost:8010/readyz
curl http://localhost:8010/api/v1/articles
```

Compose binds only loopback, uses a dedicated `content_db`, and limits DB/API memory
to 256/128 MiB. The image entrypoint runs `content-migrate up` before the API
starts (set `RUN_MIGRATIONS=false` to skip, e.g. when a release job migrates).
Local credentials in Compose/.env.example are development fixtures. Use separate
credentials and TLS for deployed databases. Do not attach another service's DB.

Without Docker, migrate and run explicitly (the executable does not load `.env`):

```powershell
$env:DATABASE_URL = 'postgres://content_dev:local_content_only@localhost:15410/content_db?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/server
```

## Migrations

`migrations/NNNNNN_name.{up,down}.sql`, embedded in the binary and recorded in
`schema_migrations`. `cmd/migrate up` applies pending files under an advisory lock
(safe with several replicas); a database created earlier by `psql` is adopted
(version 1 is recorded, not re-run). `cmd/migrate down` reverts **one** step and
**destroys that step's data**: use it only on a disposable database or with an
approved backup. Every file must be wrapped in `BEGIN; ... COMMIT;` so it also runs
through `psql -v ON_ERROR_STOP=1 -f`.

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | required | PostgreSQL DSN |
| `HTTP_PORT` | 8010 | listen port |
| `LOG_LEVEL` | info | zerolog level |
| `DB_MAX_CONNS` / `DB_MIN_CONNS` | 5 / 0 | pool bounds (max 1..50) |
| `DB_CONN_LIFETIME` / `DB_CONNECT_TIMEOUT` | 30m / 3s | pool connection limits |
| `HTTP_REQUEST_TIMEOUT` | 10s | per-request deadline |
| `READINESS_TIMEOUT` | 1s | database probe deadline for `/readyz` |
| `SHUTDOWN_TIMEOUT` | 15s | drain time on SIGINT/SIGTERM |
| `MAX_BODY_BYTES` | 1048576 | JSON request body cap |
| `CONTENT_JWT_ISSUER` | empty | pinned token issuer |
| `CONTENT_JWT_PUBLIC_KEY` | empty | path to the issuer's RSA public key (PEM) |
| `CONTENT_JWT_PUBLIC_KEY_B64` | empty | Coolify only: base64 PEM, decoded by `Dockerfile.coolify` |
| `CONTENT_STAFF_AUDIENCE` | omama-api | required `aud` |
| `RUN_MIGRATIONS` | true | container entrypoint only |

Durations use Go syntax (`10s`, `2m`). Out-of-range values fail startup.
`/readyz` and `/healthz` check connectivity and every migrated table and return 503
otherwise; `/livez` returns 200 while the process runs. A database outage does not
stop the process; readiness stays false. SIGINT/SIGTERM drains requests and closes
the pool.

Rate limiting and CORS are **not** implemented here: the gateway (Kong) owns them
for the public routes. Do not expose this service directly to browsers.

## Security boundary

Admin routes accept only an Identity-issued **staff access token** and fail closed:

- If `CONTENT_JWT_ISSUER` or the public key is missing or invalid the admin API
  answers 401 and the startup log says why. Public and health routes still work.
- The token must be RS256, carry the configured `iss` and `aud`, `exp` and `iat`,
  a `sub`, and a UUID `user_id`. Authority comes only from the signed
  `permission_grants` claim (`PERM#GLOBAL` or `PERM#SCOPE_TYPE#SCOPE_ID`, same
  contract as Order, Profile, Fulfillment). Malformed grants, a bad signature,
  expired/wrong issuer/audience → **401**.
- A valid token that is not a staff token (no `permission_grants`, a `service`
  claim, an empty or non-content grant list) → **403**, not 401. So a customer is
  403 even if an old token carries an `ADMIN` role.
- `ADMIN` is a role of an EMPLOYEE and **never bypasses** a missing grant. Role
  names are not read at all; `X-User-ID`, `X-User-Role`, `X-Internal-Token` and
  arbitrary Bearer values cannot grant access. There is no mock-auth switch.
- Only `GLOBAL` grants authorize in this version. `CATEGORY`/ownership scopes are not
  implemented and such grants are ignored (the call is 403).

| Operation | Permission |
| --- | --- |
| list, read | `CONTENT_ARTICLE_VIEW` |
| create | `CONTENT_ARTICLE_CREATE` |
| edit (PATCH) | `CONTENT_ARTICLE_EDIT` |
| submit | `CONTENT_ARTICLE_SUBMIT` |
| approve | `CONTENT_ARTICLE_APPROVE` |
| reject | `CONTENT_ARTICLE_REJECT` |
| publish, restore | `CONTENT_ARTICLE_PUBLISH` |
| hide | `CONTENT_ARTICLE_HIDE` |

Both the middleware and the use cases enforce them. These permission codes must be
seeded in Identity (IAM) and granted to roles before anyone can use the admin API;
that seed lives outside this service. Mock verifiers exist only in tests.

Body is text/Markdown. Clients must sanitize any generated HTML. Media URLs are
HTTP(S) references only; this service never fetches them or uploads objects.
SEO JSON-LD is a bounded JSON object and must be safely serialized by consumers.

## REST contracts

| Method | Path | Behavior |
| --- | --- | --- |
| GET | `/livez`, `/readyz`, `/healthz` | Liveness and migrated database readiness |
| GET | `/api/v1/articles` | Visible published snapshots only |
| GET | `/api/v1/articles/{slug}` | Published title/body/SEO/media; hidden/missing = 404 |
| GET / POST | `/api/v1/admin/articles` | Administrative list / create draft |
| GET / PATCH | `/api/v1/admin/articles/{id}` | Full history / append draft snapshot |
| POST | `/api/v1/admin/articles/{id}/submit` | DRAFT or REJECTED -> PENDING |
| POST | `/api/v1/admin/articles/{id}/approve` | PENDING -> APPROVED, audit decision |
| POST | `/api/v1/admin/articles/{id}/reject` | PENDING -> REJECTED, required reason |
| POST | `/api/v1/admin/articles/{id}/publish` | APPROVED -> PUBLISHED, replace public pointer |
| POST | `/api/v1/admin/articles/{id}/hide` | Remove public visibility, keep revisions (optional `reason`) |
| POST | `/api/v1/admin/articles/{id}/restore` | Make the hidden, already published snapshot public again |

Lists accept `page` (default 1), `limit` (default 20, max 100), `q` (literal,
case-insensitive title/body substring), and `category` (exact match). Admin also
accepts `status`: DRAFT/PENDING/APPROVED/REJECTED/PUBLISHED. Sorting is stable
`created_at DESC, id DESC`; filtering public lists uses the **published** snapshot.
Responses have `{items,total,page,limit}`; an empty page returns `items: []`.
Lists never select or return article bodies (the `q` filter does read them);
fetch the detail endpoint to read the body. Each page is one joined data query plus
a count query, without loading revision history. **Known limit:** `q` is a
sequential substring scan with no index; keep result sets small or add a
`tsvector`/trigram index before large catalogs.
Status describes the latest editorial revision; `visible` is independent, so a
published article can simultaneously have a new draft. Admin detail/mutations
return current revision, revision history, decisions, slug history, the `audit`
trail, timestamps, published pointer, and numeric version. Admin lists omit history.

Create JSON (`Content-Type: application/json`):

```json
{
  "title": "Hue heritage", "slug": "hue-heritage", "body": "Markdown story",
  "category": "heritage",
  "seo": {"title":"Hue heritage", "description":"A local story", "keywords":["Hue"], "canonical_url":"", "schema":{"@type":"Article"}},
  "media": [{"id":"21f3dfd5-62f8-43fb-8fcb-a29969836a36", "url":"https://example.com/photo.jpg", "alt":"Hue", "kind":"image"}]
}
```

Creation returns 201 with Location, ETag and version 1. PATCH requires `version`
plus at least one optional field (title/slug/body/category/seo/media). Omitted or
null fields remain unchanged; SEO and media replace their entire value, `{}` and
`[]` clear them. Example: `{"version":1,"body":"Revised story"}`.
Actions require `{"version":2}`; reject additionally requires a nonblank `reason`.
Use the latest response version for the next mutation; ETag is informational,
the expected version is passed in JSON. Stale versions return 409.

Every edit creates a new DRAFT snapshot. PENDING snapshots cannot be edited.
Editing approved/published content requires review again; the public pointer
continues to serve the previous snapshot. Publishing only selects the currently
approved revision. **Hide** keeps the pointer and history and only flips
`visible`; **restore** re-exposes that same snapshot without a new review. To
publish newer content after a hide, edit, submit, approve and publish as usual.

Slug reservations are global and permanent per article, including draft URLs.
An article may reuse its own historical slug. After publishing a different slug,
the old URL returns 404; redirect delivery is outside V1 scope.

### Idempotency-Key

Admin `POST` requests (create and every action) honor an optional
`Idempotency-Key` header (1..255 printable ASCII, no spaces). Keys are scoped to the
verified user. The same key with the same method, path and body replays the first
response (status, body, `Location`, `ETag`) with `Idempotent-Replayed: true`; the
same key with a different request is **409 `IDEMPOTENCY_KEY_REUSED`**; a request
still running is **409 `IDEMPOTENCY_IN_PROGRESS`** (`Retry-After: 1`). 5xx outcomes
are not stored, so a retry runs again. A crashed request's key can be taken over
after two minutes. A request without the header is not deduplicated (optimistic
`version` and slug reservation still prevent duplicates). Records are not purged
automatically yet (`idempotency_keys.created_at` is indexed for a cleanup job).

### Caching

Published reads (`/api/v1/articles`, `/api/v1/articles/{slug}`) return
`Cache-Control: public, max-age=60, s-maxage=300` and a strong `ETag`, and answer
`If-None-Match` with 304. Everything else, including every error and the whole admin
API, is `no-store`. A hide therefore reaches a CDN after at most `s-maxage`
(5 minutes) unless the gateway purges.

### Errors

Every failure uses the flat envelope
`{"code":"...","message":"...","requestId":"...","errors":[]}`; `requestId` equals the
`X-Request-ID` response header. This covers 400, 401, 403, 404, 405, 409, 422, 500
(panics included) and the 503 readiness body.

| Status | When |
| --- | --- |
| 400 `BAD_REQUEST` | malformed JSON, unknown field, trailing JSON, oversize body, bad pagination/ID/`Idempotency-Key` |
| 401 `UNAUTHENTICATED` | missing/invalid/expired token or malformed grants |
| 403 `FORBIDDEN` | non-staff principal or missing permission |
| 404 `NOT_FOUND` | missing, unpublished or hidden content; unknown route |
| 409 | `VERSION_CONFLICT`, `DUPLICATE_SLUG`, `INVALID_STATE` (every invalid transition), `IDEMPOTENCY_*` |
| 422 `VALIDATION_ERROR` | invalid fields |
| 500 `INTERNAL_ERROR` | unexpected errors, logged server-side and never leaked |

## Storage and tests

`articles` stores version and editorial/public pointers. `article_revisions` stores
immutable content plus revision-owned SEO/media JSONB; a write only **inserts** new
revisions and moves one revision's status at a time under a compare-and-set, and a
PUBLISHED revision can never change status (trigger). `approval_decisions` records
approver, status, reason and timestamp. `audit_events` is an append-only trail of
every action (create, edit, submit, approve, reject, publish, hide, restore) with
actor, revision and reason. `slug_history` reserves URLs. `idempotency_keys` stores
replayable responses. Composite foreign keys guarantee revision ownership; SQL
checks/indexes support integrity and bounded pagination. A mutation locks the
article, checks the expected version, writes everything and advances the version in
one transaction. Read transactions use repeatable read so pointers and content
stay coherent. Unique violations map to 409, never 500.

```sh
go fmt ./...
go test ./...
go vet ./...
go build ./...
# Run against a disposable dedicated DB with CREATE SCHEMA permission:
CONTENT_TEST_DATABASE_URL="$DATABASE_URL" go test -count=1 ./tests/integration
```

Integration tests create a random `content_test_<uuid>` schema and remove only that
schema. They verify the migration runner (up/down/up, adoption of a psql-created
database), ownership/immutability/audit constraints, DB error mapping, the Postgres
idempotency store, slug rollback, simultaneous version/slug races, the editorial
lifecycle with hide/restore, and real HTTP requests over PostgreSQL. Without the
test URL they explicitly skip; unit and HTTP tests (including real RS256 token
verification) still run. Test fixtures are never wired into main.

Layout: `cmd/{server,migrate}`; `internal/config`, `domain`, `usecase`, `repository`,
`infrastructure/{postgres,migrate}`, `transport/http/{handlers,middleware}`;
`migrations` (SQL + embed); `tests/{integration,testhelper}`;
`docs/{MASTER_PLAN,ACTIVE_PLAN,CURRENT_STATE}.md`. Optional integrations
(Kafka/outbox, Product SEO, media upload, scheduling) are outside this branch's
scope. See `docs/CURRENT_STATE.md` for actual validation and the next action.
