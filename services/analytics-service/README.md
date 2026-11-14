# Analytics Service · MS-11

Dashboard Order v1 đọc mốc PAID và COMPLETED từ Kafka, lưu checkpoint trong PostgreSQL và phục vụ Dashboard cùng drill-down từ snapshot ClickHouse. [Thiết kế MS-11](../../docs/02_architecture/services/analytics-service/README.md)

Recommendation v1 có Related ở trang chi tiết, Bought Together ở giỏ hàng và Personalized ở trang chủ, gồm category affinity và item-to-item CF. [Thiết kế runtime](../../docs/02_architecture/services/analytics-service/15_recommendation_persistence_and_delivery.md) · [Cách chạy nguồn, worker, Web và kiểm thử](../../docs/02_architecture/services/analytics-service/16_recommendation_runtime_and_verification.md). Router mặc định tắt; xem runbook để cấu hình role, scope, Identity và consent trước khi bật.

## Chạy kiểm thử cô lập

Từ thư mục gốc repo, dùng project Compose riêng trên Docker Desktop hoặc Linux.

```powershell
docker compose -p analytics-local -f services/analytics-service/compose.test.yml build api tests
docker compose -p analytics-local -f services/analytics-service/compose.test.yml run -T --name analytics-local-tests tests
docker cp analytics-local-tests:/tmp/analytics-pytest.xml analytics-pytest.xml
docker compose -p analytics-local -f services/analytics-service/compose.test.yml down --volumes --remove-orphans
```

API local bind loopback trên port 18011. Test tạo ngày dữ liệu có thể đối soát, kiểm API HTTP và dùng fixture Identity có khóa ký tạm thời. Kafka, PostgreSQL và ClickHouse chạy thật. Chỉ dùng lệnh xóa volume ở trên cho project test cô lập.

Để giữ một pilot local chạy nền, dùng project riêng `analytics-pilot-local`:

```powershell
docker compose -p analytics-pilot-local -f services/analytics-service/compose.test.yml up -d --build api collector builder
Invoke-WebRequest http://127.0.0.1:18011/readyz
docker compose -p analytics-pilot-local -f services/analytics-service/compose.test.yml ps
```

`readyz` chỉ chứng minh API nối được PostgreSQL và ClickHouse; Kafka chạy với topic fixture, Identity dùng tài khoản/khóa thử nghiệm. Đường Dashboard không có token phải trả 401. Pilot này không đọc Order/Inventory thật và không chứng minh số liệu kinh doanh. Dừng bằng `docker compose -p analytics-pilot-local -f services/analytics-service/compose.test.yml stop`; chỉ xóa volume của project này sau khi đã lưu bằng chứng cần thiết.

## Pilot dev

Dùng image build với context gốc repo và `-f services/analytics-service/Dockerfile`. Set `ANALYTICS_IMAGE` bằng image digest, `ANALYTICS_DEV_NETWORK` bằng network private có Kong và `ANALYTICS_DEV_CONFIG` bằng thư mục cấu hình ngoài repo. Tạo `migration/service.env`, `collector/service.env`, `builder/service.env`, `api/service.env`. Mỗi container chỉ mount thư mục cấu hình của role tương ứng tại `/run/analytics`.

Trước khi migrate hoặc bật API trên host shared dev, chạy `services/analytics-service/dev/preflight.ps1 -Supply` với các biến môi trường trên và `ANALYTICS_SUPPLY_INBOX`. Preflight kiểm image digest, network riêng, cấu hình ngoài repo, hai thư mục nguồn và Compose; không in secret hoặc khởi động container. Kết quả pass chỉ xác nhận hạ tầng tối thiểu, không thay cho đối soát dữ liệu và smoke test qua Kong.

`ANALYTICS_AUTH_MODE=pilot` giữ giới hạn UUID pilot. Khi chuyển sang `scoped`, Dashboard yêu cầu `ANALYTICS_DASHBOARD_VIEW#GLOBAL`, Supply yêu cầu `ANALYTICS_SUPPLY_VIEW#GLOBAL`. JWT phải mang `permission_grants` do Identity sync phát; phiên cookie dùng `permissionGrants` từ Identity. Quyền phẳng và tên role không mở API ở chế độ này. Chỉ dùng GLOBAL cho MVP một kho; scope WAREHOUSE chờ hai owner thống nhất ID kho canonical. Migration Identity V14 đăng ký hai permission và cấp cho role ADMIN seed; tạo role nghiệp vụ riêng qua IAM nếu cần.

