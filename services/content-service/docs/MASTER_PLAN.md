# Content Service roadmap

Scope: standalone MS-10 article CMS, PostgreSQL and REST only. All changes stay
inside services/content-service on feat/content-service, based on origin/dev
2291ef5a44dd3a6f7985297903bfec01b325b087. No push or PR is authorized.

1. DONE Foundation: environment configuration, pool, lifecycle, health, Docker files.
2. DONE Editorial core: immutable content snapshots, version checks, review decisions,
   publication pointer, slug reservations/history, PostgreSQL transactions.
3. DONE REST: separate public/admin DTOs, verified identity boundary and role checks.
4. DONE Core verification: domain, HTTP, database migration and concurrency integration;
   inspect final diff and update CURRENT_STATE.

Dependencies: 2 depends on 1; 3 on 2; 4 verifies all. Detailed current work is in
ACTIVE_PLAN.md. External auth verification is a separate integration after this
implementation; production admin must deny access until it exists. Container build/run
remains NOT RUN because the local Docker daemon was unavailable.

Decisions: match Profile's Go/Chi/pgx/Zerolog layers and compatible versions;
adapt Inventory's transactional callback/rollback and locking; adapt Catalog's
review transitions and optimistic versioning. Current revision and public
revision are separate. A new draft cannot alter the public snapshot. Slugs are
reserved permanently per article; old public URLs return 404 after replacement
(history is retained for a future redirect adapter). SEO/media are revision-owned
JSONB values, not another service's entities. Body is plain text/Markdown and
must never be rendered as unsanitized HTML by consumers.

Out of scope: other service edits, JWT provider selection, gateway provisioning,
Kafka/outbox, storage uploads, Redis, Elasticsearch, product SEO, frontend.
