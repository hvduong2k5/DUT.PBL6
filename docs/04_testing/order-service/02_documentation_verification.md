> **09/10/2026:** Đây là report lịch sử của docs v3.0. Viewer/helpers cũ đã bị xóa khỏi workspace; schema và runtime mới được kiểm tại [03_implementation_verification.md](03_implementation_verification.md). Các câu “chưa có implementation” bên dưới chỉ mô tả ngày 08/10.

# MS-04 v3.0 — kết quả kiểm tra tài liệu ngày 08/10/2026

## 1. Phạm vi và kết quả

Đã viết lại [bộ thiết kế Order](../../02_architecture/services/order-service/README.md) theo domain/state/persistence/saga/contracts/testing và quyết định/gaps; sinh lại master, Draw.io 9 trang, viewer offline và online URL payloads.

| Kiểm tra thực tế | Kết quả | Giới hạn |
| --- | --- | --- |
| UTF-8, Markdown fences, liên kết nội bộ | PASS bằng `verify_order_docs.py` | không chứng minh rendering của tất cả Markdown clients |
| Master chứa nguyên bảy chương nguồn | PASS | master generated, không sửa riêng |
| XML Draw.io: 9 pages, cell IDs unique, endpoints tồn tại | PASS | chưa mở diagrams.net để kiểm tra editor/import hoặc route auto-layout |
| Viewer SVG/tabs và links đồng bộ | PASS | SVG render từ cùng graph summary, Markdown chứa guards chi tiết hơn |
| 11 link diagrams.net: raw-deflate decode ra XML tương ứng | PASS | giải mã local, không gửi/publish nội dung tới external service |
| Playwright: click 9 tabs, chỉ 1 panel visible; ArrowRight/End/Home | PASS trên Chromium | kiểm viewer tài liệu, không phải D2C business UI |
| Viewer 390px, card text bounding boxes trên 9 pages | PASS; không page overflow, panel scroll ngang | sơ đồ rộng cần cuộn trên mobile |
| Visual inspection screenshot topology/lifecycle | PASS; nội dung đọc được, card text không cắt | không phải diagram UML formal validator |
| Browser console sau sửa favicon inline | 0 errors, 0 warnings | lần đầu có favicon 404, đã sửa và reload kiểm lại |
| Candidate DDL chạy trên PostgreSQL 16.13 | PASS, 18 tables và indexes | database tạm trong container network none; không migration production |
| SQL constraint smoke checks | PASS, fixtures rollback | không có distributed/service runtime tests |

PostgreSQL image `pgvector/pgvector:pg16` đã có trên máy; test chạy trong container riêng `codex-order-docs-pg16-20261008`, không publish port, không dùng database/volume Profile đang chạy. Container test được xóa khi kết thúc. DDL và smoke checks cuối cùng chạy trong database mới `order_docs_v3_final`.

## 2. SQL assertions đã chạy

- Hợp lệ khi merchandise discount và free-shipping được tách đúng: 220.000 − 20.000 + 25.000 − 25.000 = 200.000 VND.
- Từ chối merchandise discount vượt subtotal, final arithmetic sai, quantity bằng 0.
- Từ chối PENDING_PAYMENT thiếu reservation/expiry, PAID khi stock chưa COMMITTED, CONFIRMED_COD khi method không phải COD.
- Unique(provider,transaction ID) chặn receipt trùng; inbox consumer/source/event ID chặn trùng.
- Receipt unmatched với order ID NULL vẫn lưu được outbox aggregate PAYMENT.
- Fixtures nằm trong transaction và rollback sau assertions; schema vẫn tồn tại trong database tạm để kiểm catalog trước cleanup.

Test không xác nhận cross-table tổng allocation/refund hoặc snapshot immutable đã enforced đầy đủ; những invariant đó cần implementation transactions/locks/trigger và test theo chương 06.

## 3. Cách chạy lại

Từ repository root:

```powershell
python docs/02_architecture/services/order-service/tools/build_order_doc_assets.py
python docs/02_architecture/services/order-service/tools/verify_order_docs.py
```

Trong PostgreSQL 16 database **rỗng, isolated** do người chạy chuẩn bị:

```text
psql -v ON_ERROR_STOP=1 -f docs/02_architecture/services/order-service/order_schema_candidate.sql
psql -v ON_ERROR_STOP=1 -f docs/04_testing/order-service/schema_smoke_checks.sql
```

Phải cấu hình connection tới database tạm đúng mục đích; không chạy vào database dịch vụ đang dùng. Viewer mở bằng file local hoặc HTTP localhost; browser CLI có thể chặn file protocol nên test dùng HTTP server bind 127.0.0.1. Online launcher không tự redirect, viewer không tải script bên ngoài.

## 4. Chưa được kiểm chứng

ORD-T01..26 vẫn **NOT_RUN** vì `services/order-service` chưa có implementation. SLO, Inventory/Promotion terminal protocol, provider payment/refund adapter, verified Guest claim, ready/cancel barrier và completion/Case barrier chưa có bằng chứng đầu-cuối. Danh sách [gaps](../../02_architecture/services/order-service/07_decisions_and_contract_gaps.md) là điều kiện cần đóng trước feature integration.

Không sửa các thay đổi Profile/hợp đồng dùng chung đã có trong working tree. Thiết kế v3.0 phân biệt contract repo hiện tại và candidate; đợt này không tự cập nhật proto/event/OpenAPI chung hay triển khai service.
