# Đợt giao đầu Order — implementation, ADR và cách chạy

Ngày: 15/10/2026. Profile: `ORDER_RELEASE_PROFILE=mvp1`. Đây là **backend sandbox của đợt đầu Order**, chưa phải hoàn thành toàn bộ MVP 1.0 của các epic và chưa được nghiệm thu production.

Cập nhật 16/10/2026 lúc 14:17 (Việt Nam): **luồng thanh toán merchant sandbox thật đã PASS**. Tạo đơn 135.000 VND, mở checkout BANK_TRANSFER, bấm giả lập thanh toán trên SePay; IPN thật trả 200, notification APPLIED, một receipt, Order PAID / payment CONFIRMED / stock COMMITTED. Kho/profile/vận chuyển vẫn là simulator, không chuyển tiền thật. Lần đầu IPN trả 401 vì merchant chọn Auth Type “Không có”; đổi sang Secret Key và thử đơn mới đã thành công. Đã sửa mô tả đơn được ký, tra cứu detail theo invoice và timeout có giới hạn; unit/race, vet và tập integration TestMVP đã qua. Xem [bằng chứng live sandbox](../../../04_testing/order-service/live-sepay-sandbox-verification.json), [ADR-005](adr/ADR-005-local-sepay-sandbox-validation.md) và [ADR-006](adr/ADR-006-sepay-query-verified-identifiers.md). Bằng chứng 61 test bên dưới thuộc lần chạy 15/10, không phải kết quả test merchant thật.

Cập nhật đợt ngoại lệ cùng ngày: **65 test chính, 29 ca con PASS**, đã thử replay payload IPN thật qua HTTPS, hủy trên SePay, chờ hết hạn thực, thanh toán sau hết hạn, backend ngừng khi có IPN và quy trình manual refund với chứng từ giả lập. Đã sửa ba vấn đề về worker query chặn IPN, thiếu cảnh báo khi query không có transaction và cảnh báo chặn IPN hợp lệ tới sau. [Báo cáo kiểm thử đầy đủ](../../../04_testing/order-service/SEPAY_MVP_TEST_REPORT.md) · [ADR-007](adr/ADR-007-payment-recovery-exception-tests.md).

Giới hạn còn lại: thiếu/thừa tiền kiểm thử bằng payload giả lập; hoàn tiền chưa có giao dịch ngân hàng thật. CAPTURED với transactions rỗng chỉ vào review, chưa đủ để tự phục hồi mất IPN; cần xác minh với provider trước production. Tunnel chỉ phục vụ phát triển và URL sẽ cần cập nhật nếu đổi. Production vẫn bị tắt.

## 1. Đã triển khai gì?

| Phần | Kết quả |
| --- | --- |
| Phạm vi | Quote/checkout chỉ CUSTOMER + VIETQR; Guest và COD bị từ chối trước giữ hàng |
| Payment attempt | Lưu cùng transaction tạo Order, một attempt/Order; reload dùng cùng invoice |
| SePay checkout | Trả form POST có chữ ký server, BANK_TRANSFER, danh sách field có thứ tự; không trả merchant secret |
| SePay IPN | Xác thực X-Secret-Key trước lưu; mã hóa payload; ACK sau persist; worker xử lý sau |
| Tiền và kho | Chỉ PAID sau receipt đủ điều kiện và stock COMMITTED; simulator callback không thể trả tiền thay SePay |
| Ngoại lệ | Tiền trùng/đến muộn/sai số tiền giữ receipt và case; notification chưa hiểu giữ để review và đặt local hold nếu khớp đơn |
| Phục hồi | Queue IPN bền vững; query danh sách/chi tiết SePay, gồm đơn đã hủy/hết hạn; dùng transaction resource ID thống nhất |
| Duyệt hủy | Manager duyệt/từ chối request; worker dừng task và đảo kho bằng barrier, không coi approval là hủy xong |
| Manual refund | Duyệt toàn khoản thu, giữ budget, một người nhận việc, UNKNOWN không gửi lại; người thứ hai xác nhận evidence |
| Sự kiện manual | Fact riêng và outbox cùng transaction; không tạo provider refund proof giả |
| Completion | Profile không bật tự động COMPLETED; giữ DELIVERED |

