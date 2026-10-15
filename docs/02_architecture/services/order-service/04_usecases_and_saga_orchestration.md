# 04 — Use cases và saga, baseline 3.2

[Trước](03_database_and_persistence.md) · [Tiếp](05_api_contracts_and_transports.md)

## 1. Cart → quote → accepted operation

Cart không reserve. Mutate revision atomic; canonicalize duplicate SKU, limits trước external calls. Quote server-issued bind owner/cart/address/method; giá hiển thị chỉ preview. Saved address lookup đúng customer/version, Guest validation cùng master data; không trust arbitrary customer_id.

POST checkout kiểm auth/quote/key, T0 durable rồi trả 202 operation. New acceptance chưa gọi stock. Worker revalidate canonical terms; preview hết hạn sau acceptance không tự làm mất operation, nhưng Catalog/Shipping terms phải được validate lại. Giá/phí khác khách đã đồng ý → PRICE_CHANGED, không tự charge.

## 2. Placement trả trước

Stage: VALIDATE → VOUCHER_HOLD nếu bật → RESERVE → PLACE → SUCCEEDED.
- VALIDATE: Catalog coherent snapshot và Profile address độc lập có thể parallel; Shipping sau weight/address; voucher sau totals. Lưu snapshot/deadline/intent trước acquisition.
- Voucher hold/query dùng stable key, expiry remote; feature tắt trả VOUCHER_FEATURE_DISABLED.
- Reserve: stable key checkout:{operation}:reserve, planned_order_id, items digest, warehouse policy. Transport timeout → WAITING_RETRY và query/retry cùng key.
- Response phải đúng order/resource/digest/warehouse và current RESERVED, có expiry hợp lệ. Terminal expired/released/reversed không được coi success.
- PLACE: còn >30s trong min expiries; T3 tạo Order/items/history/outbox + operation SUCCEEDED. Replay đọc kết quả cũ.
- Lỗi sau acquisition → COMPENSATE ngược thứ tự, kể cả tài nguyên có thể đã giữ do lost ACK. Chỉ terminal RELEASED/EXPIRED hoặc verified cancellation tombstone mới hoàn tất. RESERVED/UNKNOWN/rỗng/mismatch giữ obligation.

Không đóng operation FAILED khi compensation còn UNKNOWN. Planned UUID của operation cancelled không dùng reserve lần nữa; checkout mới dùng operation mới.

## 3. Receipt và finalization

Verify raw callback theo provider contract; receiver/currency/reference/status/signature/replay policy. VietQR image chỉ là hướng dẫn chuyển khoản; contract callback phải đến từ provider thực đã chọn, không suy diễn mọi provider dùng HMAC-SHA256.

Ghi receipt ngay cả unmatched/late. Duplicate provider transaction cùng normalized payload no-op, altered payload conflict/security review. Exact, pending, còn hạn dưới Order lock → allocation + PAYMENT_FINALIZING + FINALIZE job.

Worker query/finalize Inventory bằng stable key; remote COMMITTED đúng identity/digest mới tiếp tục. Nếu voucher bật, finalize voucher kế tiếp. Cả hai thành công → PAID/history/outbox + ready intent atomic. Voucher failure sau stock commit giữ hold và mở cancel/reversal/refund review; không dùng release ACTIVE hoặc tự PAID.

| Receipt | MVP |
| --- | --- |
| Exact và còn hạn | allocate, finalize resources |
| Under/over/multiple partial | giữ receipt/reconciliation; không tự cộng/allocate |
| Late/cancelled/unmatched | lưu tiền, review/refund theo policy; không hồi sinh |
| Exact nhưng stock expired | giữ receipt/allocation trace, hold/reconciliation; không ready |
| Extra receipt sau PAID | case riêng, không xóa allocation hợp lệ/paid lần nữa |

Statement reconciliation độc lập callback; cursor chỉ advance sau durable outcome từng record, có overlap/dedup theo provider semantics. Không chỉ scan active Orders vì tiền có thể tới đơn đã cancelled.

## 4. COD

Eligibility kiểm channel/address/carrier/value/risk. Không QR/không chờ customer payment 15 phút. Reserve → finalize stock (voucher nếu bật) → PLACE CONFIRMED_COD/UNPAID → ready. Lease acquisition ngắn; expiry remote vẫn phải tôn trọng.

