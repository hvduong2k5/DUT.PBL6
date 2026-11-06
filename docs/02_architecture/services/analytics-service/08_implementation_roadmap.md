# 08 — Lộ trình triển khai

| Lát | Deliverable | Điều kiện hoàn tất |
| --- | --- | --- |
| 0 — Chốt nguồn | Registry metric, cohort Order, routing, quyền, retention và freshness | PO duyệt scope. Order cho fixture/pin phiên bản, môi trường test và mốc đối soát. Identity/Web Admin thống nhất permission. |
| 1 — Ingest | Collector + journal PostgreSQL + checkpoint/replay | Re-delivery, conflict, rebalance, offset ngoài retention và quarantine đều qua test PostgreSQL/Kafka. Đối soát PAID/COMPLETED với Order trong cohort cố định. |
| 2 — Đọc Dashboard nội bộ | Builder ClickHouse, serving pointer, API + daily series | Kill/retry qua cả hai store, drill-down cùng generation, zero/partial, quyền staff, đo tải. Các card chờ nguồn trả unavailable có lý do. |
| 3 — Dashboard US-DSS-01 | Finance, Customer và Inventory adapters, comparison và drill-down đúng grain | Source owner sign-off theo card, reconciliation/quality đạt và nghiệm thu Web Admin trên môi trường tích hợp. |
| 4 — DSS theo từng capability | Product/Velocity, RFM/Churn, Forecast/Expiry, Association/Mùa vụ | Dữ liệu và backtest đủ theo từng story, policy quyền/retention, benchmark trong quota. |
| 5 — Strategic Insight | Recommendation, quyết định, audit, handoff và outcome | Epic đích duyệt contract và chặn tự động thay đổi nghiệp vụ. Kiểm quyền, stale/superseded, replay và thu hồi. |

Lát 0–2 chuẩn bị phần khả thi của Dashboard và tạo bằng chứng tích hợp. Lát 3 mới đủ scope story khi các nguồn tương ứng được duyệt. Chạy shadow trước khi bật từng capability. Dùng [kế hoạch kiểm thử](../../../04_testing/analytics-service/01_acceptance_and_verification_plan.md) và [deployment](../../../05_deployment/analytics_service_deployment.md) để xác định gate và rollback.
