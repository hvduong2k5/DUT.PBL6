# Báo cáo kiểm thử Order MVP và SePay Sandbox

Ngày: **16/10/2026**, giờ Việt Nam. Profile: `mvp1`. Không sử dụng tiền thật.

## 1. Kết quả và phạm vi

**Bộ test tự động cuối cùng: 65 test chính và 29 ca con đều PASS; 0 FAIL, 0 test nghiệp vụ bị skip.** Có 18 test chính dành riêng cho MVP; các test còn lại kiểm tra domain, giao thức SePay và hồi quy runtime cũ. Không cộng ca con với test cha rồi coi thành 94 kịch bản độc lập.

- `go vet ./...`, `go test -race -count=1 -json ./...`, kiểm tra định dạng Go đã thay đổi và Docker build: PASS.
- PostgreSQL, Redis, Kafka chạy thật trong Docker. Bộ Go integration dùng database schema riêng, Redis DB 1 và dependency simulator; phản hồi provider trong bộ này là giả lập.
- Có thử trực tiếp merchant SePay Sandbox qua trình duyệt, nhận IPN thật qua HTTPS, tra cứu API thật và kiểm tra database của backend đang chạy.
- Có 10 kiểm tra HTTP bổ sung trên một receipt từ SePay thật: xác thực, replay, quyền xem, hủy đơn, đảo kho và quy trình kế toán hoàn tiền. Đây không phải 10 giao dịch ngân hàng thật.
- Kiểm tra contract: 4 OpenAPI PASS; 8 schema sự kiện, 9 ví dụ hợp lệ, 32 ví dụ sai bị từ chối. Schema manual refund riêng cũng từ chối 6 biến thể sai. Không chạy lại biên dịch proto trong đợt này vì không thay đổi proto.

**Giới hạn quan trọng:** luồng phục hồi mất IPN chưa thể tự xác nhận PAID trên sandbox khi API detail chỉ trả CAPTURED và transactions rỗng. Hệ thống đã ghi cảnh báo để xử lý, không tự tạo bằng chứng tiền. Production vẫn tắt; kết quả này không phải nghiệm thu toàn bộ epic MVP 1.0 sản phẩm.

## 2. Các ca thử trực tiếp và thử bằng dữ liệu provider thật

