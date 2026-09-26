# SƠ ĐỒ TRẠNG THÁI VÒNG ĐỜI THỰC THỂ & SAGA STATE MACHINE (STATE DIAGRAM - stateDiagram-v2)

## 1. Vị Trí & Vai Trò Trong Tài Liệu HLD
- **Cấp độ kiến trúc:** Vòng đời Entity & Máy trạng thái Giao dịch phân tán (State Transition Diagram).
- **Vị trí trong HLD:** Đặt tại **Chương 3 (Mục 3.2 & 3.3 - Saga Orchestration & State Machine)** — Trả lời câu hỏi: *Một đơn hàng, một giao dịch Saga phân tán, và một lô kẹo mè xửng trải qua những trạng thái nào trong suốt vòng đời của nó? Điều kiện gì kích hoạt việc chuyển trạng thái và kịch bản nào dẫn tới hủy/hoàn tiền?*

---

## 2. Quy Chuẩn Ký Hiệu Áp Dụng (Tuân Thủ Tuyệt Đối `regulation.md`)
- Sử dụng engine `stateDiagram-v2` chuẩn hóa của Mermaid.
- Theo Dòng 5 của `regulation.md`: *State Diagram không áp dụng phân chia sync/async*, mà tập trung vào:
  - Trạng thái khởi đầu `[*]` và trạng thái kết thúc `[*]`.
  - Nhãn chuyển trạng thái kèm Trigger và Điều kiện kiểm tra (Guard Condition): `State1 --> State2 : Trigger [Guard]`.
  - Trạng thái hợp thành (Composite States) để nhóm các pha xử lý liên quan: `state CheckoutPhase { ... }`.
  - Phân nhánh lựa chọn (Choice / Decision point) bằng `<<choice>>`.

---

## 3. Sơ Đồ 4.1: Vòng Đời Thực Thể Đơn Hàng Cốt Lõi (Order Aggregate Root Lifecycle)

```mermaid
stateDiagram-v2
    title Vòng Đời Thực Thể Đơn Hàng (Order Aggregate Root Lifecycle)

    [*] --> DRAFT : Khách bấm "Đặt hàng" (Checkout Initiated)
    
    state DRAFT {
        [*] --> ReservingStock
        ReservingStock --> StockLocked : gRPC ReserveStock [Thành công]
        ReservingStock --> StockFailed : gRPC ReserveStock [Hết hàng]
    }

    DRAFT --> PENDING_PAYMENT : Khóa tồn 15m thành công, cấp VietQR
    DRAFT --> CANCELLED_OUT_OF_STOCK : Không đủ tồn kho khả dụng

    state PENDING_PAYMENT {
        [*] --> AwaitingVietQR
        AwaitingVietQR --> Countdown15m
    }

    PENDING_PAYMENT --> PAID : Webhook VietQR xác nhận tiền về [Đúng HMAC]
    PENDING_PAYMENT --> CANCELLED_TIMEOUT : Hết hạn 15 phút không thanh toán
    PENDING_PAYMENT --> CANCELLED_BY_USER : Khách chủ động bấm "Hủy đơn"

    state PAID {
        [*] --> PickingTaskAssigned
        PickingTaskAssigned --> PackingInProgress : Thợ xưởng quét mã lô FEFO
    }

    PAID --> PROCESSING : Fulfillment Service tiếp nhận đơn
    PROCESSING --> PACKED : Quay video niêm phong tem Seal O Mạ hoàn tất
    PACKED --> SHIPPED : Bàn giao kiện kẹo cho bưu tá 3PL (GHN/ViettelPost)

    state SHIPPED {
        [*] --> InTransit
        InTransit --> OutForDelivery : Shipper đang đi giao
    }

    SHIPPED --> DELIVERED : Bưu tá xác nhận giao hàng thành công
    SHIPPED --> DELIVERY_FAILED : Giao thất bại 3 lần / Khách từ chối nhận (Bom hàng)

    DELIVERY_FAILED --> RETURNED_TO_FACTORY : Bưu kiện hoàn về kho xưởng Huế
    RETURNED_TO_FACTORY --> CANCELLED_RETURNED : Thủ kho nhập lại kho / tiêu hủy

    DELIVERED --> COMPLETED : Sau 7 ngày không khiếu nại (Tự động tích 1% Loyalty)
    DELIVERED --> RETURN_REQUESTED : Khách gửi ticket khiếu nại kẹo vỡ (CSKH tiếp nhận)

    state RETURN_REQUESTED {
        [*] --> VideoEvidenceInspected : Đối soát video đóng gói có seal
        VideoEvidenceInspected --> ReturnApproved : CSKH chấp thuận đổi trả
        VideoEvidenceInspected --> ReturnRejected : Bằng chứng seal nguyên vẹn
    }

    RETURN_REQUESTED --> REFUNDED : Kiểm định hàng trả về xưởng OK -> Finance hoàn tiền
    RETURN_REQUESTED --> COMPLETED : Bác bỏ khiếu nại không hợp lệ

    COMPLETED --> [*]
    CANCELLED_TIMEOUT --> [*]
    CANCELLED_BY_USER --> [*]
    CANCELLED_OUT_OF_STOCK --> [*]
    REFUNDED --> [*]
```

