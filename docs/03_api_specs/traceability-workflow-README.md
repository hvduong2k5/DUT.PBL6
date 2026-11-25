# Bộ đặc tả triển khai nghiệp vụ truy xuất theo lô

Cập nhật 25/11/2026 · Owner nghiệp vụ/contract: người phụ trách Traceability.

## Mục đích và thứ tự đọc

Bộ này chốt cách triển khai phần Traceability trước, rồi bàn giao đầu vào/đầu ra chính xác cho các service khác. Người nhận không phải đọc lịch sử chat hoặc tự thiết kế lại nghiệp vụ. Không có tài liệu nào có thể thay thế xác nhận khả năng provider hoặc kiểm thử; phần chưa được owner nhận vẫn được ghi rõ, không gọi là stable/đã chạy.

| Thứ tự | Tài liệu | Ai dùng / kết quả |
| --- | --- | --- |
| 1 | [Quyết định nghiệp vụ toàn luồng](traceability-workflow-design-2026-11-21.md) | Cả nhóm; xác định phạm vi, hai nghiệp vụ nhập kho/hồ sơ độc lập, QR theo lô |
| 2 | [Đặc tả Traceability](traceability-workflow-service-spec.md) | Owner Traceability; invariants, quyền, state machine, dữ liệu/migration, API, lỗi/retry, QR, vận hành |
| 3 | [REST tích hợp](traceability-workflow-draft.openapi.yaml), [admin scoped](traceability-scoped-admin-draft.openapi.yaml), [Identity BFF transports](traceability-workflow-bff-draft.openapi.yaml) | Backend/Web/Gateway; wire contract, schema, examples, security. Bản v1 hiện hành không bị thay ngầm |
| 4 | [Phiếu bàn giao từng owner](traceability-workflow-peer-handoff.md) | Inventory, Identity, Catalog, Fulfillment, Order, Web/Gateway; đầu ra OUT bắt buộc, bảng đối chiếu khả năng đã có, mapping/gap và bằng chứng nghiệm thu |
| 5 | [Proto Inventory candidate](traceability-workflow-candidates/batch_source.proto), [proto packing candidate](traceability-workflow-candidates/packing.proto) | Owner provider/consumer; field numbers và schema đề xuất, không phải canonical runtime proto |
| 6 | [Nghiệm thu và rollout](traceability-workflow-acceptance.md) | Người triển khai/reviewer/QA; test IDs, fixture, bằng chứng phải nộp, thứ tự rollout |
| 7 | [Review code và rủi ro25/11](traceability-workflow-review-2026-11-25.md) | Các xung đột bản0.2.1 đã phát hiện, cách sửa target và rủi ro còn lại |

Các state matrix chung admin/customer liên kết bộ này. `packages/contracts/mvp/registry.yaml` đăng ký các artifact draft. HLD/Kafka có đính chính để không hiểu packing kích hoạt QR.

## Những gì đã quyết định trong phạm vi này

- Nhập kho commit trước; UI nối “Lập/Xem hồ sơ nguồn gốc”, không tự tạo nháp bằng event.
- Nhân viên làm trong kho được giao; role chỉ là bundle quyền. Người đóng góp không tự duyệt.
- Publish của Traceability quyết định public; sinh hình QR tại Web để tải/in sau kiểm tra eligibility, không thêm QR generation service.
- Cùng lô dùng chung QR, không thay QR theo từng lần nhập/đơn/in lại. Không có serial chống giả hoặc QR tổng hợp kiện.
- Order dùng danh sách lô từ exact sealed proof; không dùng allocation live, không đổi event packing v2, không thêm consumer tự kích hoạt QR.
- Chuyển kho cùng Batch UUID không được tự thay scope; legacy chưa có binding kho chỉ GLOBAL đọc/xử lý an toàn. Không backfill đơn cũ bằng tồn live.
- Phần admin scoped dùng API v2 tách baseline v1 vì chuẩn hóa error/header và scope; scoped staff không fallback qua v1 GLOBAL-only.

## Baseline và mức sẵn sàng

