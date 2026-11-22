# MS-11 — Kế hoạch kiểm thử và nghiệm thu

Các scenario dưới đây là tiêu chí cho implementation. Bắt đầu với fixture nhỏ có kết quả tính tay; sau đó chạy trên Kafka, PostgreSQL, ClickHouse và producer thật cùng phiên bản/cấu hình dự kiến. Ghi SHA, schema version, feature flags, fixture và phạm vi của từng kết quả.

| Gate | Kịch bản | Điều kiện đạt |
| --- | --- | --- |
| Static contract | Parse OpenAPI, kiểm example và Markdown links, đối chiếu DDL với model | Các route, unit, quality, lỗi và generation khớp tài liệu. |
| Cohort và duplicate | Cùng event hai lần, cùng business key với ID mới, conflict digest, v1/v2/v3 cùng Order | Một PAID và một COMPLETED tối đa trong cohort được bật. Conflict tạo quarantine và ảnh hưởng coverage. |
| Checkpoint | Kill trước/sau DB commit, mất ACK Kafka, rebalance, offset mất retention, tái tạo topic | Không bỏ record, không đếm trùng. Cursor ngoài log range mở gap. Lease cũ không được commit dữ liệu. |
| Serving | Kill giữa các insert, timeout lúc publish, hai builder cùng scope, đọc qua replica, restore lệch | Chỉ generation đã đối soát được đọc. Stale writer không thay pointer. |
| Thời gian | Ngay trước/đúng biên 17:00Z, kỳ hôm nay, thiếu partition, late event, empty đầy đủ | Buckets theo ngày Việt Nam. Coverage thiếu ra partial/unavailable; zero chỉ ở ngày đã đủ. Drill-down cũ vẫn khớp snapshot cũ. |
| So sánh | Hai kỳ bằng nhau, mẫu số 0, một kỳ stale, metric version khác | Cùng ngày và scope. Mẫu số 0 chỉ có delta tuyệt đối; version khác hoặc coverage thiếu trả delta null. |
| Authorization | Token Customer, token staff thiếu quyền, header giả, snapshot ngoài scope, quyền bị thu hồi | 401/403/404 theo contract, không lộ dữ liệu hoặc sự tồn tại của snapshot ngoài scope. |
| Finance và Inventory | Refund lặp ở hai feed, thiếu Finance policy, batch sai HSD, snapshot tồn thiếu kho | Không cộng trùng. Card chưa duyệt unavailable. Expiry ghi chất lượng và đúng grain. |
| DSS | RFM guest/claim, churn thiếu dữ liệu, leakage forecast, FEFO nhiều batch, basket mẫu nhỏ, Tết lệch lịch | Đúng oracle theo [thiết kế DSS](../../02_architecture/services/analytics-service/06_dss_engines.md). Insight thiếu căn cứ không được bàn giao. |
| Tải và resilience | Truy vấn 90 ngày song song, backlog replay, Redis outage, down Kafka/OLAP | Quota và timeout hoạt động. Không tăng tải đồng bộ cho Order. Theo dõi độ mới, query p95, resource usage và RTO/RPO thực đo. |

## Fixture đối soát tối thiểu

- Kỳ hiện tại `2026-11-01..2026-11-02`, kỳ trước `2026-10-31..2026-11-01`. A PAID đúng `2026-10-31T17:00:00Z` và COMPLETED sau biên cuối kỳ. B PAID trước biên đầu 1 giây. C chỉ có COMPLETED trong kỳ, mốc PAID thuộc lịch sử ngoài hai kỳ.
- Gửi lại mốc A cùng event ID và thêm một duplicate tương đương khác ID. Đối soát hiện tại PAID = 1, COMPLETED = 1, kỳ trước PAID = 1, delta PAID = 0. Hoàn tiền của A tới sau không thay đổi hai card mốc.
- Thêm mốc PAID D đến muộn trong kỳ. Generation mới cho PAID = 2 và delta = 1. Drill-down ghim generation cũ vẫn PAID = 1.
- Chạy lại với coverage Order thiếu một partition. Mở quality gap, giữ zero chỉ ở phần đủ bằng chứng và kiểm lý do trên API.

## Cấp nghiệm thu

1. **Artifact** kiểm hợp đồng, schema, fixture, SQL candidate trên DB sạch.
2. **Infrastructure** kiểm behavior với stack test và fault injection.
3. **Owning peer** chốt contract, nguồn và coverage với Order/Finance/Inventory/Profile.
4. **Web Admin** nghiệm thu token, bộ lọc, drill-down, fallback và trạng thái từng card.

Kết quả cấp Artifact đủ để review thiết kế. Việc bật capability dùng evidence cấp Infrastructure và Owning peer, cộng Web Admin nếu có route UI.

## Kiểm tra artifact ngày 06/11/2026

| Kiểm tra | Kết quả |
| --- | --- |
| OpenAPI 3.1 | `openapi-spec-validator` xác nhận spec hợp lệ. Bảy ví dụ request parameter/response khớp schema và bảy ca giá trị theo quality đạt. Ví dụ Dashboard và series khớp generation, kỳ và tổng mốc. |
| Postman CLI 1.70.0 | Lint cục bộ với `--no-report-events --fail-severity error` báo `No issues found`. CLI yêu cầu đăng nhập để dùng tính năng workspace. |
| Schema nguồn | Hai fixture PAID/COMPLETED v3 trong repo khớp JSON Schema tương ứng. |
| DDL | PostgreSQL 16.15 tạo 9 bảng control. ClickHouse 25.8.33.6 ở chế độ local tạo 2 bảng OLAP. Hai phép kiểm chạy trên store sạch, cô lập. |
| Điều hướng | Kiểm tra đường dẫn và heading của các liên kết nội bộ trong các file Markdown có thay đổi. |

Các scenario Infrastructure, Owning peer và Web Admin vẫn chờ implementation và môi trường tích hợp.
