# AI Agent Service production deployment

## Runtime

- Coolify application: `services/ai-agent-service/Dockerfile`, target `runtime`, port `8019`.
- Health check: `GET /health/ready` (database is mandatory when `AI_FIXTURES=0`).
- Public gateway routes: shopping conversations, recommendations and consent only.
- `/api/v1/ai/knowledge/products/sync`, `/metrics` and health endpoints stay on the private network.

Set the GitHub repository variable `COOLIFY_APP_UUID_AI` to the Coolify application UUID before selecting `ai` or `all` in the CD workflow. The workflow deliberately fails instead of guessing this identifier.

## Required configuration

Start from `services/ai-agent-service/.env.example` and replace every example endpoint, issuer and audience with the deployed values. Production requires:

- `AI_FIXTURES=0`;
- PostgreSQL 16 with pgvector and a dedicated `ai_db` credential;
- Bedrock/Titan IAM permissions and the configured AWS region;
- Catalog, Order, Analytics and Identity private service URLs;
- the customer JWT issuer, audience and JWKS URL;
- `AI_AUTO_MIGRATE=1` for advisory-lock-protected startup migrations;
- `AI_IDENTITY_BASE_URL` so bearer identities are resolved as ACTIVE CUSTOMER principals.

Mount two independent random values of at least 32 bytes:

- `/run/secrets/ai_internal_token` for private knowledge ingestion;
- `/run/secrets/ai_guest_token_secret` for deterministic, retry-safe guest conversation tokens.

Do not expose either value as a browser header or configure the knowledge route in Kong. Secret rotation for `ai_guest_token_secret` invalidates outstanding guest conversation tokens, so retain the previous deployment secret until the 15-day guest retention window has elapsed or perform a coordinated session reset.

## Database rollout

Startup migration is serialized with a PostgreSQL advisory lock. Migration `002_production_readiness.sql` adds guest token binding, idempotency records and retention indexes without rewriting the initial migration.

Before first traffic, confirm `/health/ready` returns `200` and the `schema_migrations` table contains both `001_ai_agent.sql` and `002_production_readiness.sql`.

## Gateway and smoke check

Apply `services/api-gateway/config/kong.yml`, then restart Kong after the AI container is deployed. The route has an AI-specific IP rate limit and never forwards the internal ingestion endpoint.

The CD smoke check calls:

```text
GET https://api-pbl6.hvduong.qzz.io/api/v1/ai/recommendations?placement=home
```

An empty `items` array is healthy. A deployment failure is rolled back through Coolify Deployments, followed by a Kong restart.

## Operational checks

- Alert on `/health/ready != 200`, elevated `ai_tool_calls{result!="ok"}`, and p95 `ai_tool_seconds` near the configured tool timeout.
- Use `X-Request-Id` to correlate API errors and logs.
- `agent_runs` and `tool_calls` persist execution outcomes without storing bearer tokens.
- The in-process cleanup removes expired conversations hourly by default; monitor cleanup errors in service logs.
- Shipping estimates intentionally defer to checkout until complete address and parcel inputs exist.