Worktree `codex/traceability-order-flow` vẫn có nền `6f54a4ab` cùng commit tài liệu `bdb215d7`. Đã fetch và đối chiếu `origin/dev@7a80e3f4` ngày 25/11; chưa merge/rebase vào worktree có tài liệu chưa commit. Diff Traceability, Inventory, Fulfillment, Catalog, Identity proxy và canonical batch/packing proto giữa hai mốc không có thay đổi. Order có guest-session mới: credential `order-service-guest-session`, audience `order-storefront`; adapter tích hợp phải dùng provider ở đúng release, không dùng payment-read grant để đọc đơn.

Trạng thái: **business decisions recorded / direct implementation authorized / implementation pending / runtime verification pending**. Theo chỉ đạo mới của chủ yêu cầu ngày 25/11, bộ target0.2.2-draft là đầu vào triển khai trực tiếp; không có gate yêu cầu peer gửi ACCEPT/duyệt mapping trước code. Draft vẫn chưa đồng nghĩa stable hoặc đã có provider/runtime tests. Cách implement nội bộ do owner quyết định; contract đích thay đổi ngoài scope cần amendment/version theo review thông thường.

## Cách bàn giao để owner làm ngay

1. Gửi section task của [phiếu handoff](traceability-workflow-peer-handoff.md) cùng artifacts và test IDs. Owner đọc được yêu cầu OUT, wire contract và bước triển khai mà không cần lịch sử chat. Không yêu cầu họ phản hồi ACCEPT hoặc gửi phương án trước.
2. Owner tự kiểm code hiện có: đủ thì tái sử dụng và test, thiếu thì bổ sung. API nội bộ khác wire thì tự làm lớp tương thích trong phần phụ trách để consumer nhận đúng contract đích; không chờ duyệt mapping và không yêu cầu consumer đổi field/path.
3. Làm phần độc lập ngay bằng fixture đúng contract có nhãn nếu peer chưa lên; chỉ giữ runtime integration ở trạng thái chưa chạy. Chưa nhận phản hồi không là lý do dừng code.
4. Kết quả cuối/PR gồm phần đã làm, version/SHA, mapping nếu có, tests, bằng chứng và phần integration còn chờ. Chỉ báo blocker khi không thể giữ dữ liệu authoritative, backward compatibility hoặc contract trong phạm vi nhiệm vụ.
5. Review PR và promotion stable vẫn cần contract tests, đủ trạng thái/auth và bằng chứng thật theo AGENTS.md. Đây là kiểm soát sau thực hiện, không phải vòng xin duyệt trước sửa code. Chưa gửi việc qua công cụ nhắn tin bên ngoài hoặc push/deploy trong lượt tài liệu này.

Phạm vi và contract đích đã đủ để bắt đầu. Không yêu cầu owner thiết kế lại nghiệp vụ; chỉ cần quay lại quyết định chung nếu gặp mâu thuẫn thực tế vượt phạm vi không thể giải quyết bằng cách triển khai nội bộ. Không tạo thêm báo cáo hoặc deck trùng nội dung.

Bản0.2.2 bổ sung theo bằng chứng code: Catalog exact-SKU binding, Traceability batch-context, Identity cookie proxy/Origin/v2 employee guard, Web scoped projection và SKU bridge cho Order detail. Không còn giả định Inventory SKU detail đã có provider hoặc Identity chỉ cần cấu hình. Đọc review để phân biệt các gap cần triển khai với hạn chế còn được chấp nhận.

## Vị trí trong hồ sơ dự án

Bộ này được tìm thấy từ [README dự án](../../README.md), [mục lục API](README.md), [Requirements](../01_requirements/README.md), [kiến trúc Traceability](../02_architecture/services/traceability-service/README.md), [kiểm thử Traceability](../04_testing/traceability-service/README.md) và [README service](../../services/traceability-service/README.md). EPIC19/backlog/release-plan supplement/FR-20/use-case giữ liên kết nghiệp vụ; registry và named admin/customer draft giữ liên kết contract. Các index chỉ dẫn về nguồn chính, không tạo bản sao spec.

Tuân thủ AGENTS.md: các contract/handoff có nội dung Catalog vẫn trong `docs/03_api_specs/`; giữ catalog-checkout và admin-stable được registry tham chiếu; stable không import draft; generated rows/snapshot không sửa tay. Việc tổ chức vào bộ tài liệu không phải promote stable, nghiệm thu runtime hoặc merge lên dev. Mọi commit mới theo convention tiếng Anh của repository.