| Ca | Cách thử và điều kiện | Kết quả |
| --- | --- | --- |
| Thanh toán thành công | Tạo đơn 135.000 VND, gửi form ký từ Order, bấm Giả lập thanh toán trên SePay, nhận IPN thật | PASS: HTTP 200, receipt đúng một lần, PAID / CONFIRMED / COMMITTED |
| Thanh toán sau bản sửa cuối | Một đơn mới, sau khi deploy bản đã tách worker và xử lý warning | PASS: PAID / CONFIRMED / COMMITTED |
| Sửa số tiền ở trình duyệt | Đổi amount từ 135.000 thành 1 nhưng giữ chữ ký cũ | PASS: SePay hiển thị Yêu cầu không hợp lệ; form gốc cùng đơn được chấp nhận và thanh toán thành công |
| Thiếu khóa IPN | Gửi lại payload IPN thật qua endpoint HTTPS, bỏ X-Secret-Key | PASS: 401 |
| Sai khóa IPN | Gửi cùng payload với khóa giả | PASS: 401 |
| IPN lặp | Gửi lại payload đã nhận từ SePay ba lần, thêm một lần đổi timestamp gửi | PASS: vẫn một receipt và một allocation. Đây là replay chủ động, không khẳng định SePay đã tự retry bốn lần |
| Trả tiền thêm qua endpoint checkout của đơn đã PAID | Gọi API payment-checkout bằng chủ đơn | PASS: 409, không cấp checkout mới |
| Người khác xem đơn | Customer khác gọi GET order | PASS: 404 |
| Bấm Hủy giao dịch trên SePay | Bấm nút, xác nhận trong hộp thoại | PASS: quay về result=cancel; Order chưa được đánh dấu hủy chỉ vì return URL |
| Hết hạn thực | Chờ hết 5 phút của đơn thử, không sửa đồng hồ hoặc deadline trong database | PASS: CANCELLED_TIMEOUT / RELEASED |
| Trả tiền sau hết hạn | Mở lại form cũ của đơn trên, SePay còn chấp nhận, bấm giả lập thanh toán | PASS: lưu một receipt và một case đối soát; 0 allocation, đơn vẫn CANCELLED_TIMEOUT, kho RELEASED |
| Backend dừng lúc có IPN | Tạo checkout trước, dừng API, bấm thanh toán, nhận 503 rồi khởi động API lại | PASS cho khả năng báo lỗi trung thực và khởi động lại; chưa phải PASS tự phục hồi thanh toán |
| Mất IPN và tra cứu | API thật báo CAPTURED nhưng transactions=[] | LIMITED: không có receipt để tự xác nhận; tạo MANUAL_REVIEW với PROVIDER_TRANSACTION_PROOF_MISSING |
| Hoàn tiền trước khi trả kho | Receipt của đơn đã PAID, chưa hủy/đảo kho, gọi approve refund | PASS: 409 |
| Hủy đơn đã trả tiền | Chủ đơn tạo yêu cầu; admin duyệt, gửi lại quyết định; backend đảo kho mô phỏng | PASS: CANCELLED_BY_ADMIN / RELEASED |
| Approve hoàn tiền lặp | Cùng request key và nội dung | PASS: cùng refund_id |
| Hai người nhận cùng việc hoàn tiền | Người thứ hai claim refund đã có người nhận | PASS: 409 |
| Chưa rõ kết quả hoàn tiền | Đánh dấu UNKNOWN rồi thử tạo nghĩa vụ hoàn tiền lần nữa | PASS: ngân sách vẫn bị giữ, không tạo hoàn tiền trùng |
| Tự kiểm tra việc hoàn tiền của mình | Người nộp bằng chứng gọi confirm | PASS: 403 |
| Người khác kiểm tra và gửi lại | Người độc lập confirm hai lần | PASS: SUCCEEDED, payment REFUNDED, một event manual refund trong outbox |

**Hoàn tiền ở bảng trên kiểm tra workflow nội bộ trên một receipt sandbox thật. Tài khoản nhận và chứng từ chuyển khoản đều ghi rõ SYNTHETIC; không gọi API chuyển tiền hoặc thực hiện hoàn tiền ngân hàng.** Kho của các ca này cũng là simulator.

## 3. Những ngoại lệ kiểm thử tự động qua cấu hình MVP

Các ca sau dùng phản hồi provider có kiểm soát, qua HTTP handler hoặc tầng nghiệp vụ cùng PostgreSQL thật. Không gọi chúng là chuyển khoản thiếu/thừa tiền thật trên SePay.

