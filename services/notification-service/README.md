# MS-17 Notification Service — bộ kế hoạch bàn giao

**Trạng thái 2026-11-06: N1 local foundation implemented; real database acceptance pending.** Có Go module, internal HTTP API, authorization local, durable PostgreSQL intake/idempotency/audit obligation và explicit migration. Chưa có delivery worker, Kafka consumer hoặc provider gửi Email. Các phase này chờ N1 acceptance và owner gates.

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

## Chạy local foundation

Go compiler đã kiểm tra: portable Go 1.27.3; module language baseline 1.22.0 theo repo. Chưa đánh giá tương thích với compiler 1.22 hoặc security toàn dependency tree. Binary chỉ cho phép APP_MODE=local và listener loopback; không cấu hình token thì protected APIs deny toàn bộ.

Set `NOTIFICATION_DATABASE_URL` đến database riêng và `NOTIFICATION_LOCAL_TOKEN` dài ít nhất 32 byte trong environment (không commit). Chạy từ service:

```powershell
go run ./cmd/notification -migrate
go run ./cmd/notification
```

Default listener `127.0.0.1:18017`. API và response schema trong [local contract](contracts/README.md). POST nhận file `contracts/examples/intake.json`, header `Content-Type: application/json`, `Authorization: Bearer <local token>` và `Idempotency-Key`. HTTP 202 có trạng thái WAITING_CONTEXT: chưa gửi Email. Cùng key/body trả 200 và cùng request ID; body khác trả 409. GET chỉ đọc request của authenticated producer. Readiness `/health/ready` kiểm DB và migration checksum; liveness `/health/live` chỉ kiểm process.

Local PostgreSQL stack tách biệt ở `deploy/local/compose.yaml`: set `NOTIFICATION_TEST_PASSWORD` disposable trước khi chạy `docker compose -f deploy/local/compose.yaml up -d --wait`. DB bind `127.0.0.1:15417`, user/database notification_test, volume/project riêng. Đây là lệnh tùy chọn cần Docker đang hoạt động, chưa có nghĩa stack đã được chạy.

Kiểm thử: `./scripts/verify.ps1` chạy race/vet/build. `-Integration` yêu cầu `NOTIFICATION_TEST_DATABASE_URL` trỏ DB disposable; tạo schema random và chỉ xóa schema test đó. Không có URL → integration test SKIP; `-Integration` fail nếu thiếu URL. Không dùng shared/production DB. N1 chưa DONE khi required DB tests chưa thực sự chạy.
