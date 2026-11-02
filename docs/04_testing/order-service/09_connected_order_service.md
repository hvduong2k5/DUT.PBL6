# Order Service — kiểm chứng luồng đã nối

Ngày24/10/2026. Code tiếp tục từ `1fc5185` trên `feat/order-service`; thay đổi mới còn ở working tree, chưa commit/push. [ADR010](../../02_architecture/adr/order-service/adr_010_connected_checkout_lifecycle_and_test_bridge.md) ghi các gap/quyết định. [Kết quả máy đọc](09_connected_order_service.results.json), [runbook](../../../services/order-service/CONNECTED_TEST.md).

## Kết quả

**Đã nối luồng chính trong TEST:** public Cart → Quote → CheckoutOperation → Order/Snapshot/Intent → SePay reconciliation → Inventory finalize → PAID → READY → packing/shipping → DELIVERED → Care barrier → COMPLETED. Full flow sử dụng PostgreSQL16/Redis7/Kafka thật, HTTP/gRPC tới owning-service stand-ins. Không seed Order hoặc gọi API mark-paid để vượt qua placement/payment.

| Lớp kiểm chứng | Kết quả và giới hạn |
| --- | --- |
| Go vet + race regression | **205 test/subtest PASS, 0 FAIL, 0 SKIP**; gồm9 connected integration và41 integration legacy/Payment cùng các test domain/persistence/adapter/transport |
| Actual connected HTTP JSON |13 examples khớp Quote/CheckoutOperation/OrderDetail candidate, không sửa schema để pass |
| Actual connected outbox JSON |5 events: placed, receipt_recorded, paid, fulfillment_ready, completed; đúng pinned schema |
| Payment contracts |5 response examples và2 financial events khớp schema |
| Static contracts |3 baseline OpenAPI +4 candidate OpenAPI hợp lệ;8 event schemas,9 positive/32 negative; shared protobuf compile12 files gồm imports |
| Cloudflare Worker |6 tests PASS: HMAC/encrypted capture, failure handling, nguyên signed bytes, capture trước forward, ID0 riêng, runtime failure/redirect không ACK thành công |
| SePay API Sandbox thật |Go runtime client đã đọc/validate2 canonical rows bằng readonly probe. Runtime sau fix zero-page lên HEALTHY cả FAST/OVERLAP/DAILY/BACKFILL |
| Hosted webhook → runtime |URL hiện hữu → Worker capture → tunnel riêng → backend SQL observation thành công. Signed synthetic probe tạo1 observation, Order PENDING_PAYMENT/VERIFYING, allocation0; không falsely PAID |
| Canonical OMP từ Dashboard |**CHƯA NGHIỆM THU**: cần tạo giao dịch Test mode bằng OMP/amount của Intent do public checkout tạo; không coi synthetic bridge probe hoặc API read-only là PAID acceptance |

## Các ca xuyên luồng đã chạy

1. VIETQR: acceptance/replay giữ một operation/order; webhook đơn lẻ không booking, canonical UUID mới allocation; Inventory finalize ACK lost được query recovery; FIRST_PAID/outbox một lần, physical stock98 từ100. Consumer đọc placed/paid/ready từ Kafka thật. Packing seal đến trước acceptance được DEFERRED/replay; shipment delivered đi qua Kafka consumer; Care completion phát một milestone/event.
2. COD: commit stock trước PLACE, READY sau guards, không tạo Intent/allocation.
3. Customer self-cancel: đóng Intent, durable stock release, không reopen/PAID khi canonical tiền đến muộn; receipt/case vẫn được ghi.
4. Guest: ES256 order grant, không đọc payment bằng generic Guest bearer; logout/revoke làm grant và protected session hết hiệu lực. Thiếu signing key thì không nhận checkout VIETQR Guest mới.
5. Admission hold/resume: provider no-task barrier fence generation cũ; generation mới được admit. Đã nhận task thì hold admission-only trả409, chưa giả lập remote-stop proof.
6. Unpaid timeout: hạn thực khoảng40s từ trusted TEST terms; sweep hủy và release. Canonical bank time trước deadline nhưng ingestion sau timeout không booking/reopen. Không chỉnh expiry để đạt test.
7. Terms thay đổi: revalidation từ chối trước reserve; submission guard được release, fresh quote cùng selection có thể retry, key cũ vẫn replay kết quả thất bại.
8. Cart sửa sau acceptance: cleanup SKIPPED_CHANGED, giỏ mới còn nguyên; money của Intent không đổi theo cart edit.
9. Source unavailable →20 attempts → PARKED/MANUAL_REVIEW; Customer recover403, Admin scoped recover dùng lại remote identity, không tạo thêm reservation/order.

