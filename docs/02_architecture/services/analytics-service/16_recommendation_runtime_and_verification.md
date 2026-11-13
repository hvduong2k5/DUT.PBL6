# 16 — Recommendation runtime và kết quả kiểm chứng

**07/11/2026 · nhánh feat/analysis-service · môi trường cô lập.**

[Thiết kế và giới hạn bản pilot](15_recommendation_persistence_and_delivery.md) · [OpenAPI](../../../03_api_specs/analytics-recommendation.candidate.openapi.yaml) · [JSON Schema](../../../03_api_specs/analytics-recommendation.v1.schema.json)

## 1. Đã triển khai

- MS-11: API ba placement; related content score, basket support/confidence/lift, category affinity và item-to-item cosine có shrinkage.
- Customer auth qua Identity, consent version/idempotency, feedback VIEW_ITEM/ADD_TO_CART; purchase chỉ từ Order, nối account bằng principal đã xác minh.
- Source publisher/SQL cho Catalog, Inventory, Order; manifest/checksum/revision, atomic publish và worker import/build/cleanup.
- PostgreSQL schema riêng, role API/worker riêng; generation publish có source/privacy fencing, rollback và CLI status.
- Web BFF có trusted credentials, CSRF/origin, timeout; rails ở product/cart/home, consent controls và GA4 measurement opt-in.
- Cấu hình exact Gateway route; opt-in worker trong Compose dev. Chưa triển khai lên dev chung.

## 2. Chạy kiểm thử backend

Từ root repo, cần Docker. Project test phải có PostgreSQL/Kafka/ClickHouse mới; các bài Dashboard/Supply dùng fixtures cố định và không chạy lặp trên database đã seed.

~~~powershell
$env:ANALYTICS_TEST_HTTP_PORT = '18121'
$env:ANALYTICS_TEST_IDENTITY_PORT = '18126'
$env:ANALYTICS_REC_TEST_PORT = '18122'
docker compose -p pbl-recommendation-ci -f services/analytics-service/compose.test.yml build api tests
docker compose -p pbl-recommendation-ci -f services/analytics-service/compose.test.yml run --rm --no-deps --entrypoint ruff tests check --no-cache src tests scripts
docker compose -p pbl-recommendation-ci -f services/analytics-service/compose.test.yml run --rm --no-deps --entrypoint ruff tests format --no-cache --check src tests scripts
docker compose -p pbl-recommendation-ci -f services/analytics-service/compose.test.yml run --name pbl-recommendation-ci-tests tests
~~~

Muốn chạy lại toàn bộ: thu evidence trước rồi dọn **chỉ project test do mình tạo** bằng compose down --volumes --remove-orphans; không chạy lệnh dọn trên project dev. CI đã dùng stack riêng và cleanup ở cuối workflow.

Bộ test Recommendation có thể chạy lại riêng trên stack đang chạy: pytest các file test_recommendation*.py. Mỗi bài nguồn/engine tạo database UUID riêng và xóa đúng database đó trong teardown. test_recommendation_http.py gọi các HTTP runtime thật của Compose.

## 3. Pilot HTTP và trình duyệt local

Dùng project riêng với các port trong compose.test.yml: API Analytics 18111, Recommendation 18112, Identity 18116. Các cookie/key/password bên dưới là fixture công khai, chỉ dành stack cô lập.

~~~powershell
$env:ANALYTICS_TEST_HTTP_PORT = '18111'
$env:ANALYTICS_TEST_IDENTITY_PORT = '18116'
docker compose -p pbl-recommendation -f services/analytics-service/compose.test.yml up -d api recommendation-api collector builder
docker compose -p pbl-recommendation -f services/analytics-service/compose.test.yml run --rm --no-deps --entrypoint python tests tests/recommendation_pilot.py --watch
~~~

Giữ seed process chạy: cứ 30 giây công bố 6 SKU, sellability và 20 basket rồi build. Script dùng worker credential thật; cấp consent qua HTTP của API trước khi tạo purchase signal sau grant. **Chỉ chạy trên Compose fixture**, không chạy script này trên dev thật.

Một terminal khác chạy owner HTTP simulation, không mock Recommendation:

~~~powershell
postman simulation run postman/simulations/recommendation/recommendation.sim.yaml --port 18117 --no-history --output ndjson
~~~

