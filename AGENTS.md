# Agent Rules — DUT.PBL6 (Repository-wide)

## Commit Message Convention (MANDATORY)

**All commit messages MUST be written in 100% English.** No Vietnamese text in commit subject or body.

### Format
```
<type>(<scope>): <short description in English>

- Detail of change 1
- Detail of change 2
- ./gradlew test --offline: PASS
```

### Types
| Type | Meaning |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Refactor without behavior change |
| `docs` | Documentation only |
| `chore` | Build, config, maintenance |
| `build` | Build system changes |
| `ci` | CI/CD changes |
| `test` | Adding or updating tests |
| `style` | Formatting, missing semicolons, etc. |
| `perf` | Performance improvement |

### Rules
- Each commit must do **one thing** only (one type, one scope).
- Do not squash unrelated changes into one commit.
- Commit message must be clear. Do not use vague words like "fix", "update", "done".
- Subject line: `<type>(<scope>): <short description>` — English only.
- Body bullets: English only.

### Examples (correct)
```
feat(catalog): add PATCH product and SKU admin endpoints

- Implement PATCH /products/{id} with full content update including story
- Implement PATCH /skus/{id} for variant attributes and sale status
- skuCode is immutable; price changes go through /prices endpoint
- Add unit tests for service and controller layers
- ./gradlew test --offline: PASS
```

```
fix(catalog): escape XML comparison operators in ProductMapper.xml

- Replace raw '<' and '>' with '&lt;' and '&gt;' in MyBatis mapper XML
- Fix SAXParseException causing Spring context startup failure
- ./gradlew test --offline: PASS
```

## Contract Docs
- Admin catalog API contract: `docs/03_api_specs/admin-stable.openapi.yaml`
- Admin state matrix: `docs/03_api_specs/admin-api-state-matrix.md`
- Do not create new catalog doc files outside `docs/03_api_specs/`.
- `catalog-checkout.openapi.yaml` must be kept (referenced by `packages/contracts/mvp/registry.yaml`).

## Catalog Service Specific Rules
See `services/catalog-service/CATALOG_SERVICE_AGENT_RULES.md`.
