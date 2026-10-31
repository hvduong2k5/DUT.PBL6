# M1 — Phân tích khoảng trống contract MVP trước khi đổi Mobile

Ngày: 2026-10-31. Trạng thái: ANALYSIS COMPLETE / BACKEND DECISIONS OPEN. Chỉ thực hiện bước1 trong lượt này.

## Mục tiêu / lý do

Xác định đủ bằng chứng để chọn luồng Mobile phù hợp contract mới, tránh sửa checkout dựa trên giả định. Ba đầu ra: phạm vi mua một phần giỏ, nguồn mã tỉnh/phường và hợp đồng lịch sử/theo dõi đơn. Phân biệt kết luận từ schema, đề xuất UI và quyết định cần backend xác nhận.

## Phụ thuộc / nguồn chuẩn

- Mobile feature/mobile-ui, HEAD trước bước này: 3f206027dc8de3b992bfc327aeb9dcb27db9a86f; ban đầu sạch.
- Dev chỉ đọc: 7f22226c7b2f4363d3ba376b715feb0e63f69355, đã fetch theo quyền người dùng. Không checkout/merge/cherry-pick.
- Target: docs/03_api_specs/mvp-customer.openapi.yaml; packages/contracts/mvp/README.md, MIGRATION.md, registry.yaml, TEAM-DECISIONS.md ở commit trên.
- ZIP hiện tại là baseline lịch sử; giữ nguyên nội dung/hash. Provider-required/adapter-required/pending không phải runtime-ready.

## Trong / ngoài phạm vi

Trong: đọc nguồn public và provider liên quan qua git show, đối chiếu đúng caller/model Mobile; lập bảng thiếu/đủ và đề xuất quyết định, cập nhật master/handoff.

Ngoài: sửa Flutter/API/mock/tests, chạy app/scrcpy, nâng dependency, sửa schema/backend/dev/Web, mở feature gate, liên hệ team hoặc gửi issue/PR. Không tự biến đề xuất thành quyết định nghiệp vụ đã duyệt.

## Các bước phân tích

- [x] Đối chiếu cart revision, quote input và phạm vi items được đặt; xác định cách giữ selection hiện tại.
- [x] Đối chiếu address code/name/version và API dữ liệu địa giới; phân biệt provider/internal với public route.
- [x] Đối chiếu order list/cursor/filter/search và tracking DTO; xác định dữ liệu hiện UI đang cần.
- [x] Ghi bằng chứng có đường dẫn + vị trí ở Git commit cố định; đề xuất an toàn, câu hỏi backend, điều kiện mở M2.
- [x] Kiểm diff chỉ tài liệu, cập nhật điểm tiếp tục và chuẩn bị checkpoint Mobile; đối chiếu HEAD/origin khi bàn giao.

## Các vùng Mobile cần đọc

features/cart/domain/cart_models.dart, data/rest_cart_repository.dart, presentation/cart_cubit.dart; features/checkout/data/rest_checkout_repository.dart, domain/checkout_location.dart; features/account/domain/address.dart, data/rest_address_repository.dart; features/orders/data/rest_orders_repository.dart, presentation/orders_cubit.dart, domain/order_models.dart.

## UX / API / bảo mật cần giữ

Không âm thầm xóa dòng chưa chọn để giả lập partial checkout. Giá/phí/expiry đến từ quote; code địa giới đến từ nguồn chuẩn, không suy từ tên. List/tracking thiếu dữ liệu phải có unavailable/retry, không vẽ mốc giao hàng giả. Ownership do server kiểm; không dùng header tự khai hay endpoint nội bộ để né public contract.

## Kiểm chứng / Definition of Done

Đọc và phân tích YAML cục bộ từ Git object, không chạy script tải từ dev. Kiểm tra các request/response/parameter cụ thể và caller Mobile. Mỗi nhóm có kết luận, rủi ro và tiêu chí đủ để triển khai; những điểm thiếu cần team chốt được ghi OPEN, không tuyên bố đã chốt backend. git diff --check và kiểm chỉ tài liệu thay đổi; không cần chạy Flutter test/build cho tài liệu. Hoàn thành phân tích M1 không đồng nghĩa đã tháo hết blocker hoặc cho phép làm M2.