Finalize success mất ACK: query và tiếp tục place. Nếu đã COMMITTED nhưng local place không thể hoàn tất vì business terms invalid, giữ MANUAL_REVIEW/reversal intent; không release như hold. COD cũng phải có canonical terms hợp lệ tại confirmation; không tự bỏ qua Catalog/Shipping validity vì không chờ thanh toán.

COD settlement chỉ từ evidence trusted của carrier/provider; delivery không tự bằng collected/settled. Nếu receipt đến khi cancel pending, giữ reconciliation/financial hold theo guard, không tự đảo cancellation.

## 5. Cancel, ready và hold

Pending self-cancel/timeout: lock Order cạnh tranh receipt; transaction cancelled + release job; response compensation PENDING, UI không nói kho đã hoàn ngay. ReleaseByOrder tạo tombstone nếu chưa thấy reserve.

Paid/COD cancel: lưu hold/cancel intent; Fulfillment CancelTask(order,generation,key) serialize với AcceptTask. CANCELLED → reverse committed inventory sau kiểm approval/policy; target REVERSED; mới commit terminal Order cancel. ACCEPTED → reject self-cancel và chuyển Care theo policy; không restart task từ event khác. UNKNOWN → query barrier.

Ready flow: Order đủ guards phát ready fact với generation. Fulfillment nhận và query ready permit Order; permit chỉ là quyền thử accept, không thắng được cancellation tombstone. AcceptTask commit tại Fulfillment là điểm tuyến tính hóa với CancelTask. Mất ACK query task. Hard hold cũng cần remote barrier; soft local hold không được quảng cáo là đã dừng task remote. Resume generation mới và xác nhận trạng thái task cũ trước phát ready.

Giữa barrier cancel và reversal nếu crash, job giữ evidence và resume key cũ; không được tạo ready mới trong lúc cancel/reversal pending. Refund là nghĩa vụ độc lập; money refund không tự restock.

## 6. Fulfillment / Shipping / completion

Source event có publisher identity, resource association, sequence/version; duplicate event ID khác nhưng same checkpoint không tạo effect mới. Thiếu prerequisite → DEFERRED; query source, không bỏ mất hoặc quay state ngược.

Packing accepted → PROCESSING; sealed → PACKED; shipment created chỉ reference; dispatched → SHIPPED; delivered/failure từ checkpoint; retry attempt mới có ID mới.

Care completion decision serialize case-open với completion; có decision ID/version/query. Order chỉ completed sau settlement policy, elapsed window, không refund obligation còn UNKNOWN và Care cho phép. Loyalty consumer dùng completion/reversal facts idempotent; không hard-code earn 1% trong Order.

## 7. Refund và mở rộng

Care approval bind receipt/order/scope/amount/currency/policy/actor/version, không tin chuỗi approval_id trống nghĩa. Payment component reserve refundable budget dưới lock trước provider call. UNKNOWN giữ budget; query trước retry, key không đổi. FAILED terminal verified mới giải phóng obligation theo policy. Partial/full refund không tự đổi lifecycle hoặc stock.

Guest claim cần verified proof bind order/customer và anti-replay; Order apply ack/query để Profile biết ownership đã đổi. Marketplace unique(channel,store,external_id), payment/fulfillment guarantee do Channel normalize, không mặc nhiên PAID. POS trusted sale/terminal/cashier, offline retry cùng sale ID; không bán bỏ stock check khi mất mạng. B2B deposit/credit và gifts multi-address chưa MVP; phải có ADR mở rộng.

## 8. Retry và ngân sách

Target retry full jitter trong [0,min(30s,0.5s*2^attempt)], tối đa 10 automatic attempts hoặc 15 phút mỗi stage rồi MANUAL_REVIEW; không xóa nghĩa vụ. Finalize ưu tiên trong remote expiry; sau deadline query outcome, không blind mutate. 429/Unavailable/DeadlineExceeded retry cùng key; validation/conflict definitive không retry để “vượt lỗi”. Breaker không tính business rejection.

HTTP acceptance deadline3s, external attempt2s, worker lease10s là default versioned, cần đo. Runtime3.1 dùng backoff khác, ghi gap R05. Receipt accepted ACK nghĩa đã persist receipt, không PAID. UI phân biệt đang xử lý, đã nhận tiền/đang chốt đơn, cần đối soát và compensation pending.
