# Danh sách Kịch bản Unit Test - Inventory Service

Tài liệu này định nghĩa các kịch bản kiểm thử mức Unit (Unit Tests) cho `inventory-service`, tuân thủ theo kiến trúc Clean Architecture bằng Go. Các test case bao phủ từ Domain Logic (bảo vệ invariants) cho đến Application Layer (Use Cases) và Background Workers.

---

## 1. Domain Entities & Invariants Tests

Mục tiêu: Đảm bảo các quy tắc kinh doanh cốt lõi (Business Invariants) không bao giờ bị vi phạm tại các Entity.
Thư viện/Công cụ: `testing`, `testify/assert`.

### **Mã Test Case: UT-INV-DOMAIN-01**
* **Tầng kiểm thử:** Domain Entity (`ProductInventory` / `Batch`)
* **Mô tả mục tiêu:** Kiểm tra INV-BI-01: Cấm tuyệt đối tồn kho âm ($Q_{avail} = Q_{phy} - Q_{res} \ge 0$).
* **Điều kiện tiên quyết (Given):** Một sản phẩm có Physical Quantity = 100, Reserved Quantity = 90. 
* **Hành động (When):** Thực hiện yêu cầu giữ thêm (reserve) số lượng 20 đơn vị.
* **Kỳ vọng (Then):** Hàm trả về lỗi `ErrInsufficientStock`, các thuộc tính số lượng của Entity không bị thay đổi.

### **Mã Test Case: UT-INV-DOMAIN-02**
* **Tầng kiểm thử:** Domain Entity (`Batch`)
* **Mô tả mục tiêu:** Kiểm tra INV-BI-02: Lô có hạn sử dụng quá khứ tự động chuyển `EXPIRED`.
* **Điều kiện tiên quyết (Given):** Một lô hàng đang ở trạng thái `ACTIVE`, ngày hết hạn (`exp_date`) là ngày hôm qua.
* **Hành động (When):** Gọi phương thức `CheckExpiry(current_date)`.
* **Kỳ vọng (Then):** Trạng thái lô hàng đổi thành `EXPIRED` và phương thức kiểm tra tính hợp lệ phân bổ `CanAllocate()` trả về `false`.

### **Mã Test Case: UT-INV-DOMAIN-03**
* **Tầng kiểm thử:** Domain Entity (`Batch`)
* **Mô tả mục tiêu:** Kiểm tra INV-BI-04: Tính hợp lệ của dữ liệu lô khi tạo mới.
* **Điều kiện tiên quyết (Given):** Thông khởi tạo lô hàng với `mfg_date` sau `exp_date` hoặc `physical_qty` = 0.
* **Hành động (When):** Gọi hàm constructor `NewBatch(...)`.
* **Kỳ vọng (Then):** Trả về lỗi `ErrInvalidBatchData` và đối tượng trả về là `nil`.

---

## 2. Domain Service (FEFO Allocation Algorithm) Tests

Mục tiêu: Đảm bảo thuật toán phân bổ tồn kho (First Expired, First Out) hoạt động chính xác dưới mọi điều kiện biên.
Thư viện/Công cụ: `testing`, `testify/assert`.

### **Mã Test Case: UT-INV-FEFO-01**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case đủ hàng từ 1 lô duy nhất.
* **Điều kiện tiên quyết (Given):** 2 lô hàng (Lô A: 100 chiếc, hết hạn T+30; Lô B: 100 chiếc, hết hạn T+60). Yêu cầu = 50.
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, 50)`.
* **Kỳ vọng (Then):** Trả về kết quả phân bổ lấy 50 chiếc từ Lô A. Lô B không bị trừ.

### **Mã Test Case: UT-INV-FEFO-02**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case cần gom từ nhiều lô theo thứ tự FEFO (`exp_date ASC`).
* **Điều kiện tiên quyết (Given):** Lô A (50 chiếc, hết hạn sớm), Lô B (100 chiếc, hết hạn muộn hơn). Yêu cầu = 120.
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, 120)`.
* **Kỳ vọng (Then):** Phân bổ thành công: Lấy 50 chiếc từ Lô A, 70 chiếc từ Lô B. 

