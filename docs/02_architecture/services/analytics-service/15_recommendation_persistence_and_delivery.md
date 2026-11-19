# 15 — Persistence, vận hành và bàn giao Recommendation

**Hiện thực pilot v1 · 07/11/2026.** [Engine](12_recommendation_engine_design.md) · [Nguồn](13_recommendation_source_contracts.md) · [API/Web](14_recommendation_api_and_storefront.md) · [Runbook](16_recommendation_runtime_and_verification.md)

Chương này mô tả code đang có. Các chương 12–14 giữ đặc tả nghiệp vụ; những phương án scale lớn trong đó chưa mặc nhiên là chức năng đã triển khai. MS-11 phục vụ đủ Related, Bought Together và Personalized (category affinity + item-to-item CF), với nguồn FILE_JSONL của pilot.

## 1. PostgreSQL và ClickHouse

Migration [004_recommendation.sql](../../../../services/analytics-service/migrations/postgres/004_recommendation.sql) tạo schema riêng `analytics_recommendation`. Connection Recommendation đặt search_path cố định, timeout truy vấn 5 giây và credential riêng. Không thay migration Dashboard/Supply cũ.

| Bảng thực tế | Key / vai trò |
| --- | --- |
| source_snapshot | snapshot UUID; unique scope/dataset/revision; manifest, checksum, records JSONB, cutoff/received_at |
| source_pointer | scope/dataset → snapshot đã validate đầy đủ |
| consent | subject HMAC; enabled, version, policy, granted_at/revoked_at |
| consent_request | subject/idempotency UUID → request digest và created_at |
| privacy_state | singleton id=1, epoch tăng khi revoke |
| erasure | subject → requested_at, through_version; xóa được retry qua transaction |
| behavior_event | subject/event UUID; digest, consent version, scope, SKU, type, occurred/effective/received time |
| generation | UUID; input digest, scope, source vector, privacy epoch, built_at và payload JSONB |
| serving_pointer | một bundle generation cho mỗi scope |
| decision_trace | recommendation UUID, scope/placement/generation/strategy, SKU đã trả; không ghi subject |

Source projections lưu nguyên tập records đã kiểm tra trong JSONB. Basket records loại raw principal trước khi commit; chỉ thêm HMAC + consentVersion nếu có consent và completedAt sau grant. Payload generation chứa basket neighbors và item neighbors; related được tính từ Catalog hiện hành, personal history được tính trong cửa sổ 30 ngày.

Đây là lựa chọn pilot giảm số bảng và bước vận hành, không bỏ ba thuật toán. Không có các bảng catalog_sku, subject_profile hoặc neighbor riêng, không có recommendation pair tables trên ClickHouse ở bản này. Dashboard/Supply tiếp tục dùng ClickHouse như trước. Khi mở rộng phải benchmark với kích thước thật rồi mới chọn normalized projections/ClickHouse aggregation.

Public query không tải toàn bộ basket export vào Python. Home chỉ lấy basket của subject đang xác minh bằng JSON path trong PostgreSQL; truy vấn này vẫn có thể quét JSONB và cần đo lại khi tăng dữ liệu.

### 1.1. Role

Migration runner nhận `ANALYTICS_REC_POSTGRES_WORKER_ROLE` và `ANALYTICS_REC_POSTGRES_API_ROLE`. Tạo role trước khi migrate.

- Worker: SELECT Recommendation, ghi/xóa source, models, erasure, events/TTL và pointers; chỉ đọc consent.
- API: SELECT Recommendation và Supply export; INSERT/UPDATE consent, request idempotency, privacy, erasure, events và trace. Không được ghi source/model/pointers hay DELETE.
- Owner exporter: credential chỉ đọc database do owner vận hành, không dùng credential Analytics.

Kiểm thử Compose xác minh privilege thực tế. Owner không đọc chéo database của nhau; Analytics chỉ đọc file đã công bố.

## 2. Worker và vòng đời publish

| CLI thực tế | Công việc |
| --- | --- |
| recommendation-sync | Mỗi vòng: cleanup → scan/import → build, mặc định 60 giây |
| recommendation-build-once | Build một lần từ source pointers hiện hành |
| recommendation-status | Scope, pointer generation, revision/cutoff/count từng nguồn |
| recommendation-cleanup | Erasure và TTL |
| recommendation-rollback GENERATION_UUID | Chuyển bundle pointer cùng scope, generation dưới 24 giờ, epoch còn hợp lệ và nguồn đủ mới |

Không có process `recommendation-build` riêng trong bản pilot. API gắn router khi `ANALYTICS_REC_ENABLED=1`; worker chỉ chạy khi operator khởi động command tương ứng.

### 2.1. Source snapshots

