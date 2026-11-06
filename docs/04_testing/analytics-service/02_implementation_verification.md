# MS-11 — Xác minh runtime Dashboard Order v1

**Ngày 06/11/2026 · Nhánh `feat/analysis-service` · Baseline `89c6d64`.**

## Môi trường và bằng chứng

- Docker Compose cô lập chạy Python 3.12, Kafka 3.8.0, PostgreSQL 16.15 và ClickHouse 25.8.33.6. Source events dùng fixture bám schema Order v3 của repo. HTTP Identity dùng fixture riêng với khóa ký RSA sinh trong container.
- Lượt cuối trên stack sạch đạt **12/12 test** trong 51,74 giây. Mốc coverage được kiểm cả tại cutoff đã xác nhận lẫn checkpoint hiện hành. Nộp lại cùng manifest giữ nguyên approval.
- Ruff check, Ruff format, kiểm OpenAPI 3.1 và đối chiếu schema với response runtime. Postman CLI báo `No issues found` ở chế độ lint cục bộ.
- Docker build tạo image chạy API, collector và builder. Migration áp trên hai store sạch với role test riêng. Workflow GitHub Actions dùng cùng Compose và ghi JUnit vào artifact.
- Cấu hình Compose dev và Kong được kiểm tra cú pháp. Smoke qua Kong cục bộ trả 200 cho Dashboard và daily series với token pilot hợp lệ, 401 khi thiếu token và 404 cho đường dẫn ngoài hai route.

## Kết quả hành vi

| Nhóm | Bằng chứng |
| --- | --- |
| Kafka và checkpoint | Phát lại cùng event ID và phát lại cùng mốc bằng ID mới vẫn giữ một PAID. Consumer rebalance qua hai instance. Kiểm rollback transaction, phát lại sau commit, lease fencing và offset gap. |
| Coverage và dữ liệu đến muộn | Kiểm ngày Việt Nam, zero có coverage, partial và unavailable. Sau khi nhập lại bằng chứng cho ngày có PAID đến muộn, Dashboard đổi từ 1 lên 2. Snapshot cũ vẫn trả 1. |
| Snapshot | Mất ACK khi ghi ClickHouse và hai builder cạnh tranh giữ con trỏ an toàn. Rollup của generation ghim bị mất trả 503. |
| API và bảo mật | Schema Dashboard, series và lỗi khớp hợp đồng. Customer và Warehouse Admin nhận 403, Identity fixture lỗi trả 503, header giả không thay thế bearer token. |

## Kiểm tra trước PR vào dev

- Chạy lại bộ Analytics trên stack sạch với base `89c6d64`. **12/12 test đạt**. Smoke qua cấu hình Kong ghép với cấu hình Gateway hiện có đạt 200, 401, 403 và 404 theo từng ca.
- Bộ contract MVP đạt với 42 OpenAPI spec, 52 event schema, 15 file protobuf và các kiểm tra codegen/wire. Kiểm contract thiết kế Order cũng đạt.
- `go vet ./...` và `go test -race -count=1 ./...` của Order hoàn thành với 271 test/subtest đạt. Hai ca live Inventory TLS và Keycloak được skip vì stack sandbox chưa cấu hình hai provider đó.
- Phát lại payload PAID và COMPLETED lấy từ outbox của runtime Order trong test Connected Checkout vào Kafka của Analytics. Journal Analytics ghi đúng một mốc mỗi loại cho cùng Order, không tăng quarantine. Bằng chứng này kiểm tương thích producer/consumer trong sandbox.

## Điều kiện còn chờ trên dev

| Điều kiện | Việc cần xác minh hoặc cung cấp |
| --- | --- |
| Nguồn Order v3 | Kiểm tra deployment Order dev thực sự phát cả PAID và COMPLETED trên `order.events.v3` qua luồng thanh toán đến hoàn tất. Code Order trên nhánh `dev` có publisher v3, còn profile Order Core phụ thuộc SePay TEST runtime. Xác nhận topic ID, partition, retention và ACL đọc/topic/group cho Analytics. |
| Hạ tầng Analytics | Bootstrap repo có tên database `om_analytics_db`. Cần xác nhận database có trên dev, cấp role migration/worker/API, và cung cấp ClickHouse riêng hoặc dùng chung với database/role Analytics riêng. Compose infra chung trong repo chưa khai báo ClickHouse. |
| Coverage | Order owner cung cấp offset bootstrap và manifest theo ngày đã đóng, với cutoff từng partition, số PAID/COMPLETED và tham chiếu đối soát. Dùng group và scope Analytics riêng cho giai đoạn chạy đối soát. |
| Gateway và Identity | Chuẩn bị image theo digest, cấu hình trên private network chung với Kong, JWT issuer/key/audience và tài khoản `ACTIVE GLOBAL ADMIN` trong danh sách pilot. Mở fragment route sau khi số liệu khớp và chạy smoke bằng token Identity dev thật. |

Bốn card ngoài Order vẫn trả `UNAVAILABLE` theo phạm vi Dashboard Order v1. Mở các card này cần nguồn Inventory, Customer và Finance được duyệt riêng.

[Hướng dẫn chạy lại và triển khai pilot](../../../services/analytics-service/README.md)
