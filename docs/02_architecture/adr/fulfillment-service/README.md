# Fulfillment ADR index

Ngày30/10/2026. Status các ADR: **Proposed**; chưa thay quyết định của owning peer hoặc requirement baseline. [Service design](../../services/fulfillment-service/README.md).

| ADR | Quyết định |
| --- | --- |
| [001](adr_001_terminal_admission_and_packing_lifecycle.md) | Tách terminal admission khỏi packing; DB guard/order |
| [002](adr_002_package_identity_and_exact_revisions.md) | One task/package MVP; giữ immutable exact revisions và extension lineage |
| [003](adr_003_private_evidence_and_video_cohort.md) | Private verified evidence; video-required integrated-v2 |
| [004](adr_004_peer_contracts_and_capability_gates.md) | Peer authority, candidate artifacts và explicit acceptance gates |

Thay đổi ADR cần context, alternatives, consequences, test/migration impact; không dùng note “accepted” nếu peers chưa sign-off.