| Nhóm | Điều kiện đã kiểm tra | Kỳ vọng đã qua |
| --- | --- | --- |
| Phạm vi MVP | Guest, COD, automatic refund, callback simulator cũ | Bị chặn, không gây giữ hàng hay tạo nghĩa vụ ngoài phạm vi |
| Checkout | Một attempt tạo cùng Order; tải lại form; khác chủ; đơn đã PAID | Invoice/form ổn định; không lộ secret; đúng quyền và trạng thái |
| Chữ ký và dữ liệu tiền | Chuỗi ký cố định đối chiếu HMAC; số tiền chuỗi/int; VND có .00; số lẻ, âm, exponent, vượt giới hạn | Không dùng float cho tiền; từ chối dữ liệu không hợp lệ |
| IPN HTTP | Thiếu/sai key; object rỗng; timestamp=0; thiếu type; array/string thay object | 401/400, không lưu như thông báo hợp lệ |
| Payload xác thực nhưng hỏng | Amount null, ID boolean và dữ liệu transaction không hợp lệ | ACK sau lưu bền vững; worker chuyển manual review, không nhận là tiền |
| Thiếu tiền | Một transaction thấp hơn tổng đơn | Receipt được giữ để đối soát, không allocation/PAID |
| Thừa tiền | Một transaction cao hơn tổng đơn | Không tự phân bổ một phần; giữ khoản tiền để đối soát |
| Chuyển chia nhiều lần | Hai transaction mỗi cái bằng nửa tổng đơn | Không tự cộng dồn trái quy tắc một receipt đủ tiền; hai case đối soát |
| Sai mã đơn | Invoice không tồn tại | Giữ receipt UNMATCHED, không gán vào đơn khác |
| Đổi provider order | Cùng invoice, transaction sau mang provider order_id khác attempt đã bind | Không phân bổ vào attempt không khớp |
| Sai phương thức/loại tiền/trạng thái | CARD, USD, DECLINED | Không tạo receipt hợp lệ, đưa vào review |
| Thông báo bất lợi/không hỗ trợ | TRANSACTION_VOID, loại sự kiện lạ | Giữ dấu vết và ngăn cấp quyền giao hàng mới theo quy tắc hold hiện có |
| Lặp và đồng thời | 20 payload khác timestamp cho cùng transaction, 8 worker xử lý đồng thời | Một receipt, một allocation, physical stock chỉ trừ một lần |
| Transaction bị sửa | Cùng ID transaction nhưng đổi amount sau đó | Xung đột bất biến được lưu audit/review, không sửa receipt gốc |
| Chuyển tiền lần hai | Transaction mới cho đơn đã PAID | Giữ khoản dư riêng, không allocation lần hai |
| Tiền tới trễ | Đã timeout hoặc hủy trước nhận tiền | Không phục hồi đơn đã hủy; tiền được giữ để đối soát |
| Hết giữ chỗ trước finalize | Nhận tiền rồi Inventory trả EXPIRED | Không phát paid/ready; đối soát và trả tồn khả dụng |
| Đua timeout với receipt | Timeout và xử lý IPN chạy đồng thời khi deadline đã qua | Không chốt hàng cho đơn hết hạn |
| Mất phản hồi finalize | Inventory đã chốt nhưng ACK bị mất | Retry dùng cùng nghiệp vụ; không trừ kho hai lần |
| Query và IPN cùng giao dịch | Query tạo bằng chứng trước, service restart, IPN tới sau | Một transaction identity, một receipt/allocation |
| Provider query lỗi | HTTP 503, detail sai invoice/ID | Không tiến cursor sai, không tạo tiền |
| CAPTURED thiếu transaction | Query trả trạng thái nhưng không có chi tiết | Có review, không bịa receipt; IPN hợp lệ tới sau trong hạn vẫn PAID |
| Query chậm | Giữ HTTP query đang mở trong khi gửi IPN mới | IPN được xử lý độc lập, không chờ query |
| Từ chối yêu cầu hủy | Manager reject, gửi lại cùng quyết định, thử đổi ý | Đơn còn PAID và được giao; replay ổn định; đổi quyết định bị chặn |
| Quyền duyệt hủy | Customer, packing staff, manager khác kho, thiếu approve | 403/404/422, không đổi đơn |
| Duyệt hủy và trả tiền | Duyệt trước packing, reverse tồn, approve/claim/evidence/confirm | Đơn hủy, tồn trả lại, payment REFUNDED sau xác nhận |
| Quyền và replay hoàn tiền | Customer, manager khác kho; thiếu key; cùng key cùng/khác payload | Không lộ chi tiết; không tạo nghĩa vụ trùng |
| Ngân sách hoàn tiền | Hai approve đồng thời; UNKNOWN; hoàn một phần | Không vượt receipt; UNKNOWN giữ ngân sách; chỉ full receipt trong MVP |
| Nhận việc hoàn tiền | Hai người claim đồng thời | Một người thắng; người khác không thể thực hiện cùng việc |
| Chứng từ hoàn tiền | Sai amount, thời gian tương lai, gửi lại, sửa chứng từ, dùng cùng giao dịch ngân hàng cho hai refund | Sai bị chặn; replay ổn định; chứng từ bất biến và không dùng hai lần |
| Kiểm tra độc lập | Chưa có chứng từ, tự confirm, người khác confirm rồi replay | Không settle sớm; cần người độc lập; một event |
| Khoản trả dư | Hoàn khoản dư trong khi giữ giao dịch bán gốc | Không thay đổi trạng thái đã trả tiền/chốt hàng của khoản bán gốc |
| Giao hàng MVP | Packing → sealed → shipped → delivered | Đến DELIVERED; không tự chuyển COMPLETED khi quá hạn hoàn tất |

