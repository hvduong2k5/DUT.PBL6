# ADR-002 — Checkpoint và công bố snapshot

**Trạng thái đề xuất · 06/11/2026 · Chủ review Data Team và Platform.**

## Bối cảnh

Outbox có thể phát lại cùng event sau mất ACK. Một lần ghi ClickHouse hoặc cập nhật con trỏ có thể thành công dù caller bị timeout. Dashboard và drill-down cần đọc đúng cùng dữ liệu.

## Quyết định

Lưu khóa nguồn, grain nghiệp vụ và checkpoint SQL cùng transaction. Dựng generation bất biến từ SQL snapshot, kiểm count, digest và rollup qua đường đọc thực tế. Công bố bằng CAS trên con trỏ SQL. Client dùng `snapshotId` để ghim drill-down; repair/replay dựng generation mới.

## Hệ quả

Cần thêm journal có retention và dung lượng cho staging. Chu kỳ rebuild và budget query được đo trước rollout. Consumer phục hồi từ SQL cursor và so với broker retention. Thiết kế chi tiết ở [persistence](../../services/analytics-service/03_database_and_persistence.md) và [messaging](../../services/analytics-service/04_messaging_and_recovery.md).
