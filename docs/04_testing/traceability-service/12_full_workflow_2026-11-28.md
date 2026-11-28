# Kiểm chứng toàn luồng Traceability và rule — 28/11/2026

Checkpoint lịch sử trước PR176. [Report13 sau khi ghép dev mới](13_after_pr176_2026-11-28.md) là nguồn trạng thái hiện hành; giữ nguyên source SHA và kết quả của lượt này.

Trạng thái: **PASS toàn luồng phần mềm trong môi trường local cô lập, chờ review/tích hợp release**. Worktree `DUT.PBL6-traceability-dev`, nhánh `codex/traceability-order-flow`; không merge dev/release hoặc deploy production. Đây là kết quả của lượt chạy được nêu dưới đây; CI theo HEAD phải xem trực tiếp tại [PR177](https://github.com/hvduong2k5/DUT.PBL6/pull/177), không suy CI xanh từ báo cáo local.

## Phạm vi

Đóng các phần chưa có bằng chứng trong report11: receipt thật và replay, Order thật được đóng gói bởi Fulfillment, hai lô FEFO, resolver TLS thật, hồ sơ và chứng nhận thật trong môi trường fixture, phân quyền kho qua IAM API, trạng thái công khai/ngừng công khai và hành vi khi resolver dừng. Dữ liệu, tài khoản, chứng nhận và thanh toán đều thuộc sandbox; chứng nhận PDF ghi rõ không có giá trị pháp lý. SePay statements TEST/carrier FAKE không được gọi là giao dịch ngân hàng/vận chuyển thật.

## Tuân thủ rule

- Order/Fulfillment giữ đúng hai Dockerfile; local image dùng artifact Linux, user không đặc quyền. Coolify giữ source build độc lập.
- Năm topology và codegen/race toolchain chuyển sang `infra/docker/`; service không còn Compose hoặc Dockerfile.tools. CI và lệnh local trỏ hạ tầng trung tâm. Không xóa dữ liệu volume.
- Regenerate bằng protoc3.21.12/Go plugin1.34.1/gRPC plugin1.3.0; LF cho Go/shell scripts trên Windows. Không sửa tay generated code.
- Commit mới dùng tiếng Anh, tách mục đích. Commit lịch sử bdb215d7 giữ nguyên theo yêu cầu tiếng Việt của chủ repository tại thời điểm tạo; không rewrite lịch sử đã push hoặc phá source SHA đã phát cho owner.
- Contract/state matrix theo bộ0.4.0; chưa dùng CI xanh để tự promote stable. Các checkpoint cũ được đánh dấu lịch sử.
- Rule chủ repository gửi yêu cầu giữ tên `catalog-checkout.openapi.yaml`, trong khi PR175 đã xóa hợp đồng REST đó. File được giữ lại dưới dạng marker `catalog-checkout-2.0-retired`, `paths: {}`, trỏ canonical proto; không khôi phục internal REST hoặc thay registry hiện hành.

## Bài chạy lặp lại

`python scripts/test-web-identity.py --traceability-focused --fulfillment --output-dir <ignored-folder>` dựng provider riêng, cổng động, HTTPS ingress với CA fixture, các RPC nội bộ TLS. Traceability dùng **cmd/server ở integration mode**, runtime PostgreSQL role không có DDL/superuser; không nạp profile công khai trực tiếp vào DB.

Identity, Catalog, Inventory, Order, Fulfillment và Media dùng handler/logic thật. Catalog/Media giữ canonical RPC theo PR175; không viết lại GetSkuBinding đã gỡ. Flow vẫn độc lập: lỗi truy xuất không rollback receipt hoặc sửa lifecycle đơn hàng.

Lượt local này dùng MinIO versioned S3 đã cache và image ClamAV thực với signature DB fixture nhỏ; CI mặc định giữ RustFS/ClamAV image pin của suite. Đây không phải nghiệm thu bộ chữ ký antivirus production. Media PDF chuyển sang READY theo xử lý hiện hành của provider, không có nghĩa đã quét antivirus; ClamAV được kiểm cho video Fulfillment. Media fixture tắt Kafka riêng của Media; các đường Kafka Inventory→Catalog, Order/Shipping và Traceability outbox vẫn dùng broker thật. `--reuse-built-jars` chỉ dành cho lượt lặp local, kiểm timestamp tất cả nguồn Java/build/proto trước khi dùng artifact hiện có và không báo đã chạy lại Java unit tests. CI mặc định build/test Java.

## Kết quả

Run `traceability-order-live-20261128-v8`: API PASS ngày28/11, source head `c16548a3c8b29b3f28bb191fbc03b92ff5fe17b2`; browser kiểm lại sau sửa CSS tại source `1ba126f8` cùng ngày. [Kết quả đã lọc bí mật](assets/2026-11-28-order-e2e/result.json) giữ các assertion và source SHA. Cookie/JWT/password/private key/signed object URL và file cấu hình runtime vẫn ở thư mục `.sandbox` bị ignore.

| Ca kiểm chứng | Kết quả quan sát |
| --- | --- |
| Receipt thật và replay | Nhập lô mới qua Web→Identity→Inventory; replay trả nguyên kết quả, không nhập trùng; binding đúng kho |
| Kho và maker/reviewer | Nhân viên kho khác403; người lập tự duyệt403; người khác cùng kho duyệt/publish200 |
| Media | Hai PDF fixture qua upload-session→presigned PUT→complete→READY; validate/bind bằng canonical Media RPC |
| Đơn thật và FEFO | Hai lượt quote/accept/replay/payment TEST, Inventory TLS, Fulfillment thật, versioned video, ffprobe/ClamAV, niêm phong, Shipping FAKE/Kafka→DELIVERED/history |
| Hai lô trong cùng đơn | Đơn thứ hai lấy hai lô, mỗi lô1 sản phẩm; hai link khớp exact sealed proof |
| Công khai và lịch sử | Draft2 không lộ qua QR/public1; suspend hồ sơ thứ hai tạo AVAILABLE + NO_LONGER_PUBLIC, public410; đổi tên lô live không sửa batch_code đã niêm phong |
| Resolver outage | Dừng cmd/server thật: row UNAVAILABLE, Order status/version giữ nguyên; khởi động lại khôi phục AVAILABLE |
| IAM hiện hành | Thu hồi qua API và assignment hết hạn đều chặn403 với JWT chưa hết hạn |
| Outbox/Kafka | Đọc được hai `vn.omama.traceability.profile.published.v1` từ broker thật; sau suspend có3/3 outbox PUBLISHED,0 pending/failed |
| Browser | Desktop và390px: đơn hai lô, trang public/410, draft isolation, PNG/SVG, QR canvas và PNG tải mới giải mã đúng URL |
| In phần mềm, chưa nghiệm thu thiết bị | Portal in dùng bản public; PDF A4 đúng1 trang, render Poppler và kiểm ảnh; không đưa draft2 vào tem |

Ảnh đối chiếu: [đơn desktop](assets/2026-11-28-order-e2e/order-two-sealed-batches.png), [đơn mobile](assets/2026-11-28-order-e2e/order-two-sealed-batches-mobile.png), [public mobile](assets/2026-11-28-order-e2e/public-source-mobile.png), [public410](assets/2026-11-28-order-e2e/public-source-gone-mobile.png), [bản in render](assets/2026-11-28-order-e2e/print-render.png). Ảnh cho thấy lỗi panel nguồn gốc bị đặt vào cột icon46px của `.order-lines`; CSS riêng đã sửa để nội dung dùng toàn bộ chiều rộng card và chụp lại.

## Đóng gói và kiểm tra bổ sung

- Năm Compose topology tập trung đều qua `docker compose config`; hai local artifact images build thành công và chạy user không đặc quyền. Hai Coolify source images build thành công: Order `sha256:ce1b40638d80321f5b55f6fa02bc4d3e044fc85e75c322f709959e3967af8d80`, Fulfillment `sha256:7d5fb25c7461ee76f99b1c67bf64f4603a2b1ee87f7364754a748ea689e2c44f`.
- Order Coolify giải mã token/CA fixture đúng nội dung, file mode600; base64 lỗi làm startup thất bại. Không dùng credential thật để probe.
- Toolchain trung tâm regenerate/gofmt PASS, không diff generated stubs. Go build trong container với mounted worktree PASS sau `GOFLAGS=-buildvcs=false`; provenance lấy từ manifest, không phụ thuộc Git ownership/đường dẫn `.git` của host.
- Bốn contract frontend bắt buộc lint PASS; retired Catalog marker lint PASS với một warning license metadata. Không đổi schema REST, permission hoặc trạng thái giao diện từ thay đổi CSS này.
- Lượt CI `5f69244c` phát hiện VCS stamping trong Order/Fulfillment và listener chỉ bind loopback không tới được từ Kong Linux. Hai lỗi fixture/build đã sửa; dedicated CI workflow sẽ chạy lại trên HEAD. HTTP fixture dùng host interface để Docker gateway truy cập, các RPC TLS/native checks vẫn dùng loopback. HTTPS proxy chuyển cả HMR WebSocket để Next dev hydrate thật.

Các lần dựng môi trường lỗi, kỳ vọng fixture nhầm200 thay vì201 khi tạo revision, và lần proxy thiếu WebSocket không được tính PASS. CI cũ hoặc gate MVP chung không thay thế ca Traceability toàn luồng này.

## Cách lặp lại và bàn giao

- Chuẩn bị toolchain/phụ thuộc và Fulfillment image theo [hạ tầng trung tâm](../../../infra/docker/README.md) và workflow `.github/workflows/web-identity.yml`; Media dùng `Dockerfile.coolify` canonical.
- Chạy `python scripts/test-web-identity.py --gradle <gradle> --traceability-focused --keep-running --output-dir <ignored-folder>`. Cấu hình/kết quả riêng tư nằm trong folder được chọn; chỉ trích xuất danh sách file bằng chứng đã duyệt.
- Browser tùy chọn: cài `puppeteer-core` và `jsqr` vào tool directory, đặt `PUPPETEER_MODULE`, `JSQR_MODULE`, `CHROME_BINARY`, rồi chạy `node scripts/test-traceability-browser.cjs <ignored-folder>`. Dùng Chrome profile fixture riêng, không điều khiển profile cá nhân.
- Bài chạy đối chiếu dev@`ac5395c5`; Dương Order@`20322d1c` vẫn là phần pricing/checkout riêng chưa merge vào dev tại lần fetch28/11. Khi ghép nhánh đó, chạy lại regression trước release; không sửa core canonicalizer hoặc migration000013 để né conflict.
- Các phiếu owner được sinh lại từ source commit đã ghim; phần đã có trên nhánh này là đầu vào tích hợp, không giao owner code trùng. Feature flags, migration/regrant và cấu hình TLS theo runbook report11; review/merge/release do nhóm thực hiện.

## Giới hạn thiết bị

In và QR được kiểm bằng browser/giải mã ảnh; chưa có máy in và điện thoại vật lý nối tới máy này để nghiệm thu phần cứng. Điều này không được mô tả thành ca camera/in giấy đã chạy.
