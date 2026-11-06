# MS-11 Analytics Service

**Thiết kế đề xuất v0.1 · 06/11/2026 · baseline `origin/dev` tại `89c6d649e8955050db39e4742da1c720995f337a`.**

MS-11 tổng hợp chỉ số điều hành và tạo kết quả hỗ trợ quyết định (DSS). Nhánh `feat/analysis-service` có [runtime Python của Dashboard Order v1](../../../../services/analytics-service/README.md). Các artifact dưới đây mô tả thiết kế Dashboard, điều kiện mở nguồn và lộ trình DSS.

| Tài liệu | Nội dung |
| --- | --- |
| [Master index](ms11_analytics_service_design.md) | Nguồn, ưu tiên và truy vết yêu cầu |
| [00 — HLD](00_analytics_service_hld.md) | Ranh giới, luồng và phân kỳ |
| [01 — Domain và metric](01_domain_and_metrics.md) | Định nghĩa KPI, coverage và so sánh kỳ |
| [02 — Use case và vòng đời](02_usecases_and_lifecycle.md) | Dashboard, phiên phân tích và khuyến nghị |
| [03 — Persistence](03_database_and_persistence.md) | Grain, schema, snapshot và đối soát |
| [04 — Messaging và recovery](04_messaging_and_recovery.md) | Checkpoint, dedup, watermark và replay |
| [05 — API và bảo mật](05_api_contracts_and_security.md) | REST, quyền, masking và export |
| [06 — DSS engines](06_dss_engines.md) | Thiết kế Phase 3 cho US-DSS-02 đến 10 |
| [07 — Tích hợp và khoảng trống](07_integrations_and_contract_gaps.md) | Topic, event, nguồn chuẩn và điều kiện bật |
| [08 — Roadmap](08_implementation_roadmap.md) | Các lát triển khai và tiêu chí nghiệm thu |

- [DDL PostgreSQL](analytics_control.candidate.sql) và [DDL ClickHouse](analytics_olap.candidate.sql) mô tả phần nền tảng Dashboard.
- [OpenAPI Dashboard](../../../03_api_specs/analytics-service.candidate.openapi.yaml), [ADRs](../../adr/analytics-service/README.md), [kế hoạch kiểm thử](../../../04_testing/analytics-service/01_acceptance_and_verification_plan.md) và [kết quả runtime](../../../04_testing/analytics-service/02_implementation_verification.md).
- [Kế hoạch triển khai](../../../05_deployment/analytics_service_deployment.md) và [runbook](../../../06_operations/analytics_service_runbook.md).

Lát triển khai đầu tiên đề xuất hiển thị số đơn ghi nhận mốc PAID và COMPLETED theo cohort Order v3 đã xác minh. Doanh thu và lợi nhuận cần định nghĩa cùng dữ liệu do Finance phê duyệt. Các card Customer và Inventory có điều kiện bật riêng. Việc xếp Dashboard vào Giai đoạn 2 vẫn cần Product Owner xác nhận.