| Admin route | Quyền ở `scoped` |
| --- | --- |
| `GET /dashboard`, `GET /metrics/{metricId}/series` | `ANALYTICS_DASHBOARD_VIEW#GLOBAL` |
| `GET /supply/workbench`, `GET /supply/velocity`, `GET /supply/expiry-risk` | `ANALYTICS_SUPPLY_VIEW#GLOBAL` |

Sau khi deploy Identity có V14, migration SQL trực tiếp **không tự chạy PermissionSync**. Role ADMIN được bảo vệ, nên `PUT /api/v1/admin/roles/{roleId}/permissions` sẽ bị từ chối. Với mỗi principal ADMIN cần dùng Analytics, người có `PERMISSION_MANAGE` gọi `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions`, rồi yêu cầu nhân viên đăng nhập lại. Kiểm token mới có hai `ANALYTICS_*#GLOBAL`; ADMIN seed có 37 permission thay vì 35. Token cũ giữ grant cũ cho đến khi hết hạn hoặc được cấp lại. Chưa đồng bộ xong thì chưa bật `scoped` cho nhân viên đó.

| Role | Biến cần cấu hình |
| --- | --- |
| Migration | `ANALYTICS_POSTGRES_DSN_FILE`, `ANALYTICS_POSTGRES_WORKER_ROLE`, `ANALYTICS_POSTGRES_API_ROLE`, `ANALYTICS_CLICKHOUSE_HOST`, `ANALYTICS_CLICKHOUSE_USER`, `ANALYTICS_CLICKHOUSE_PASSWORD_FILE` |
| Collector và builder | `ANALYTICS_POSTGRES_DSN_FILE`, `ANALYTICS_PIPELINE`, `ANALYTICS_SCOPE`. Collector thêm `ANALYTICS_KAFKA_BROKERS`, `ANALYTICS_KAFKA_OPTIONS_FILE` nếu dùng TLS/SASL. Builder thêm cấu hình ClickHouse với role ghi. |
| API | DSN và ClickHouse với role chỉ đọc, cùng pipeline/scope. Thêm `ANALYTICS_IDENTITY_URL`, `ANALYTICS_JWT_ISSUER`, `ANALYTICS_JWT_PUBLIC_KEY_FILE`, `ANALYTICS_PILOT_IDS` (UUID phân cách bằng dấu phẩy), `ANALYTICS_JWT_AUDIENCE` nếu khác `omama-api`. |

Đặt `ANALYTICS_CLICKHOUSE_SECURE=1` và `ANALYTICS_CLICKHOUSE_PORT=8443` nếu dùng HTTPS. Tạo role PostgreSQL trước migrate. Role ClickHouse của API có SELECT, role builder có SELECT, INSERT và ALTER DELETE trên database Analytics. Migration cần quyền tạo database/bảng. Kafka cần Describe/Read cho `order.events.v3` và quyền nhóm của pipeline. Bản đầu dùng topic có `cleanup.policy=delete` và producer Order không giao dịch Kafka, nên offset nhảy bất thường sẽ khóa coverage.

1. Chạy `docker compose -f services/analytics-service/compose.dev.yml run --rm migrate`. Dùng cùng Compose file để chạy các lệnh bên dưới với service `collector` hoặc `builder`.
2. Chạy `run --rm collector topic` để xác minh topic ID và partitions. Lưu manifest bootstrap gồm `topicId` và `offsets` theo bằng chứng Order. Mount manifest trong thư mục collector và chạy `run --rm collector bootstrap /run/analytics/bootstrap.json`. Khởi động `up -d collector builder`, theo dõi `run --rm collector status`.
3. Đối soát ngày đã đóng với Order. Coverage manifest bổ sung `businessDate`, `sourceAsOf` có timezone, `counts` với `PAID` và `COMPLETED`, `approvedBy` và `evidenceRef`. Dùng `run --rm collector coverage /run/analytics/coverage.json`, rồi `run --rm builder build-once`. Topic và partitions phải khớp, journal phải đọc tới các checkpoint trong manifest và số đếm phải khớp. Xác nhận riêng từng ngày. Freshness mặc định 24 giờ kể từ lần xác nhận. Nộp lại cùng bằng chứng giữ nguyên thời điểm xác nhận.
4. Sau khi đủ bằng chứng nguồn và pilot, chạy `--profile pilot up -d api` và thêm [đoạn cấu hình Kong pilot](dev/kong-pilot-service.yml) vào cấu hình riêng của dev. Smoke test qua Kong với token thật. Ghi image digest, checkpoint và kết quả đối soát.
5. Khi cần rollback, gỡ route pilot hoặc dừng builder và dùng `run --rm builder rollback <generation-uuid>`. Chỉ chọn snapshot cùng scope còn hiệu lực. Worker và journal duy trì checkpoint để tiếp tục sau khi xử lý.

