# MS-11 — Nguồn và truy vết thiết kế

Đọc [HLD](00_analytics_service_hld.md) để nắm phạm vi, [hợp đồng nguồn](07_integrations_and_contract_gaps.md) để kiểm tra mức bằng chứng, rồi [roadmap](08_implementation_roadmap.md) để lập PR. Các đường dẫn nguồn dưới đây thuộc baseline ghi trong [README](README.md).

## Thứ tự ưu tiên khi có khác biệt

| Câu hỏi | Nguồn quyết định | Cách áp dụng |
| --- | --- | --- |
| DSS phục vụ ai, gồm những gì? | [EPIC 25](../../../01_requirements/epics/EPIC_25_Customer_Analytics.md), [FR-25/26](../../../01_requirements/04_Functional_Requirements.md), [NFR](../../../01_requirements/03_Non_Functional_Requirements.md) | Theo story và tiêu chí chi tiết. Nhánh Phase 2 ở [Sprint Planning](../../../01_requirements/02_Sprint_Planning.md) chờ PO vì [Epic Map](../../../01_requirements/01_Product_Backlog.md) xếp EPIC 25 vào Phase 3. |
| Tích hợp với peer qua gì? | [Service boundary](../../service_boundary.md#ms-11-analytics-service), [BC-11](../../bounded_context.md#bc-11-analytics-dss--ai-context), [HLD hệ thống](../../high_level_design.md), [MVP guide](../../../../packages/contracts/mvp/README.md) | HLD cho hướng kiến trúc. Với payload và routing, đối chiếu schema phiên bản, code producer và evidence của cohort trước khi bật. |
| Doanh thu, refund và lợi nhuận lấy ở đâu? | [EPIC 24](../../../01_requirements/epics/EPIC_24_Finance.md), [EPIC 14](../../../01_requirements/epics/EPIC_14_Return_Refund.md) | Finance cung cấp metric đã duyệt. Payment/return là chứng từ nguồn với vòng đời riêng. |
| DSS có tự chạy Campaign hoặc xử lý hội thoại? | [Use case tổng](../../../01_requirements/use_case_specification.md), [EPIC 25](../../../01_requirements/epics/EPIC_25_Customer_Analytics.md), [đề xuất AI Agent](../../ai_agent_specification_and_proposal.md) | Chọn EPIC 25 cho thao tác chấp nhận/từ chối và bàn giao có phê duyệt. Runtime hội thoại thuộc đề xuất AI Agent riêng. |
| Recommendation cho khách đặt ở đâu? | Quyết định MS-11 đảm nhận engine/API, [thiết kế triển khai](12_recommendation_engine_design.md), [source contracts](13_recommendation_source_contracts.md) | Ba placement Product Detail, Cart và Home. Chương 12–15 cụ thể hóa bản nghiên cứu 10–11 và được dùng khi có khác biệt về interface/thuật toán. |

## Story, chương và gate

| Story | FR-DSS | Thiết kế | Gate dữ liệu chính |
| --- | --- | --- | --- |
| US-DSS-01 Dashboard | 01–03 | [Metric](01_domain_and_metrics.md), [API](05_api_contracts_and_security.md) | Order cohort; Finance, Profile và Inventory theo từng card |
| US-DSS-02 RFM | 04–06 | [DSS engines](06_dss_engines.md#customer-rfm-và-churn) | Liên kết khách có thẩm quyền, dòng giao dịch và hoàn tiền |
| US-DSS-03 Churn | 07–09 | [DSS engines](06_dss_engines.md#customer-rfm-và-churn) | Cửa sổ nhãn, consent và luồng Marketing |
| US-DSS-04 Product/SKU | 10–12 | [DSS engines](06_dss_engines.md#product-performance-và-velocity) | Dòng đơn, Catalog history, stockout |
| US-DSS-05 Velocity | 13–15 | [DSS engines](06_dss_engines.md#product-performance-và-velocity) | Đơn vị bán, return line, coverage theo ngày |
| US-DSS-06 Forecast | 16–18 | [DSS engines](06_dss_engines.md#demand-forecast-và-expiry-risk) | Actual, lịch, stockout, backtest |
| US-DSS-07 Expiry Risk | 19–21 | [DSS engines](06_dss_engines.md#demand-forecast-và-expiry-risk) | Snapshot batch, HSD, số lượng khả dụng |
| US-DSS-08 Association | 22–24 | [DSS engines](06_dss_engines.md#market-basket-và-mùa-vụ) | Giỏ hàng đã lọc, SKU đang bán, Promotion handoff |
| US-DSS-09 Mùa vụ | 25–27 | [DSS engines](06_dss_engines.md#market-basket-và-mùa-vụ) | Lịch phiên bản theo năm và ngữ cảnh Promotion |
| US-DSS-10 Khuyến nghị | 28–30 | [Lifecycle](02_usecases_and_lifecycle.md), [DSS engines](06_dss_engines.md#khuyến-nghị-chiến-lược) | Căn cứ có phiên bản, quyết định có audit |
| Cắt ngang | 31–35 | [API và bảo mật](05_api_contracts_and_security.md), [Acceptance](../../../04_testing/analytics-service/01_acceptance_and_verification_plan.md) | Quyền, version, freshness, cô lập tải, bàn giao có phê duyệt |
| US-AI-04 Product Recommendation | FR-31 / EPIC 03 | [Engine](12_recommendation_engine_design.md), [API](14_recommendation_api_and_storefront.md), [Delivery](15_recommendation_persistence_and_delivery.md) | Catalog/Inventory projections, Order baskets và consent/history theo từng strategy |

Metric ID, topic allowlist và permission trong bộ này là thiết kế đề xuất. Chủ dữ liệu xác nhận ngữ nghĩa và Analytics kiểm thử contract trước khi mở từng nguồn.
