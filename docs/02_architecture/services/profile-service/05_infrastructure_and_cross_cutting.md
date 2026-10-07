# HẠ TẦNG KỸ THUẬT & CƠ CHẾ PHÒNG VỆ XUYÊN SUỐT (CROSS-CUTTING CONCERNS)
## KIẾN TRÚC PHÂN TÁN, BỘ ĐỆM 2 TẦNG, TRANSACTIONAL OUTBOX & KHẢ NĂNG PHỤC HỒI
### MS-15: PROFILE SERVICE — MÈ XỬNG O MẠ OCOP

---

## 1. MÔ HÌNH BỘ ĐỆM 2 TẦNG (DUAL-LAYER CACHE ARCHITECTURE)

Nhằm đáp ứng yêu cầu khắc khe của chuỗi cung ứng thương mại điện tử (đặc biệt là luồng đặt hàng trực tiếp D2C gọi sang gRPC lấy địa chỉ mặc định với ngân sách **$P99 \le 5\text{ms}$**):

```text
               CLIENT (MS-04 CHECKOUT / REST API)
                               │
                               ▼
        ┌──────────────────────────────────────────────┐
        │        TẦNG 1: L1 IN-MEMORY LRU CACHE        │
        │        (Nằm trên RAM của Pod hiện tại)       │
        │  • Thư viện: hashicorp/golang-lru/v2         │
        │  • Tốc độ đọc: 421.9 ns/op (0.0004 ms)       │
        │  • Dung lượng: 5.000 phần tử                 │
        └──────────────────────┬───────────────────────┘
                               │
                       [L1 Cache Miss]
                               │
                               ▼
        ┌──────────────────────────────────────────────┐
        │        TẦNG 2: L2 DISTRIBUTED CACHE          │
        │             (Redis 7 Cluster)                │
        │  • Thư viện: redis/go-redis/v9               │
        │  • Tốc độ đọc: 0.8 ms - 1.2 ms qua mạng TCP  │
        │  • TTL: 15 phút (Profile), 30 phút (Address) │
        └──────────────────────┬───────────────────────┘
                               │
                       [L2 Cache Miss]
                               │
                               ▼
        ┌──────────────────────────────────────────────┐
        │            POSTGRESQL 16 DATABASE            │
        │  • Truy vấn B-Tree Index: 2.0 ms - 3.5 ms    │
        │  • Tự động Backfill lên cả L2 và L1          │
        └──────────────────────────────────────────────┘
```

### Đồng Bộ Xóa Cache Giữa Các Pod (Pub/Sub Sync Invalidation)
Khi một khách hàng cập nhật hồ sơ hoặc đổi địa chỉ mặc định trên Pod A:
1. Pod A xóa key tương ứng trong L1 của chính mình và gọi `DEL` trên Redis L2.
2. Pod A xuất bản thông điệp vào Redis Pub/Sub channel: `cache:invalidate:profile`.
3. Toàn bộ các Pod khác (Pod B, Pod C) đang lắng nghe channel này sẽ lập tức xóa key đó khỏi L1 của mình, triệt tiêu hoàn toàn rủi ro đọc dữ liệu cũ (Stale Data).

---

## 2. TRANSACTIONAL OUTBOX PATTERN & WORKER NGẦM

Để đảm bảo tính nhất quán tuyệt đối giữa Cơ sở dữ liệu và Kafka mà không cần tới cơ chế Two-Phase Commit (2PC) nặng nề:

