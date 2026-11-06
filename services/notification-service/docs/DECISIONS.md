# Decision register

Status vocabulary: **FIXED BY REQUEST** = user yêu cầu; **PROPOSED BASELINE** = đề xuất của plan, được triển khai khi user giao implementation và chấp nhận baseline; **OPEN/GATED** = chưa được coi là khả năng thật. Không có provider/infrastructure approval ngầm.

## Baseline decisions

Implementation follow-up 2026-11-06: user instructed “dựa vào kế hoạch đó tiến hành làm notification service đi”, authorizing the planned implementation. Local N0/N1 proceeds without re-asking architecture approval. G01 resolved for local validation with user-authorized portable Go 1.27.3 (official archive SHA256 checked); module baseline stays 1.22.0 with existing repository Chi/pgx/UUID pins. No claim of full dependency security audit or production readiness. G02/G03/G08 remain peer-owner integration gates; the executable is local-only and persists opaque references in WAITING_CONTEXT, without resolving recipients or sending. Runtime uses standard-library slog for metadata-only startup failure rather than adding Zerolog for the foundation; no sensitive underlying connection error is logged. Redis/Kafka/provider dependencies are added only when their phases open.

| ID | Status | Quyết định / alternatives / lý do |
|---|---|---|
| D01 | FIXED BY REQUEST | Nhánh mới, không merge/push, plan kỹ cho model khác, docs tách biệt. `codex/notification-service` từ dev snapshot; toàn bộ writes dưới service. Không dùng Mobile branch làm base vì kéo theo task docs không liên quan |
| D02 | PROPOSED BASELINE | Go module + Chi/pgx/kafka-go/Zerolog theo Go services đã có. Alternatives Java/Spring hoặc Node/BullMQ làm tăng stack mới so với lựa chọn Go; exact supported toolchain/library versions là G01, không copy version cũ mù quáng |
| D03 | PROPOSED BASELINE | PostgreSQL durable source + Redis hint/rebuildable limiter thay vì Redis-only delivery storage. DB-only scheduler có thể dùng cho local mode; architecture Redis của MS-17 vẫn có adapter, không đưa dual-write mất dữ liệu vào correctness path. Nếu team bắt buộc Redis là dispatch driver: outbox → hint + periodic DB recovery bắt buộc |
| D04 | PROPOSED BASELINE | Event-driven + trusted internal command port. Không đồng bộ gửi Email trong Order HTTP transaction. Không thêm gRPC khi chưa có caller yêu cầu |
| D05 | PROPOSED BASELINE | MVP Email đúng EPIC 28; provider=fake default. SMS/ZNS/Push/alerts là phases riêng, không bỏ các FR này khỏi roadmap |
| D06 | PROPOSED BASELINE | Versioned producer adapters, route allowlist, topic/cohort pins. Không universal decoder đoán field hoặc dual-consume cùng aggregate cohort |
| D07 | PROPOSED BASELINE | At-least-once intake + durable dedup; provider UNKNOWN/reconciliation. Không hứa exactly-once email khi provider không có idempotency |
| D08 | PROPOSED BASELINE | Template deploy/versioned files cho MVP; full live admin template editor deferred. Immutable template/policy pin cho retry/history; template update không tạo business intent mới |
| D09 | PROPOSED BASELINE | Recipient/audience từ source authority; encryption port và scoped auth. Không join DB nguồn, không default customer=principal, không derive consent từ có email |
| D10 | PROPOSED BASELINE | Local event/context/provider fixtures cho core; production/public routes closed cho tới auth/owner/provider gate. Local test không chứng minh real delivery |
| D11 | PROPOSED BASELINE | Kafka ingestion retry khác delivery retry; 5 attempts candidate cho known-safe provider failures. Giải quyết số 3/5 lệch giữa tài liệu ở G06, không silently overwrite shared docs |
| D12 | PROPOSED BASELINE | Future detailed phases được viết khi vào phase, giữ master ổn định. N0/N1 có plan chi tiết; phần schema/state/test hiện tại là design contract, không execution log |

## Blocking gates và người cần chốt

| Gate | Câu hỏi / artifact phải có | Owner cần xác nhận | Chặn phần nào; có thể làm trước |
|---|---|---|---|
| G01 | Go toolchain support, exact dependencies, security check, module name | Service maintainer/user | N1 executable foundation; docs/fixtures độc lập làm trước |
| G02 | Event→purpose→audience→channel matrix; Order cohort đang dùng; baseline 3.2 vs sandbox; same intent event+command identity | Order + Notification owner | Real Order adapter; core decoder fixtures làm trước |
| G03 | Authorized recipient/context resolver hay private dispatch command? Guest/buyer/recipient/contact revision/hide-price/expiry fields | Order/Profile/Identity + Notification owner | Real send, đặc biệt Guest/gift; fake resolver làm trước |
| G04 | Email provider, budget/quota, sender/domain, sandbox, idempotency retention/query/callback signature/bounce semantics | User/platform owner | Real provider adapter/activation. So sánh API provider có callbacks/idempotency (ưu tiên) với SMTP (thường thiếu query; timeout cần manual review). Local capture không cần chọn hãng |
| G05 | OTP send owner là Keycloak hay Notification; expiry/resend/rate-limit; contact authorization, secure secret transport | Identity owner | OTP adapter/live tests. Chưa chốt → disabled, không tuyên bố full Email MVP |
| G06 | Retry/timeouts/unknown/reconcile policy, SLO, traffic, backlog cap, shared quotas | Product/platform + provider owner | Production tuning/acceptance; fake error tests vẫn làm được |
| G07 | Invoice producer, protected artifact access và audience/legal-entity binding | Finance owner | Invoice integration; schema-only không đủ |
| G08 | Service-to-service auth; public issuer/audience/JWKS/current status; notification permissions + resource scopes, gateway routing | Identity/Gateway owner | Protected real APIs, peer resolver. Local tests use injected principals; no production header bypass |
| G09 | Consent/preference authority; mandatory classification, bounce/complaint override policy; data retention/key management/rotation | Product/security + Notification owner | Real channel enable/production purge; Marketing deny default |
| G10 | Audit/status schema/topic ACL/retention; replay permissions; history visibility | Audit/platform owner | Outgoing integration; durable local obligation hoạt động trước |
| G11 | Full US-NOTI-01 target hay scope giảm chỉ Order Email trong đợt đầu? | User/product owner | Cách tuyên bố DONE; mặc định giữ cả OTP/Invoice trong mục tiêu, OPEN nếu thiếu |
| G12 | N5–N7 provider, Push endpoint ownership, quiet hours, alert audiences/escalation SLA/current resource authorization | Owning teams/product | Phases sau; không chặn Email local foundation |

Không cần hỏi user toàn bộ 12 gates cùng lúc. Trước implementation hỏi gộp các lựa chọn ảnh hưởng baseline còn thiếu (phạm vi đợt đầu, provider thật hay local, authoritative owner contact). Sau đó làm phần độc lập; từng integration chỉ dừng ở gate tương ứng. Không đoán câu trả lời từ elapsed time.

## Khi cần thay đổi baseline

Ghi entry Dxx mới: lý do có bằng chứng, alternatives, modules/contracts bị ảnh hưởng, migration/rollback, compatibility với event đã lưu và old template/key/dedup identities. Cập nhật master chỉ nếu dependencies/milestones đổi, active plan cho file-level work. Không sửa state machine đã nghiệm thu chỉ để vượt một test mới.
