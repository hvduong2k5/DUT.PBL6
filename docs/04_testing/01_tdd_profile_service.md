# CHIẾN LƯỢC KIỂM THỬ THEO TDD: MS-15 PROFILE SERVICE
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

---

## 1. KIM TỰ THÁP KIỂM THỬ ĐẶC THÙ (CUSTOM TEST PYRAMID FOR MS-15)

Khác với các microservice thuần Domain Logic (nơi phần lớn nghiệp vụ nằm ở Pure Functions trong RAM), `MS-15 profile-service` có đặc thù: **Các nghiệp vụ quan trọng và rủi ro nhất lại gắn chặt với hành vi của Cơ sở dữ liệu và Cơ chế hạ tầng**:
- Khóa dòng đồng thời (`SELECT ... FOR UPDATE`).
- Ràng buộc toàn vẹn có điều kiện (**PostgreSQL Partial Unique Index**).
- Khóa lạc quan nguyên tử (**Compare-And-Swap**).
- Vòng đời khóa mật mã phân tán (**HashiCorp Vault Transit Engine DEK/KEK**).
- Đồng bộ hóa xóa bộ đệm phân tán (**Redis Pub/Sub across Pods**).

Do đó, việc mock cơ sở dữ liệu (ví dụ dùng `sqlmock` thông thường) là **vô nghĩa và tạo ra cảm giác an toàn giả tạo (False Sense of Security)**: mock sẽ luôn luôn "giả vờ pass" ngay cả khi câu lệnh SQL chết vì race condition hay syntax error trên PostgreSQL thật.

```text
                           ▲
                          / \           E2E Tests (2-3 Luồng Khách Hàng E2E qua Gateway)
                         /   \
                        /-----\         Contract Tests (Pact Provider Verification với Order Service)
                       /       \
                      /---------\       Integration Tests (Testcontainers: PG 16, Redis 7, Vault Dev)
                     /           \      ← TẦNG NẶNG NHẤT VÀ QUAN TRỌNG NHẤT CỦA MS-15
                    /-------------\
                   /               \    Unit Tests (Fuzzy Match Levenshtein, AES-GCM In-Memory, DTO Validation)
                  /_________________\
```

### 1.1. Ma Trận Lựa Chọn Test Double (Nguyên Tắc: Đừng Mock Những Gì Mock Không Mô Phỏng Được)

| Phụ thuộc hạ tầng | Test Double Sử Dụng | Lý Do Kỹ Thuật Bắt Buộc |
| :--- | :--- | :--- |
| **PostgreSQL 16** (`profile_db`) | **Testcontainers Go (Postgres 16 thật)** | Mock không thể mô phỏng: (1) `SELECT ... FOR UPDATE` có thực sự chặn luồng thứ 2 hay không, (2) `Partial Unique Index` có loại trừ đúng trạng thái `REVOKED` hay không, (3) `pg_trgm` GIN index có lọc đúng chuỗi tiếng Việt hay không. |
| **Redis Cluster** (`profile_cache`) | **Testcontainers Go (Redis 7 thật)** | Cần kiểm chứng: (1) TTL 15 phút có thực sự hết hạn, (2) Lệnh `Publish/Subscribe` có thực sự phát thông điệp sang Pod thứ 2 trong môi trường mạng đồng thời hay không. |
| **HashiCorp Vault** (Transit Engine) | **Vault Dev Server Container** (Integration) + **Interface Mock** (Unit Test) | Unit test nghiệp vụ thông thường chỉ cần Mock Interface; nhưng test round-trip Encrypt $\rightarrow$ Rotate KEK $\rightarrow$ Re-wrap $\rightarrow$ Decrypt bắt buộc chạy trên Vault Transit Engine thật để xác thực tính tương thích versioning. |
| **Apache Kafka** (`profile.events.v1`) | **Mock Producer Interface** (Unit) + **Testcontainers Kafka** (Outbox E2E) | Verify bản ghi trong bảng `outbox_events` được quét bằng `SKIP LOCKED` và đẩy sang Kafka broker thật mà không bị mất tin. |

---

## 2. MINH HỌA CHU TRÌNH RED - GREEN - REFACTOR TRÊN BUG THỰC TẾ

Đây là minh chứng rõ ràng nhất cho giá trị của **Test-Driven Development (TDD)**: Viết test trước để kiểm chứng lỗi kiến trúc, sau đó sửa code và refactor.

