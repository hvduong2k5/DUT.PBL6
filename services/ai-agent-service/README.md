# ai-agent-service (MS-19)

One FastAPI service hosting the platform's AI agent modules. It is **one**
service: the business modules inside it are code organisation, not separate
microservices.

| Module | Phase | Scope |
|---|---|---|
| `shopping_advisor` | 1 | Customer shopping & gift advice chat (US-AI-01) |
| `recommendation` | 1 | Personalised suggestions (US-AI-04) via the Analytics recommendation API |
| `knowledge_assistant` | 2 | Internal knowledge Q&A (US-AI-06) |
| `support_copilot` | 2 | CSKH ticket copilot (US-AI-02) |
| `content_assistant` | 3 | Content & SEO drafts (US-AI-03) |
| `strategic_advisor` | 3 | Executive DSS action cards (EPIC 25) |

## Decided architecture (2026-11-09)

- Python + FastAPI, port **8019**, own PostgreSQL database `ai_db` + **pgvector**.
- Production LLM: **Amazon Bedrock Converse API**, single model
  `amazon.nova-pro-v1:0`; embeddings: Titan Embeddings V2.
- Dev/test LLM: `FakeProvider` (deterministic); Gemini API optional for manual
  dev. Business code never branches on the provider.
- The LLM never invents prices or stock: facts come from **Catalog** and
  **Inventory** through the Tool Execution Gateway (read-only tools only).
- Controlled agent runtime: deterministic workflow steps (stock check is
  mandatory), the LLM decides only inside its allowance.
- Conversation retention: 30 days (logged-in customers), 15 days (guests).

## Run (dev)

```bash
docker compose -f compose.dev.yml up --build
# http://localhost:8019/health
python -m ai_agent_service migrate   # apply migrations/postgres/*.sql
```

## Test

```bash
docker compose -f compose.test.yml up --build --abort-on-container-exit
```
