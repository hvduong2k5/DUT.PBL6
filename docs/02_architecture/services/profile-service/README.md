> **Cập nhật 08/10/2026:** Hành vi implementation hiện tại được mô tả tại [06_hardening_and_contract.md](06_hardening_and_contract.md). Các mô tả cache authoritative, OTP claim đã hoàn thiện và SLA từ mock benchmark trong bản v2.0 dưới đây đã được thay thế. Xem ADR-004/005/006 cho quyết định mới.

# BỘ TÀI LIỆU THIẾT KẾ KIẾN TRÚC CHI TIẾT (LLD): MS-15 PROFILE SERVICE
## TRUNG TÂM DỮ LIỆU THỰC THỂ, ĐỊA CHỈ 2 CẤP & GIÁM SÁT TUÂN THỦ OCOP — MÈ XỬNG O MẠ
### PHIÊN BẢN: 2.0 (MODULAR DOMAIN-DRIVEN DESIGN SPECIFICATION)

---

## 📌 BẢN ĐỒ TÀI LIỆU KIẾN TRÚC THEO TỪNG SUB-DOMAIN

Nhằm nâng cao khả năng đọc hiểu, bảo trì và phân công công việc độc lập giữa các kỹ sư phần mềm, toàn bộ thiết kế chi tiết của dịch vụ `profile-service` được phân rã thành **5 Phân Hệ Miền Nghiệp Vụ (Domain Modules)** chuẩn DDD:

```text
docs/02_architecture/services/profile-service/
├── README.md                                  # [BẠN ĐANG Ở ĐÂY] Bản đồ điều hướng & Kiến trúc tổng thể
├── ms15_profile_service_design.md             # Tài liệu đặc tả tổng hợp 11 bước (Master Index)
│
├── 01_customer_profile_domain.md              # [DOMAIN 1] Hồ sơ Khách hàng & Sở thích Dị ứng Mè OCOP
├── 02_shipping_address_domain.md              # [DOMAIN 2] Sổ địa chỉ 2 cấp, gRPC Checkout (P99 <= 5ms) & Fuzzy Match Huế
├── 03_guest_order_claim_domain.md             # [DOMAIN 3] Khôi phục & Liên kết Đơn hàng Vãng lai (OTP Verification)
├── 04_employee_compliance_domain.md           # [DOMAIN 4] Quản lý Nhân sự Xưởng kẹo, Envelope Encryption & Giám sát VSATTP
└── 05_infrastructure_and_cross_cutting.md     # [HẠ TẦNG] Dual-Layer Cache, Outbox Worker & Phục hồi Phân tán
```

---

## 🗺️ TỔNG HỢP NHANH CÁC PHÂN HỆ NGHIỆP VỤ (DOMAIN SUMMARY)

### 1. [Phân Hệ 1: Hồ Sơ Khách Hàng & Khẩu Vị OCOP](01_customer_profile_domain.md)
- **Nghiệp vụ cốt lõi:** Quản lý thông tin định danh người tiêu dùng D2C, sở thích sản phẩm và cảnh báo dị ứng mè vừng / đậu phụng (`DietaryPreference`).
- **Kỹ thuật then chốt:** Khóa lạc quan CAS (`UpdateProfileWithOCCUseCase`), FSM trạng thái tài khoản, chỉ mục JSONB GIN Index, phân biệt lỗi 404 Not Found và 409 Conflict.
- **Xem chi tiết:** 👉 [Đọc tài liệu Domain 1](01_customer_profile_domain.md)

### 2. [Phân Hệ 2: Sổ Địa Chỉ 2 Cấp & gRPC Checkout Critical Path](02_shipping_address_domain.md)
- **Nghiệp vụ cốt lõi:** Địa chỉ giao nhận chuẩn hành chính 2 cấp (Hậu sáp nhập 01/07/2025: Xã/Phường $\rightarrow$ Tỉnh/TP TW, bãi bỏ hoàn toàn cấp huyện).
- **Kỹ thuật then chốt:**
  - Chuyển đổi địa chỉ mặc định nguyên tử (Atomic Switch) với `SELECT FOR UPDATE` + Partial Unique Index.
  - Cổng gRPC nội bộ phục vụ `MS-04 order-service` với cam kết độ trễ **$P99 \le 5\text{ms}$** (thực đo **421.9 ns/op**).
  - Thuật toán so khớp mờ địa chỉ Huế ADR-003: Giảm 50% chi phí phạt cho cặp đồng âm `i`/`y` (*Vỹ Dạ* $\leftrightarrow$ *Vi Dạ*).