## Kết quả / quyết định / blocker

### G1 — Mua một phần giỏ

**Đã xác minh:** public target dòng129 POST quote chỉ nhận cart_revision/payment_method và shipping_address_id hoặc address, voucher_code; additionalProperties=false. Không có selected_item_ids, items hay checkout scope. OrderCart dòng2550 có revision/items; OrderQuote dòng2564 trả items và cart_revision. Provider order-service.openapi.yaml dòng120 cũng không định nghĩa partial selection. Đây là thiếu contract cho partial checkout, không phải bằng chứng runtime đã xử lý toàn bộ giỏ đúng.

**Mobile hiện tại:** cart_cubit.dart:29–34 dựng selectedCart từ selectedIds; RestCheckoutRepository.submit gửi danh sách items đã chọn trực tiếp. Cart model chưa có revision; adapter hiện đọc Money object và dữ liệu in_stock/variant_name/thumbnail mà OrderItem mới không đảm bảo. Vì vậy chỉ đổi màu/nút hoặc thêm quote_id sẽ không tương thích.

**Đề xuất (chưa được duyệt làm policy):** MVP dùng toàn bộ giỏ đã được server báo giá; CTA ghi “Xem báo giá cho giỏ hàng”, tóm tắt đúng items từ quote. Nếu muốn giữ mua một phần, cần Order/BFF bổ sung contract phạm vi lựa chọn, revision binding, quote items, thay đổi giỏ và hành vi sau thành công trước khi bật checkbox trong cohort MVP. Giữ luồng/mock cũ tách biệt cho tới khi migrate có chủ ý.

**Không làm:** xóa tạm dòng bỏ chọn rồi thêm lại, tạo giỏ phụ không có contract, gửi thêm selected IDs bất chấp additionalProperties=false, hoặc báo thành công rồi tự clear giỏ theo logic cũ. Sau operation thành công phải đọc lại giỏ server; việc tiêu thụ giữ/xóa dòng cần provider xác nhận.

**Cần Order/BFF chốt G1:** vòng đầu full-cart hay partial; nếu full-cart, xác nhận quote bao phủ đúng revision và semantics tiêu thụ giỏ sau placement; nếu partial, cung cấp schema + conflict/retry và tests cho phạm vi. Kiểm thử đầu ra: đổi giỏ trên máy khác gây conflict/requote, quote hiển thị đúng hàng, retry không mua trùng, dòng còn lại không bị mất. **Gate M2:** có thể thiết kế typed Cart/revision/quote; không thay hành vi selection thật trước quyết định G1.

### G2 — Nguồn dữ liệu tỉnh/phường và địa chỉ có phiên bản

**Đã xác minh:** public target không có /locations/checkout hoặc public province/ward lookup. ProfileAddressWrite dòng3186 nhận ward_code/province_code; tên gửi lên bị server bỏ qua để dùng canonical master data. ShippingAddress dòng3276 có code/name/version. Update/delete/default bắt buộc If-Match dạng quoted version; list trả array. Quote nhận saved shipping_address_id hoặc inline address mã hóa; response có source_address_id/source_address_version optional. Không thấy ràng buộc oneOf để loại trường hợp gửi cả hai hoặc không gửi địa chỉ.

Provider profile-service.openapi.yaml chỉ có /profile/addresses/validate (gợi ý ward) và validate-consistency (kiểm tính nhất quán). Đây không phải danh mục tỉnh/phường public và không có trong public MVP. customer-profile-mvp reference cũng không cung cấp nguồn lookup. Kết luận chỉ trong các contract đã đọc; không khẳng định backend/deployment không có master data.

**Mobile hiện tại:** rest_checkout_repository.dart:14 gọi /locations/checkout; address.dart:52 gửi tên ward/province, chưa lưu code/version; rest_address_repository.dart đọc wrapper addresses và thiếu If-Match. Picker và adapter phải đổi cùng nhau; dữ liệu tên cũ không đủ để tự suy mã.