Simulation dùng các response shape của Web contracts đã có trong repo, proxy Identity tới fixture. Có cart state theo account, unknown product 404. Postman CLI local đang signed out nên bỏ qua tìm mock của owner trên cloud. Review API của service thật vẫn là gate dev riêng.

Tại apps/web-user, đặt env:

~~~dotenv
NEXT_PUBLIC_RECOMMENDATIONS_ENABLED=true
ANALYTICS_RECOMMENDATION_UPSTREAM_URL=http://127.0.0.1:18112
WEB_AUTH_MODE=identity-bff
WEB_PUBLIC_URL=http://127.0.0.1:13046
WEB_OIDC_ALLOW_LOCAL_HTTP=true
IDENTITY_UPSTREAM_URL=http://127.0.0.1:18117/upstreams/api/v1
CUSTOMER_CORE_UPSTREAM_URL=http://127.0.0.1:18117/upstreams/api/v1
CUSTOMER_EXTENSIONS_UPSTREAM_URL=http://127.0.0.1:18117/upstreams/api/v1
~~~

Chạy npm run dev -- --hostname 127.0.0.1 --port 13046. Từ root repo:

~~~powershell
npx --yes --package @playwright/cli playwright-cli -s=recommendation open http://127.0.0.1:13046
npx --yes --package @playwright/cli playwright-cli -s=recommendation run-code --filename scripts/verify-recommendation-browser.js
npx --yes --package @playwright/cli playwright-cli -s=recommendation close
~~~

Browser script kiểm ba rail, server nhận view feedback, impression một lần sau visibility, click/add provenance, revoke chuyển fallback, CSRF sai bị chặn và rail lỗi 503 không chặn trang sản phẩm. Đo lường dùng gtag spy ở trình duyệt, không giả lập đã gửi đến Google.

Load test, khi seed process vẫn đang chạy:

~~~powershell
services/analytics-service/.venv/Scripts/python.exe services/analytics-service/tests/recommendation_http_pilot.py --duration 600 --rps 20 --output output/recommendation/load.json
~~~

API test tăng query rate limit lên 100.000/phút; cấu hình thường vẫn là 60/phút. Phép đo dùng hai Customer, chia đều ba placement; fallback được coi hợp lệ khi rebuild/consent yêu cầu, còn cart không được trả SKU anchor.

## 4. Bật trên dev có nguồn thật

Tạo scope JSON từ [example](../../../../services/analytics-service/dev/recommendation-scope.json), thay kho/scope/Supply alias và identityMappingVersion bằng giá trị owner đã xác nhận. Không suy ra D2C từ tất cả đơn COMPLETED; cohort phải là order-core-1 và channel D2C_WEB. Các đơn không dùng thống kê đưa vào excludedOrderIds.

Owner chạy publisher bằng Python và psql; cung cấp libpq credentials read-only qua PGHOST/PGPORT/PGDATABASE/PGUSER và pgpass hoặc secret manager. Mỗi command chỉ kết nối DB của chính owner. Root outbox dành riêng scope, có ba thư mục owner.

~~~powershell
python scripts/publish-analytics-recommendation.py catalog scope.json E:/analytics-exports/d2c-hue --watch
python scripts/publish-analytics-recommendation.py sellability scope.json E:/analytics-exports/d2c-hue --watch
python scripts/publish-analytics-recommendation.py completed_baskets scope.json E:/analytics-exports/d2c-hue --watch
~~~

Mỗi command chạy trong môi trường credential của owner tương ứng, không dùng chung một tài khoản. Chu kỳ mặc định lần lượt 300/60/900 giây, có --interval để điều chỉnh. Có thể dùng scheduler/supervisor của owner gọi một lần thay --watch. Theo dõi exit/error, source lag và retention outbox tối đa 24 giờ theo quy trình owner.

Chuẩn bị thư mục config ngoài repo:

- migration/service.env: role migrator hiện có, thêm ANALYTICS_REC_POSTGRES_WORKER_ROLE và ANALYTICS_REC_POSTGRES_API_ROLE.
- recommendation/service.env: REC_POSTGRES_DSN_FILE trỏ worker role, REC_SUBJECT_KEY_FILE, REC_SCOPE_CONFIG_FILE; các đường dẫn là bên trong /run/analytics.
- api/service.env: ANALYTICS_REC_ENABLED=1, DSN Recommendation của API role, cùng scope/subject key namespace, Identity settings và ANALYTICS_SESSION_ORIGIN.
- API và worker dùng cùng secret HMAC, keyVersion và mapping; không log secret. Nếu dùng fallback file, dùng cùng file đã Catalog duyệt. File chỉ chứa object SKU → integer rank.

