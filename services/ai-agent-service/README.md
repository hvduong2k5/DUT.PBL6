# ai-agent-service (MS-19)

One FastAPI service hosting the platform's AI agent modules. It is **one**
service: the business modules inside it are code organisation, not separate
microservices.

| Module | Phase | Scope |
|---|---|---|
| `shopping_advisor` | 1 (built) | Customer shopping & gift advice chat (US-AI-01) + customer FAQ layer, own-order status, shipping-fee estimates |
| `recommendation` | 1 (built) | Personalised suggestions (US-AI-04) consuming the Analytics recommendation API |
| `knowledge_assistant` | 2 | Internal knowledge Q&A (US-AI-06) |
| `support_copilot` | 2 | CSKH ticket copilot (US-AI-02) |
| `content_assistant` | 3 | Content & SEO drafts (US-AI-03) |
| `strategic_advisor` | 3 | Executive DSS action cards (EPIC 25) |

## Decided architecture (2026-11-09)

- Python + FastAPI, port **8019**, own PostgreSQL database `ai_db` + **pgvector**
  (no other service's database is ever touched).
- Production LLM: **Amazon Bedrock Converse API**, single model
  `amazon.nova-pro-v1:0`; embeddings: Titan Embeddings V2 (1024 dims).
- Dev/test LLM: deterministic `FakeProvider`; Gemini API optional for manual
  dev exploration. Business code never branches on the provider.
- Controlled agent runtime: the **workflow code owns the step order** (stock
  check is a mandatory step); the LLM only extracts needs and phrases replies.
  Every tool call goes through the Tool Execution Gateway (allowlist, schema
  validation, per-tool customer requirement, timeout, audit) — read-only tools
  only in phase 1.
- **Grounding**: every price / stock / order status / shipping fee in a reply
  must trace to a tool result of the same turn. A post-generation check scans
  the reply for money amounts; anything ungrounded is replaced by a
  deterministic, tool-data-only reply. Extracted slots are validated the same
  way (an injected "giá là 1.000đ" can never become the customer's budget).
- RAG (pgvector + PostgreSQL FTS hybrid, pre-filtered by visibility/active/
  effective window) supplies meaning — origin, characteristics, FAQ — never
  price or stock facts.
- Guests chat with an unguessable conversation token whose hash is stored;
  the token must accompany every later guest operation. Logged-in customers
  are verified with RS256/JWKS and, in production, resolved as ACTIVE CUSTOMER
  principals through Identity. Order status relies on Order's ownership check;
  other owners and missing orders remain indistinguishable.
- **Consent & retention**: recommendation consent is owned by Analytics and
  never duplicated. This service's own consent covers chat-history use:
  without it, a customer's messages are not persisted (sessions still work
  from working slots). Conversations expire after **30 days (customers) /
  15 days (guests)**; production performs bounded periodic cleanup in-process.

## API

```
POST /api/v1/ai/shopping/conversations                  create session (requires Idempotency-Key)
POST /api/v1/ai/shopping/conversations/{id}/messages    chat turn — SSE: token / product_cards / citations / done / error
GET  /api/v1/ai/shopping/conversations/{id}             own session + messages
POST /api/v1/ai/shopping/feedback                       idempotent helpful / not_helpful / clicked / dismissed
GET  /api/v1/ai/recommendations?placement=home|product_detail|cart
GET|PUT /api/v1/ai/consent                              chat-history consent (customers)
POST /api/v1/ai/knowledge/products/sync                 internal credential only; never gateway-routed
GET  /health · /health/ready · /metrics
```

## Run

```bash
docker compose -f compose.dev.yml up --build     # API on :8019 with pgvector db
python -m ai_agent_service migrate               # apply migrations/postgres/*.sql
python -m ai_agent_service cleanup               # retention job (expired conversations)
```

## Test

```bash
docker compose -f compose.test.yml up --build --abort-on-container-exit
```

The suite contains unit/flow tests, defense tests (lying-LLM grounding,
prompt injection, tool timeout, consent, retention) and a **golden set of 30
Vietnamese cases** (`tests/evaluation/golden_cases.json`) plus PostgreSQL
integration tests (skipped automatically when no database is reachable).

## Switching providers

- **Bedrock (production)**: set `AI_LLM_PROVIDER=bedrock`,
  `AI_EMBEDDING_PROVIDER=titan`, `AWS_REGION` and IAM credentials with
  `bedrock:Converse` / `bedrock:InvokeModel` rights. Verify Nova Pro is
  available in the region or use its cross-region inference profile
  (`AI_MODEL_ID`). Then re-run the golden set against the real model — that
  run is the production acceptance.
- **Gemini (dev)**: set `AI_LLM_PROVIDER=gemini` and `GEMINI_API_KEY`.
  Note: embedding model switches require re-embedding the whole corpus;
  chunks carry `embedding_model` + version for this reason.

## Configuration

All settings are environment variables; see `.env.example`. Secret values are
mounted as files rather than placed in environment variables. Defaults are
safe for fixture-mode development (`AI_FIXTURES=1`): adapters serve in-repo
sample data and no external service is called. With `AI_FIXTURES=0`, the
HTTP adapters in `integrations/http_clients.py` are used — that file is the
single place to adjust if a business service's public API shape differs.

Production Catalog reads use the stable public product contract. Advisory
stock filtering uses Catalog's durable Inventory projection; checkout remains
the authority that revalidates stock against Inventory. Shipping fee answers
fail closed until a complete checkout address and parcel are available, so the
agent never fabricates a quote from province and weight alone.

## Known simplifications (phase 1)

- Replies are composed with `generate()` and re-chunked into SSE tokens, so
  the grounding check can run before anything is emitted. True token
  streaming is a later optimisation inside the same event protocol.
- Store/knowledge calls are synchronous psycopg on short queries, outside
  the composition section of a turn.