**Đề xuất:** giữ picker hai cấp hiện có nhưng dữ liệu do nguồn canonical đã được team chỉ định cung cấp; lưu code và hiển thị name. Chọn saved address thì gửi ID và hiển thị snapshot từ quote; không lấy address book mới ghi đè địa chỉ đơn đã đặt. 409 khi địa chỉ đổi: báo “Địa chỉ đã được cập nhật”, tải lại và cho khách xem lại; 428 là thiếu tiền điều kiện, không tự retry ghi đè.

**Fallback phát hành đề xuất:** nếu chưa có lookup, chỉ cho chọn địa chỉ có code chuẩn đã lưu. Không có địa chỉ phù hợp thì thông báo chưa thể tiếp tục và cho quay lại/giữ giỏ; không hứa khách có thể thêm mới khi picker chưa có nguồn. Không nhập mã địa giới thô cho khách, không hard-code danh mục production hoặc tự ghép mã theo tên. Fallback này cũng cần được duyệt trước khi giới hạn tính năng thực tế.

**Cần Profile/BFF chốt G2:** public lookup hay bộ dữ liệu được quản lý phiên bản; chủ sở hữu, auth, schema codes/names, lọc ward theo province và hành vi dữ liệu cũ; chỉ ID hay inline và độ ưu tiên nếu cả hai xuất hiện; trường hợp chưa có địa chỉ, version drift và error codes. Nếu là bundled dataset phải có nguồn/version/quy trình cập nhật. **Gate M2:** mapper saved-address/version có thể phân tích triển khai; picker create/edit/live quote phụ thuộc nguồn canonical và policy được chốt.

### G3 — Lịch sử đơn, cursor, tìm kiếm và tracking

**Đã xác minh:** public GET /orders dòng441 trả items/next_cursor, không khai báo query hoặc path-level pagination params. Provider order-service.openapi.yaml dòng427 cũng thiếu query này. Không có quy ước công khai gửi cursor tiếp, limit, thứ tự ổn định, hết trang, status hay q. Không thể suy next_cursor là URL, token tự giải mã hoặc tự gửi page/pageSize của ZIP.

GET /orders/{id}/tracking dòng551 chỉ có response type:object, chưa có schema carrier/checkpoints/package/seal/media. Không đủ để khẳng định UI timeline cũ được backend mới hỗ trợ. OrderOrder dòng2629 có status, payment_status, cancellation_status, allowed_actions và snapshot optional; không chứa lịch sử timestamp đầy đủ cho mọi mốc. DELIVERED và COMPLETED là trạng thái khác nhau; các field nội bộ/generation không cần lộ ra UI.

**Mobile hiện tại:** RestOrdersRepository.orderPage gửi q/status/page/pageSize và đọc orders/total/page/page_size; OrdersCubit dùng tổng và số trang. Tracking.fromJson trông đợi code/carrier/status/seal/video/checkpoints. Adapter có thể crash hoặc làm mất dữ liệu nếu dùng nguyên với DTO mới. Order detail dùng UUID để gọi API nhưng nên hiển thị order_code cho khách; không tìm cách parse mã hiển thị thành UUID.

**Đề xuất:** cursor opaque do server quản lý; danh sách chỉ ghi “Đã tải N đơn”, có Tải thêm khi contract đã quy định cursor tiếp. Nếu chưa chốt request cursor thì chỉ hiển thị lô đầu và nói rõ phạm vi, không coi đó là toàn bộ lịch sử. Tìm kiếm/bộ lọc server chỉ bật khi có query contract; lọc cục bộ phải ghi “trong các đơn đã tải”, không đưa empty toàn cục gây hiểu nhầm. Bảo toàn refresh sau hủy và query/filter state phù hợp, không quay lại badge cũ.

**Tracking fallback đề xuất:** hiển thị trạng thái đơn thật và thời điểm updated_at có nhãn “Cập nhật gần nhất”, không gán nó thành thời điểm giao/đóng gói. Chưa có lịch sử vận chuyển thì ghi unavailable + thử lại; chỉ hiện action theo allowed_actions và trạng thái. Không gọi Shipping nội bộ từ Mobile hay tự dựng tuyến đường/mốc tương lai từ enum.