### 2.1. Bước 1: RED (Viết Test Cho Lỗi `guest_order_claims` Khi Chưa Sửa DDL)
Nếu chạy trên DDL cũ (`CREATE UNIQUE INDEX uq_order_claim ON guest_order_claims(order_id)`):
```go
func TestGuestOrderClaim_RevokedClaim_CanBeReclaimedByCorrectOwner(t *testing.T) {
    ctx := context.Background()
    db := testhelper.SetupPostgresContainer(t)

    repo := repository.NewClaimRepository(db)

    // Bước 1: Gán nhầm khách hàng A, sau đó phát hiện và thu hồi (REVOKED)
    wrongCustID := uuid.NewV7()
    err := repo.CreateClaim(ctx, wrongCustID, "ORD-20261015-0042", "0905111111")
    require.NoError(t, err)

    err = repo.RevokeClaim(ctx, "ORD-20261015-0042")
    require.NoError(t, err)

    // Bước 2: Khách hàng B (chủ sở hữu hợp pháp thực tế) thực hiện claim lại
    legitimateCustID := uuid.NewV7()
    err = repo.CreateClaim(ctx, legitimateCustID, "ORD-20261015-0042", "0905222222")

    // TRÊN DDL CŨ: Lệnh này ném lỗi duplicate key value violates unique constraint "uq_order_claim"
    // -> TEST BẬT ĐÈN ĐỎ (FAIL / RED)!
    require.NoError(t, err, "Chủ đơn hợp pháp phải claim lại được đơn hàng sau khi claim sai đã bị REVOKED")
}
```

### 2.2. Bước 2: GREEN (Áp Dụng Bản Sửa DDL Partial Unique Index)
Chạy migration áp dụng DDL mới:
```sql
DROP INDEX IF EXISTS uq_order_claim;
CREATE UNIQUE INDEX uq_order_claim_active ON guest_order_claims(order_id) WHERE claim_status = 'VERIFIED';
```
$\rightarrow$ Chạy lại test: **BẬT ĐÈN XANH (PASS / GREEN)!**

### 2.3. Bước 3: REFACTOR (Bổ Sung Test Biên Chặn Trùng Khi Đang VERIFIED)
Đảm bảo rằng Partial Index không làm mất đi tính năng chặn claim trùng ban đầu:
```go
func TestGuestOrderClaim_AlreadyVerifiedClaim_RejectsDoubleClaim(t *testing.T) {
    ctx := context.Background()
    db := testhelper.SetupPostgresContainer(t)
    repo := repository.NewClaimRepository(db)

    custA := uuid.NewV7()
    custB := uuid.NewV7()

    _ = repo.CreateClaim(ctx, custA, "ORD-20261015-0099", "0905111111")
    
    // Khách B cố tình claim đơn đang VERIFIED của khách A
    err := repo.CreateClaim(ctx, custB, "ORD-20261015-0099", "0905222222")
    require.ErrorIs(t, err, repository.ErrOrderAlreadyClaimed)
}
```

---

## 3. BẢNG ĐẶC TẢ CÁC BỘ TEST BẮT BUỘC THEO 5 THUẬT TOÁN LLD

### 3.1. Thuật Toán 1: Atomic Default Address Switcher (Mục 4.1)
- **Tệp test:** `tests/integration/address_switch_test.go`
- **Mục tiêu:** Kiểm tra điều kiện tranh chấp đồng thời (Concurrency Race Condition).
- **Kịch bản:**
  - Khởi tạo 1 khách hàng và 5 địa chỉ trong cơ sở dữ liệu thật.
  - Kích hoạt **100 Goroutines chạy song song cùng lúc**, liên tục gửi yêu cầu đặt địa chỉ mặc định ngẫu nhiên trong 5 địa chỉ đó.
  - Sử dụng `sync.WaitGroup` để đồng bộ và đợi toàn bộ 100 luồng hoàn tất.
  - **Khẳng định (Assertion):** Câu lệnh SQL `SELECT COUNT(*) FROM shipping_addresses WHERE customer_id = $1 AND is_default = TRUE AND is_deleted = FALSE` bắt buộc phải trả về **chính xác 1** (không bao giờ được là 0 hoặc $> 1$).

