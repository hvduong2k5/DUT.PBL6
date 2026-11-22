# 09 — Tốc độ bán và rủi ro cận hạn

Supply Analytics phục vụ quản lý kinh doanh và quản lý kho. Analytics sở hữu chỉ số, phạm vi dữ liệu và kết quả ước lượng. Order xác nhận lượng bán và lượng trả đã đối soát. Inventory xác nhận lịch sử hết hàng và số lượng bán được theo lô. Tồn vật lý, ngày hết hạn và quyết định xử lý hàng thuộc Inventory.

## Hợp đồng đầu vào pilot

Ba loại bản xuất được mô tả trong [JSON Schema](../../../03_api_specs/analytics-supply-source.v1.schema.json). Mỗi bản xuất là một revision đầy đủ cho cùng loại, SKU, kho và ngày Việt Nam, có `sourceAsOf`, `approvedBy` và `evidenceRef`. Chỉ operator đã được cấp quyền ghi mới nhập bản xuất đã được owner đối soát.

| Loại | Owner | Ý nghĩa |
| --- | --- | --- |
| `order_day` | Order | `soldUnits` là số đơn vị theo SKU/kho đã hoàn tất trong ngày sau khi lọc đơn test/hủy. `returnedUnits` là số đơn vị trả vật lý đã xác nhận trong ngày trả. Order đối soát dữ liệu trả với bên sở hữu trước khi phát bản xuất. Refund tiền riêng không suy ra số lượng trả. |
| `inventory_day` | Inventory | `stockoutMinutes` là số phút hết tồn bán được trong ngày đã đóng và đủ quan sát. Giá trị 0 xác nhận cả ngày có hàng. Thiếu quan sát thì bỏ bản xuất ngày đó. |
| `inventory_batches` | Inventory | Danh sách đầy đủ các lô bán được tại `sourceAsOf` cho một SKU/kho. `sellableUnits` đã loại hàng giữ chỗ, bị chặn hoặc không còn bán được. `expiresOn` là ngày đầu tiên lô không còn bán được theo giờ Việt Nam. Danh sách rỗng xác nhận kho/SKU không có lô bán được tại thời điểm chụp. |

Dùng số lượng nguyên theo cùng đơn vị bán của SKU. Bản xuất ngày phải có `sourceAsOf` từ biên đóng ngày trở đi. Nhập lại cùng revision và nội dung là idempotent. Nội dung khác ở cùng revision bị từ chối. Sửa dữ liệu bằng revision cao hơn, với `sourceAsOf` không lùi. Định danh scope và pipeline đến từ cấu hình service. PostgreSQL giữ các revision để truy vết và phục vụ các truy vấn giới hạn một SKU/kho. Các bản xuất chứa số đếm đã đối soát và mã lô, không chứa thông tin khách hàng.

Bản export pilot cần được Order/Inventory chấp thuận trước khi dùng trên dev. Exporter phía Order đếm dòng đơn thuộc cohort `order-core-1` đã hoàn tất theo ngày và so sánh với tổng đã đối soát. Exporter phía Inventory đọc số lượng bán được hiện tại từ dữ liệu lô theo mã kho nguồn, trạng thái và hạn dùng.

Ví dụ bản xuất Order cho ngày 31/10/2026.

```json
{
  "schemaVersion": "1",
  "kind": "order_day",
  "source": "order-service",
  "skuCode": "SKU-01",
  "warehouseId": "warehouse-hue",
  "businessDate": "2026-10-31",
  "revision": 1,
  "sourceAsOf": "2026-11-01T00:15:00+07:00",
  "approvedBy": "order-data-owner",
  "evidenceRef": "order-reconciliation/2026-10-31/warehouse-hue",
  "soldUnits": 20,
  "returnedUnits": 2
}
```

## Publisher phía owner

Chạy job từ checkout chứa SQL của owner. Mỗi manifest có `warehouseId` dùng chung trong Analytics, `sourceWarehouseId` theo định danh của owner, `approvedBy` và mảng `exports`. Mỗi phần tử có `kind`, `skuCode`, `businessDate`, `revision` và `evidenceRef`.

| Loại | Trường bổ sung của manifest |
| --- | --- |
| `order_day` | `expectedSoldUnits`, `returnedUnits` và `sourceAsOf`. Query chỉ công bố nếu tổng bán khớp. Owner lấy số lượng trả vật lý từ nguồn đối soát. |
| `inventory_day` | `stockoutMinutes` và `sourceAsOf` từ quan sát đủ ngày. Schema Inventory hiện thiếu lịch sử quan sát liên tục để tự dựng lại con số này. |
| `inventory_batches` | Dùng ngày Việt Nam hiện tại và mã kho Inventory. Publisher lấy giờ chụp từ database. Mỗi lần chụp mới trong ngày dùng revision cao hơn. |