Bộ hồi quy cũ còn kiểm tra giỏ/quote, idempotency checkout, quyền gRPC, lease worker, transaction/outbox atomicity, Kafka inbox deduplication, event sai thứ tự, barrier packing/cancel và audit bất biến. Các ca COD/Guest của profile cũ chỉ bảo vệ tương thích; chúng không có nghĩa COD/Guest được bật trong MVP.

## 4. Lỗi phát hiện và đã sửa

| Vấn đề | Trước sửa | Sau sửa |
| --- | --- | --- |
| Query chậm chặn IPN | Cùng worker nên một query treo khiến IPN đã lưu phải đợi | Hai worker độc lập, shutdown có chờ; test giữ query mở vẫn PAID |
| Mất IPN, query thiếu transaction | Không có receipt, cũng không có cảnh báo nghiệp vụ | Lưu observation mã hóa, có lý do PROVIDER_TRANSACTION_PROOF_MISSING trong review |
| Cảnh báo query chặn IPN tới sau | Nếu hold ngay, receipt hợp lệ tới sau bị coi không khả dụng | Query observation không tự hold; PENDING chưa được giao hàng, vẫn có thể nhận IPN hợp lệ |

Không sửa deadline hoặc tự tạo transaction để biến ca mất IPN thành PASS. [ADR-007](../../02_architecture/services/order-service/adr/ADR-007-payment-recovery-exception-tests.md) giải thích quyết định và đánh đổi.

## 5. Giới hạn còn lại và tiêu chí trước production

1. **Phục hồi mất IPN:** cần xác nhận với SePay cách lấy transaction của BANK_TRANSFER sandbox/production và cách gửi lại IPN. Khi detail rỗng, hiện chỉ có review; chưa tự PAID và chưa thể tạo nghĩa vụ hoàn tiền từ một receipt chưa tồn tại.
2. **Ca thiếu/thừa tiền thật:** UI giả lập hiện quan sát được chỉ có một nút trả đủ số tiền. Đã kiểm thử logic bằng dữ liệu có kiểm soát; chưa chứng minh hành vi ngân hàng/provider cho chuyển tiền lệch số tiền.
3. **Tiền hoàn thực tế:** chưa test chuyển tiền ngân hàng hay API hoàn tiền; thiết kế MVP chọn thao tác thủ công với người kiểm tra độc lập. Test chứng từ giả không thay thế kiểm chứng ngân hàng.
4. **Notification review:** observation query cũ còn trong lịch sử review kể cả khi sau đó có IPN hợp lệ. Chưa có cơ chế tự đóng hoặc UI vận hành xử lý toàn bộ vòng đời notification.
5. **Tích hợp khác:** inventory/profile/shipping vẫn simulator; chưa nghiệm thu giao tiếp production với các service thật, failover hạ tầng, tải cao hay khóa production. Bộ test logic không thay thế các kiểm tra này.
6. **Phân trang provider:** có kiểm tra metadata và không tiến cursor khi lỗi; chưa chứng minh quét toàn lịch sử merchant lớn trong khi liên tục có đơn mới. Đây là bước nghiệm thu riêng khi tăng quy mô.

## 6. Bằng chứng và cách chạy lại