- **Xem chi tiết:** 👉 [Đọc tài liệu Domain 2](02_shipping_address_domain.md)

### 3. [Phân Hệ 3: Khôi Phục & Liên Kết Đơn Hàng Vãng Lai](03_guest_order_claim_domain.md)
- **Nghiệp vụ cốt lõi:** Cho phép khách hàng mua thử dạng Guest khôi phục và liên kết đơn hàng cũ vào tài khoản chính thức để tích lũy **Mè Xửng Xu**.
- **Kỹ thuật then chốt:** Guest Order Claim FSM, Xác thực OTP SMS qua điện thoại, Partial Unique Index chống claim trùng lặp đơn hàng (`uq_order_claim_active`).
- **Xem chi tiết:** 👉 [Đọc tài liệu Domain 3](03_guest_order_claim_domain.md)

### 4. [Phân Hệ 4: Nhân Sự Xưởng Kẹo, Envelope Encryption & Giám Sát VSATTP](04_employee_compliance_domain.md)
- **Nghiệp vụ cốt lõi:** Quản lý công nhân quấy kẹo, nhân viên đóng gói niêm phong tem QR OCOP, giám sát hạn hiệu lực Giấy chứng nhận tập huấn VSATTP.
- **Kỹ thuật then chốt:**
  - Bảo mật PII theo Nghị định 13/2023/NĐ-CP: **Envelope Encryption AES-256-GCM** kết hợp **HashiCorp Vault KMS Transit Engine**.
  - Xóa sạch mảng byte DEK khỏi RAM khi hết hạn (**In-Memory Zeroize on Eviction**).
  - Cron Worker quét hạn VSATTP 30 ngày / 7 ngày, phát cảnh báo trước khi hết hạn.
- **Xem chi tiết:** 👉 [Đọc tài liệu Domain 4](04_employee_compliance_domain.md)

### 5. [Hạ Tầng Kỹ Thuật, Bộ Đệm 2 Tầng & Khả Năng Phục Hồi](05_infrastructure_and_cross_cutting.md)
- **Nghiệp vụ cốt lõi:** Đảm bảo độ sẵn sàng cao, hiệu năng siêu tốc và khả năng phục hồi khi có sự cố mạng.
- **Kỹ thuật then chốt:**
  - Dual-Layer Cache: L1 In-Memory LRU (421 ns) + L2 Redis (0.95 ms) + Redis Pub/Sub đồng bộ xóa cache giữa các Pod.
  - Transactional Outbox Pattern: Polling bằng `SELECT FOR UPDATE SKIP LOCKED` phát sự kiện lên Kafka `profile.events.v1`.
  - Giám sát viễn trắc Prometheus `/metrics`.
  - Ma trận sự cố phân tán: Tự phục hồi khi Vault KMS sập, Redis sập hoặc Kafka broker bảo trì.
- **Xem chi tiết:** 👉 [Đọc tài liệu Hạ Tầng](05_infrastructure_and_cross_cutting.md)

---

## 🏛️ SƠ ĐỒ ĐỊNH VỊ TOÀN CẢNH HỆ THỐNG

```text
                                        KONG API GATEWAY
                                               │
                       ┌───────────────────────┴───────────────────────┐
                       │ HTTP REST (North-South)                       │ gRPC (East-West)
                       ▼                                               ▼
          ┌─────────────────────────────────────────────────────────────────────────┐
          │                    MS-15: PROFILE SERVICE (CORE DOMAIN)                 │
          │  ┌───────────────────────────────┐     ┌─────────────────────────────┐  │
          │  │     PROFILE & ADDRESS ENGINE  │     │      COMPLIANCE & SECURITY  │  │
          │  │  • Quản lý Aggregate Profile  │     │  • Envelope Encryption PII  │  │
          │  │  • Sổ địa chỉ giao hàng 2 cấp │     │  • Cron Giám sát VSATTP     │  │
          │  │  • Atomic Default Switcher    │     │  • In-Memory DEK Zeroize    │  │
          │  │  • Khóa lạc quan CAS (OCC)    │     │  • Outbox Publisher Kafka   │  │
          │  └───────────────────────────────┘     └─────────────────────────────┘  │
          └─────────────────────────────────────────────────────────────────────────┘
                               │                                       ▲
                               ▼                                       │
                      POSTGRESQL 16 (profile_db)               HASHICORP VAULT KMS
```
