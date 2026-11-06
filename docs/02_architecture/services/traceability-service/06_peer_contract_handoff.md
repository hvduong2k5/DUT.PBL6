# Contract cần xác nhận để triển khai P8

Ngày 2026-10-17. R1 backend sandbox đã có code/tests/runbook; tài liệu này dành
cho bước integration thật, chưa phải thỏa thuận của các owner và chưa gửi team.

**Xác nhận từ người dùng:** nhóm chưa quyết định P8 và chưa chốt contract liên
service. Các phương án dưới đây dùng để thảo luận, không phải yêu cầu đã được
nhóm cam kết thực hiện. Trạng thái P8 hiện là NOT_DECIDED; review R1 có thể tiến
hành độc lập trước khi nhóm chọn phạm vi và owner cho integration.
Bằng chứng code và SHA nằm trong [quyết định](./01_decisions_and_peer_code_review.md)
và [final audit](../../../04_testing/traceability-service/04_final_r1_audit.md).

## 1. Hai đầu vào cần trước tiên

### Inventory — X-01/X-02

Đề nghị cung cấp query theo **Batch UUID ổn định**, trả được SKU, batch code,
ngày sản xuất/hết hạn, trạng thái nguồn và version có thể phát hiện correction.
Cần xác nhận cùng UUID vẫn lookup được sau khi hết tồn, quá hạn hoặc cách ly;
nếu chuyển kho thì UUID là lô gốc hay một bản ghi phân bổ kho. Không coi code
không unique là UUID hoặc suy ra một lô bằng cách lấy phần tử đầu FEFO.

Code đã trả lời: nhánh `feat/inventory-service@2b9f1790` có UUID trong DB nhưng
query FEFO không công khai UUID; batch code chưa có ràng buộc unique phù hợp.
Nhánh `copilot/trin-khai-inventory-service@e23a9199` chỉ có skeleton và tài liệu
stock/reservation, chưa giải quyết điểm này.

Đề nghị QR: Traceability sở hữu canonical opaque token/URL sau binding; Inventory
nhận binding/lookup đã cấp thay cho tự ghép `/batch/{code}`. Owner xác nhận có
URL đã in cần giữ tương thích hay không; không tạo alias nhập nhằng khi chưa biết.

**Đầu vào đủ để code**: ref/provider proto hoặc OpenAPI, endpoint chạy thử, fixture
Batch UUID + SKU/Product tương ứng, raw status/date semantics, retention/correction
policy và owner QR. **Chứng minh đạt**: exact UUID match/mismatch, depleted lookup,
expired/restricted handling, source change chặn publish và QR giữ đúng binding.

### Identity/Gateway — X-04

Chưa thấy implementation Identity riêng trong remote repo hiện tại. Ví dụ token
nội bộ Profile và verifier Content không chứng minh contract xác thực chung.

**Đầu vào đủ để code**: ref/repo Identity, issuer/audience, key discovery/rotation
hoặc cơ chế verify chính thức, claim principal UUID, permission mapping, test actors
và route admin/public Gateway. Có thể gửi public contract/config và lệnh cấp token;
không đưa signing secret vào tài liệu hay source.

Đề nghị giữ permission `traceability.*` của API hiện tại, mapping từ role do owner
Identity xác nhận. Public anonymous, admin phải verify identity; header client
không thể thay actor/role. **Chứng minh đạt**: token thật đủ/thiếu quyền, sai audience,
expired/rotated key, forged header, hai actor duyệt độc lập qua Gateway thật.

## 2. Các bước còn lại theo kế hoạch

| Gate | Phương án Traceability đề xuất | Owner cần trả lời gì để thực hiện | Bằng chứng đóng gate |
| --- | --- | --- | --- |
| X-03 Catalog | Dùng public implementation `hvduong.catalog.v1` và GetProduct chứa Active SKU để verify quan hệ hiện tại | Ref/proto chính thức; cách xác minh SKU/Product không còn public và version lịch sử | Codegen/query thật; positive, mismatch, unavailable và historical behavior |
| X-05 Web | Giữ `/trace/{code}`, public DTO nullable/manual/sandbox và error410 | Có nhận DTO hiện tại và sửa status UI/proxy theo OpenAPI service không? | Browser flow manual/null/404/410 không gắn badge hợp lệ sai |
| X-06 Catalog/Content | Consumer service-local CloudEvents draft, dedup event ID + so version | Topic/type/schema chính thức, owner consumer và projection nào được cập nhật | Broker → consumer → projection; duplicate/out-of-order không làm lùi trạng thái |
| X-09 Audit | Local append-only audit + outbox, không bịa prev/current hash | Provider/ref và schema ingest thực nhận; ai tính chuỗi hash, privacy/outcome/order mapping? | Audit consumer ingest/dedup thật với payload đã thống nhất |

X-07 Fulfillment (packing/seal activation) và X-08 Procurement (receipt/genealogy)
vẫn thuộc backlog đã hoãn, không thêm thành điều kiện R1. EXT-10 chưa tích hợp.

## 3. Cách tiếp tục khi có phản hồi

Ghi owner/ref/ngày và câu trả lời từng X vào file quyết định. Sau đó hiện thực
adapter đúng contract đã xác nhận, tests trên provider thật và ghi endpoint/image/
ref/config không chứa secret trong report. Chỉ mở integration mode khi các dependency
bắt buộc của flow đã đủ; không bật từng phần bằng fallback sang sandbox fixture.

Không cần team duyệt lại SQL nội bộ, revision/CAS/idempotency, contributor policy,
manual certificates hoặc cách tổ chức code đã thuộc quyền quyết định Traceability.
Chưa có quyết định phạm vi và các đầu vào trên thì giữ P8 NOT_DECIDED;
không ghi R2 PASS.

## 4. Kiểm tra lại khi chưa có câu trả lời contract

Remote mới xuất hiện `feat/care-service@94d8731e969cc6a3966df63784b26d4632e5d4a8`.
Đã đọc tree, `services/care-service/internal/transport/httpapi/auth.go` và Inventory
proto ở đúng ref này qua GitHub API. Care dùng `TrustedGatewayAuth`: internal
token cùng `X-User-ID`/`X-Permissions`. Đây là implementation riêng, không chứng
minh Gateway chung đã strip/verify header, có route Traceability hoặc đã thống nhất
permission mapping. Không tự thay verifier Traceability theo ví dụ này.

Inventory proto trong nhánh Care vẫn dùng `GetBatchFEFODetails` theo SKU/warehouse,
trả `batch_code`, chưa công khai Batch UUID lookup. Không có source implementation
Identity/Audit trong tree đọc ở ref này. Các refs Catalog/Inventory/Web/Content/
Profile/Order đã đọc trước vẫn giữ nguyên SHA. Vì vậy X-01/X-04 chưa được giải đáp;
X-03/X-05/X-06/X-09 cũng chưa có xác nhận contract hoặc roundtrip mới.

Backend R1 giữ nguyên kết quả review readiness. Mục tiêu tổng thể chưa hoàn tất:
P8 đang chờ input từ owner/user; cần ref/provider contract hoặc quyết định phối hợp
như mục 1–2 để tiếp tục, không thêm fixture hay adapter giả để đóng gate.