**Cần Order/BFF (+ Shipping cho projection) chốt G3:** input cursor/limit và opaque token, thứ tự/tie-break, hết trang và retry; q tìm order_code hay UUID, status nào là order/payment, kết hợp filter/search; typed tracking response, thiếu/null/chưa có shipment, multi-package, checkpoints/timestamp/timezone, public-safe seal/media. Đây là đề nghị nội dung cần đặc tả, không phải endpoint/field đã tồn tại. **Gate M4:** chưa triển khai cursor load-more hoặc timeline thật trước schema; có thể migrate đọc trạng thái/allowed_actions riêng khi contract/provider sẵn sàng.

### Bộ câu hỏi sẵn để người dùng chuyển team (chưa gửi)

| ID | Owner dự kiến | Quyết định cần trả lời | Đề xuất mặc định để review | Bằng chứng cần có |
|---|---|---|---|---|
| G1 | Order + BFF, PO xác nhận UX | Full-cart hay partial; giỏ thay đổi/tiêu thụ sau placement | Full-cart theo quote, không giả lập partial | Schema/semantics được duyệt và revision/retry/remaining-items cases |
| G2 | Profile + BFF, nguồn master data | Public nguồn code/name/version; saved vs inline; không có địa chỉ | Picker canonical; saved-address-only là fallback có điều kiện | Lookup/data version, ownership/concurrency/error examples |
| G3a | Order + BFF | Cursor input/end/order/search/filter | Opaque cursor; không dùng pageSize cũ | OpenAPI query + examples trang đầu/cuối/retry/filter |
| G3b | Order + Shipping + BFF | Typed tracking projection và dữ liệu thiếu | Chỉ trạng thái thật khi chưa có timeline | DTO + examples chưa có shipment/đang giao/đã giao/nhiều kiện |

Mọi quyết định G1–G3 hiện **OPEN**: user đã duyệt việc lập plan/phân tích, chưa có trả lời owning-team. Không gửi tin nhắn, mở issue hoặc sửa dev để tự chốt. Schema merge và CI xanh không thay cho runtime acceptance.

### Chuyển giao cho các bước sau

M2: phân tích model/repository/UI/tests cụ thể ở thời điểm vào bước; import/pin target snapshot có provenance và hashes khi thực sự migrate, giữ ZIP bất biến. Thiết kế để có thể tách local legacy mock và MVP fixtures, không fallback từ remote failure sang mock tài chính. Chỉ mở phần không bị G1/G2 chặn hoặc sau khi đã có quyết định. M3/M4 giữ ở mức master roadmap, không triển khai/premature-detail trong M1.

QA-01 cũ chuyển sang kiểm thử payment contract mới ở M4; kết quả144/144 của baseline không chứng minh adapter MVP tương thích. QA-02 media vẫn unverified và ngoài lượt này. Không cần computer use cho phân tích contract.

## Điểm tiếp tục

Phân tích ba nhóm đã hoàn tất. Người dùng/team review và trả lời G1–G3; sau đó mới mở plan chi tiết M2 theo phần phụ thuộc được giải quyết. Chưa thực hiện M2. Checkpoint tài liệu rồi dừng; không đánh dấu các quyết định backend đã chốt.

## Evidence truy cập lại

Tất cả link dưới đây pin đúng commit đã phân tích, không trỏ dev đang thay đổi:

- [Public quote input](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/mvp-customer.openapi.yaml#L129), [cart/quote schemas](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/mvp-customer.openapi.yaml#L2550).
- [Public order list](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/mvp-customer.openapi.yaml#L441), [tracking](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/mvp-customer.openapi.yaml#L551).
- [Profile address write schema](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/mvp-customer.openapi.yaml#L3186).
- [Provider Order list](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/order-service.openapi.yaml#L427), [Profile provider](https://github.com/hvduong2k5/DUT.PBL6/blob/7f22226c7b2f4363d3ba376b715feb0e63f69355/docs/03_api_specs/profile-service.openapi.yaml#L596).

Kiểm chứng trong lượt:8assertion tĩnh trên YAML pinned PASS; kiểm schema quote/revision, thiếu lookup, list params/response, tracking placeholder và If-Match. Không chạy contract generator/CI, không gọi provider, không chạy Flutter tests/build; không có thay đổi app source.
