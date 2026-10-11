# Content Service operating contract

- Modify this directory only, while on the designated new branch feat/content-service.
- No push, PR, merge, other branch/service edits or destructive Git operations.
- Resume from docs/CURRENT_STATE.md, MASTER_PLAN.md and ACTIVE_PLAN.md, then
  inspect only the relevant implementation.
- Go 1.22; Chi v5, pgx/v5, Zerolog. Keep the production identity boundary closed
  until a trusted verifier contract is available. Test mock identities stay in tests.
- Content snapshots are immutable, mutations versioned and transactional, public
  reads use only the visible published revision. Preserve these contracts.
- From this module: go fmt ./..., go test ./..., go vet ./..., go build ./....
  CONTENT_TEST_DATABASE_URL enables random-schema real PostgreSQL tests.
- Run integration migrations only against a dedicated disposable database.