### 3.2. Thuật Toán 2: Envelope Encryption & Key Rotation (Mục 4.2)
- **Tệp test:** `internal/infrastructure/security/envelope_encryptor_test.go`
- **5 Nhóm Test Bắt Buộc:**
  1. **Tamper Detection (GCM Authentication Tag):** Mã hóa số CCCD, cố tình can thiệp làm sai lệch 1 byte cuối cùng của Ciphertext (`ciphertext[len-1] ^= 0xFF`). Gọi hàm giải mã: Bắt buộc phải trả về lỗi `ErrTamperDetected`, tuyệt đối không được `panic` hay trả về dữ liệu rác.
  2. **Nonce Uniqueness (Chống Tái Sử Dụng Nonce):** Thực hiện vòng lặp 1.000 lần mã hóa trên cùng một chuỗi plaintext CCCD. Kiểm tra toàn bộ 1.000 giá trị Nonce (12-byte) sinh ra không được phép trùng lặp bất kỳ lần nào.
  3. **Key Rotation Re-wrap Round-Trip (Vault Transit Container):** Tạo DEK v1, mã hóa dữ liệu. Kích hoạt xoay khóa KEK sang v2 trên Vault. Thực hiện Re-wrap DEK. Dùng DEK đã re-wrap giải mã lại ciphertext cũ: Kết quả phải ra đúng chuỗi CCCD ban đầu mà không cần giải mã lại ciphertext trong DB.
  4. **In-Memory Short-lived DEK Cache & Injectable TTL Zeroize:** Tham số `cacheTTL time.Duration` được inject trực tiếp vào constructor (`NewEnvelopeEncryptor`). Trong test, truyền TTL ngắn `20ms` - `50ms` để kiểm chứng nhanh hook `OnEvict` tự động `zeroize` toàn bộ byte của DEK trong RAM (`val[i] = 0`) mà không làm chậm quá trình chạy test suite.
  5. **Mixed Versions Coexistence During Re-wrap (Zero-Downtime Guarantee):** Mô phỏng Background Job đang chạy dở: Bản ghi 1 đã re-wrap (KEK v2), bản ghi 2 chưa kịp re-wrap (vẫn KEK v1). Cả 2 bản ghi cùng giải mã thành công đồng thời, chứng minh 100% tính sẵn sàng Zero-Downtime.

### 3.3. Thuật Toán 3: Chuẩn Hóa & Tìm Kiếm Mờ Địa Chỉ 2 Giai Đoạn (Mục 4.3)
- **Tệp test:** `internal/usecase/address_fuzzy_matcher_test.go`
- **Tệp dữ liệu mẫu (Golden File):** `internal/usecase/testdata/address_fuzzy_cases.csv`
- **Mô hình Dữ liệu Hành chính 2 Cấp (Chuẩn hóa Quốc gia từ 01/07/2025):**
  - Hệ thống áp dụng triệt để mô hình **2 cấp (Tỉnh/Thành phố trực thuộc TW $\rightarrow$ Xã/Phường)**.
  - Thành phố Huế là Thành phố trực thuộc Trung ương, trực tiếp quản lý 40 xã/phường (bãi bỏ hoàn toàn cấp huyện/thị xã như Hương Thủy cũ). Bảng `shipping_addresses` được tinh giản, loại bỏ cột `district_code` và `district_name`.
- **Mục tiêu:** Kiểm tra độ chính xác của Pipeline 2 giai đoạn (Pre-filter GIN Trigram trên Postgres + Levenshtein tinh chỉnh trong Go).
- **Phương pháp Golden Dataset:** Sử dụng file CSV nhúng (`//go:embed`) gồm 50+ ca kiểm thử thực tế địa danh Thành phố Huế chuẩn 2 cấp và thói quen nhập liệu lịch sử:
  - Định dạng 2 cấp chính thức: `"P. Thuan Hoa TP Hue"` $\rightarrow$ `WARD-TH-001`, `"P. Phu Bai TP Hue"` $\rightarrow$ `WARD-PB-003`.
  - Thói quen gọi theo mốc cũ: `"Phú Bài Hương Thủy"` $\rightarrow$ `WARD-PB-003` (nhờ cơ chế lọc stopwords thông minh).
  - Khử dấu: `"thuan hoa"` $\rightarrow$ `WARD-TH-001`, `"vinh loc"` $\rightarrow$ `WARD-VL-002`.
  - Sai chính tả nhẹ: `"thuan hooa"` $\rightarrow$ `WARD-TH-001` (Score $\ge 0.85$).
  - Gõ dính chữ: `"phuongthuanhoa"` $\rightarrow$ `WARD-TH-001`.
  - **Khẳng định Ranh giới Critical Path:** Test case `TestFuzzyMatch_CheckoutCriticalPathBoundary` xác nhận luồng Checkout chỉ tra cứu Primary Key B-Tree $O(1)$, hoàn toàn không chạy Fuzzy Matching ($P99 \le 5\text{ms}$).

