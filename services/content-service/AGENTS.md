# Content Service operating contract

- Work on the designated branch feat/content-service. Modify this directory, plus
  the contract and state-matrix updates the repository rules require in
  `docs/03_api_specs/` when the REST API changes. Ask before editing contracts,
  `infra/`, `.github/workflows/` or other services.
- No push, PR, merge or destructive Git operations without explicit approval.
  The repository rules take precedence over this file.
- Resume from docs/CURRENT_STATE.md, MASTER_PLAN.md and ACTIVE_PLAN.md, then
  inspect only the relevant implementation.
- Go 1.22; Chi v5, pgx/v5, Zerolog. The admin boundary is the Identity staff token
  (RS256, `permission_grants`); keep it fail-closed and never trust client headers
  or role names. Test mock identities stay in tests.
- Content snapshots are immutable, mutations versioned and transactional, public
  reads use only the visible published revision. Preserve these contracts.
- From this module: go fmt ./..., go test ./..., go vet ./..., go build ./....
  CONTENT_TEST_DATABASE_URL enables random-schema real PostgreSQL tests.
- Run integration migrations only against a dedicated disposable database.
