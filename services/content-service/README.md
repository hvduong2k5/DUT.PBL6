# Content Service (MS-10)

Standalone article CMS for DUT.PBL6. Go 1.22, Chi v5, pgx/v5, PostgreSQL 16,
Zerolog and Testify. No Redis, Kafka, ORM, gateway or paid service is required.

## Run locally

From this directory:

```sh
docker compose up --build -d
curl http://localhost:8010/livez
curl http://localhost:8010/readyz
curl http://localhost:8010/api/v1/articles
```

Compose binds only loopback, uses a dedicated `content_db`, limits DB/API memory
to 256/128 MiB, and initializes the up migration on **a fresh volume only**. Local
credentials in Compose/.env.example are development fixtures. Use separate
credentials and TLS for deployed databases. Do not attach another service's DB.

For an existing dedicated PostgreSQL database, apply the migration explicitly:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_content.up.sql
go run ./cmd/server
```

Set environment variables first; the executable does not load `.env` files:

```powershell
$env:DATABASE_URL = 'postgres://content_dev:local_content_only@localhost:15410/content_db?sslmode=disable'
$env:HTTP_PORT = '8010'
go run ./cmd/server
```

`DATABASE_URL` is required. `HTTP_PORT` defaults to 8010, `DB_MAX_CONNS` to 5
(accepted 1..50), and `LOG_LEVEL` to info. Health/readiness check connectivity and
the four migrated tables; they return 503 if unavailable. Liveness returns 200
while the HTTP process runs. DB unavailability does not prevent starting the
process; readiness remains false and requests return sanitized errors.
SIGINT/SIGTERM drains HTTP requests for up to 15 seconds and closes the pool.

The down migration removes **this service's tables and data**. Use only on a
disposable database or with an explicitly approved backup/rollback plan:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_content.down.sql
```

## Security boundary

The shipped executable deliberately has **no identity verifier**: all admin
routes return 401. Public and health routes work normally. `X-User-ID`,
`X-User-Role`, `X-Internal-Token` and arbitrary Bearer values cannot grant access.
There is no mock-auth environment switch, development bypass or shared token.

`transport/http/middleware.Verifier` is the integration boundary. A real adapter
must validate credentials and produce a verified UUID identity and mapped role;
wire it through `RouterConfig.Verifier` in `cmd/server`. JWT integration must
validate signature, trusted issuer, audience and expiry; gateway integration
must authenticate the gateway and prevent direct forwarded-header access.
The external Auth/Gateway contract is not available in this implementation.
Mock verifiers exist only in automated tests. Both middleware and usecases
enforce permissions:

| Operation | Roles |
| --- | --- |
| Read/create/edit/submit/hide | ADMIN, CONTENT_EDITOR, CONTENT_APPROVER |
| Approve/reject/publish | ADMIN, CONTENT_APPROVER |

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
| POST | `/api/v1/admin/articles/{id}/hide` | Remove public visibility, keep revisions |

Lists accept `page` (default 1), `limit` (default 20, max 100), `q` (literal,
case-insensitive title/body substring), and `category` (exact match). Admin also
accepts `status`: DRAFT/PENDING/APPROVED/REJECTED/PUBLISHED. Sorting is stable
`created_at DESC, id DESC`; filtering public lists uses the **published** snapshot.
Responses have `{items,total,page,limit}`; an empty page returns `items: []`.
Lists omit article bodies; fetch the detail endpoint to read the body. Each page
uses one joined data query plus a count query, without loading revision history.
Status describes the latest editorial revision; `visible` is independent, so a
published article can simultaneously have a new draft. Admin detail/mutations
return current revision, revision history, decisions, slug history, timestamps,
published pointer, and numeric version. Admin lists omit history.

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
approved revision. Hiding retains the pointer/history; to republish hidden
content, edit, submit, approve and publish a new revision.

Slug reservations are global and permanent per article, including draft URLs.
An article may reuse its own historical slug. After publishing a different slug,
the old URL returns 404; redirect delivery is outside V1 scope.

Errors use `{"error":{"code":"...","message":"..."}}`: 400 malformed input,
401 missing/invalid verified identity, 403 insufficient role, 404 missing content,
409 slug/version conflict or invalid publication state, 422 invalid fields or other
invalid state transitions, 500 unexpected server errors. An authenticated editor
attempting publication receives 403, including when the revision is approved.
Unknown JSON fields, trailing JSON and oversized bodies are rejected. Database
errors are logged server-side and never sent to clients.

## Storage and tests

`articles` stores version and editorial/public pointers. `article_revisions`
stores immutable content plus revision-owned SEO/media JSONB. `approval_decisions`
records approver, status, reason and timestamp. `slug_history` reserves URLs.
Composite foreign keys guarantee revision ownership; SQL checks/indexes support
integrity and bounded pagination. Mutation locks the article, checks expected
version, writes snapshot/decision/slug and advances version in one transaction.
Read transactions use repeatable read to keep public pointers and content coherent.

```sh
go fmt ./...
go test ./...
go vet ./...
go build ./...
# Run against a disposable dedicated DB with CREATE SCHEMA permission:
CONTENT_TEST_DATABASE_URL="$DATABASE_URL" go test -count=1 ./tests/integration
```

Integration tests create a random `content_test_<uuid>` schema and remove only
that schema. They verify up/down/up, ownership/immutability constraints, slug
rollback, simultaneous version/slug races, editorial lifecycle and real HTTP
requests over PostgreSQL. Without the test URL these tests explicitly skip;
unit/HTTP authorization tests still run. Test fixtures are never wired into main.

Layout: `cmd/server`; `internal/config`, `domain`, `usecase`, `repository`,
`infrastructure/postgres`, `transport/http/{handlers,middleware}`; `migrations`;
`tests/{integration,testhelper}`; `docs/{MASTER_PLAN,ACTIVE_PLAN,CURRENT_STATE}.md`.
Optional integrations and other service source changes are outside this branch's
scope. See `docs/CURRENT_STATE.md` for actual validation and the next action.