### 3.4. Thuật Toán 4: Dual-Layer Cache Invalidation (Mục 4.4)
- **Tệp test:** `tests/integration/cache_invalidation_test.go`
- **Mục tiêu:** Kiểm tra việc xóa cache L1 (RAM) và L2 (Redis) nhất quán giữa nhiều Pod mà không gây flaky test trên CI.
- **Kịch bản Polling Không Dùng Sleep Cố Định:**
  - Khởi tạo 2 thực thể `podA` và `podB` độc lập cùng kết nối cụm Redis container thật.
  - **Khử Flaky Test:** Sử dụng `require.Eventually` kết hợp lệnh Redis `PUBSUB NUMSUB` để bảo đảm cả 2 Pod đã hoàn tất đăng ký kênh trước khi phát tín hiệu.
  - Cả 2 Pod cùng nạp dữ liệu khách hàng vào L1 RAM.
  - `podA` thực hiện cập nhật và gọi `Invalidate`.
  - Dùng `require.Eventually` polling định kỳ 20ms (timeout 1s) kiểm tra L1 của `podB` bị xóa bỏ. Khẳng định `podB.Get()` trả về cache miss buộc đọc DB mới, ngăn chặn hoàn toàn lỗi đọc stale address.

### 3.5. Thuật Toán 5: Optimistic Concurrency Control (Mục 4.6)
- **Tệp test:** `internal/repository/customer_repository_test.go`
- **Mục tiêu:** Kiểm tra tính đúng đắn của cơ chế phân biệt lỗi HTTP 404 và HTTP 409.
- **Table-Driven Test Cases:**
  1. `Case 1 (Conflict):` `expected_version = 1`, nhưng bản ghi trong DB đã bị ai đó sửa lên `version = 2` $\rightarrow$ Trả về `ErrOptimisticLockConflict` (Map sang 409).
  2. `Case 2 (Not Found):` `customer_id` không tồn tại trong hệ thống $\rightarrow$ Trả về `ErrCustomerNotFound` (Map sang 404).
  3. `Case 3 (Success):` `expected_version = 1`, DB đang ở `version = 1` $\rightarrow$ Thành công, `version` mới là 2.

---

## 4. CONTRACT TESTING VỚI PACT BROKER & MUTATION TESTING

### 4.1. Pact Provider Verification (gRPC Contract Test)
- **Tệp test:** `tests/contract/pact_provider_test.go`
- **Mục tiêu:** Bảo đảm hợp đồng giữa `MS-04 order-service` (Consumer) và `MS-15 profile-service` (Provider) luôn tương thích mà không cần khởi động toàn bộ 18 service.
- **Cách thức vận hành:** Sử dụng Pact Go SDK tải hợp đồng Pact file từ Pact Broker, khởi chạy gRPC server của `profile-service` cục bộ và xác thực mọi RPC `GetCustomerProfile`, `GetDeliveryAddress` phản hồi đúng schema cam kết.

### 4.2. Mutation Testing Cho Module Mật Mã (`infrastructure/security/`)
- **Vấn đề:** Test Coverage 80% không có ý nghĩa nếu các câu lệnh `require.NoError` hoặc `require.Equal` bị viết lỏng lẻo (Assertion-free tests).
- **Giải pháp:** Áp dụng công cụ **`gremlins`** hoặc **`go-mutesting`** riêng cho gói `internal/infrastructure/security/`:
  - Công cụ tự động sinh đột biến (Mutant): Đổi `!=` thành `==`, xóa câu lệnh kiểm tra lỗi `if err != nil`, thay đổi số byte Nonce từ 12 thành 11.
  - Chạy lại test suite: Test **phải FAIL** thì Mutant mới bị tiêu diệt (Killed).
  - **Mục tiêu Mutation Score:** $\ge 90\%$ đối với module mã hóa PII.
