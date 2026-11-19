# Web Packing → Fulfillment

Baseline: `dev@89c6d649e8955050db39e4742da1c720995f337a`.

## Runtime

In `WEB_AUTH_MODE=identity-bff`, `/admin/packing` is the management view and
`/admin/work/packing` handles self-assignment, start, counted allocations,
checklist, video upload/verification and sealing. The legacy Mockoon screen
remains available only through the legacy local auth configuration. There is
no automatic fallback when Fulfillment fails.

Web calls `/api/admin/fulfillment/*` with the HttpOnly session cookie. Its BFF
calls Identity `/api/v1/admin/packing-api/*` through Kong; Identity resolves the
ACTIVE Employee and forwards its server-held JWT to the Fulfillment Gateway
route. Fulfillment rechecks current ADMIN GLOBAL/WAREHOUSE membership and the
resource's warehouse. No new permission claims or bearer tokens reach browser
JavaScript. The packing menu mirrors the service's current ADMIN mapping;
fine-grained Packing roles are still outside this accepted backend profile.

The API exposes only queue/detail/history, self-assignment/start/unassign,
picks, checklist, seal and media metadata/completion. Browser mutations require
the configured Web origin. Task commands carry their original `If-Match` and
`Idempotency-Key`; an ambiguous command is retained in sessionStorage for an
explicit retry, including across reload. It is not retried automatically.

## Video

`POST /api/admin/fulfillment/tasks/{task_id}/video` streams MP4/WebM bytes.
Required headers: `X-Video-Size`, `X-Video-Sha256`, `X-Package-ID`,
`Idempotency-Key`, and the video `Content-Type`. Maximum size is 200 MiB;
Fulfillment also checks the packing policy. The BFF creates the staff-authorized
media session, streams exactly the declared bytes to its signed PUT URL, and
returns only `file_id/status`. The browser then completes the upload and polls
metadata until VERIFIED/REJECTED. A stopped scanner never makes a video valid.

Configure `FULFILLMENT_STORAGE_UPLOAD_ORIGIN` as the exact origin used by the
storage signer, e.g. `http://packing-storage:9000`. A Web process outside Docker
may additionally set `FULFILLMENT_STORAGE_UPLOAD_CONNECT_ORIGIN` to the published
loopback origin; the relay preserves the signed Host/path/query. The browser
cannot choose a target URL, storage key or credential. No CORS setup is needed
for this server relay. Metadata/session calls still go through Identity.

Sealing reads the current task/package, confirmed quantities, required checklist,
verified evidence and entered parcel measurements. The UI does not equate
COMPLETED with shipment creation: Order creates the shipment asynchronously.
The existing Fulfillment `integration` fields remain PENDING and are not used
as proof of shipping success.

## Run and verify

Prerequisites: Docker, Python, Node 22, Go 1.27.3, Java 21 (`JAVA_HOME`) and
Gradle 9.2.0. From the repository root on Windows:

```powershell
./scripts/start-packing-demo.ps1
```

Executable paths can be passed using `-Go`, `-Java`, `-Gradle`. The command
builds the Fulfillment image, runs the isolated E2E with `--fulfillment`, and
retains the stack. Infrastructure/Fulfillment/storage/scanner run in containers;
the existing fixture runs the remaining apps as native processes. This is not
an all-services Compose deployment. Inspect the ignored
`.system_generated/packing-demo/ready.json` locally for the URL/accounts;
create `.system_generated/packing-demo/stop` to stop and clean this fixture.
No secrets or signed URLs belong in committed reports.

The integrated packing acceptance now uses the Web cookie BFF, including binary
video upload. It checks Customer denial, forged browser bearer rejection,
cross-origin mutation denial, unsafe media, scanner outage, seal replay and
automatic recovery from Shipping outage without a duplicate shipment. Payment
statements remain TEST and the carrier remains FAKE. COD, paid cancellation,
reopen, multiple packages and assign-other remain gated.

Web regression: `npm run verify:mvp` and `npm run build` in `apps/web-user`.
The fixture runs Identity tests with its own PostgreSQL before booting apps.
