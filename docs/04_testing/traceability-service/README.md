# Traceability — mục lục kiểm thử và bằng chứng

Cập nhật25/11/2026. Mục lục này không phải báo cáo PASS cho target mới.

## Target0.2.2: việc phải nghiệm thu

- [Acceptance/test IDs và rollout](../../03_api_specs/traceability-workflow-acceptance.md): nguồn duy nhất của T-*, INV-*, IAM-*, CAT-*, FUL-*, EVT-*, ORD-*, WEB-*.
- [Review code/rủi ro và kết quả kiểm artifact](../../03_api_specs/traceability-workflow-review-2026-11-25.md): phân biệt static checks đã chạy với runtime chưa chạy.
- [Bộ contract và task owner](../../03_api_specs/traceability-workflow-README.md): đầu vào triển khai và kiểm thử.

Trạng thái runtime target: **NOT RUN / NOT VERIFIED**. Lint/schema/proto compatibility xanh không chứng minh scope kho, cookie transport, in tem hoặc source proof chạy thật. Không dùng evidence sandbox/v1/P8 bên dưới để đóng các case target. Mỗi run mới phải ghi SHA/image, mode/fixture/peer thật, testID, requestId và ảnh khi áp dụng; không đưa secrets/PII vào report. Bằng chứng mới thuộc `docs/04_testing/traceability-service/` theo runID, không ghi đè report lịch sử hoặc tạo catalog contract tại đây.

## Baseline đã có tài liệu bằng chứng

| Tài liệu | Cách sử dụng |
| --- | --- |
| [01 verification](01_verification.md) | Checkpoint sandbox P7, đọc phạm vi/ngày trong file |
| [02 reviewer runbook](02_reviewer_runbook.md) | Quy trình chạy baseline và điều kiện môi trường |
| [03 readiness/PR draft](03_review_readiness_and_pr_draft.md) | Trạng thái review tại checkpoint cũ |
| [04 final R1 audit](04_final_r1_audit.md) | Kết quả trong phạm vi R1, không thay runtime acceptance target0.2.2 |
| [05 MVP alignment](05_mvp_contract_alignment.md) | Đối chiếu contract cũ |
| [06 P8 implementation](06_p8_implementation_2026-11-13.md) | Bằng chứng adapter/gateway P8 theo phạm vi ghi trong report |
| [07 P8 staging plan](07_p8_staging_test_plan_2026-11-13.md) | Kế hoạch, không mặc định tất cả ca đã chạy |

Unit/integration tests triển khai vẫn nằm trong từng service; báo cáo ở đây dẫn tới kết quả thật. Chưa có run mới thì ghi chưa chạy, không copy chữ PASS từ lịch sử. Review PR/deploy theo quy trình nhóm, không thử trên deploy release cũ rồi nhận là bản sửa mới.
