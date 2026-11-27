# Traceability — mục lục thiết kế

**Checkpoint27/11: target0.3.0 gRPC East-West.** [Report09](../../../04_testing/traceability-service/09_grpc_transport_2026-11-27.md) thay transport nội bộ của checkpoint08; nghiệp vụ/REST Web giữ nguyên, peer/UI vẫn pending.

Cập nhật26/11/2026. Phân biệt baseline đã có code/bằng chứng với target mới; không lấy trạng thái P8/R1 làm nghiệm thu scoped v2 hoặc nguồn gốc từ đơn hàng.

## Thiết kế hiện hành cho nghiệp vụ bổ sung

Đọc [mục lục bộ đặc tả dự án](../../../03_api_specs/traceability-workflow-README.md), sau đó:

| Nội dung | Nguồn chính |
| --- | --- |
| Quyết định nghiệp vụ | [EPIC19](../../../01_requirements/epics/EPIC_19_OCOP_Traceability.md), mục6.5 |
| Hành vi service, scope, state machine, data/migration, replay | [Service spec target0.2.2](../../../03_api_specs/traceability-workflow-service-spec.md) |
| Tương tác liên service và ranh giới transaction | [Workflow design](../../../03_api_specs/traceability-workflow-design-2026-11-21.md) |
| REST/gRPC/BFF và task từng owner | [API/contract index](../../../03_api_specs/README.md), [handoff trực tiếp](../../../03_api_specs/traceability-workflow-peer-handoff.md), [MVP registry](../../../../packages/contracts/mvp/registry.yaml) |
| Rủi ro đã đối chiếu code | [Review25/11](../../../03_api_specs/traceability-workflow-review-2026-11-25.md) |
| Kiểm thử phải thực hiện | [Testing index](../../../04_testing/traceability-service/README.md) |

Backend scoped v2/resolver đã implement và có [bằng chứng PostgreSQL/signed HTTP local](../../../04_testing/traceability-service/08_scoped_backend_2026-11-26.md); peer/UI integration vẫn pending. Target có thể được giao triển khai trực tiếp, không chờ owner gửi ACCEPT. Contract/handoff, kể cả nội dung phụ thuộc Catalog, nằm tại `docs/03_api_specs/` theo AGENTS.md; thư mục kiến trúc này chỉ liên kết nguồn, không sao chép lại các bảng wire/schema.

## Baseline và lịch sử thiết kế

- [00 readiness](00_readiness_review.md), [01 decisions](01_decisions_and_peer_code_review.md), [02 development plan](02_development_plan.md): quyết định/checkpoint giai đoạn đầu.
- [03 service design v1](03_service_design.md), [04 database design](04_database_design.md), [05 test matrix](05_test_matrix.md): baseline và invariants đã có; target sửa/bổ sung theo service spec mới, không đổi ngầm contract v1.
- [06 peer handoff](06_peer_contract_handoff.md), [07 P8 integration plan](07_p8_integration_plan_2026-11-13.md), [08 batch snapshot](08_batch_snapshot_2026-11-13.md), [09 authorization](09_staff_authorization_2026-11-13.md), [10 gateway/public QR](10_gateway_web_qr_2026-11-13.md): tài liệu P8. GLOBAL-only và route v1 là baseline, không phải scoped capability đã triển khai.

Khi sửa nội dung mới, sửa nguồn ở mục trên và cập nhật reference/version liên quan; không tạo một bản spec thứ hai ở đây. Canonical runtime proto và generated code chỉ thay theo implementation task và compatibility tests, không thay bằng việc di chuyển proto candidate.

Traceability 0.4.0 (27/11/2026) follows Duong's canonical Catalog/Media in PR #175/dev@ac5395c5. The [current guide](../../../03_api_specs/traceability-workflow-grpc.md) replaces the unshipped GetSkuBinding proposal, adds OCOP validation and durable post-publication binding, and preserves other workflow contracts. See [report10](../../../04_testing/traceability-service/10_catalog_media_alignment_2026-11-27.md) and reissued owner briefs; fixture results are not real-provider acceptance.
