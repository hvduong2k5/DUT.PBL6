# Inventory — cập nhật theo MVP (31/10/2026)

Trạng thái: đã có bài kiểm tra tích hợp Order–Inventory local qua TLS/PostgreSQL; CHƯA nghiệm thu deployment/toàn bộ MVP. Không đổi registry thành verified. Xem [bằng chứng và cách chạy chung](../order-service/order-inventory-integration.md).

## Đã triển khai
- Generate lại Go message/gRPC từ Inventory và Common workflow contract trên dev.
- Reserve profile order-3.2: normalize/consolidate/sort SKU, kiểm overflow int32, items_digest, request_hash, TTL 1–15 phút và warehouse.
- GetReservationByOrder, FinalizeReservation, ReleaseReservationByOrder với resource version, DB observation time và command evidence bền vững; replay giữ decision ID/time, key khác hash bị conflict.
- Ledger PostgreSQL riêng cho cohort mới; advisory lock theo order kể cả chưa có reservation; cancellation tombstone chặn reserve đến muộn. Trigger bảo vệ không tạo reservation legacy trên order thuộc cohort mới.
- Khóa item/batch để cập nhật physical/reserved atomic; outbox nằm cùng transaction. Expiry worker và finalize/release dùng cùng order lock; expired finalize trả REJECTED, không commit kho.
- ReverseCommittedStock vẫn khóa vì approval/remote stop chưa có authority được nghiệm thu.
- Terminal profile mặc định tắt. Bật yêu cầu TLS + credential Order riêng và schema đã migrate. Không nhận contract_version mới qua handler legacy.

## Cách chạy
1. Apply 000002_terminal_profile.up.sql bằng công cụ migration của môi trường (không chạy seed/TRUNCATE).
2. Cấu hình INVENTORY_TERMINAL_ENABLED=true, INVENTORY_ORDER_TOKEN (ít nhất 32 bytes), INVENTORY_TLS_CERT và INVENTORY_TLS_KEY.
3. Order phải dùng TLS, metadata authorization: Bearer <credential>, contract_version=order-3.2 và canonical hash trong packages/contracts/order-3.2.md.
4. Không bật cohort Order mới trước khi test hai bên và nghiệm thu credential/transport.

## Kiểm tra
- go test ./..., go vet ./..., go build ./...: PASS local.
- Tests mới: canonical/hash validation, tombstone, original decision replay, same-key/hash conflict, authentication và reversal gate.
- Có test PostgreSQL thật TestTerminalPostgresLifecycle: concurrency finalize, không double deduction, release committed reject và tombstone. Cần INVENTORY_TEST_DATABASE_URL trỏ DB test có quyền CREATE SCHEMA; không lấy DATABASE_URL tự động. Mỗi test tạo schema UUID riêng và xóa schema đó khi xong.
- Bản candidate ban đầu chưa chạy PostgreSQL thật. Bài `scripts/test-order-inventory.py` nay chạy `TestTerminalPostgresLifecycle` và client Order qua TLS với PostgreSQL riêng; có kiểm worker đặt đơn, mất ACK/replay và số tồn/outbox. Chưa có nghiệm thu live broker hoặc UI E2E; không coi mock test là acceptance.

## Việc còn phải nghiệm thu
- Run PostgreSQL concurrency/rollback/expiry/restart và real broker outbox với owning team.
- Owner duyệt shared internal credential/TLS deployment, rotate secret và integration rollout. Hiện một credential đại diện Order; chưa có shared Identity service mapping.
- Expiry hiện scan ledger mỗi 30s; cần projection/index và retention policy trước tải lớn.
- Reverse/paid cancellation không được bật. Legacy order-paid consumer chỉ dùng order.events.v1; không nối v3 vào consumer này.
