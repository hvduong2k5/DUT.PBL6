# Frontend API contract handoff

This directory is the backend handoff for the Customer Web and Internal Employee Web APIs. It is independent from any local fixture or mock-server implementation.

## Files

- `customer-stable.openapi.yaml`: implementation-ready Customer API.
- `customer-draft-planned.openapi.yaml`: proposed Customer capabilities that still need product, security, storage, or workflow decisions.
- `admin-stable.openapi.yaml`: implementation-ready employee session, management-read, and approved workbench API.
- `admin-draft-planned.openapi.yaml`: proposed employee mutations that are not yet policy-stable.
- `customer-api-state-matrix.md`: required Customer success/loading/empty/error/auth state coverage.
- `admin-api-state-matrix.md`: required Admin success/loading/empty/error/auth state coverage.

## Lifecycle rules

- `stable`: backend can implement now; breaking changes require a contract version change.
- `draft`: endpoint shape is concrete enough for review, but at least one named policy or workflow decision is unresolved.
- `planned`: capability is required by the backlog, but endpoint shape and/or workflow may still change substantially.

The draft-planned files are not imported by the stable files. Moving an operation to stable requires an explicit decision, completed authorization rules, examples for every applicable state, and contract tests.

## Shared rules

- All errors use the shared `Error` envelope with stable `code`, safe `message`, and `requestId`.
- Empty collections return `200` with empty arrays and zero totals.
- `401` means authentication is absent or invalid; `403` means identity is known but permission, capability, ownership, or scope is insufficient.
- Admin authorization is permission-and-scope based. Role names are assignment bundles and must not be used as the only authorization check.
- `Idempotency-Key` is required by implementation policy for retryable create/action operations even where an older client has not supplied it yet.
- Upload proposals use direct-to-object-storage sessions; business APIs receive immutable `fileId` references, not binary bodies or arbitrary URLs.

## Validation

Run:

```powershell
npx @redocly/cli lint .\docs\03_api_specs\frontend-contracts\customer-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\frontend-contracts\customer-draft-planned.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\frontend-contracts\admin-stable.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\frontend-contracts\admin-draft-planned.openapi.yaml
```

The generated artifacts can be refreshed with `node scripts/generate-frontend-api-contracts.mjs` after an approved contract change. Do not edit generated rows or inferred stable response examples without updating the generator as well.
