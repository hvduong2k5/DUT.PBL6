# Frontend contract references

The MVP target is [../mvp-customer.openapi.yaml](../mvp-customer.openapi.yaml). Read [the migration](../../../packages/contracts/mvp/MIGRATION.md) before changing clients or BFFs.

`customer-stable.openapi.yaml`, `customer-draft-planned.openapi.yaml` and `customer-api-state-matrix.md` are content-preserved snapshots with LF-normalized newlines from the pinned Mobile Git commit recorded in `packages/contracts/mvp/sources.json`. Their original stable/draft labels describe that historical handoff. Overlapping MVP routes now follow the integration target, especially async checkout and Profile DTO/concurrency.

`web-reference/` preserves earlier Web feature proposals. Neither reference folder is a claim of runtime provider acceptance. See `snapshot-hashes.json` for both upstream and normalized SHA-256 values. Edit the MVP generator/provider contracts and regenerate the target; do not rewrite historical snapshots to hide differences.