---

## 4. Sơ Đồ 4.2: Máy Trạng Thái Điều Phối Saga Phân Tán (Distributed Saga State Machine)

Sơ đồ đặc tả bảng `saga_states` bên trong `order-service`, thể hiện cơ chế **Centralized Compensation (Mục 3.4)**:

```mermaid
stateDiagram-v2
    title Máy Trạng Thái Điều Phối Saga Phân Tán (Saga Orchestration State Machine)

    [*] --> SAGA_STARTED : Khởi tạo giao dịch phân tán

    state SAGA_STARTED {
        [*] --> Step1_ReserveInventory
    }

    Step1_ReserveInventory --> SAGA_FAILED_OUT_OF_STOCK : gRPC ReserveStock [Thất bại]
    Step1_ReserveInventory --> INVENTORY_RESERVED : gRPC ReserveStock [Thành công]

    state INVENTORY_RESERVED {
        [*] --> Step2_ValidatePricingAndVoucher
        Step2_ValidatePricingAndVoucher --> AwaitingPaymentConfirmation
    }

    state AwaitingPaymentConfirmation {
        state check_payment <<choice>>
        [*] --> check_payment
        check_payment --> PaymentReceived : Webhook VietQR [Thành công]
        check_payment --> PaymentExpired : Quá 15 phút không nhận được tiền
        check_payment --> UserCancelled : Nhận lệnh hủy từ Client
    }

    %% Happy Path:
    PaymentReceived --> SAGA_COMPLETED : Saga kết thúc thành công (Phát OrderPaidEvent)
    SAGA_COMPLETED --> [*]

    %% Compensation Paths (Nguyên tắc Tập trung hóa đền bù 100% tại Saga):
    PaymentExpired --> COMPENSATING_INVENTORY : Kích hoạt kịch bản đền bù
    UserCancelled --> COMPENSATING_INVENTORY : Kích hoạt kịch bản đền bù

    state COMPENSATING_INVENTORY {
        [*] --> ExecutingReleaseReservation : Gọi gRPC ReleaseReservation() sang Kho
        ExecutingReleaseReservation --> ReservationReleased : Kho xác nhận nhả tồn
        ExecutingReleaseReservation --> RetryCompensation : Sự cố mạng (Exponential Backoff)
        RetryCompensation --> ExecutingReleaseReservation
    }

    ReservationReleased --> SAGA_COMPENSATED : Hoàn tất đền bù, dữ liệu nhất quán 100%
    SAGA_COMPENSATED --> [*]
    SAGA_FAILED_OUT_OF_STOCK --> [*]
```

---

## 5. Sơ Đồ 4.3: Vòng Đời Lô Hàng & Chiến Lược Xuất Kho FEFO (Batch Lifecycle & FEFO Strategy)

Quản lý xuất kho kẹo mè xửng Huế bảo đảm chất lượng OCOP theo nguyên tắc **Hạn dùng sớm nhất xuất trước (FEFO - First Expired, First Out)**:

```mermaid
stateDiagram-v2
    title Vòng Đời Lô Hàng Kẹo Mè Xửng OCOP (Batch Lifecycle & FEFO Strategy)

    [*] --> RAW_INSPECTED : Nhập nguyên liệu mè/đậu từ Hợp tác xã Huế
    RAW_INSPECTED --> IN_PRODUCTION : Đưa vào xưởng nấu kẹo theo công thức O Mạ
    IN_PRODUCTION --> QA_TESTING : Kẹo ra lò, kiểm định vệ sinh ATTP & độ giòn
    
    QA_TESTING --> QA_REJECTED : Không đạt tiêu chuẩn OCOP (Hủy mẻ nấu)
    QA_REJECTED --> [*]

    QA_TESTING --> ACTIVE_IN_STOCK : Đạt chuẩn -> In tem truy xuất nguồn gốc QR OCOP
    
    state ACTIVE_IN_STOCK {
        [*] --> NormalFEFO : Hạn sử dụng > 60 ngày (Ưu tiên xuất bán thông thường)
        NormalFEFO --> ApproachingExpiry : Hạn sử dụng còn từ 30 - 45 ngày
    }

    ApproachingExpiry --> EXPIRY_WARNING_EMITTED : Scheduled Job 01:00 AM phát ExpiryWarningEvent
    EXPIRY_WARNING_EMITTED --> FLASH_SALE_PROMOTION : Promotion Service tự động áp mã Flash Sale xả hàng

    FLASH_SALE_PROMOTION --> SOLD_OUT : Bán hết toàn bộ lô hàng
    FLASH_SALE_PROMOTION --> EXPIRED_QUARANTINE : Quá hạn sử dụng (Còn dưới 15 ngày)

    EXPIRED_QUARANTINE --> DESTROYED_AUDITED : Chuyển vào khu cách ly tiêu hủy (Có biên bản kiểm toán)

    SOLD_OUT --> [*]
    DESTROYED_AUDITED --> [*]
```