Mã tiền đọc từ chuỗi và ghi số nguyên VND. Định danh provider, cấu hình sandbox và receiver được kiểm ở module Payment. Không dùng trang redirect success của khách làm chứng cứ nhận tiền.

## 2. Cách chạy

**Kiểm thử tự động không cần khóa SePay thật:**

```powershell
docker compose -f services/order-service/docker-compose.test.yml up -d db redis kafka-init
docker compose -f services/order-service/docker-compose.test.yml run --rm tests
python -m pip install -r services/order-service/scripts/requirements-verification.txt
python services/order-service/scripts/verify-mvp.py
```

Cần image công cụ `omamx/order-go-tools:local`; nếu chưa có:

```powershell
docker build -f services/order-service/Dockerfile.tools -t omamx/order-go-tools:local services/order-service
```

Tests dùng phản hồi SePay giả lập theo protocol và PostgreSQL/Redis/Kafka thật trong môi trường test. Không chuyển tiền hoặc gọi merchant sandbox của bạn.

**Chạy API với tài khoản SePay sandbox của bạn:** đặt `SEPAY_MERCHANT_ID`, `SEPAY_SECRET_KEY`, `SEPAY_IPN_SECRET`, `SEPAY_RETURN_URL` trong môi trường shell. Không commit khóa. IPN key phải đúng giá trị cấu hình auth type SECRET_KEY bên SePay; không giả định có thể tự chọn key khác mà không cấu hình ở merchant.

```powershell
docker compose -f services/order-service/docker-compose.test.yml -f services/order-service/docker-compose.mvp.yml up -d --build api
```

Sử dụng database sandbox riêng cho profile này; không chuyển một checkout đang chạy của sandbox 3.1 sang profile mới. Các enum và nghĩa vụ lịch sử vẫn được giữ, nhưng chưa có công cụ migrate cohort đang chạy. `APP_MODE=production` và `SEPAY_ENVIRONMENT=production` tiếp tục bị từ chối.

Cấu hình IPN URL phía SePay trỏ tới `/api/v1/payments/sepay/ipn` qua địa chỉ HTTPS mà SePay gọi được. Return URL là trang kết quả của frontend, chỉ dùng để hướng dẫn khách quay lại xem đơn. Không có callback nào từ trình duyệt ghi PAID.

Frontend dev có thể dùng Return URL HTTP trên localhost/127.0.0.1; IPN từ SePay vẫn cần địa chỉ HTTPS công khai. Không đăng thông tin xác thực hoặc nội dung form của merchant thật lên báo cáo test.

## 3. Frontend cần gọi gì?

1. Đăng nhập CUSTOMER, cập nhật giỏ và tạo quote như API cũ.
2. POST checkout nhận 202; poll checkout operation đến khi có order_id.
3. POST `/api/v1/orders/{id}/payment-checkout` bằng token của chủ đơn.
4. Dựng form HTML method POST tới `action`; tạo hidden input theo **đúng thứ tự** trong `fields`. Không tự sửa số tiền, invoice hay tên trường. Gửi form sang SePay bằng thao tác của khách.
5. Khi quay lại website, GET Order để xem trạng thái. Nếu đang PAYMENT_FINALIZING, hiển thị “Đã nhận tiền, đang xác nhận đơn”. Không cho trả lần nữa khi kết quả chưa rõ.

Không có ứng dụng frontend mới trong thay đổi này. [OpenAPI profile](../../../03_api_specs/order-service.mvp1.openapi.yaml) mô tả API để frontend tích hợp.

## 4. Nhân viên xử lý ngoại lệ thế nào?

GET `/api/v1/admin/payments/review` trả khoản thu, lý do đối soát và ID refund/cancellation đang chờ, giới hạn 100 mục. API này là snapshot vận hành, không phải sao kê hoặc báo cáo đầy đủ. ADMIN xem được ID notification chưa xử lý được và GET `/api/v1/admin/payments/notifications/{id}` để đọc dữ liệu gốc có quyền.

