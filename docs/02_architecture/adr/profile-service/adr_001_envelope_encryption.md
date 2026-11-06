# ADR-001: ÁP DỤNG ENVELOPE ENCRYPTION (DEK/KEK) CHO DỮ LIỆU NHẠY CẢM PII
## HỆ THỐNG THƯƠNG MẠI ĐIỆN TỬ ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

- **Trạng thái:** ACCEPTED
- **Ngày quyết định:** 23/09/2026
- **Người đề xuất:** Tech Lead & Security Specialist
- **Phạm vi:** `MS-15 profile-service` (Dữ liệu CCCD/Định danh cá nhân thợ xưởng & nhân sự)

---

## 1. Ngữ Cảnh & Vấn Đề (Context & Problem Statement)

Theo quy định pháp luật Việt Nam về bảo vệ dữ liệu cá nhân (Nghị định 13/2023/NĐ-CP) và yêu cầu chứng nhận OCOP 4 sao về quản lý nhân sự sản xuất, thông tin số Căn cước công dân (CCCD) của thợ xưởng làm kẹo mè xửng và nhân viên giao hàng là dữ liệu cá nhân nhạy cảm (PII), bắt buộc phải được mã hóa khi lưu trữ trong cơ sở dữ liệu (Encryption at Rest).

### Các Phương Án Đã Xem Xét:
1. **Phương án A: Dùng `pgcrypto` của PostgreSQL**
   - *Ưu điểm:* Cú pháp SQL đơn giản (`pgp_sym_encrypt`).
   - *Nhược điểm chí mạng:* Khóa giải mã phải truyền trực tiếp trong câu lệnh SQL hoặc lưu trên Database Server. Quản trị viên DB (DBA) hoặc hacker chiếm quyền DB có thể đọc toàn bộ khóa trong RAM hoặc logs; không hỗ trợ xoay khóa không gián đoạn (Zero-downtime key rotation).
2. **Phương án B: Mã hóa đối xứng 1 lớp tại Application (`AES-256-GCM` dùng Master Key tĩnh)**
   - *Ưu điểm:* Mã hóa ngoài ứng dụng trước khi ghi xuống DB.
   - *Nhược điểm chí mạng:* Nếu Master Key bị lộ hoặc đến chu kỳ tuân thủ phải xoay khóa (Key Rotation), toàn bộ hàng trăm ngàn bản ghi CCCD lịch sử phải giải mã lại và mã hóa lại đồng loạt $\rightarrow$ gây downtime hệ thống và rủi ro mất mát dữ liệu.
3. **Phương án C: Envelope Encryption 2 lớp (DEK cục bộ + KEK quản lý bởi HashiCorp Vault Transit)**

---

## 2. Quyết Định Kiến Trúc (Decision)

Hệ thống quyết định áp dụng **Phương án C: Envelope Encryption 2 lớp kết hợp HashiCorp Vault Transit Engine và Short-lived In-memory DEK Cache**:

1. **Vòng đời mã hóa (Write Path):**
   - Mỗi bản ghi nhân viên sinh một khóa **DEK (Data Encryption Key)** ngẫu nhiên 256-bit trong RAM.
   - Dùng DEK mã hóa số CCCD với giải thuật **AES-256-GCM** (12-byte Nonce ngẫu nhiên duy nhất cho mỗi lần ghi, kèm Authentication Tag chống tamper).
   - Gọi HashiCorp Vault Transit API (`POST /v1/transit/encrypt/profile-pii-kek`) để khóa **KEK (Key Encryption Key)** trong Vault bọc lại DEK thành `encrypted_dek`.
   - Lưu trữ vào DB 4 trường: `id_card_encrypted`, `id_card_nonce`, `encrypted_dek`, `kek_version`.
   - Ngay sau đó, toàn bộ vùng nhớ RAM chứa DEK plaintext được xóa sạch (`zeroize`).

2. **Vòng đời xoay khóa (Key Rotation - Zero Downtime):**
   - Khi xoay KEK từ version 1 lên version 2 trên Vault, Background Job chỉ cần gọi lệnh Vault `RewrapDEK` để bọc lại `encrypted_dek` với KEK v2.
   - **Tuyệt đối không giải mã lại ciphertext CCCD trong DB**, đạt SLA Zero-Downtime $100\%$. Cả 2 phiên bản KEK v1 và v2 cùng giải mã được đồng thời trong quá trình chuyển tiếp.

3. **Tối ưu luồng đọc với Short-Lived In-Memory DEK Cache:**
   - Để tránh việc HR duyệt danh sách thợ xưởng gọi Vault liên tục gây nghẽn, triển khai bộ đệm LRU (`golang-lru/v2`) lưu DEK unwrap trong RAM của Pod với TTL ngắn (3 - 5 phút).
   - Key cache: `SHA-256(encrypted_dek)`.
   - Cài đặt hook `OnEvict` tự động `zeroize` vùng nhớ (`dek[i] = 0`). **Tuyệt đối không lưu DEK ra đĩa cứng hay Redis dùng chung**.

---

## 3. Hệ Quả & Đánh Đổi (Consequences & Trade-offs)

- **Tích cực:**
  - Bảo mật tối đa: Dù hacker có dump toàn bộ Database PostgreSQL cũng không thể giải mã được CCCD vì không có KEK (nằm trong Vault) và không có DEK plaintext.
  - Xoay khóa định kỳ không cần downtime.
  - Đáp ứng 100% tiêu chuẩn kiểm toán bảo mật Nghị định 13/2023/NĐ-CP.
- **Đánh đổi có chủ đích (Security > Availability cho Admin Plane):**
  - Màn hình HR xem CCCD có phụ thuộc ngoài vào HashiCorp Vault. Nếu Vault tạm thời mất kết nối, HR không thể xem số CCCD dù DB vẫn chạy.
  - Đánh đổi này hoàn toàn chấp nhận được vì luồng xem CCCD chỉ là tác vụ nội bộ của HR, hoàn toàn nằm ngoài luồng mua kẹo của khách hàng (Checkout Critical Path).