Owner publisher dùng PostgreSQL transaction repeatable-read/read-only, timezone UTC. Ba SQL exporter đọc schema hiện có của Catalog, Inventory, Order. Publisher ghi JSONL canonical + manifest trong thư mục ẩn, fsync rồi rename nguyên thư mục. Revision counter và lock riêng owner/scope ngăn hai publisher cấp cùng revision. Mất counter khi đã có snapshot phải được đối soát, không tự reset.

Importer kiểm path/symlink, owner, scope/kho, thời gian, version mapping identity, record schema, sort/unique keys, count/hash. Chỉ nhận thư mục UUID có manifest. Source time không được ở tương lai; snapshot hết hạn bị từ chối. Giới hạn 16 KiB manifest, 64 MiB/file, 2 MiB/record, 1.000 Catalog/Inventory SKU và 100.000 basket.

Sau validate trong memory, transaction advisory lock theo scope/dataset serialize imports; privacy lock bảo vệ bước gắn subject. Insert snapshot + đổi source pointer cùng transaction. Cùng UUID/revision phải có cùng toàn bộ manifest và digest; revision cũ không ghi đè mới. File lỗi chỉ ghi mã lỗi đã giới hạn; không thay pointer.

HTTP export-job transport, durable staging leases và cursor recovery là phương án mở rộng, chưa triển khai. Worker hiện quét lại file; owner cần giữ outbox có giới hạn và xóa snapshot retired sau thời gian giữ đã thỏa thuận.

### 2.2. Model generations

Builder capture một repeatable-read snapshot gồm source vector, consent, events và privacy epoch. Tính pair statistics ngoài transaction giữ lock. Input digest gồm algorithm version, source vector, mapping version, privacy epoch, event IDs/digests, cutoff bucket 5 phút và fallback policy.

Khi publish: lock theo scope, khóa privacy_state, so epoch/source vector/pointer cũ, insert generation và chuyển pointer cùng transaction. Nếu hai builder tạo cùng input, dùng chung generation. Nếu source/privacy/pointer đã đổi, hủy publish với BUILD_INPUT_CHANGED để vòng sau thử lại. Crash trước commit không lộ generation dở dang.

Một bundle chứa basket và CF; query kiểm freshness/epoch riêng trước khi sử dụng từng thành phần. Không có ba pointer độc lập. Không có state BUILDING persist hoặc ClickHouse insert cần phục hồi trong bản pilot.

Budget: tối đa 1 triệu behavior events/build, tối đa 100 history SKU/subject, top 100 neighbor/anchor và 5 triệu pair updates cho mỗi thuật toán. Vượt budget dừng publish, giữ generation trước; API vẫn áp freshness và eligibility. Chưa benchmark quy mô tối đa.

## 3. Privacy, freshness và retention

Customer identity được xác minh qua JWT ký RS256 + Identity users/me, hoặc cookie UUID OMAMA_SESSION + Identity. Không nhận subject từ query/header tùy ý. Principal chỉ được HMAC với secret >=32 byte và keyVersion phía server. Home không có consent dùng danh sách fallback đã duyệt.

PUT consent yêu cầu If-Match và Idempotency-Key. Consent version serialize theo subject; revoke commit cùng privacy epoch và erasure. Event cũ/revoke version bị từ chối; regrant mở grantedAt mới. Không dùng mua hàng trước thời điểm consent để tái tạo profile. Erasure xóa event/linkage thuộc các version đã thu hồi, giữ dữ liệu Customer khác và dữ liệu phát sinh sau regrant.

API không dùng CF thuộc epoch cũ. Category profile của Customer còn consent vẫn tính được từ history hiện hành. Trace không chứa subject. Worker thực hiện erasure ở vòng kế tiếp, mục tiêu vận hành dưới 24 giờ; cần cảnh báo khi worker ngừng.

| Dữ liệu | Chính sách code hiện tại |
| --- | --- |
| Catalog / Inventory | Source max age 600 / 120 giây; hết hạn manifest cũng chặn serving |
| Sellability | Deadline từng SKU phải còn hiệu lực; thiếu SKU không được coi có hàng |
| Order | Max age 1.800 giây; stale loại basket/CF và purchase history, vẫn có baseline từ behavior hợp lệ |
| Model | Max age 86.400 giây |
| Behavior | 30 ngày theo effective time; future <=5 phút clamp về received time |
| Consent idempotency | 24 giờ |
| Decision trace | 7 ngày |
| Non-current source/model | Xóa sau 24 giờ; current pointers được giữ |

Không có audit journal 90 ngày riêng ở bản pilot. Giữ metadata hiện tại và trace theo bảng trên. Backup restore phải áp erasure/epoch mới nhất trước khi mở serving. Rotation HMAC đổi namespace cần kế hoạch chuyển đổi/rebuild; không tự ghép lịch sử hai key.