Với đơn đã trả tiền, khách tạo cancellation request; Manager POST `/api/v1/admin/cancellation-requests/{id}/decision` để duyệt/từ chối. Sau duyệt, chờ worker hoàn tất cancellation và kiểm kết quả kho rồi mới tạo hoàn tiền cho receipt đã allocate. Nếu task remote đã được nhận, hủy bị từ chối; không nhả kho hoặc giả vờ đã dừng đóng gói.

Hoàn khoản dư/đến muộn hoặc đơn đã hủy:

1. Manager kiểm tra receipt và xác minh tài khoản nhận với khách; POST `/api/v1/admin/manual-refunds` cùng key chống trùng. Đây là duyệt và giữ ngân sách, chưa gửi tiền.
2. Người thực hiện POST `/{id}/claim` trước khi chuyển khoản ở ngân hàng. Không để hai người cùng thao tác.
3. Không rõ đã chuyển chưa: POST `/{id}/unknown`, kiểm sao kê trước. Hệ thống giữ ngân sách và người phụ trách.
4. Người thực hiện POST `/{id}/evidence` với mã giao dịch ngân hàng, nguồn tiền, số tiền, thời điểm và tham chiếu sao kê đã đối chiếu. Evidence không được âm thầm sửa sau ghi.
5. Một Manager khác GET chi tiết, đối chiếu ngân hàng rồi POST `/{id}/confirm` với evidence_hash đã xem. Chỉ lúc đó ledger ghi SUCCEEDED và tạo một sự kiện manual.

API không tự chuyển tiền ngân hàng. Ảnh chụp hoặc một nút “đã chuyển” không tự xác nhận thành công. Tính năng manual đòi hai tài khoản nhân viên có quyền để thực hiện bước xác nhận độc lập.

## 5. ADR và trade-off

Đọc [chỉ mục ADR](adr/README.md). Bốn quyết định được ghi cùng đợt implementation, có nguồn provider và giới hạn bằng chứng. Các lựa chọn chưa được chứng minh không ghi thành capability đã có.

| Đánh đổi | Tại sao chấp nhận ở đợt đầu? |
| --- | --- |
| Chỉ QR và login | Giảm số nhánh tích hợp; ghi rõ thiếu Guest của MVP sản phẩm |
| Trang thanh toán SePay | Frontend xây ít hơn, đổi lại phải giữ form order và cấu hình callback |
| Manual refund | Không chờ API refund cho đúng tài khoản; cần người vận hành và chưa hỗ trợ partial |
| Fact manual riêng | Không làm sai semantics v3; cần consumer mới nếu Finance/Notification dùng |
| Giữ dependency simulator | Cho phép kiểm recovery cục bộ; production vẫn bị chặn |

## 6. Bằng chứng và giới hạn

- Go vet và toàn bộ Go tests có race detector: xem [tóm tắt kiểm chứng](../../../04_testing/order-service/mvp1-verification.json).
- OpenAPI và response thật từ integration tests: xem [contract verification](../../../04_testing/order-service/mvp1-contracts-verification.json).
- SePay merchant sandbox thực: **NOT_RUN — chưa có credentials**. Không coi fixture protocol là kết quả nhà cung cấp chấp nhận.
- Identity, Inventory/Fulfillment/Shipping production và UI quản trị/frontend mới: chưa nghiệm thu bởi đợt này.
- Shared proto/v3 target không đổi; runtime vẫn có adapter HTTP/sự kiện v2 lịch sử. Không tự công bố đã triển khai toàn bộ baseline 3.2.

Các mục còn thiếu so với epic được ghi trong [ADR-001](adr/ADR-001-mvp-delivery-profile.md): Guest, Return Case/media/thẩm định, hoàn một phần, UI vận hành, email xác nhận và nhà vận chuyển thực. Không sửa epic để xóa những yêu cầu đó khỏi lịch sử sản phẩm.