### **Mã Test Case: UT-INV-FEFO-03**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case tổng tồn khả dụng không đủ (`ErrInsufficientStock`).
* **Điều kiện tiên quyết (Given):** Lô A (50), Lô B (50). Yêu cầu = 150.
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, 150)`.
* **Kỳ vọng (Then):** Trả về lỗi `ErrInsufficientStock` và không có danh sách phân bổ nào được tạo.

### **Mã Test Case: UT-INV-FEFO-04**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case có lô xen kẽ bị quá hạn hoặc trạng thái không phải `ACTIVE`.
* **Điều kiện tiên quyết (Given):** Lô A (50, `EXPIRED`), Lô B (50, `QUARANTINE`), Lô C (100, `ACTIVE`, hết hạn xa nhất). Yêu cầu = 50.
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, 50)`.
* **Kỳ vọng (Then):** Bỏ qua Lô A và Lô B. Phân bổ lấy toàn bộ 50 chiếc từ Lô C.

### **Mã Test Case: UT-INV-FEFO-05**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case biên (Edge Cases) - Lấy số lượng <= 0.
* **Điều kiện tiên quyết (Given):** Danh sách lô hàng hợp lệ. Yêu cầu = 0 hoặc -10.
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, req_qty)`.
* **Kỳ vọng (Then):** Trả về lỗi `ErrInvalidAllocationRequest`.

### **Mã Test Case: UT-INV-FEFO-06**
* **Tầng kiểm thử:** Domain Service (`AllocationService`)
* **Mô tả mục tiêu:** Case biên (Edge Cases) - Lấy chính xác 1 đơn vị khả dụng cuối cùng của kho hàng.
* **Điều kiện tiên quyết (Given):** Duy nhất 1 lô còn đúng 1 đơn vị khả dụng ($Q_{avail} = 1$). Yêu cầu mua đúng 1 đơn vị ($req\_qty = 1$).
* **Hành động (When):** Chạy hàm `AllocateByFEFO(batches, 1)`.
* **Kỳ vọng (Then):** Phân bổ thành công đúng 1 đơn vị từ lô này, số dư khả dụng còn lại của lô đạt đúng bằng 0 ($Q_{avail} = 0$), không xảy ra lỗi out-of-bounds hay tồn âm.


---

## 3. Application Layer (Use Cases / Commands) Tests

Mục tiêu: Kiểm tra luồng phối hợp (orchestration), tính idempotency, concurrency lock và cơ chế transaction tại lớp Application.
Thư viện/Công cụ: `testing`, `testify/assert`, `testify/mock` (để mock DB, Redis lock, Message Broker).

### **Mã Test Case: UT-INV-APP-01**
* **Tầng kiểm thử:** Application Use Case (`ReserveStockUseCase`)
* **Mô tả mục tiêu:** Luồng cơ bản (Happy path) cho việc giữ chỗ tồn kho.
* **Điều kiện tiên quyết (Given):** Mock LockService trả về true. Mock DB Repository trả về danh sách lô hợp lệ, Mock DB transaction commit thành công. 
* **Hành động (When):** Gọi `Execute(order_id, items)`.
* **Kỳ vọng (Then):** 
    - Gọi LockService `AcquireLock`.
    - Gọi Repository để lấy dữ liệu.
    - Chạy FEFO Alloc.
    - Gọi Repository cập nhật DB và chèn Reservation Record (cùng 1 Transaction).
    - Gọi LockService `ReleaseLock`.
    - Trả về Success.

### **Mã Test Case: UT-INV-APP-02**
* **Tầng kiểm thử:** Application Use Case (`ReserveStockUseCase`)
* **Mô tả mục tiêu:** Xử lý tranh chấp khóa (Lock contention).
* **Điều kiện tiên quyết (Given):** Mock LockService trả về lỗi/false khi gọi `AcquireLock` (đang có process khác lock `product_id`).
* **Hành động (When):** Gọi `Execute(order_id, items)`.
* **Kỳ vọng (Then):** Trả về lỗi `ErrConcurrentUpdate`. Hệ thống kết thúc sớm luồng thực thi, không thực hiện bất kỳ lệnh query nào vào CSDL Repository.

### **Mã Test Case: UT-INV-APP-03**
* **Tầng kiểm thử:** Application Use Case (`ReserveStockUseCase`)
* **Mô tả mục tiêu:** Đảm bảo tính Idempotency khi mạng bị chập chờn.
* **Điều kiện tiên quyết (Given):** Lệnh reserve với `order_id` X đã tồn tại trong DB (Mock Repository trả về bản ghi Reservation X).
* **Hành động (When):** Gửi lại request `Execute(order_id=X, items)`.
* **Kỳ vọng (Then):** Use Case trả về kết quả thành công ngay lập tức dựa trên bản ghi cũ. Tồn kho không bị trừ/giữ thêm lần 2.

### **Mã Test Case: UT-INV-APP-04**
* **Tầng kiểm thử:** Application Use Case (`ReleaseReservationUseCase`)
* **Mô tả mục tiêu:** Giải phóng tồn kho thành công do đơn hàng bị hủy hoặc timeout.
* **Điều kiện tiên quyết (Given):** Mock Repository trả về Reservation hợp lệ ở trạng thái `RESERVED`.
* **Hành động (When):** Gọi `Execute(order_id)`.
* **Kỳ vọng (Then):** Các lô được hoàn trả `reserved_qty`, Reservation record chuyển sang `CANCELLED`. Commit transaction thành công.

### **Mã Test Case: UT-INV-APP-05**
* **Tầng kiểm thử:** Application Use Case (`ReleaseReservationUseCase`)
* **Mô tả mục tiêu:** Đảm bảo Idempotency khi giải phóng.
* **Điều kiện tiên quyết (Given):** Reservation cho `order_id` đã ở trạng thái `CANCELLED` hoặc `COMMITTED`.
* **Hành động (When):** Gọi `Execute(order_id)`.
* **Kỳ vọng (Then):** Kết thúc thành công sớm, không gọi DB Update. Không có số lượng nào bị trừ/cộng sai lệnh.

### **Mã Test Case: UT-INV-APP-06**
* **Tầng kiểm thử:** Application Use Case (`CommitStockDeductionUseCase`)
* **Mô tả mục tiêu:** Đơn hàng thanh toán thành công, xác nhận trừ thật số lượng.
* **Điều kiện tiên quyết (Given):** Đơn hàng đang có trạng thái `RESERVED`.
* **Hành động (When):** Gọi `Execute(order_id)`.
* **Kỳ vọng (Then):** `Q_phy` giảm xuống, `Q_res` giảm tương ứng cho các lô đã phân bổ. Chuyển Reservation status thành `COMMITTED`.

---

## 4. Background Workers Tests

Mục tiêu: Đảm bảo các tiến trình nền chạy định kỳ xử lý đúng đắn các tác vụ dọn dẹp và cảnh báo tự động.
Thư viện/Công cụ: `testing`, `testify/assert`, `testify/mock`.

### **Mã Test Case: UT-INV-WORKER-01**
* **Tầng kiểm thử:** Background Worker (`TTLReservationCleanupWorker`)
* **Mô tả mục tiêu:** Tự động giải phóng các Reservation quá hạn 15 phút chưa được commit.
* **Điều kiện tiên quyết (Given):** Mock DB trả về danh sách 5 Reservations có `created_at` < (Now - 15 phút) và status = `RESERVED`.
* **Hành động (When):** Worker gọi hàm `Run()`.
* **Kỳ vọng (Then):** `ReleaseReservationUseCase` được trigger cho 5 `order_id` này. In log thành công.

### **Mã Test Case: UT-INV-WORKER-02**
* **Tầng kiểm thử:** Background Worker (`ExpiryCheckWorker`)
* **Mô tả mục tiêu:** Quét và gắn cờ các lô cận hạn.
* **Điều kiện tiên quyết (Given):** Lô hàng A còn 40 ngày là hết hạn, lô B còn 50 ngày.
* **Hành động (When):** Worker chạy hàm `CheckNearExpiryBatches(threshold=45 days)`.
* **Kỳ vọng (Then):** Lô A được đổi trạng thái thành `NEAR_EXPIRY` và mock EventPublisher bắn event `BatchNearExpiryEvent` đi. Lô B giữ nguyên trạng thái `ACTIVE`.