- [Kết quả đầy đủ, tên test và hash source](mvp1-expanded-verification.json).
- [Các ca trực tiếp với SePay và giới hạn mất IPN](sepay-live-exceptions-verification.json).
- [Gọi lại API độc lập, đối chiếu log IPN #38279](sepay-api-recheck.json): ngày 16/10/2026 khoảng 15:19 Việt Nam, Python và curl cùng nhận HTTP 200 với data.transactions rỗng cho đúng invoice trong log; thử thêm hai đơn đã nhận IPN thành công cũng rỗng. Không phải lỗi deserialize của Go. Chưa xác định nguyên nhân nội bộ SePay, không suy rộng sang production hay khẳng định SePay không lưu transaction.
- [Thử trực tiếp SDK chính thức và hướng API khác](sepay-sdk-alternative-check.json): chạy `sepay-pg-node` 1.0.0 ở sandbox, `order.retrieve(invoice)` trả HTTP 200 và transactions rỗng; `order.all` tìm đúng một đơn, không có trường transaction chi tiết. SDK cài riêng trong thư mục được Git bỏ qua, không thêm dependency vào Order. API giao dịch ngân hàng v2 dùng token Test mode riêng; chưa có token và chưa xác nhận giao dịch giả lập Gateway xuất hiện trong nguồn dữ liệu này.
- [Thử User API với token được cung cấp sau đó](sepay-userapi-sandbox-check.json): Python và curl cùng nhận HTTP 401 unauthorized từ User API Sandbox. Chưa lấy được dữ liệu để kiểm tra giao dịch; cần xác minh token được tạo trong Test mode API Access và còn hoạt động. Không coi HTTP 401 là danh sách giao dịch rỗng; không thử token trên production.
- [Đối chiếu ba trang docs.sepay.vn do người dùng cung cấp](sepay-legacy-userapi-check.json): sau khi được hướng sang endpoint User API v1 trong tài liệu, GET có lọc reference_number tại `my.sepay.vn/userapi/transactions/list` nhận 200, success=true, transactions rỗng. Token được endpoint này chấp nhận; không thể kết luận token sai chỉ từ 401 của Sandbox. Mã transaction Gateway chỉ được thử như một candidate reference_number, chưa chứng minh ánh xạ sang mã ngân hàng; kết quả rỗng có lọc không chứng minh mọi nguồn dữ liệu đều thiếu giao dịch. Không đọc lịch sử tài khoản không lọc, không sửa giao dịch hay logic Order.
- [Thử token Sandbox mới](sepay-userapi-new-sandbox-token-check.json): User API Sandbox nhận token và trả 200. Tìm theo invoice, PAY reference và ngày 16/10/2026 với amount_in=135000 đều không có bản ghi; GET theo UUID transaction trong log Gateway trả 404. Xác thực đã thành công nhưng vẫn chưa xác minh được nguồn khôi phục transaction của Gateway. Chỉ thực hiện GET, không tạo tiền mô phỏng hay gọi production trong lần thử này.
- [10 kiểm tra replay/hủy/hoàn tiền](live-sepay-additional-workflow.json).
- [Lần merchant sandbox thành công đầu tiên](live-sepay-sandbox-verification.json).
- [Kiểm tra OpenAPI và manual event](mvp1-contracts-verification.json).
- [Các test ngoại lệ MVP mới](../../../services/order-service/tests/integration/mvp_exceptions_test.go).
- [Script workflow HTTP](../../../services/order-service/scripts/verify-sepay-sandbox-workflow.py).

Từ thư mục gốc repository, khi PostgreSQL/Redis/Kafka sandbox đã chạy:

```powershell
docker compose -f services/order-service/docker-compose.test.yml run --rm --no-deps tests sh -c 'go vet ./... && go test -race -count=1 ./...'
python services/order-service/scripts/verify-mvp.py
python packages/contracts/verify_order_design.py
```

Script dưới đây **hủy và ghi nhận hoàn tiền nội bộ cho đơn sandbox được chỉ định**; chỉ chọn một đơn fixture PAID chưa packing. Không đưa đơn người dùng thật vào script:

```powershell
python services/order-service/scripts/verify-sepay-sandbox-workflow.py --order-id UUID-CUA-DON-SANDBOX-RIENG
```

Log thô, token demo và các trang snapshot nằm trong `.system_generated/order-sepay/` được Git bỏ qua. Các báo cáo được chia sẻ không chứa merchant secret, bearer token hay body IPN có thông tin người dùng.
