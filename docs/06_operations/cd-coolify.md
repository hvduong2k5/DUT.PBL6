# CD to Coolify (GitHub Actions)

Workflow: `.github/workflows/cd-deploy.yml` — prepared 2026-11-13.

## How it works

1. A push lands on a release branch (the team's release flow stays manual:
   fixes are cherry-picked to the release branch after review, exactly as
   today) — or a human starts the workflow manually with an app list.
2. The workflow works out which apps changed:
   - `release/v0.1.0-rc.1` → **identity-service only** (its Coolify app is
     configured to build this branch; every other app builds rc.2).
   - `release/v0.1.0-rc.2` → per-path mapping to catalog / profile / order /
     fulfillment / shipping / inventory / web-user. Changes under
     `packages/` (shared proto/contracts) redeploy all backend services.
     Identity changes on rc.2 are reported but not deployed — identity
     ships from rc.1.
3. For each app it calls the Coolify deploy API
   (`GET /api/v1/deploy?uuid=<app>&force=true`) with the API token.
4. It restarts Kong — required after any container replacement
   (finding F-08: Kong otherwise serves stale upstream IPs).
5. Smoke gate: the public product list and the storefront must answer
   HTTP 200 (retried for ~3 minutes). A failed smoke fails the run and
   prints the rollback instructions in the run summary.

Rollback is a deliberate manual step in v1: Coolify → the app →
Deployments → roll back to the previous deployment, then restart Kong
(the W4 R3 path, already proven). If the team wants it automated, that
is a v2 decision.

## One-time setup (repo/Coolify owner)

1. In Coolify: **Keys & Tokens → API tokens** → create a token allowed to
   deploy. Copy the value once — Coolify shows it only at creation.
2. In GitHub: **Settings → Secrets and variables → Actions → New
   repository secret** → name `COOLIFY_API_TOKEN`, value = the token.
3. First run: start the workflow manually (Actions → "CD - Deploy to
   Coolify" → Run workflow) with `web` and watch it end-to-end once
   before trusting it on pushes.

App UUIDs are written into the workflow file itself: they are plain
identifiers (visible in Coolify URLs), not secrets.

## What this does NOT change

- CI on pull requests is untouched; making CI a *required* check on
  `dev` is a separate branch-protection setting (Settings → Branches),
  still a team decision.
- Merging remains a human/team action. CD starts only after code has
  reached a release branch.
