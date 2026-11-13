# Xác minh sản phẩm, SKU và lô trước khi publish QR

**Ngày lập:** 13/11/2026 (UTC+7) · **Mốc:** `dev@239c8db` · **Trạng thái:** Chốt thiết kế ngày 13/11/2026

`GetBatchFEFODetails` đọc theo SKU, trả mã lô và thông tin FEFO nhưng thiếu UUID và revision. `GetCommittedPackingAllocation` có UUID và eligibility revision nhưng yêu cầu reservation đã commit. Traceability cần kiểm tra lô trước khi có đơn hàng. Contract checkout của Catalog phục vụ Order và dữ liệu bán hàng.

## Contract đọc

| Provider | Khóa tra cứu | Dữ liệu trả về |
| --- | --- | --- |
| Catalog · `GET /api/v1/internal/traceability/products/{product_id}/skus/{sku_code}` | `product_id`, `sku_code` | Binding chính xác và `catalog_version`. Trả binding cho SKU đã ẩn hoặc ngừng bán còn lưu trong Catalog. |
| Inventory · gRPC `GetTraceabilityBatch` | `batch_id`, `sku_code` | UUID, SKU, mã lô, ngày sản xuất, hạn dùng, trạng thái hạn chế, trạng thái hết tồn và `batch_version`. Traceability tính `best_before` và ghi thời điểm bắt đầu đọc hai nguồn. |

Catalog dùng `X-Internal-Token` với credential riêng cho Traceability. Inventory dùng TLS và bearer token riêng theo mẫu RPC nội bộ hiện có. Hai API chỉ mở trên mạng nội bộ. Inventory dùng `batches.id` làm định danh, đối chiếu SKU chính xác và đọc được cả khi hết tồn hoặc cách ly. Nếu chuyển kho tạo bản ghi mới, hồ sơ cũ tiếp tục trỏ UUID ban đầu. Giữ dữ liệu nguồn trong thời gian QR còn được phục vụ. Ngày sản xuất và hạn dùng dùng `YYYY-MM-DD`; `best_before` là đầu ngày kế tiếp sau hạn dùng theo giờ Việt Nam, biểu diễn UTC. Phiên bản tăng khi dữ liệu ảnh hưởng đến xác minh thay đổi. Hai lượt đọc dùng chung ngân sách 2 giây hiện có của publish.

Traceability đối chiếu cả hai kết quả với binding hồ sơ khi publish, áp dụng các điều kiện hiện có về độ mới, hạn dùng và trạng thái hạn chế, rồi lưu phiên bản nguồn cùng bản công bố. Tách lỗi dữ liệu không khớp, dữ liệu không đủ điều kiện và lỗi provider tạm thời để nhân viên biết cần sửa hồ sơ hay thử lại. Khi provider lỗi hoặc dữ liệu không đủ, publish phải dừng trước khi ghi bản công bố hoặc outbox. Sau publish, đính chính đi qua revision mới. Nếu phát hiện lô bị cách ly, nhân viên có quyền đình chỉ hồ sơ để QR trả `410`. Bản đọc trước publish chỉ chứng minh trạng thái nguồn tại thời điểm đọc.

## Nghiệm thu

- Lô và SKU khớp trên hai provider thật, trả đủ revision và publish thành công.
- Sai UUID, sai SKU hoặc SKU thuộc sản phẩm khác làm publish thất bại và không tạo event.
- Lô hết tồn vẫn đọc đúng UUID. Kiểm tra riêng lô hết hạn, cách ly và ngày bị đính chính.
- Publish đọc lại hai nguồn và so với snapshot đã gửi duyệt. Nếu dữ liệu hoặc revision thay đổi, trả conflict và yêu cầu gửi duyệt revision mới. Provider timeout và mất quyền đọc làm publish thất bại trước khi ghi bản công bố.
- Kiểm thử quyền đọc theo danh tính service và kiểm tra kết quả sau khi đổi credential.

Nguồn đối chiếu [Inventory proto](../../../../packages/proto/inventory/v1/inventory.proto), [Packing proto](../../../../packages/proto/packing/v1/packing.proto) và [dữ liệu Traceability yêu cầu](../../../../services/traceability-service/internal/domain/model.go).