## 4. Cấu hình, health và đo tải

`setting()` hỗ trợ secret dưới dạng giá trị env hoặc biến cùng tên thêm hậu tố `_FILE`.

| Biến | Giá trị / ý nghĩa |
| --- | --- |
| ANALYTICS_REC_ENABLED | Opt-in router, mặc định tắt |
| ANALYTICS_REC_RELATED_ENABLED / BASKET_ENABLED / PERSONALIZED_ENABLED | Mặc định 1 khi đã bật router; đặt 0 để tắt riêng placement |
| ANALYTICS_REC_POSTGRES_DSN[_FILE] | Role riêng của API/worker |
| ANALYTICS_REC_SUBJECT_KEY[_FILE], REC_KEY_VERSION | Secret HMAC; namespace mặc định subject-v1 |
| ANALYTICS_REC_SCOPE_CONFIG_FILE | JSON scope/kho/cohort/identity mapping đã duyệt |
| ANALYTICS_REC_EXPORT_ROOT | Root exports read-only của worker |
| ANALYTICS_REC_FALLBACK_POLICY_FILE | JSON object SKU → positive rank do Catalog duyệt |
| ANALYTICS_REC_SYNC_INTERVAL_SECONDS | 60 giây; lịch exporter do owner lập riêng |
| ANALYTICS_REC_CATALOG_MAX_AGE_SECONDS | 600 |
| ANALYTICS_REC_INVENTORY_MAX_AGE_SECONDS | 120 |
| ANALYTICS_REC_ORDER_MAX_AGE_SECONDS | 1800 |
| ANALYTICS_REC_MODEL_MAX_AGE_SECONDS | 86400 |
| ANALYTICS_REC_QUERY_RATE_LIMIT | 60/phút theo IP, thêm Customer limit cho home; per-process |
| ANALYTICS_SESSION_ORIGIN | Origin Web chính xác cho cookie mutation |

Identity timeout 500 ms; query timeout BFF 1 giây; body mutations tối đa 4 KiB. Event limit 120/phút/subject; consent-write 10/phút/subject. Các limiter hiện là memory từng instance; gateway có limit chung riêng. Chỉ test load mới tăng query limit.

Metrics: API query duration/result strategy và event outcomes trên /metrics hiện hành; worker port 8015 có source cutoff, import outcomes và build duration. Label không chứa principal, subject hay event ID. Đặt cảnh báo source lag theo max age và cảnh báo worker scrape thất bại. Chưa có metric erasure lag hoặc readiness capability riêng; dùng status + query 503 và kiểm backlog bằng operator trước release.

Readiness Dashboard giữ cơ chế cũ. Source Recommendation mất freshness trả 503 cho query; BFF/Web ẩn rail khi lỗi, không chặn mua hàng.

## 5. Kiểm thử và gate phát hành

Bộ kiểm thử gồm thuật toán, schema/OpenAPI khớp runtime, snapshot conflict/stale/replacement, actual owner SQL trên migration thật, nối purchase principal, consent/idempotency, revoke/build race, parallel build, isolation và quyền runtime. Web có BFF unit tests và được kiểm tra trình duyệt với MS-11 thật, owner HTTP fixtures.

Mục tiêu load: 20 RPS trong 10 phút, chia đều ba placement, ít nhất hai Customer và worker refresh/build đồng thời; p95 <300 ms. Phép đo chỉ xác nhận fixture đã chạy, không ngoại suy tới 100.000 basket. [Evidence và cách chạy lại](16_recommendation_runtime_and_verification.md).

Các gate còn cần dữ liệu/hạ tầng thật:

1. Owner đối soát scope D2C/core, Inventory warehouse/batches/reservations và principal mapping.
2. Catalog duyệt fallback policy; endpoint ảnh cần public/signed URL do Catalog cung cấp. SQL pilot trả thumbnailUrl rỗng vì DB chỉ có object_key; UI dùng placeholder.
3. Catalog/Sales chấm top 6, time-split Recall@6/NDCG@6/coverage trên dữ liệu đủ mẫu; fixture chưa chứng minh chất lượng kinh doanh.
4. Property GA4 dev xác minh delivery/consent trong DebugView; test hiện xác minh lời gọi đo lường tại trình duyệt.
5. Cấp secrets, lập lịch export, cảnh báo và smoke test qua Kong trên dev.

Bật cờ trên dev sau đối soát; rollout related → basket → personalized theo gate nguồn tương ứng. Tắt cờ riêng khi lỗi hoặc rollback bundle còn hợp lệ. Chưa deploy môi trường chung, chưa mặc nhiên nghiệm thu các gate trên.