```text
[HTTP REQUEST / GRPC] ──► Mở DB Transaction
                             ├── 1. Thực thi UPDATE / INSERT nghiệp vụ
                             ├── 2. INSERT sự kiện vào bảng outbox_events
                             └── 3. COMMIT Transaction thành công
                                         │
                                         ▼ (Dữ liệu đã nằm an toàn trong DB)
┌────────────────────────────────────────────────────────────────────────┐
│                   OUTBOX PUBLISHER BACKGROUND WORKER                   │
│                                                                        │
│  Lặp định kỳ (Polling Interval: 500ms):                                │
│  SELECT id, aggregate_id, event_type, payload, topic                   │
│  FROM outbox_events                                                    │
│  WHERE published_at IS NULL                                            │
│  ORDER BY created_at ASC                                               │
│  LIMIT 50                                                              │
│  FOR UPDATE SKIP LOCKED; ◄─── (Nhiều Pod chạy đồng thời không bị khóa) │
│                                                                        │
│  └──> Gửi sang Kafka Broker (segmentio/kafka-go)                       │
│  └──> Khi Kafka ACK thành công: UPDATE SET published_at = NOW()        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. GIÁM SÁT VIỄN TRẮC & CHỈ SỐ PROMETHEUS (OBSERVABILITY)

Service tích hợp sẵn endpoint chuẩn viễn trắc `/metrics`:

| Metric Name | Loại | Ý Nghĩa Kỹ Thuật | Ngưỡng Cảnh Báo (Alert Threshold) |
| :--- | :--- | :--- | :--- |
| `profile_http_requests_total` | Counter | Tổng lượt gọi API phân loại theo method, path, status | Tỷ lệ lỗi 5xx > 1% trong 5 phút |
| `profile_http_request_duration_seconds`| Histogram | Phân bố độ trễ xử lý HTTP request | $P99 > 200\text{ms}$ |
| `profile_address_cache_hit_total` | Counter | Đo lường tỷ lệ Hit/Miss của L1 và L2 Cache | Hit ratio tổng $< 80\%$ |
| `profile_outbox_lag` | Gauge | Số lượng sự kiện Outbox đang tồn đọng chưa gửi lên Kafka | Lag $> 500$ events liên tục trong 3 phút |
| `profile_fuzzy_match_duration_seconds` | Histogram | Thời gian thực thi giải thuật Levenshtein trọng số | $P95 > 20\text{ms}$ |

---

## 4. MA TRẬN SỰ CỐ PHÂN TÁN & CƠ CHẾ PHỤC HỒI (DISTRIBUTED FAILURE MATRIX)

```text
┌───────────────────────────┬──────────────────────────────────┬──────────────────────────────────────────┐
│ Thành Phần Gặp Sự Cố      │ Biểu Hiện & Tác Động             │ Cơ Chế Tự Phục Hồi & Đối Phó (Fallback)  │
├───────────────────────────┼──────────────────────────────────┼──────────────────────────────────────────┤
│ HashiCorp Vault KMS Down  │ Lỗi kết nối khi giải mã CCCD     │ • Luồng Checkout KHÔNG bị ảnh hưởng.     │
│                           │ HR không xem được CCCD nhân sự   │ • Circuit Breaker mở mạch sau 5 lỗi liên │
│                           │                                  │   tiếp trong 30s để bảo vệ Goroutines.   │
│                           │                                  │ • Trả về HTTP 503 kèm log cảnh báo rõ.   │
├───────────────────────────┼──────────────────────────────────┼──────────────────────────────────────────┤
│ Redis Cache Cluster Sập   │ Cache Miss 100%                  │ • Tự động Fallback đọc PostgreSQL 16.    │
│                           │ Độ trễ gRPC tăng nhẹ (~2.5ms)    │ • Nhờ B-Tree index, độ trễ vẫn nằm trong │
│                           │                                  │   ngân sách an toàn P99 ≤ 5ms.           │
│                           │                                  │ • Tự động kết nối lại khi Redis hồi sinh.│
├───────────────────────────┼──────────────────────────────────┼──────────────────────────────────────────┤
│ PostgreSQL Pool Nghẽn     │ Số connection chạm max (50)      │ • Strict Acquire Timeout 2.0 giây.       │
│                           │ Request mới bị nghẽn chờ         │ • Áp dụng cơ chế singleflight chống bão  │
│                           │                                  │   truy vấn cùng 1 customer_id.           │
├───────────────────────────┼──────────────────────────────────┼──────────────────────────────────────────┤
│ Apache Kafka Broker Mất   │ Không gửi được sự kiện tức thì   │ • Giao dịch sửa hồ sơ / đổi địa chỉ VẪN  │
│ Kết Nối                   │                                  │   thành công 100% nhờ bảng outbox_events.│
│                           │                                  │ • Outbox Worker tự động đẩy bù khi mạng  │
│                           │                                  │   phục hồi (Backlog Catch-up).           │
├───────────────────────────┼──────────────────────────────────┼──────────────────────────────────────────┤
│ Xung Đột Phiên Bản OCC    │ 2 thiết bị cùng sửa hồ sơ đồng   │ • Câu lệnh CAS trả về RowsAffected == 0. │
│                           │ thời                             │ • Trả về HTTP 409 Conflict với thông điệp│
│                           │                                  │   yêu cầu client tải lại dữ liệu mới.    │
└───────────────────────────┴──────────────────────────────────┴──────────────────────────────────────────┘
```

---

## 5. CHIẾN LƯỢC KIỂM THỬ TDD & MA TRẬN TRUY VẾT (TESTING BLUEPRINT)

### 5.1. Kim Tự Tháp Kiểm Thử MS-15
1. **Unit Tests (Thuần Logic Miền & Giải Thuật):**
   - Thuật toán Envelope Encryption AES-256-GCM.
   - Thao tác Zeroize ghi đè mảng byte bộ đệm trong RAM.
   - Kiểm thử 50 ca kiểm thử so khớp mờ địa danh Thừa Thiên Huế (Pass 100%).
2. **Integration Tests (Testcontainers Thật):**
   - Chạy PostgreSQL 16 và Redis 7 thật trong Docker container.
   - Bật 50 Goroutines đua nhau gọi `SetDefaultAddress` $\rightarrow$ Xác nhận DB chỉ có duy nhất 1 địa chỉ mặc định.
   - Kiểm thử chống chiếm đoạt đơn vãng lai bằng Partial Unique Index.
3. **Benchmark Test:**
   - Kiểm thử đo lường vi mô luồng đọc địa chỉ Checkout:
   - **Kết quả thực đo:** `BenchmarkGetDeliveryAddress_DualLayerCache: 421.9 ns/op` (nhanh gấp 11.000 lần mức trần SLA 5ms).
