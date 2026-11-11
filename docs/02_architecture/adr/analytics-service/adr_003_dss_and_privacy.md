# ADR-003 — DSS theo phiên và bàn giao có phê duyệt

**Trạng thái đề xuất · 06/11/2026 · Chủ review Data Team, Identity, Marketing, Promotion và Procurement.**

## Bối cảnh

Một kết quả DSS có thể đổi khi cutoff, model hoặc quyền dùng dữ liệu đổi. Một số tài liệu use case cũ mô tả kích hoạt Campaign trực tiếp. EPIC 25 yêu cầu khuyến nghị có căn cứ và phê duyệt theo Epic đích.

## Quyết định

Ghim dataset, model, scope, policy và thời hạn cho mỗi run/insight. Chỉ dùng trường tối thiểu, khóa khách hàng giả danh và quyền có mục đích. Bàn giao snapshot hoặc tham chiếu qua contract của Epic đích sau quyết định có audit. Tách luồng hội thoại theo đề xuất AI Agent.

## Hệ quả

Phải kiểm privacy cả lúc build lẫn khi đọc/xuất. Bên nhận tự áp quyền và phê duyệt nghiệp vụ. Thu hồi policy có thể yêu cầu vô hiệu snapshot/export. [Thiết kế DSS](../../services/analytics-service/06_dss_engines.md)
