# Analytics Service · MS-11

Dashboard Order v1 đọc mốc PAID và COMPLETED từ Kafka, lưu checkpoint trong PostgreSQL và phục vụ Dashboard cùng drill-down từ snapshot ClickHouse. [Thiết kế MS-11](../../docs/02_architecture/services/analytics-service/README.md)

## Chạy kiểm thử cô lập

Từ thư mục gốc repo, dùng project Compose riêng trên Docker Desktop hoặc Linux.

```powershell
docker compose -p analytics-local -f services/analytics-service/compose.test.yml build api tests
docker compose -p analytics-local -f services/analytics-service/compose.test.yml run -T --name analytics-local-tests tests
docker cp analytics-local-tests:/tmp/analytics-pytest.xml analytics-pytest.xml
docker compose -p analytics-local -f services/analytics-service/compose.test.yml down --volumes --remove-orphans
```

API local bind loopback trên port 18011. Test tạo ngày dữ liệu có thể đối soát, kiểm API HTTP và dùng fixture Identity có khóa ký tạm thời. Kafka, PostgreSQL và ClickHouse chạy thật. Chỉ dùng lệnh xóa volume ở trên cho project test cô lập.

## Pilot dev

Dùng image build với context gốc repo và `-f services/analytics-service/Dockerfile`. Set `ANALYTICS_IMAGE` bằng image digest, `ANALYTICS_DEV_NETWORK` bằng network private có Kong và `ANALYTICS_DEV_CONFIG` bằng thư mục cấu hình ngoài repo. Tạo `migration/service.env`, `collector/service.env`, `builder/service.env`, `api/service.env`. Mỗi container chỉ mount thư mục cấu hình của role tương ứng tại `/run/analytics`.

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
