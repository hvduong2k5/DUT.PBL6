# Đề xuất chốt 6 điểm để ghép thành MVP mua hàng

Ngày viết: **28/10/2026**. Người đọc: Web, Mobile, Gateway/BFF và các service backend.

**Đây là phương án triển khai đề xuất, chưa phải xác nhận các nhóm đã đồng ý hoặc code đã chạy xuyên service.** Contract hiện tại vẫn là `integration-target`; việc merge tài liệu không đổi trạng thái nghiệm thu. Khi triển khai, mỗi nhóm ghi tên người phụ trách, PR và bằng chứng kiểm thử vào bảng cuối tài liệu.

## Đọc phần này trước: MVP sẽ làm được gì?

Đề xuất vòng đầu: **khách đăng nhập → chọn hàng và địa chỉ → xem giá/phí giao hàng thật → đặt hàng → chuyển khoản SePay → hệ thống xác nhận tiền và kho → đóng gói → giao hàng → ghi nhận đã giao**.

Khởi đầu bằng Web; Mobile dùng cùng public contract và được bật khi qua cùng bộ kiểm thử. Không bắt buộc hai giao diện phải hoàn thành đồng thời để nghiệm thu vòng Web.

Guest (khách không đăng nhập), COD (thu tiền khi giao), voucher, hoàn tiền và tự động chuyển sang `Completed` chưa nằm trong vòng đầu được bật. Đây là **đề xuất thu hẹp phạm vi**, không phải yêu cầu xóa các chức năng đã có. Chúng có điều kiện mở ở dưới. Vẫn cần xử lý giữ đơn/vụ việc chặn trong phạm vi vận hành đã bật; không dùng việc chưa làm hoàn trả để giả định mọi đơn đều an toàn.

`Delivered` nghĩa là đã giao hàng; `Completed` nghĩa là đã đủ điều kiện đóng nghiệp vụ đơn. MVP vòng đầu có thể kết thúc ở `Delivered` và phải ghi đúng giới hạn đó khi demo/nghiệm thu.

```mermaid
sequenceDiagram
    actor K as Khách mua
    participant FE as Web / Mobile
    participant G as Gateway / BFF
    participant O as Order
    participant P as Các service cung cấp dữ liệu
    K->>FE: Chọn hàng và địa chỉ
    FE->>G: Yêu cầu báo giá
    G->>O: Quote theo giỏ hàng hiện tại
    O->>P: Kiểm giá, địa chỉ, phí giao hàng
    O-->>FE: Quote có hạn dùng và tổng tiền
    K->>FE: Bấm đặt hàng
    FE->>G: Quote + revision + khóa chống trùng
    G->>O: Checkout
    O-->>FE: CheckoutOperation để theo dõi
    loop Trong khi đang xử lý
        FE->>G: Hỏi kết quả operation
        G->>O: Đọc operation
        O-->>FE: Đang xử lý / thành công / thất bại
    end
    O-->>FE: Khi thành công: đọc đơn và hướng dẫn thanh toán
    K->>P: Chuyển khoản SePay
    P-->>O: Bằng chứng thanh toán và kết quả kho
    O->>P: Cho phép đóng gói, rồi giao hàng
    P-->>O: Bằng chứng đã giao theo đúng phiên bản
    O-->>FE: Đã giao; chưa tự suy ra Completed
```

Các mũi tên phản hồi qua Gateway được rút gọn để dễ đọc. Đây là luồng đích đề xuất, không phải sơ đồ chứng minh các service đã nối thật.