Dùng `repair <topic-id> <partition> <offset> <evidence-ref>` để đọc lại bản ghi gốc khi adapter đã xử lý được. Dùng `--discard` sau khi source owner duyệt bỏ bản ghi. Topic đổi ID hoặc offset không còn trong retention cần pipeline/bootstrap mới theo bằng chứng nguồn. Xác nhận coverage lại sau khi sửa.

## Pilot tốc độ bán và cận hạn

Pilot dev chạy trên môi trường phát triển chung với Order, Inventory và Identity của nhóm. Chọn một kho và vài SKU có 14 ngày Việt Nam đã đóng và đối soát. Stack `compose.test.yml` là môi trường cô lập với dữ liệu mẫu để kiểm đường truyền và công thức.

1. Hoàn tất migration PostgreSQL và ClickHouse theo phần trên. Cấu hình `ANALYTICS_SUPPLY_ENABLED=1` và `ANALYTICS_SESSION_ORIGIN` bằng origin chính xác của Web Admin ở role API. Cấp tài khoản nhân viên `ACTIVE`, role `ADMIN` phạm vi `GLOBAL` và thêm UUID vào `ANALYTICS_PILOT_IDS`. Tài khoản mở màn hình cần cả `INVENTORY_MANAGEMENT_VIEW` và `BATCH_VIEW`.
2. Đặt `ANALYTICS_SUPPLY_INBOX` là thư mục chia sẻ ngoài repo với hai thư mục con `order-service` và `inventory-service`. Cấp quyền ghi riêng cho owner của từng thư mục. Cấp quyền đọc qua nhóm chung hoặc ACL để UID 10001 đọc được file mode 0640. Worker Analytics mount thư mục chỉ đọc.
3. Chạy publisher ở phía từng owner với `psql`, Python và role đọc PostgreSQL của owner. Scheduler chạy Order và Inventory daily sau khi bằng chứng của ngày đã đóng sẵn sàng. Chạy snapshot lô theo giờ với revision tăng dần trong ngày. Lần chạy lại cùng revision và nội dung giữ nguyên thời điểm chụp đã công bố. Dùng secret store hoặc cơ chế libpq để cấp credential cho job. [Cách lập manifest cho từng owner](../../docs/02_architecture/services/analytics-service/09_supply_analytics_v1.md)
4. Bật API, worker và [route Kong](dev/kong-pilot-service.yml). Trên Web Admin cấu hình `ANALYTICS_ADMIN_UPSTREAM_URL` bằng URL nội bộ đầy đủ tới prefix Analytics, ví dụ `http://kong:8000/api/v1/admin/analytics`. Có thể đặt `ANALYTICS_PILOT_WAREHOUSE` để điền sẵn mã kho. Màn hình ở `/admin/analytics`.

```powershell
docker compose -f services/analytics-service/compose.dev.yml --profile supply --profile pilot up -d supply-sync api
```

Worker quét mặc định mỗi 60 giây, điều chỉnh bằng `ANALYTICS_SUPPLY_SYNC_INTERVAL_SECONDS`. Đọc metrics nội bộ ở port 8014. `analytics_supply_inbox_available` bằng 0 báo thư mục không khả dụng. `analytics_supply_source_as_of_seconds` theo loại bản xuất bằng 0 khi worker chưa nhận dữ liệu và có thể dùng để theo dõi độ trễ theo lịch owner. Theo dõi `analytics_supply_files_total` với `result="rejected"`, `"read_error"` và `"retry"`. Trên Web Admin kiểm cột coverage từng SKU và lý do không tính được rủi ro. Khi cần backfill thủ công, mount file đã đối soát vào cấu hình collector và chạy.

```powershell
docker compose -f services/analytics-service/compose.dev.yml run --rm collector import-supply /run/analytics/supply-export.json
```

Các truy vấn Supply hỗ trợ JWT staff đã có và phiên Identity của Web Admin khi bật origin.

- `GET /api/v1/admin/analytics/supply/workbench?warehouseId=warehouse-hue&from=2026-10-18&to=2026-11-01&horizonDays=14`
- `GET /api/v1/admin/analytics/supply/velocity?skuCode=SKU-01&warehouseId=warehouse-hue&from=2026-10-25&to=2026-11-01&comparison=previous_period`
- `GET /api/v1/admin/analytics/supply/expiry-risk?skuCode=SKU-01&warehouseId=warehouse-hue&horizonDays=30`

[Định nghĩa chỉ số và điều kiện dữ liệu](../../docs/02_architecture/services/analytics-service/09_supply_analytics_v1.md)
