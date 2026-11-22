# ADR-001 — Nguồn chuẩn KPI và phạm vi Dashboard

**Trạng thái đề xuất · 06/11/2026 · Chủ review Data Team, PO, Order và Finance.**

## Bối cảnh

EPIC 25 cần Dashboard đa nguồn. Kế hoạch chi tiết đề xuất Phase 2 nhưng Epic Map xếp toàn EPIC vào Phase 3. Payload PAID/COMPLETED hiện hỗ trợ đếm mốc, còn doanh thu và lợi nhuận cần rule, kỳ và đối soát từ Finance.

## Quyết định

Lát đầu dùng hai metric đếm mốc Order v3 có phạm vi rõ ràng. Customer, Inventory và Finance có gate theo card. Khi không có coverage đã chứng minh, giá trị card để trống với lý do. Finance duyệt định nghĩa và cung cấp dữ liệu có phiên bản trước khi công bố tài chính.

## Hệ quả

Cần đối soát theo cohort, đo coverage và thống nhất với Web Admin về trạng thái partial/unavailable. PO duyệt phạm vi phát hành sau khi xem [lộ trình](../../services/analytics-service/08_implementation_roadmap.md). Việc dùng receipt để tính GMV cũng cần một metric và nguồn được duyệt riêng.
