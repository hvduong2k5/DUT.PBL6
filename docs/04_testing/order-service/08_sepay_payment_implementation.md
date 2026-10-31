# Triển khai Payment SePay/VietQR — 24/10/2026

Đã có code cho TEST runtime theo thiết kế09 và resolution10. **Chưa nghiệm thu luồng public checkout → giao dịch SePay chứa OMP hợp lệ → PAID trên môi trường deploy.** [ADR009](../../02_architecture/adr/order-service/adr_009_sepay_payment_runtime_and_capability_gates.md) ghi rõ những quyết định/gap khi coding; [runbook](../../../services/order-service/SEPAY_TEST.md) ghi cấu hình và cách chạy.

## Những phần đã triển khai

| Phần | Hành vi đã có |
| --- | --- |
| Protocol/QR | Raw-body HMAC/timestamp/current+previous key; exact int64, duplicate JSON rejection; whole-token OMP; trusted VietQR URL và receiver config |
| Webhook/P1 | Observation mã hóa + durable SQL work trước ACK; duplicate/conflict/wrong receiver/outgoing; không tạo receipt/PAID |
| Canonical API/P2 | API v2 Sandbox UUID; financial facts immutable; encrypted evidence; dedup, receipt/case, exact allocation; không cộng partial hoặc reopen đơn |
| Sync/recovery | Four account streams, fixed windows, page dispositions, fenced lease/checkpoint restart, shared2req/s quota; independent sync/finalize loops |
| Finalization/P3 | Inventory3.2 authenticated query/command proof; stable operation key; stock COMMITTED mới PAID/milestone/outbox; release/expired và refund race giữ reconciliation |
| Owner/Guest reads | Actual schema-aligned status/instructions; DB-local polling; health/expiry guard; bounded verification watch; ES256 order-bound grants, SQL sessions/revoke/version/current owner |
| Staff controls | Permissions + receiver scope, Care canonical approval, reserved refund budget, claim/start/unknown, independent verification/MFA, evidence/vault ACL, manual case queue/assignment/resolve |
| Persistence/wiring | Migration005 và legacy compatibility; opt-in startup; Kong customer/staff/webhook routes; public internal endpoint không expose; production startup/auto-refund vẫn tắt |

## Kiểm chứng đã chạy

Project Docker riêng `codex-order-payment-20261024`, PostgreSQL16/Redis7/Kafka thật, Go1.22.12 Linux amd64 với race detector. Từng test tự dùng schema riêng; HTTP/gRPC owning services là stand-ins có authentication và failure cases. Không thực hiện bank mutation/chuyển tiền.

| Check | Kết quả |
| --- | --- |
| `go vet ./...` | PASS |
| `go test -race -json -count=1 ./...` | **188 test/subtest PASS**, 0 fail, 0 test skip; trong đó **41 integration tests cũ** PASS |
| Bổ sung `TestRefundReservationDuringInventoryCommitBlocksPaid` | PASS với real PostgreSQL + race detector; refund được đặt trong lúc Inventory commit, không phát FIRST_PAID; case và parked job giữ evidence |
| Actual Payment JSON so với unchanged candidate OpenAPI | **5 examples PASS**: PaymentStatus, PaymentInstructions, Refund, StartInstruction, Case |
| Actual receipt/paid outbox payload so với candidate financial schema và pinned Order v3 schema | **2 payloads PASS** |
| `python packages/contracts/verify_order_design.py` | PASS3 existing specs,8 v3 event schemas,9 examples,32 negative examples; protobuf compile không chạy ở lệnh này |
| SePay/Cloudflare live delivery, OMP recognition, user-facing target checkout | **NOT_RUN trong lượt coding này**; giữ evidence/capture cũ độc lập |
| Postman regression và deployed Kong probe | **NOT_RUN lại trong lượt này**; không cộng kết quả baseline cũ thành test cho code mới |

Protocol HTTP ingress test chạy qua socket thật:401 unsigned,415 media,413 oversize,400 invalid JSON, valid/replay ACK, outgoing/wrong receiver durable cases và previous-HMAC overlap. Nó xác nhận4 observations,0 receipts,2 review cases. Inventory gRPC test kiểm legacy/wrong hash/rejected COMMITTED và ACK lost recovery cùng key, không lặp mutation.

Manual refund tests kiểm approval replay/body conflict, concurrent budget trên receipt unmatched, approver/self verification/MFA denial, stable attempt replay, UNKNOWN không retry/không nhả budget, batch evidence atomicity, SENT/NOT_SENT/INCONCLUSIVE, summary/outbox once, allowed-host/scan/current-scope checks. Case tests kiểm assignment, cross-actor cursor và không resolve REFUNDED khi chưa có settlement. Không gọi các test này là nghiệm thu human workflow hoặc đầy đủ toàn35 SP scenarios.

## Lỗi phát hiện và sửa trong quá trình kiểm thử

- Migration005 thay unique receipt/case identity làm một legacy `ON CONFLICT(receipt_id)` không còn hợp lệ: sửa writer và chạy lại41 legacy tests đạt.
- SQL `jsonb` normalize key order khiến replay vault ticket so sánh bytes báo conflict: đổi sang so sánh metadata ID/expiry có kiểu; replay test đạt.
- Concurrent first receipt có thể pin Order khác giữa pre-read và financial identity lock: recheck original association và retry; không sửa root chưa khóa.
- Docker clock có lần lùi khoảng2 giây: diagnostic ghi `accepted=07:37:31.575953`, canonical `07:37:31.596446`, DB đọc sau đó `07:37:29.484201`. Domain từ chối snapshot/deadline đúng guard. Giữ nguyên guard, cố định DB/test CPU0 của **project test riêng** và chạy lại toàn suite PASS. Đây là workaround môi trường test, chưa là chứng minh production timekeeping/restore-safe behavior. Không giấu các lần fail bằng cách bỏ guard.

Machine summary và source hashes ở [08_sepay_payment_implementation.results.json](08_sepay_payment_implementation.results.json). Raw logs chỉ có dữ liệu synthetic và để trong `%TEMP%/order-payment-implementation-20261024`; không chép token/real bank account vào báo cáo.

## Phần còn phải nối trước nghiệm thu end-to-end

1. Target checkout/placement/timeout/control executors tạo Order+Intent từ verified dependencies; hiện legacy public checkout không tạo target Order. Khi bật SePay, new legacy VIETQR quote/accept trả503 `TARGET_CHECKOUT_NOT_READY` để tránh nhận tiền vào flow chưa nối.
2. Inventory owner hỗ trợ và nghiệm thu command evidence/hash/query3.2; adapter fail closed với legacy simulator. Không dùng receipt để sinh COMMITTED proof.
3. Generic target outbox publisher/consumer Finance acceptance; outbox persisted chưa đồng nghĩa sự kiện được Kafka/Finance xử lý.
4. Owning Care/Finance/evidence và Identity production authority, UI/manual operational drill, provider account coverage/rotation/restore acceptance.
5. Sau khi runtime deploy/bridge đúng và gates đạt, cấu hình endpoint nhận webhook rồi thực hiện giao dịch Test mode chứa đúng OMP từ Intent thật. URL Cloudflare hiện tại vẫn là capture, chưa nối DB/worker của runtime này.

Do đó **code Payment đã được triển khai và kiểm thử ở phạm vi nêu trên; toàn luồng deploy vẫn còn dependency gates**, chưa bật live hoặc automatic refund.