Ví dụ env worker (mọi tên biến đều có prefix ANALYTICS_):

~~~dotenv
ANALYTICS_REC_POSTGRES_DSN_FILE=/run/analytics/rec-dsn
ANALYTICS_REC_SUBJECT_KEY_FILE=/run/analytics/rec-subject-key
ANALYTICS_REC_SCOPE_CONFIG_FILE=/run/analytics/scope.json
ANALYTICS_REC_FALLBACK_POLICY_FILE=/run/analytics/fallback.json
~~~

~~~powershell
docker compose -f services/analytics-service/compose.dev.yml run --rm migrate
docker compose -f services/analytics-service/compose.dev.yml --profile recommendation --profile pilot up -d recommendation-sync api
docker compose -f services/analytics-service/compose.dev.yml run --rm recommendation-sync recommendation-status
docker compose -f services/analytics-service/compose.dev.yml run --rm recommendation-sync recommendation-build-once
docker compose -f services/analytics-service/compose.dev.yml run --rm recommendation-sync recommendation-cleanup
docker compose -f services/analytics-service/compose.dev.yml run --rm recommendation-sync recommendation-rollback GENERATION_UUID
~~~

Đặt ANALYTICS_REC_EXPORT_ROOT ở host thành outbox có thật. Compose mount read-only và không tự tạo thư mục thiếu. Nhập source tự động trong vòng sync; không có CLI import-recommendation riêng.

API Web cấu hình upstream nội bộ tới host root của MS-11 hoặc Kong, không thêm /api/v1 vào ANALYTICS_RECOMMENDATION_UPSTREAM_URL. NEXT_PUBLIC_RECOMMENDATIONS_ENABLED được chốt lúc build Web. GA4 thêm NEXT_PUBLIC_GA4_MEASUREMENT_ID của property dev; mặc định không có tracking nếu chưa opt-in.

Đoạn routes trong services/api-gateway/config/kong.yml dành /api/v1/products/recommended chính xác cho MS-11; prefix sản phẩm khác giữ Catalog. Trước rollout kiểm không có product slug “recommended” trong Catalog, vì đường dẫn này đã dành cho API. Shared dev cần smoke test qua gateway và kiểm rate limit phù hợp.

## 5. Evidence và giới hạn

| Kiểm chứng | Kết quả |
| --- | --- |
| Full backend trên stack mới | 52 passed; gồm Dashboard/Supply regression, Recommendation và owner exporter SQL |
| Recommendation trên code cuối | 28 passed; thuật toán, contracts, sources, account purchase mapping, consent/concurrency và HTTP/DB permissions |
| Ruff trong test image | Check/format pass |
| Web verify:mvp | Typecheck/lint pass; 145 tests trên 46 files pass |
| Web production build | Pass |
| Playwright script | Pass cả ba placement, feedback, measurement spy, revoke, CSRF và lỗi rail |
| Load 20 RPS × 600 giây | 12.000 HTTP 200; invalidResults 0; p95 47 ms, p99 63 ms, trên fixture nhỏ |
| Smoke load code cuối, 20 RPS × 30 giây | 600 HTTP 200; invalidResults 0; p95 47 ms, p99 62 ms |
| Liên kết tài liệu / Gateway config | 173 local links hợp lệ; exact-path regex và YAML đã kiểm, chưa chạy Kong dev |

Load chạy với sync/build fixture đồng thời. Các thay đổi tiếp theo về source fencing có smoke test riêng; không dùng số đo này để cam kết latency ở quy mô tối đa.

Artifacts local ở output/recommendation/ (load.json, pytest.xml) và output/playwright/ (ba ảnh rail). Không có dữ liệu khách hàng thật trong fixture. Warning duy nhất backend là Starlette báo deprecation của httpx TestClient; integration network tests vẫn dùng HTTP runtime riêng.

Chưa xác nhận: owner dev reconciliation, quality ranking trên tập thật/time split, ảnh Catalog public URL, delivery tới property GA4 và shared-dev rollout. Không có credential/property dev trong session để nghiệm thu các phần này. Các bài owner SQL dùng migration thật + dữ liệu relational synthetic; không phải đã thực hiện checkout/payment/fulfillment thật của từng service.