| Điểm cần chốt | Phương án đề xuất mặc định | Ai phụ trách triển khai |
| --- | --- | --- |
| 1. Checkout với frontend | Quote → CheckoutOperation → đọc Order; FE thể hiện trạng thái chờ | Web/Mobile, Gateway/BFF, Order |
| 2. Shipping và event version | Shipping phục vụ v1/v2 song song; đơn MVP dùng v2; consumer sẵn sàng trước producer | Shipping, Fulfillment, Order |
| 3. Identity/Gateway | Keycloak RS256/JWKS; kiểm token ở Gateway và backend; ánh xạ ID phía server | Identity/Gateway, Order, Profile |
| 4. Adapter cho Order | Nhóm Order sở hữu một facade nội bộ, gọi provider thật | Order chủ trì; từng provider sở hữu dữ liệu gốc |
| 5. Hoàn tất đơn | Chỉ hoàn tất khi đủ bằng chứng; vòng đầu chưa bật tự động Completed/COD | Order, Shipping, Care, Finance |
| 6. Ký event và replay | Adapter chuẩn hóa/ký theo codec Order hiện tại; provider lưu revision gốc | Order chủ trì codec; Shipping/Fulfillment/Care giữ lịch sử |

## 1. Checkout: khách bấm mua thì màn hình và Order làm gì?

### Hiện tại khác nhau ở đâu?

Trong code đã đối chiếu, Web/Mobile gửi danh sách hàng và địa chỉ rồi chờ nhận đơn/QR ngay. Order nhận `quote_id`, `cart_revision`, phương thức thanh toán, rồi trả **CheckoutOperation**. Đây là mã theo dõi một lần đặt hàng, có thể chưa có `order_id`. Web còn có phần tính báo phí tạm tại BFF; đó chưa phải quote thật của Order.

Điều này cho thấy hai cách gọi chưa khớp. Nó không chứng minh hai nhóm có ý định nghiệp vụ trái nhau.

### Chọn cách làm nào?

Giữ mô hình của Order, chỉnh FE/BFF theo public MVP contract. Không thêm một luồng đặt hàng trực tiếp thứ hai chỉ để giữ màn hình cũ.

1. FE lấy báo giá thật trước khi khách bấm đặt: giá hàng, phí giao, tổng tiền và hạn dùng. **Quote** là báo giá có định danh và hạn dùng; chưa có nghĩa đã giữ hàng trong kho.
2. Khách đổi số lượng, địa chỉ hoặc lựa chọn giao hàng thì lấy quote mới. **Cart revision** là phiên bản giỏ hàng, giúp phát hiện khách đang xác nhận một giỏ đã thay đổi.
3. Khi bấm đặt, FE tạo và lưu một `Idempotency-Key` cho lần xác nhận đó, gửi quote/revision. Key là mã chống tạo đơn trùng khi cùng yêu cầu phải gửi lại.
4. Cả HTTP `201` và `202` đều đọc DTO CheckoutOperation theo contract. `202` nghĩa là đã nhận xử lý; không hiển thị “đặt hàng thành công” chỉ từ mã này.
5. FE hiển thị “Đang tạo đơn”, hỏi lại operation theo `Retry-After` khi có; mặc định 2 giây, giãn nhịp khi lỗi mạng. Dừng khi operation kết thúc. Dùng đúng enum trong OpenAPI, không tự thêm trạng thái wire theo nhãn giao diện.
6. Khi operation thành công, đọc đơn và thông tin thanh toán. Khi thất bại vì giá/quote hết hạn hoặc hết hàng, giải thích lý do và cho khách kiểm tra lại trước lần đặt mới.

| Tình huống khách gặp | Cách xử lý đề xuất |
| --- | --- |
| Bấm hai lần | Khóa nút khi gửi; backend vẫn chống trùng bằng cùng key |
| Mạng mất sau khi gửi | Giữ key và payload cũ; phục hồi operation nếu đã biết ID, hoặc gửi lại cùng lệnh để lấy kết quả |
| Khách sửa giỏ sau lần thất bại đã xác định | Lấy quote mới, xác nhận lại và tạo key mới |
| Chờ lâu nhưng chưa rõ kết quả | Hiện “Đang kiểm tra kết quả”, cho quay lại xem; không tự coi đơn thất bại và tạo đơn khác |
| Nhìn thấy QR | Chỉ nghĩa là có hướng dẫn trả tiền, chưa phải đã thanh toán |
| Webhook thanh toán trả 200 | Chỉ là đã tiếp nhận thông báo; trạng thái trả tiền dựa trên kiểm chứng backend |

