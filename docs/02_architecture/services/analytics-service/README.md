# MS-11 Analytics Service

**Thiết kế đề xuất v0.1 · 06/11/2026 · baseline `origin/dev` tại `89c6d649e8955050db39e4742da1c720995f337a`.**

MS-11 tổng hợp chỉ số điều hành, tạo kết quả hỗ trợ quyết định (DSS) và đảm nhận Recommendation Engine cùng API gợi ý cho Storefront. Nhánh `feat/analysis-service` có [runtime Python của Dashboard Order v1 và Supply](../../../../services/analytics-service/README.md). Bộ Recommendation v1 được bổ sung ngày 07/11/2026 theo kế hoạch triển khai ba kiểu gợi ý. Chương 12–15 là tài liệu thiết kế để triển khai, với contract nguồn và gate riêng.

Đọc [12 — Thiết kế Recommendation Engine](12_recommendation_engine_design.md) để bắt đầu, rồi chuyển [13 — Dữ liệu cần cung cấp từ các service](13_recommendation_source_contracts.md) cho các nhóm Catalog, Inventory, Order và Identity/Web.

Pilot ưu tiên owner export commands và file JSONL/manifest theo lịch, tái sử dụng mẫu vận hành Supply. Chương 13 có đánh giá khả thi từ code, phần dữ liệu cần owner xác minh và phương án HTTP khi mở rộng.

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
| [09 — Supply Analytics v1](09_supply_analytics_v1.md) | Tốc độ bán, tồn đến hạn và hợp đồng bản xuất Order/Inventory |
| [10 — GA4 và Recommendation](10_ga4_and_recommendations.md) | Ranh giới Marketing/MS-11, tracking và ba vị trí Storefront |
| [11 — Nghiên cứu thuật toán](11_recommendation_algorithm_proposal.md) | So sánh content-based, basket rules, item-to-item và mô hình học máy |
| [12 — Thiết kế Recommendation Engine](12_recommendation_engine_design.md) | Ownership, kiến trúc, ba thuật toán, scope và chất lượng |
| [13 — Hợp đồng nguồn Recommendation](13_recommendation_source_contracts.md) | Trường dữ liệu, payload ví dụ, snapshot/revision, scope và checklist từng owner |
| [14 — Recommendation API và Storefront](14_recommendation_api_and_storefront.md) | Query/response, consent/events, xác thực, Gateway, BFF và GA4 |
| [15 — Persistence và bàn giao Recommendation](15_recommendation_persistence_and_delivery.md) | Schema/grain, worker, publish/rollback, privacy, retention, kiểm thử và rollout |
| [16 — Runtime và kiểm chứng Recommendation](16_recommendation_runtime_and_verification.md) | Lệnh chạy exporter/worker/Web, fixture, kiểm thử và gate dev còn lại |

- [DDL PostgreSQL](analytics_control.candidate.sql) và [DDL ClickHouse](analytics_olap.candidate.sql) mô tả phần nền tảng Dashboard.
- [OpenAPI Dashboard](../../../03_api_specs/analytics-service.candidate.openapi.yaml), [ADRs](../../adr/analytics-service/README.md), [kế hoạch kiểm thử](../../../04_testing/analytics-service/01_acceptance_and_verification_plan.md) và [kết quả runtime](../../../04_testing/analytics-service/02_implementation_verification.md).
- [Kế hoạch triển khai](../../../05_deployment/analytics_service_deployment.md) và [runbook](../../../06_operations/analytics_service_runbook.md).

Lát triển khai đầu tiên đề xuất hiển thị số đơn ghi nhận mốc PAID và COMPLETED theo cohort Order v3 đã xác minh. Doanh thu và lợi nhuận cần định nghĩa cùng dữ liệu do Finance phê duyệt. Các card Customer và Inventory có điều kiện bật riêng. Việc xếp Dashboard vào Giai đoạn 2 vẫn cần Product Owner xác nhận.
