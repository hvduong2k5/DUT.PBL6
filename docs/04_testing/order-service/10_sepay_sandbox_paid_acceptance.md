# Nghiệm thu SePay Sandbox → Order PAID

Ngày25/10/2026. Tiếp tục HEAD `6e03b58` với local fix date filters và verifier; [ADR011](../../02_architecture/adr/order-service/adr_011_sepay_local_date_filters_and_sandbox_paid_acceptance.md). [Evidence JSON](10_sepay_sandbox_paid_acceptance.results.json).

## Kết quả chính

**PASS luồng public checkout → giao dịch tạo trên Dashboard SePay Test mode → webhook → canonical SePay API → allocation → Inventory finalize → PAID/READY.**

| Evidence | Giá trị |
| --- | --- |
| Order | `9b5f62f2-ebf1-4326-874a-3b9b316e7a23` |
| Payment code | `OMP7WQ426GMN7Z4` |
| Expected / allocated |250000 VND /250000 VND |
| Canonical SePay transaction UUID | `27c6ddaa-d04e-11f1-8f53-a6006ab65aca` |
| Expiry gốc |25/10/2026 15:40:39 giờ Việt Nam |
| Xác nhận PAID/READY bằng API + SQL |25/10/2026 15:33:03 giờ Việt Nam |
| Order / stock / ready |PAID / COMMITTED / true |
| Allocation / FIRST_PAID |1 /1 |
| Paid / ready events |1 /1, pending outbox0 |
| Webhook → canonical receipt |1 linked observation |

Token, account number/holder, QR URL và raw signed webhook không đưa vào report/repository.

## Luồng đã thực hiện

1. Build lại backend ở source hiện tại, khởi động stack và tunnel với cấu hình/key/UUID/volume cũ. Không reset dữ liệu hoặc financial identity.
2. Tạo Customer TEST qua simulator, sau đó gọi **public Cart/Quote/Checkout/CheckoutOperation/payment-instructions**. Order/Intent được worker tạo thật; không seed financial rows.
3. User đăng nhập vào browser automation session. Xác nhận Dashboard URL dưới `/testmode/`, checkbox Test mode bật, tài khoản chọn khớp trusted receiver. Nhập Tiền vào250000 VND và đúng OMP; bấm Gửi giao dịch thử một lần.
4. Dashboard nhận diện code và báo webhook thành công. Backend chuyển WAITING_PAYMENT → VERIFYING. Chưa coi webhook ACK là PAID.
5. Phát hiện UTC date-filter mismatch trên API thật. Ba GET probes cùng window trả0/1/1 theo format; sửa adapter theo ADR011, chạy client race test rồi redeploy API.
6. Scanner tự tìm canonical receipt, allocation và finalize reservation qua Inventory TEST gRPC. Verifier đọc owner APIs, canonical API và SQL để xác nhận PAID/READY và các counts trên. Không thay amount/expiry/bank time hoặc gọi mark-paid.

## Lỗi tìm thấy nhờ provider acceptance

Client trước gửi UTC RFC3339 vào date filters. Sandbox so sánh các filter theo wall time, khiến scanner có thể không thấy giao dịch mới trong cửa sổ mong muốn dù API trả200/empty-page hợp lệ. Sau chuyển format query sang giờ Việt Nam, cùng giao dịch được phát hiện và xử lý trước hạn gốc. Chi tiết raw probe counts ở evidence JSON; unit test mới kiểm local-date conversion qua calendar boundary.

Lượt verifier đầu timeout khi Order còn chờ API reconciliation; giữ failure log riêng. Lượt sau fix PASS. Không thay kết quả failure bằng kết quả rerun.

## Kiểm chứng và phạm vi

Verifier `scripts/verify-sepay-sandbox-payment.mjs` chạy PASS, đối chiếu cùng canonical UUID giữa SePay API và receipt của allocation. SePay client race tests chạy PASS. Kết quả full regression sau fix được ghi riêng trong evidence JSON; không dùng acceptance Payment này để gọi toàn bộ test suite xanh.

Đợt sau fix chạy `go vet ./... && go test -race -json -count=1 ./...`: **213 test/subtest PASS,0 FAIL,0 SKIP**, không ghi nhận data race. Actual connected responses13/events5 và Payment responses5/events2 đều khớp pinned schemas. Postman owner payment GET có2 assertions PASS (HTTP200 và PAID/exact allocation), cloud event upload tắt.

Ca Kafka issue33 PASS trong đợt full này; không có thay đổi consumer nhằm sửa issue33 trong lượt Payment, nên không kết luận đã tìm ra nguyên nhân intermittent failure trước đó. Các log vận hành vẫn có retry của manual-refund sweep; manual-refund authority/ops không được nghiệm thu bởi kết quả thanh toán này.

Source fix/verifier/ADR/report hiện ở working tree để review, chưa commit/push lên PR32. Image API đã được build và chạy từ source có fix; baseline Git HEAD vẫn6e03b58.

**SePay và webhook ingress trong lượt này là Sandbox thật; Inventory/Catalog/Profile/Shipping/Care vẫn là owning-service TEST facades.** Giao dịch Sandbox không phải tiền Live. Lượt này nghiệm thu đến PAID/READY, chưa gửi source packing/shipping mới để gọi toàn bộ lifecycle COMPLETED trên giao dịch này; lifecycle stand-in có evidence riêng trong báo cáo09.

Các gate còn mở: real owning-service/Identity/Finance acceptance, full paid-cancel remote-stop/committed-stock reversal, restore/production security/ops và regression Kafka issue33. Gate canonical OMP Sandbox → PAID trong báo cáo09/ADR010 đã được evidence mới này đáp ứng.

## Tái hiện

Theo [CONNECTED_TEST.md](../../../services/order-service/CONNECTED_TEST.md): tạo một checkout/Intent mới, tạo đúng giao dịch trên Dashboard Test mode, rồi chạy verifier với private-directory và report-file. Mỗi Intent có hạn riêng; không reuse một mã mẫu/đơn đã hết hạn để nghiệm thu.

Logs local: `%TEMP%\order-sepay-acceptance-20261025` (`checkout.json`, `date-filter-probe.json`, `watch.log`, `paid-verification.json`, `watch-fixed.log`, `paid-verification-fixed.json`, `client-test.log`, `full-race.jsonl`). Các file báo cáo commit được sanitize; browser profile/session và private instructions vẫn nằm ngoài Git.