```bash
python scripts/publish-analytics-supply.py order-service /run/owner/order-manifest.json /srv/supply-inbox
python scripts/publish-analytics-supply.py inventory-service /run/owner/inventory-manifest.json /srv/supply-inbox
```

Manifest cần được ghi xong trước khi scheduler gọi publisher. Giữ nguyên file revision đã công bố. Khi dữ liệu đổi, công bố revision cao hơn. Có thể chuyển file đã nhập sang kho lưu trữ theo chính sách giữ bằng chứng. Publisher ghi từng file nguyên tử và có thể chạy lại manifest sau lỗi giữa chừng.

## Tốc độ bán

`GET /api/v1/admin/analytics/supply/velocity` nhận `skuCode`, `warehouseId`, `from`, `to` và tùy chọn `comparison=previous_period`. Kỳ có từ 1 đến 90 ngày, `from` bao gồm và `to` loại trừ.

- Tốc độ bán ròng là (số lượng bán − số lượng trả) / số ngày có bản xuất Order đủ coverage. Số ròng có thể âm nếu lượng trả ghi nhận trong kỳ lớn hơn lượng bán. Mỗi bucket có revision của hai nguồn khi có dữ liệu.
- Ngày thiếu dữ liệu dùng `null`. Ngày có số 0 cần bản xuất xác nhận. `stockoutDays` đếm ngày có ít nhất một phút hết hàng trong tập ngày Inventory đã xác nhận. Báo cáo trả coverage theo từng nguồn và `PARTIAL` khi có dữ liệu bán nhưng chưa đủ kỳ. Khi cả hai kỳ đủ coverage, API trả chênh lệch lượng bán ròng.
- Ví dụ hai ngày bán 20 hộp và trả 2 hộp cho tốc độ bán ròng 9 hộp/ngày. Nếu Inventory mới có bản xuất cho một ngày, kết quả bán vẫn hiện kèm coverage thiếu.

## Rủi ro cận hạn

`GET /api/v1/admin/analytics/supply/expiry-risk` nhận `skuCode`, `warehouseId` và `horizonDays` từ 1 đến 90, mặc định 30. API lấy snapshot lô mới nhất, yêu cầu độ mới trong ngưỡng cấu hình và quan sát đủ 14 ngày trước ngày snapshot cho cả Order và Inventory. Baseline lấy lượng bán hoàn tất trung bình trong các ngày có hàng suốt ngày, tối thiểu 7 ngày. Lượng trả vẫn xuất hiện trong báo cáo tốc độ bán ròng riêng.

- Mô phỏng từ ngày sau ngày snapshot, theo FEFO. Mỗi đơn vị nhu cầu chỉ phân bổ một lần. Nhu cầu tích lũy làm tròn xuống theo đơn vị nguyên. Khi hết hạn, lượng còn lại của lô là `unitsAtRisk`. Các lô hết hạn trong horizon được trả theo thứ tự FEFO.
- Ước lượng chỉ dùng tồn bán được tại snapshot, bỏ phần còn lại của ngày chụp và chưa tính nhập hàng sau snapshot. Ví dụ lô A còn 25 hộp và hết hạn sau 3 ngày, nhu cầu 10 hộp/ngày cho 2 ngày bán trước hạn thì ước tính còn 5 hộp. Lô B hết hạn sau 5 ngày nhận nhu cầu còn lại theo thứ tự FEFO.
- Trả `UNAVAILABLE` với lý do khi snapshot thiếu, cũ hoặc không đủ baseline. Nếu snapshot còn mới và không có lô hết hạn trong horizon, trả danh sách rỗng. API trả kết quả phân tích để quản lý chọn hành động.

## Nghiệm thu một kho

1. Chọn vài SKU và 14 ngày liên tiếp đã đóng. Kiểm tổng số bán và hàng trả trong chi tiết từng SKU với bằng chứng Order và tổng tồn bán được từng lô với snapshot Inventory cùng thời điểm.
2. Dùng `/admin/analytics` để xem xếp hạng nhanh/chậm, danh sách lô có lượng dự kiến còn khi đến hạn và chi tiết từ SKU xuống lô. Chọn một lô và kiểm lại bằng công thức FEFO với quản lý kho. Ghi ngày chụp, nhu cầu mỗi ngày, số đơn vị đã phân bổ và số đơn vị có nguy cơ.
3. Kiểm một ngày thiếu dữ liệu hoặc một snapshot quá hạn hiển thị lý do đúng. Công bố một revision sửa, đợi chu kỳ nhập và kiểm lại số liệu. Ghi kết quả cùng danh tính staff được cấp quyền và image digest chạy trên dev.

[OpenAPI](../../../03_api_specs/analytics-service.candidate.openapi.yaml) · [Cách chạy pilot](../../../../services/analytics-service/README.md)