Domain/persistence bổ sung kiểm forged fulfillment state/restore thiếu proof, completion chronology, stream broker-ACK unknown giữ bytes/sequence và clock-regression không làm mất operation hoặc ghi Order.

Completion test rút ngắn **riêng operational completion_due_at** sau DELIVERED để kiểm barrier trong thời gian hợp lý. Immutable delivered_at/payment deadline không bị backdate; đây không phải bằng chứng đã chờ7 ngày thực tế.

## Gap phát hiện và xử lý

- Checkout claim SQL có cột lease/attempts ambiguous và thiếu Scan destination: đã sửa, chạy qua public acceptance.
- PLACE/finalize/recovery từng dùng Reserve làm lookup: tách Lookup query-only, không recreate remote reservation khi recover/cancel.
- API payment instructions từng có thể được tạo từ Guest thiếu signing authority: gate trước nhận đơn mới.
- Hydration trước đây chưa đủ dữ kiện phục hồi lifecycle: lưu proof chain vào encrypted state và immutable lifecycle/source facts ở migration006.
- API SePay thật trả empty page `last_page=0`; fixture cũ chỉ có `last_page=1`. Client/fixture đã sửa exact shape và thêm4 positive/negative cases; runtime scanner thật được recheck HEALTHY.
- Clock Docker/WSL có lúc lùi khoảng1–2s. Guard tài chính vẫn giữ. Worker retry khi DB decision time trước accepted/canonical evidence; không sửa timestamp nguồn hoặc force compensation vì lỗi clock.

Có các lượt chưa đạt trước final: một lượt Payment deadline guard, một lượt thiếu Kafka topic trước start consumer, và một lượt test bị gián đoạn khi đổi cấu hình/recreate hạ tầng. Những lượt này không tính PASS; raw logs giữ riêng. Topic hiện được tạo trước consumer; đợt final dùng hạ tầng ổn định, `--no-deps`, CPU0, đạt205/205. Không xóa failure log để thay bằng summary thành công.

## Runtime đang chạy để review

Project Docker `codex-order-connected-20261024` được giữ chạy với API/simulator/DB/Redis/Kafka và tunnel. Image app `omamx/order-service:connected-20261024`; private settings/keys ở `%LOCALAPPDATA%\PBL\sepay-test`, không trong Git. Máy dev/Docker phải bật; chưa phải deploy24/7.

Giữ URL SePay: `https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook`.

Worker `order-sepay-sandbox` giữ D1/secrets cũ, `FORWARD_ENABLED=true`; origin `sepay-origin-test.npanhbao.dev` trỏ tunnel `99dfc3ac-444a-4542-b5c5-49fe8e2ee5ce`. Chỉ exact webhook path được route. Đã probe unsigned webhook401, các đường `/internal/events`, `/readyz`, `/api/v1/cart`404 bằng HTTP client thực. Tunnel c2c không bị chỉnh. Bản Worker trước deploy được backup ngoài Git; D1/ledger không bị xóa.

Mức deployed evidence mới chỉ durable observation + account scanner hoạt động. Full PAID/COMPLETED acceptance ở integration harness dùng SePay API stand-in và owning-service facades. Trước khi gọi production hoàn thành còn cần service-owner acceptance, Identity thật, full cancellation remote-stop/reversal, Finance consumer, advanced control/read contracts, restore/security/ops drill. Postman suite mới và deployed Kong probe chưa chạy trong lượt này.

## Tái hiện

Theo [CONNECTED_TEST.md](../../../services/order-service/CONNECTED_TEST.md). Raw JSON test logs nằm tại `%TEMP%\order-connected-20261024`; file final `acceptance-complete.jsonl`. Kết quả máy đọc lưu count, SHA-256 log/source LF, scope, deployed evidence và not-run riêng. Git recovery trước lượt này đã có backup riêng; không reset code local hoặc force-push thêm trong lượt tiếp tục.
