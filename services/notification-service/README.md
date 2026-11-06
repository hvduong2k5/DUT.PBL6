# MS-17 Notification Service — bộ kế hoạch bàn giao

**Trạng thái 2026-11-06: DESIGN / NOT IMPLEMENTED.** Chỉ có tài liệu và `src/.gitkeep` có sẵn. Chưa có Go module, API, migration, consumer hoặc provider chạy thật.

Nhánh riêng `codex/notification-service`, base `origin/dev` local snapshot `a24492b63c364f93ad19d880798137e554898c3f`. Không fetch, merge, push hoặc sửa service khác. Các remote-tracking refs được đọc là bản đã có trên máy, không phải xác nhận tip mới nhất trên server.

## Bắt đầu ở đâu

| Tài liệu | Trách nhiệm |
|---|---|
| [AGENTS.md](AGENTS.md) | Phạm vi thao tác và quy tắc dành cho model |
| [CURRENT_STATE](docs/CURRENT_STATE.md) | Handoff hiện tại, trạng thái, bước tiếp theo |
| [SOURCE_AUDIT](docs/SOURCE_AUDIT.md) | Commit/path đã kiểm tra, khác biệt tài liệu và code |
| [PROJECT_SPEC](docs/PROJECT_SPEC.md) | Scope, yêu cầu, traceability EPIC 28, non-goals |
| [ARCHITECTURE](docs/ARCHITECTURE.md) | Thành phần Go, boundaries, luồng dữ liệu và vận hành |
| [CONTRACTS](docs/CONTRACTS.md) | Event/API/recipient contracts ứng viên và integration gaps |
| [DATA_AND_DELIVERY](docs/DATA_AND_DELIVERY.md) | Schema ứng viên, transaction, dedup, retry, callback |
| [DECISIONS](docs/DECISIONS.md) | Quyết định thiết kế, phương án thay thế, approval gates |
| [VALIDATION](docs/VALIDATION.md) | Acceptance, failure matrix, bằng chứng cần thu |
| [MASTER_PLAN](docs/plans/MASTER_PLAN.md) | Phases và dependencies ổn định |
| [N0_N1_CONTRACT_FOUNDATION](docs/plans/active/N0_N1_CONTRACT_FOUNDATION.md) | Kế hoạch chi tiết cho bước hiện tại/tiếp theo |
| [IMPLEMENTATION_PROMPT](docs/IMPLEMENTATION_PROMPT.md) | Prompt bàn giao copy-paste cho model khác |

## Đề xuất tóm tắt

Go + Chi + pgx/PostgreSQL, Kafka inbox/outbox, Redis dispatch hint có thể phục hồi từ DB, provider qua ports. Email là MVP theo `US-NOTI-01`; SMS/Zalo, Push/Notification Center và Internal Alert là các phase sau. Core local có thể phát triển với fake provider và dữ liệu tổng hợp; tích hợp thật phải chốt các gate trong DECISIONS.

Hai rủi ro tích hợp lớn nhất: các event Order/Shipping hiện có không chứa đủ contact/audience; đồng thời catalog cũ không đồng nhất với event/cohort hiện hành. Không suy đoán email từ customer ID, không coi `order.events.v1` là nguồn mặc định cho mọi Order, không dùng normalized Order ingress như producer event.

Tài liệu mới chỉ nằm trong thư mục service này. Các docs của service khác được dẫn bằng Git revision/path, không sao chép thành kế hoạch của Notification.
