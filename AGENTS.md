# Customer Mobile parity work

## Scope and authority

- Follow the global instructions supplied by the user and `E:/.vscode/PBL6/AGENTS.md`.
- Current task approved 2026-10-19: implement Flutter Customer UI/flow parity with `origin/feature/ui-web` on `feature/mobile-ui` only.
- Web source is read-only. Read it using `git show origin/feature/ui-web:<path>`; never edit, commit or push that branch. No merges or PRs are authorized.
- The current request supersedes the ancestor's historical Stitch prompt/review sequence: use the existing Customer Web as the approved visual/flow reference and implement/test Flutter directly. Do not update the ancestor's Stitch task checkboxes.
- Only intentionally modified mobile source, mobile tests and task documentation/contracts may be committed. Push only `feature/mobile-ui`.

## Resume

User steering 2026-10-31: read-only dev contract analysis is authorized. MVP migration planning target is dev7f22226; follow the four-step roadmap in MASTER_PLAN and current M1 plan. Keep historical ZIP snapshots; runtime adapters are not migrated by a planning-only step. Dev remains read-only. No automatic merge or feature activation. Later user steering authorizes implementation of agreed Mobile portions; active M2_MVP_PURCHASE.md defines the bounded foundation slice. Runtime activation remains separate.

Read `docs/CURRENT_STATE.md`, `docs/plans/MASTER_PLAN.md`, then the active phase linked there. Inspect branch, HEAD and status before editing.

## Commands

Run from `apps/mobile-app`: `flutter analyze`, `flutter test`, `flutter build apk --debug`. SDK baseline Flutter 3.44.9 / Dart 3.12.2; no dependency upgrades without a verified need.

## Contracts and implementation

- Existing runtime baseline: the user-supplied API contract ZIP; see `docs/03_api_specs/frontend-contracts/SOURCE.md`. New migration planning uses the pinned public MVP target in MASTER_PLAN for overlapping routes; keep ZIP provenance unchanged and do not claim runtime migration from planning alone.
- Preserve stable/draft/planned distinctions. Never invent production endpoints or promote draft policies silently.
- Keep UI, Cubits/domain and REST/mock adapters separate. Default local mock mode must remain usable without backend.
- Match Customer Web tokens and flow intent while adapting layout, navigation and touch targets to phones. Reuse Flutter primitives.
- Validate loading, empty, error/retry, pending/disabled, keyboard, back navigation, long text and unavailable data. Record native evidence separately from automated checks.