**Phân công:** FE sở hữu màn hình và phục hồi; BFF đổi public `Idempotency-Key` sang `X-Idempotency-Key` của Order, chuẩn hóa lỗi; Order sở hữu quote, operation và quyết định tạo đơn. BFF không tự bịa phí 0 để thay phí chưa lấy được.

**Đủ điều kiện bật khi:** kiểm thử thành công một đơn thật trên stack tích hợp, bấm đúp không tạo hai đơn, mất phản hồi rồi retry lấy lại cùng kết quả, quote hết hạn không âm thầm đổi giá, người khác không đọc được operation/đơn.

Đọc [public MVP OpenAPI](../../../docs/03_api_specs/mvp-customer.openapi.yaml) và [migration public API](MIGRATION.md#public-api-và-client).

## 2. Shipping và event: chọn v2 nhưng giữ v1 bằng cách nào?

**Version contract** là phiên bản cấu trúc dữ liệu. Shipping v1 và đề xuất v1 cũ của Order đã dùng cùng số trường protobuf cho hai nghĩa khác nhau. Vì vậy không sửa v1 tại chỗ; v2 có tên package riêng.

**Producer** là bên phát sự kiện, ví dụ Shipping báo giao xong. **Consumer** là bên đọc sự kiện, ví dụ Order. **Rollout** là thứ tự đưa các thay đổi vào chạy.

### Đề xuất: dual-serve RPC, chuyển event theo nhóm đơn

**Dual-serve** nghĩa là một Shipping triển khai và cung cấp cả API v1 lẫn v2. Không có nghĩa cùng một đơn phải tạo hai vận đơn. Giữ v1 phục vụ caller cũ, Order MVP gọi `omamx.shipping.v2`. Không chọn adapter “đổi tên v1 thành v2” vì v1 thiếu bằng chứng phiên bản/generation mà v2 cần.

Với event, giữ schema/topic v1 cho luồng cũ; đơn thuộc nhóm MVP dùng packing/shipping v2 và Order facts v3. Một **nhóm đơn** là các đơn được đánh dấu đi theo cùng luồng tích hợp; lựa chọn này phải lưu bền và không đổi giữa chừng. Mỗi tác động như giữ/trừ kho hoặc tạo vận đơn chỉ có một bên thực hiện có thẩm quyền.

Thứ tự triển khai:

1. Shipping/Fulfillment lưu revision và dữ liệu proof thật, triển khai RPC mới; Order triển khai `GetShippingSnapshot`.
2. Deploy consumer v2 ở Shipping và adapter Order trước. Kiểm schema, replay và chống trùng khi chưa mở đơn MVP.
3. Bật producer v2 cho nhóm đơn thử nghiệm. Đơn MVP không chạy thêm consumer v1 gây cùng tác động; Order v3 không kích hoạt cơ chế trừ kho legacy.
4. Kiểm từ đóng gói tới giao hàng. Sau khi đạt mới mở thêm đơn mới.
5. Nếu có lỗi, dừng nhận **đơn mới** vào nhóm v2; tiếp tục xử lý hoặc giữ các đơn v2 đang chạy bằng đường v2. Không đổi chúng về v1 khi đang giao.

Không đặt ngày chuyển cứng trước khi provider và consumer có bằng chứng đạt. Chưa xóa v1 trong đợt MVP này; chỉ gỡ khi kiểm kê được không còn caller/đơn đang chạy cần nó.

**Đủ điều kiện bật khi:** v1 vẫn tương thích baseline; v2 có quote bền và proof đúng revision; phát lại event không tạo thêm shipment/trừ kho; bản tin v1 không bị dùng để giả lập đủ bằng chứng v2.

Đọc [chi tiết version và topics](MIGRATION.md), [Shipping v2](../../proto/shipping/v2/shipping.proto).

## 3. Identity/Gateway: thống nhất khách là ai và ai được gọi nội bộ

Nhánh Identity đã có Keycloak và thiết kế RS256/JWKS. Order hiện còn verifier HS256 theo profile sandbox. Hai kiểu token này chưa tự tương thích chỉ vì cùng được gọi là JWT.

| Thuật ngữ | Nghĩa dễ hiểu | Đề xuất chốt |
| --- | --- | --- |
| Issuer | Nơi cấp token | Cấu hình đúng issuer của realm Keycloak cho từng môi trường; so khớp chính xác |
| Audience | Token được cấp để gọi hệ thống nào | Dùng `order-api` làm audience bắt buộc cho Order; Identity cấu hình mapper tương ứng, không nhầm với client ID của FE |
| Key/JWKS | Khóa để kiểm token thật hay giả; JWKS là nơi công bố khóa công khai | RS256, lấy/cached JWKS từ issuer đã cấu hình; không lấy URL khóa tùy ý từ token |
| Subject (`sub`) | ID tài khoản tại Keycloak | Không mặc định đây chính là customer ID của Order/Profile |
| `user_id` | Principal nội bộ theo thiết kế Identity | Chỉ dùng sau khi token đã xác minh và mapper Identity được kiểm thử |
| Customer ID | ID khách dùng để sở hữu giỏ/đơn/địa chỉ | Ánh xạ principal → customer ở server; nếu khác ID thì tra mapping có thẩm quyền |
| Credential nội bộ | Giấy thông hành giữa các service | Tách khỏi token người mua và khỏi header tự khai của trình duyệt |

**Đề xuất đường public:** Gateway kiểm token; Order/Profile cũng kiểm token gốc hoặc credential chuyển tiếp có xác thực theo profile đã thống nhất. Vòng đầu ưu tiên chuyển tiếp token gốc, backend tự kiểm chữ ký, issuer, audience, expiry và quyền sở hữu. Việc Gateway đã kiểm không làm cho `X-User-Id` bất kỳ trở thành đáng tin.

Identity chịu trách nhiệm ánh xạ Keycloak subject → principal; Profile chịu trách nhiệm principal → customer nếu domain có ID riêng. Order nhận customer đã giải quyết bằng mapping đó. Cần một fixture tài khoản xuyên ba bên để chứng minh cùng một người nhận đúng giỏ, địa chỉ và đơn; không tự coi `sub = user_id = customer_id`.

**Đề xuất đường nội bộ:** credential riêng theo caller trong secret store/env triển khai, kiểm quyền theo API, truyền trên kết nối được bảo vệ. Với facade HTTP hiện có có thể giữ cơ chế internal token cho MVP nhưng phải cấu hình/kiểm thật và giới hạn đúng caller; nếu muốn nhiều caller/key thì cần sửa verifier rõ ràng. Không dùng một secret public trong repo cho mọi service. `requesting_service` trong body chỉ là thông tin ngữ cảnh, không phải bằng chứng danh tính.

**Guest:** chưa bật vòng đầu. Khi triển khai, server cấp session ngẫu nhiên có hạn dùng, bind với giỏ/operation/đơn; không chấp nhận ID session khách tự chọn như một quyền sở hữu. Web dùng cookie bảo vệ phù hợp; quyền xem QR/thanh toán được giới hạn đúng đơn và có hạn. Gateway/Identity sở hữu cơ chế session, Order kiểm ownership. Phải bổ sung và kiểm contract guest trước khi mở UI.

**Đủ điều kiện bật khi:** token Keycloak thật gọi được Order; token sai issuer/audience, hết hạn hoặc chữ ký sai bị từ chối; tài khoản B không đọc được dữ liệu A; caller nội bộ không có quyền không gọi được snapshot/facade. Cấu hình sandbox không được fallback khi xác minh Keycloak lỗi.

## 4. Adapter cho Order: ai viết và đặt ở đâu?

**Adapter** là lớp nối cách hỏi của Order với API thật của từng service. Ví dụ Order hỏi “báo giá này còn hợp lệ không?” thì adapter gọi Catalog/Shipping để kiểm, không tự trả `true`.

### Đề xuất: nhóm Order sở hữu facade, provider sở hữu sự thật

Thay owner vai trò chung `order-integration-adapters` bằng **`order-service`** trong registry. Đây là phân công đề xuất ở cấp nhóm, chưa gán cho một cá nhân.

Đặt mã adapter trong vùng mã nguồn do nhóm Order quản lý, tách khỏi domain core. Triển khai MVP dưới một process facade nội bộ do nhóm Order vận hành, giữ nguyên HTTP facade mà core đang gọi qua `DEPENDENCY_URL`. Như vậy không phải vừa đổi transport của core vừa nối nhiều provider. Có thể cùng repository/build pipeline với Order; không cần tạo một nhóm sản phẩm mới chỉ để vận hành facade.

`DEPENDENCY_URL` phải trỏ vào facade này ở môi trường tích hợp, không trỏ vào simulator hoặc tùy tiện trỏ vào root của Catalog/Shipping. Đường dẫn thư mục và entrypoint cụ thể sẽ chọn trong PR code sau khi đối chiếu cấu trúc service; tài liệu này không tuyên bố process đó đã tồn tại.

| API facade | Câu hỏi bằng tiếng người | Gọi nguồn nào / ai cung cấp dữ liệu |
| --- | --- | --- |
| `/terms` | Giá, địa chỉ, phí giao dùng để lập đơn là gì? | Catalog, Profile, Shipping; trả CommercialTerms. ID và ownership do Order điền |
| `/revalidate` | Ngay trước đặt hàng, báo giá đã nhận còn được chấp nhận không? | Kiểm lại version/expiry/binding ở các provider liên quan; không âm thầm thay giá |
| `/resources/{id}?source=...&version=...` | Bản ghi gốc nào chứng minh sự kiện này? | Shipping `GetShipmentRevision`, Fulfillment `GetTaskRevision`; Care phải có giao thức lịch sử được chốt riêng |
| `/admission-barrier` | Có thể cho tác vụ bắt đầu mà không đua với hủy/chặn không? | Fulfillment và trạng thái/permit Order theo giao thức accept/cancel; không dùng một lần đọc “chưa hủy” thay cho phối hợp nguyên tử |
| `/completion` | Đơn có đủ bằng chứng để đóng nghiệp vụ chưa? | Shipping + Care + settlement/Finance, gắn quyết định với đúng order version |

Inventory vẫn phải triển khai các RPC reserve/finalize/release và bằng chứng của mình; việc có facade không thay thế giao thức kho.

Provider chịu trách nhiệm định nghĩa đúng và lưu bền dữ liệu gốc. Nhóm Order chịu trách nhiệm timeout, mapping, retry có kiểm soát, lỗi dependency và kiểm bằng chứng. Provider timeout/thiếu dữ liệu thì trả trạng thái chưa thể quyết định hoặc lỗi phù hợp contract; không gọi simulator dự phòng.

**Đủ điều kiện bật khi:** mỗi API cần cho phạm vi MVP có test với provider thật, kèm trường hợp provider lỗi và dữ liệu stale. Các API chưa nghiệm thu vẫn đóng capability liên quan; không cần dựng completion giả để mở checkout.

Đọc [OpenAPI facade](../../../docs/03_api_specs/order-core-dependencies.mvp.openapi.yaml).

## 5. Hoàn tất đơn: giao xong chưa chắc đã đóng được đơn

Đề xuất giữ sự khác biệt:

| Trạng thái | Chứng minh được điều gì? | Chưa chứng minh điều gì? |
| --- | --- | --- |
| Đã thanh toán | Backend xác minh khoản tiền cho đơn | Hàng đã giao |
| `Delivered` | Shipping có bằng chứng giao hàng | Không có vụ việc chặn; COD đã đối soát |
| `Completed` | Đủ điều kiện kết thúc nghiệp vụ | Không phải trạng thái suy ra từ một webhook đơn lẻ |

Code Order đang có thời gian chờ **7 ngày tính từ thời điểm giao được chứng minh** trước khi xét hoàn tất. Đề xuất giữ quy tắc này cho môi trường MVP để tránh đổi hành vi giữa lúc tích hợp; đây chưa phải chính sách hoàn trả được doanh nghiệp phê duyệt hoặc một kết luận pháp lý.

Để bật tự động Completed, facade phải lấy đủ:

1. Shipping: đúng shipment/order/generation đã giao, kèm revision và thời điểm gốc.
2. Care: kết quả có thẩm quyền về vụ việc đang chặn, gắn phiên bản/thời điểm kiểm tra. “Không kết nối được Care” khác “Care xác nhận không có vụ việc”.
3. Thanh toán/Finance: trả trước đã được xác minh theo đường thanh toán; với COD phải có kết quả đối soát tiền thu hộ cho đúng đơn, không chỉ `delivered=true` hoặc “shipper đã thu”.
4. Order: kho đã commit đúng, hết thời gian chờ, không có hold/refund chưa giải quyết và order version vẫn khớp lúc áp dụng quyết định.

Quyết định từ facade cần decision ID, hạn dùng và order version theo contract. Order kiểm lại điều kiện cục bộ khi áp dụng để tránh quyết định cũ dùng cho đơn vừa bị chặn.

**Chọn cho MVP:** chưa có Care/Finance thật thì không bật automatic completion; UI thể hiện “Đã giao” và không quảng bá đây là vòng đời đóng đơn đầy đủ. Không thêm nút admin cưỡng ép Completed như cách vượt kiểm chứng.

**Điều kiện mở COD:** Finance và Shipping cung cấp settlement reference, số tiền/đơn vị tiền, trạng thái và lịch sử điều chỉnh được kiểm chứng; hợp đồng truy vấn và xử lý chênh lệch phải được bổ sung trong PR riêng. Contract facade hiện có không tự định nghĩa xong giao thức nghiệp vụ Finance/Care. Có thể triển khai COD sau vòng SePay mà không sửa lại luồng quote/operation của FE.

**Đủ điều kiện bật Completed khi:** thử đơn đủ điều kiện; đơn đang có case; thiếu settlement COD; Care timeout; quyết định hết hạn; order đổi version trước khi áp dụng. Chỉ trường hợp đủ bằng chứng được hoàn tất.

## 6. Ký event và replay: dữ liệu đúng nguồn, đọc lại vẫn đúng

**Chữ ký event** giúp Order kiểm bản tin đến qua bên được tin cậy và chưa bị sửa. **Replay** là xử lý lại bản tin đã có, thường sau mất mạng hoặc khi consumer khởi động lại. **Revision** là phiên bản dữ liệu của một task/shipment; nó khác version v1/v2 của contract.

Ví dụ shipment ở revision 3 ghi nhận đã giao, sau đó revision 4 có cập nhật khác. Khi đọc lại sự kiện revision 3, Order phải hỏi được bản ghi **3**. Lấy trạng thái mới nhất là **4** rồi gắn nhãn **3** sẽ làm sai bằng chứng.

### Đề xuất: một nơi chuẩn hóa/ký, lịch sử nằm ở provider

1. Shipping/Fulfillment/Care ghi sự thay đổi và revision lịch sử bền vững. Đề xuất transactional outbox: lưu dữ liệu và bản tin cần phát cùng giao dịch, rồi worker phát event; tránh dữ liệu đã đổi nhưng event mất khi process chết.
2. Adapter ingress do nhóm Order sở hữu đọc event từ đường nội bộ có xác thực/ACL. Nó kiểm schema, nguồn và mapping; broker credentials phải giới hạn ai được publish topic nguồn. Không nhận payload công khai tự khai `source=shipping` rồi ký hộ.
3. Adapter lấy proof revision gốc và chuẩn hóa về Event mà Order hiểu. Shipping/Fulfillment không cần tự viết lại codec HMAC của Go trong từng ngôn ngữ ở vòng đầu.
4. Adapter ký theo **codec Order hiện tại** rồi đưa vào ingress nội bộ. Order vẫn kiểm chữ ký và đối chiếu bằng chứng gốc; chữ ký không thay thế proof provider.
5. Consumer lưu kết quả xử lý/chống trùng bền vững trước khi xác nhận đã nhận bản tin. Thiếu proof tạm thời thì thử lại có giới hạn rồi giữ trong hàng chờ xử lý lỗi; sai chữ ký hoặc sai payload thì ghi nhận và cách ly, không ép chuyển trạng thái.

**Codec** là quy tắc chuyển Event thành đúng chuỗi byte để ký. Hiện Order dùng HMAC-SHA256 trên JSON của Go struct, với `signature` rỗng và quy tắc bỏ trường rỗng cụ thể. Chọn giữ codec đó trong đợt này. Không tự thay bằng JSON sắp xếp key vì chữ ký sẽ khác.

Nhóm Order phải xuất **test vectors**: các mẫu gồm input, chính xác bytes đem ký, khóa thử không phải secret triển khai và chữ ký kỳ vọng. Tối thiểu có mẫu bình thường, trường optional bị bỏ, generation/seal có giá trị, payload bị sửa, chữ ký sai và ID trùng nhưng nội dung khác. Provider/adapter dùng các mẫu này làm kiểm thử chung trước khi bật ingress.

**Khóa ký:** tách theo môi trường, lưu trong secret store, chỉ signer và Order verifier được truy cập. HMAC dùng khóa chung nên verifier cũng có khả năng ký; đây là mô hình tin cậy của MVP, không phải chữ ký chứng minh độc lập từng provider. Không dùng secret JWT hoặc credential gọi API làm khóa event.

Envelope hiện chưa chốt giao thức `key_id`/key ring dùng nhiều khóa. Với vòng MVP nội bộ, đề xuất xoay khóa trong cửa sổ bảo trì có phối hợp: dừng tiếp nhận ký mới, xử lý hết bản tin đã ký đang chờ theo kiểm kê cả hàng lỗi, chuyển signer/verifier, kiểm thử rồi mở lại. Nếu cần xử lý lại bản tin ký bằng khóa cũ sau đó, phải có quy trình kiểm chứng và ký lại từ proof gốc có audit; không bỏ kiểm chữ ký. Trước vận hành liên tục cần PR thiết kế key ID + khoảng chồng lấn khóa, không tự thêm field vào codec hiện tại.

**Giữ lịch sử:** giữ revision gốc và hồ sơ chống trùng trong suốt MVP, chưa bật dọn tự động. Trước triển khai lâu dài, chốt thời hạn theo cửa sổ replay/đối soát, dung lượng và yêu cầu dữ liệu; không chỉ giữ latest state.

**Đủ điều kiện bật khi:** phát hai lần cùng event chỉ có một tác động; cùng ID khác nội dung bị từ chối; revision cũ vẫn truy vấn được sau revision mới; bản tin đến sai thứ tự không làm lùi/sai trạng thái; codec tests qua ở signer/verifier; sai nguồn/chữ ký không qua được.

Đọc [normalized Event schema](../../events/profiles/mvp/order-source.event.json) và [migration ingress](MIGRATION.md#events-và-ingress-adapter). Kind nhận tác vụ đóng gói là `PACKING_ACCEPTED`, không phải `FULFILLMENT_ACCEPTED`.

## Thứ tự làm để không chờ nhau vô hạn

| Đợt | Việc cần có | Kết quả nghiệm thu |
| --- | --- | --- |
| A — Nền chung | Review lựa chọn trong tài liệu; gán người; Identity mapping; codec fixtures; giữ v1/v2 đúng package | Mỗi nhóm biết input/output, quyền và phần việc; contract CI qua |
| B — Tạo đơn | Catalog snapshot, Profile ownership, Shipping quote v2, Inventory protocol, facade terms/revalidate, FE operation | Khách đăng nhập tạo một đơn bằng provider thật; retry và hết hạn chạy đúng |
| C — Trả tiền và giao hàng | SePay verification, Inventory commit, Fulfillment permit/barrier, Order snapshot, Shipping v2 và ingress/proof | Tiền xác minh, kho đúng, một task/shipment, đi tới Delivered |
| D — Mở rộng | Guest, COD/settlement, Care completion authority và chính sách 7 ngày | Chỉ mở từng capability sau kiểm thử riêng; lúc này mới nghiệm thu Completed nếu đủ |

Các nhóm có thể viết code song song; thứ tự trên là **thứ tự mở và nghiệm thu chức năng**. Mocks dùng để phát triển FE/test tình huống lỗi, không dùng làm bằng chứng provider thật đã hoàn thành.

Merge contract trước để các nhánh service cùng dựa vào một mốc. Merge PR triển khai vào `dev` theo đợt, với feature gate mặc định tắt nếu dependency chưa sẵn sàng. Không merge tất cả rồi coi CI contract xanh là một MVP hoạt động.

## Bảng bàn giao: người nhận việc cập nhật ngay trong PR

Tên nhóm dưới đây là đề xuất trách nhiệm, **chưa phải xác nhận nhóm đã nhận việc**. Không đánh dấu hoàn tất khi chỉ có generated stub hoặc simulator.

| Hạng mục | Nhóm chịu trách nhiệm chính | Người nhận | PR triển khai + commit | Bằng chứng provider thật | Trạng thái |
| --- | --- | --- | --- | --- | --- |
| FE quote/operation và khôi phục retry | Web, sau đó Mobile | Chưa gán | — | — | Đề xuất |
| Public DTO/error/header mapping | Gateway/BFF | Chưa gán | — | — | Đề xuất |
| Keycloak, audience, principal/customer mapping | Identity/Gateway + Profile + Order | Chưa gán | — | — | Đề xuất |
| Catalog snapshot và Profile address | Catalog + Profile | Chưa gán | — | — | Đề xuất |
| Inventory reserve/finalize/release | Inventory | Chưa gán | — | — | Đề xuất |
| Facade và dependency adapters | Order | Chưa gán | — | — | Đề xuất |
| Shipping v2, lịch sử revision và event | Shipping | Chưa gán | — | — | Đề xuất |
| Permit/barrier, packing v2 và revision | Fulfillment + Order | Chưa gán | — | — | Đề xuất |
| Ingress, codec vectors và key procedure | Order | Chưa gán | — | — | Đề xuất |
| SePay → commit kho → cho phép đóng gói | Order + Inventory + Fulfillment | Chưa gán | — | — | Đề xuất |
| Care/settlement/Completed; COD | Care + Finance + Shipping + Order | Chưa gán | — | — | Chưa bật vòng đầu |
| Guest session/grant | Identity/Gateway + Order + FE | Chưa gán | — | — | Chưa bật vòng đầu |

Mỗi PR triển khai ghi: capability nào đã làm, phiên bản contract, commit các service đã chạy cùng, lệnh/test và kết quả, dependency nào vẫn giả lập, feature nào còn tắt. Chỉ cập nhật `implementation_acceptance` sau khi có bằng chứng tương ứng; nếu chỉ chạy một phần thì ghi đúng phần đó.

## Cách đọc cùng các file khác

- **Tài liệu này:** lý do lựa chọn, trách nhiệm và thứ tự triển khai bằng ngôn ngữ dễ đọc.
- [README](README.md): điểm vào và danh sách contract theo vai trò.
- [Registry](registry.yaml): artifact/owner/task và trạng thái máy kiểm được.
- [MIGRATION](MIGRATION.md): version, wire fields và quy tắc chuyển đổi kỹ thuật.
- [Public OpenAPI](../../../docs/03_api_specs/mvp-customer.openapi.yaml), [facade OpenAPI](../../../docs/03_api_specs/order-core-dependencies.mvp.openapi.yaml), proto và event schema: cấu trúc request/response chính xác.

Nếu đề xuất trong tài liệu cần đổi wire schema hoặc thêm endpoint, phải sửa artifact và kiểm thử trong cùng PR triển khai; prose không tự tạo capability hay thay thế schema. Review lựa chọn nào khác thì ghi rõ lý do, tác động các consumer và cập nhật cả tài liệu lẫn registry/migration liên quan.

Mốc đối chiếu cho nhận xét hiện trạng: Order `c753a52`, Web `b1ff291`, Mobile `3f20602`, Shipping `1242209`, Catalog `db95c36`, Identity `bbb78b0`; contract PR #34, trước tài liệu này là `d24e5d6`. Đây là bằng chứng từ code/contract tại các mốc đó, chưa phải khảo sát deployment đang chạy.
