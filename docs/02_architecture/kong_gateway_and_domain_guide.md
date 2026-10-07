# HƯỚNG DẪN CẤU HÌNH DOMAIN MIỄN PHÍ & VẬN HÀNH KONG API GATEWAY
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

---

## MỤC LỤC CHI TIẾT

1. [Tổng Quan Bối Cảnh Hệ Thống & Vai Trò API Gateway](#1-tổng-quan-bối-cảnh-hệ-thống--vai-trò-api-gateway)
   - 1.1. Bối cảnh kiến trúc hệ thống Mè Xửng O Mạ
   - 1.2. Vị trí và nhiệm vụ cốt lõi của Kong Gateway tại Tầng Biên (North - South)
2. [Chiến Lược Chọn Dịch Vụ Domain & Phân Giải Tên Miền Miễn Phí](#2-chiến-lược-chọn-dịch-vụ-domain--phân-giải-tên-miền-miễn-phí)
   - 2.1. Phân loại nhu cầu phân giải tên miền trong dự án
   - 2.2. Kịch bản 1: Môi trường Local / Dev nội bộ (Không cần Internet)
   - 2.3. Kịch bản 2: Môi trường Webhook / Internet Dev (VietQR, Shopee, TikTok Shop)
   - 2.4. Kịch bản 3: Tên miền và DNS miễn phí cho Demo / Staging công khai
   - 2.5. Ma trận đánh giá và khuyến nghị giải pháp theo từng giai đoạn
3. [Hướng Dẫn Cài Đặt Kong Gateway Bằng Docker Compose](#3-hướng-dẫn-cài-đặt-kong-gateway-bằng-docker-compose)
   - 3.1. So sánh Kong DB-less Mode vs Kong DB Mode (PostgreSQL)
   - 3.2. Cấu trúc thư mục triển khai chuẩn trong `services/api-gateway`
   - 3.3. File `docker-compose.yml` triển khai Kong Gateway DB-less
4. [Đặc Tả Cấu Hình Kong Gateway (`kong.yml` Declarative Configuration)](#4-đặc-tả-cấu-hình-kong-gateway-kongyml-declarative-configuration)
   - 4.1. File cấu hình mẫu đầy đủ cho 18 Microservices
   - 4.2. Cấu hình định tuyến (Services & Routes)
   - 4.3. Cấu hình các Plugins trọng yếu (CORS, JWT, Header Sanitizer, Rate Limit, Prometheus, Trace)
5. [Giải Thích Chi Tiết Kiến Trúc: Tại Sao Cấu Hình Như Vậy?](#5-giải-thích-chi-tiết-kiến-trúc-tại-sao-cấu-hình-như-vậy)
   - 5.1. Triết lý DB-less và GitOps trong quản lý hạ tầng Gateway
   - 5.2. Chống giả mạo danh tính (Header Spoofing Prevention) & Zero Trust
   - 5.3. Xác thực JWT phi tập trung (Decentralized JWT Validation) qua JWKS
   - 5.4. Chiến lược phân tầng Rate Limiting theo nhóm tác nhân
   - 5.5. Cơ chế cách ly lỗi và bảo vệ dịch vụ hạ nguồn (Resilience & Upstream Timeout)
6. [Cẩm Nang Debug & Vận Hành Kong Gateway](#6-cẩm-nang-debug--vận-hành-kong-gateway)
   - 6.1. Kiểm tra tính đúng đắn của tệp cấu hình (Config Validation)
   - 6.2. Kiểm tra trạng thái và truy vấn Kong Admin API
   - 6.3. Đọc hiểu và phân tích Header chẩn đoán của Kong (`X-Kong-*`)
   - 6.4. Tra cứu và xử lý các mã lỗi kinh điển (502, 504, 401, 403, 404, 429, CORS)
   - 6.5. Kịch bản kiểm thử tự động bằng cURL và PowerShell
7. [Lộ Trình Triển Khai Thực Tế Cho Dự Án](#7-lộ-trình-triển-khai-thực-tế-cho-dự-án)

---

## 1. TỔNG QUAN BỐI CẢNH HỆ THỐNG & VAI TRÒ API GATEWAY

### 1.1. Bối Cảnh Kiến Trúc Hệ Thống Mè Xửng O Mạ

Hệ thống **Mè Xửng O Mạ** là nền tảng thương mại điện tử đa kênh và quản trị chuỗi cung ứng sản xuất đặc sản Huế đạt chuẩn OCOP. Kiến trúc hệ thống bao gồm:
- **18 Microservices độc lập** phục vụ các miền nghiệp vụ: Bán hàng D2C (`order-service`, `catalog-service`, `promotion-service`), Kho xưởng & Chuỗi cung ứng (`inventory-service`, `fulfillment-service`, `traceability-service`, `procurement-service`, `shipping-service`), Kênh sàn TMĐT & POS (`channel-service`), Tài chính & Khách hàng (`profile-service`, `finance-service`, `care-service`, `identity-service`, `notification-service`, `marketing-service`, `content-service`, `analytics-service`, `audit-service`).
- **3 Mặt phẳng kiến trúc độc lập (Three Architectural Planes):**
  1. *Business Plane:* Xử lý giao dịch mua sắm, xuất nhập kho theo nguyên tắc FEFO, điều phối luồng Hybrid Saga.
  2. *Audit Plane:* Lưu trữ bằng chứng pháp lý và an ninh bất biến (Tamper-evident Hash Chain) độc lập hoàn toàn với luồng bán hàng.
  3. *Observability Plane:* Thu thập Metrics, Traces, Logs qua giao thức OTLP về OpenTelemetry Collector, Jaeger và Prometheus.
- **Đa dạng kênh tương tác (Client Surface):**
  - Web D2C cho khách hàng cá nhân (`omamx.vn` / `api.omama.vn`)
  - Web B2B cho khách hàng sỉ và đối tác quà tặng (`b2b.omamx.vn`)
  - Web Admin CMS nội bộ xưởng (`admin.omamx.vn`)
  - Tablet Offline POS tại quầy xưởng Hương Thủy
  - Native Mobile App (Flutter) thông qua Mobile BFF (GraphQL)
  - Webhook đối tác bên ngoài: Cổng thanh toán VietQR / Ngân hàng, Sàn TMĐT Shopee/TikTok Shop, và Đơn vị vận chuyển 3PL (GHN, ViettelPost).

```text
                                CLIENT SURFACES
       ┌───────────────────────────────┬───────────────────────────────┐
       │ Web D2C / B2B Portal (REST)   │ Mobile App (GraphQL via BFF) │
       │ Web Admin CMS / POS Quầy      │ Webhooks (VietQR, Shopee...)  │
       └───────────────────────────────┴───────────────────────────────┘
                                       │ HTTPS (Port 443 / 8000)
                                       ▼
                       ┌───────────────────────────────┐
                       │       KONG API GATEWAY        │
                       │ ───────────────────────────── │
                       │ • TLS Termination             │
                       │ • Strip X-User-* Headers      │
                       │ • Local JWT Validation (JWKS) │
                       │ • Inject Trusted Context      │
                       │ • Global Rate Limiting        │
                       │ • Correlation ID (Trace-ID)   │
                       │ • Prometheus Metrics Export   │
                       └───────────────┬───────────────┘
                                       │ Mạng nội bộ Docker / Private Network
        ┌──────────────────────────────┼──────────────────────────────┐
        ▼                              ▼                              ▼
 MS-16: identity-service        MS-04: order-service           MS-15: profile-service
 (Port: 8016)                   (Port: 8004)                   (Port: 8080/8015)
        │                              │                              │
        ▼                              ▼                              ▼
 MS-05: catalog-service         MS-01: inventory-service       MS-13: channel-service
 (Port: 8005)                   (Port: 8001)                   (Port: 8013)
```

---

### 1.2. Vị Trí Và Nhiệm Vụ Cốt Lõi Của Kong Gateway Tại Tầng Biên (North - South)

Theo thiết kế tại tài liệu kiến trúc [High-Level Design](./high_level_design.md) (Mục 2.2) và [System Design](./system_design.md) (Mục 10), Kong Gateway đóng vai trò là "người gác cổng" duy nhất tiếp nhận lưu lượng từ Internet vào hạ tầng nội bộ:

1. **Chấm dứt mã hóa SSL/TLS (TLS Termination):** Tiếp nhận kết nối HTTPS từ Client, giải mã chứng chỉ SSL và chuyển tiếp các gói tin HTTP trong mạng riêng (Docker bridge network), giải phóng năng lực tính toán mã hóa cho 18 microservices.
2. **Loại bỏ dữ liệu mạo danh (Header Sanitization):** Client bên ngoài có thể cố tình gửi các header giả mạo như `X-User-Id: 1` hoặc `X-User-Role: ADMIN`. Kong Gateway bắt buộc phải xóa sạch (`strip`) toàn bộ header bắt đầu bằng tiền tố `X-User-*` và `X-Internal-*` trước khi chuyển tiếp vào bên trong.
3. **Xác thực JWT phi tập trung (Decentralized Local JWT Validation):** Thay vì mỗi request đi vào lại gọi RPC sang `identity-service` (gây nghẽn hệ thống và biến Identity thành điểm sụp đổ dây chuyền), Kong Gateway tải bộ khóa công khai (**JWKS**) và tự thẩm định chữ ký mật mã (Cryptographic Signature) trong **< 0.1ms**.
4. **Tiêm ngữ cảnh tin cậy (Trusted Context Injection):** Sau khi xác thực JWT thành công, Kong tự động giải mã payload và tiêm các header định danh chính thống (`X-User-Id`, `X-User-Role`, `X-User-Permissions`) vào request gửi tới backend.
5. **Giới hạn lưu lượng (Global Rate Limiting):** Bảo vệ các endpoint nhạy cảm (Đăng nhập, Quét giỏ hàng, Checkout thanh toán, Webhook VietQR) khỏi các cuộc tấn công Brute-force hoặc quét dữ liệu (Web Scraping).
6. **Truy vết phân tán (Distributed Tracing Context):** Tự động sinh `X-Request-Id` (UUID) và gắn nhãn W3C Trace Context nếu client chưa có, chuyển tiếp xuyên suốt qua microservices và log.
7. **Chia sẻ tài nguyên liên nguồn gốc (CORS):** Quản lý tập trung các chính sách CORS cho Web Next.js, Web Admin Vite và POS Tablet, tránh lỗi cấu hình rải rác ở từng service.

---

## 2. CHIẾN LƯỢC CHỌN DỊCH VỤ DOMAIN & PHÂN GIẢI TÊN MIỀN MIỄN PHÍ

Khi phát triển hệ thống trong đồ án kỹ thuật / môi trường học tập (PBL), việc mua domain quốc tế (`.com`, `.vn`) và thuê IP tĩnh (Static IP) thường tốn kém và không bắt buộc. Dưới đây là chiến lược cấu hình domain và DNS hoàn toàn **MIỄN PHÍ (100% FREE)** cho từng giai đoạn của dự án.

### 2.1. Phân Loại Nhu Cầu Phân Giải Tên Miền Trong Dự Án

Hệ sinh thái Mè Xửng O Mạ cần phân giải các domain sau:
- `omamx.vn` hoặc `omamx.local`: Cổng thông tin & Website thương mại D2C.
- `api.omamx.vn` hoặc `api.omamx.local`: Cổng API Gateway tập trung.
- `admin.omamx.vn` hoặc `admin.omamx.local`: Web Admin CMS cho Quản đốc xưởng, Kế toán, CSKH.
- `b2b.omamx.vn`: Cổng báo giá quà tặng doanh nghiệp B2B.
- Endpoint nhận Webhook từ Internet: Cần một URL công khai có HTTPS hợp lệ để ngân hàng (VietQR Callback) và sàn TMĐT (Shopee/TikTok Shop) bắn dữ liệu về máy local của nhóm phát triển.

---

### 2.2. Kịch Bản 1: Môi Trường Local / Dev Nội Bộ (Không Cần Internet)

Áp dụng khi lập trình trên một máy cá nhân (Localhost) hoặc kiểm thử qua mạng Wi-Fi phòng lab trường Đại học Bách Khoa (DUT).

#### Giải Pháp 1A: Sử Dụng File `hosts` Cục Bộ (Đơn Giản Nhất Cho Máy Cá Nhân)
Hệ điều hành Windows và Linux đều có file `hosts` để ghi đè phân giải DNS cục bộ trước khi hỏi DNS Server ngoài Internet.

**Các bước cấu hình trên Windows:**
1. Nhấn phím `Windows`, gõ `Notepad`, click chuột phải chọn **Run as administrator**.
2. Mở file theo đường dẫn: `C:\Windows\System32\drivers\etc\hosts`
3. Thêm các dòng sau vào cuối file:
   ```text
   # Phân giải domain cho Hệ thống Mè Xửng O Mạ (Local Development)
   127.0.0.1   omamx.local
   127.0.0.1   api.omamx.local
   127.0.0.1   admin.omamx.local
   127.0.0.1   b2b.omamx.local
   ```
4. Lưu file lại.
5. Kiểm tra kết quả trong terminal PowerShell:
   ```powershell
   ping api.omamx.local
   # Kết quả trả về: Reply from 127.0.0.1
   ```

*Ưu điểm:* Hoạt động ngay lập tức không cần mạng Internet.  
*Hạn chế:* Chỉ có tác dụng trên máy cá nhân đã chỉnh file `hosts`; điện thoại chạy Mobile App Flutter hoặc máy của thành viên khác trong nhóm không truy cập được.

---

#### Giải Pháp 1B: Sử Dụng Magic Wildcard DNS `nip.io` hoặc `sslip.io` (Khuyến Nghị Hàng Đầu Cho Team Dev)
Nếu bạn không muốn chỉnh sửa file `hosts` trên từng máy tính, hoặc cần điện thoại di động kết nối vào API Gateway chạy trên laptop qua mạng Wi-Fi nội bộ:
- `nip.io` và `sslip.io` là các dịch vụ DNS công cộng miễn phí. Bất kỳ tên miền nào có định dạng `<IP>.nip.io` sẽ tự động được máy chủ DNS toàn cầu phân giải về đúng `<IP>` đó.

**Ví dụ ứng dụng:**
- Chạy trên localhost:
  - `127.0.0.1.nip.io` $\rightarrow$ phân giải về `127.0.0.1`
  - `api.127.0.0.1.nip.io` $\rightarrow$ phân giải về `127.0.0.1`
  - `admin.127.0.0.1.nip.io` $\rightarrow$ phân giải về `127.0.0.1`
- Chạy trong mạng Wi-Fi Lab (giả sử laptop của bạn có IP mạng LAN là `192.168.1.50`):
  - `api.192.168.1.50.nip.io` $\rightarrow$ phân giải về `192.168.1.50`
  - Khi này, bạn có thể cầm điện thoại chạy app Flutter, nhập Base URL là `http://api.192.168.1.50.nip.io:8000` và kết nối trực tiếp vào Kong Gateway mà không cần cấu hình gì trên điện thoại!

*Ưu điểm:* 100% miễn phí, không cần cài đặt phần mềm nào, hoạt động trên mọi thiết bị trong mạng.

---

### 2.3. Kịch Bản 2: Môi Trường Webhook / Internet Dev (VietQR, Shopee, TikTok Shop)

Khi tích hợp cổng thanh toán VietQR hoặc Webhook sàn TMĐT, ngân hàng hoặc Shopee bắt buộc phải gọi vào một **Public HTTPS URL** (cổng thanh toán cấm gọi về `localhost` hay IP riêng `192.168.x.x`).

#### Giải Pháp 2A: Cloudflare Tunnels (Zero Trust / `cloudflared`) - LỰA CHỌN TỐT NHẤT & AN TOÀN NHẤT
Cloudflare cung cấp tính năng **Cloudflare Tunnel (trước đây là Argo Tunnel)** hoàn toàn **MIỄN PHÍ 100%**. Nó tạo một đường hầm mã hóa bảo mật từ máy của bạn lên mạng lưới biên của Cloudflare mà **KHÔNG CẦN mở port modem (Port Forwarding), KHÔNG CẦN IP tĩnh**.

**Cách 1: Quick Tunnel (Không cần đăng ký tài khoản, có ngay URL HTTPS tạm thời):**
1. Tải file thực thi `cloudflared.exe` từ trang chủ Cloudflare (hoặc qua `winget install --id Cloudflare.cloudflared`).
2. Mở PowerShell và chạy lệnh trỏ về cổng HTTP của Kong (Port 8000):
   ```powershell
   cloudflared tunnel --url http://localhost:8000
   ```
3. Cloudflare sẽ in ra màn hình một đường dẫn dạng:
   ```text
   https://random-words-huemexung.trycloudflare.com
   ```
4. Copy link này dán vào cấu hình Webhook URL của VietQR Sandbox hoặc Shopee Open Platform. Mọi request gửi tới link này sẽ được Cloudflare tự động chuyển vào Kong Gateway cổng 8000 trên máy của bạn với chứng chỉ SSL hợp lệ.

**Cách 2: Named Tunnel với tên miền riêng miễn phí (Cố định, ổn định cho cả đồ án):**
- Nếu bạn có một tên miền miễn phí (qua DuckDNS hoặc gói GitHub Student), bạn gán domain đó vào tài khoản Cloudflare Free.
- Chạy `cloudflared tunnel run <tên-tunnel>` để cố định URL `https://api.omamx-dev.yourdomain.com` trỏ về Kong Gateway.

---

#### Giải Pháp 2B: Dịch Vụ Ngrok / LocalXpose / Pinggy (Dự Phòng Nhanh)
- **Ngrok:** Đăng ký tài khoản free tại `ngrok.com`, chạy lệnh:
  ```powershell
  ngrok http 8000
  ```
  Ngrok cung cấp 1 domain ngẫu nhiên kèm HTTPS. Bản free giới hạn số lượng request/phút và cảnh báo màn hình trung gian, phù hợp cho việc test nhanh trong 15 phút.

---

### 2.4. Kịch Bản 3: Tên Miền Và DNS Miễn Phí Cho Demo / Staging Công Khai

Khi cần dựng một môi trường Demo hoàn chỉnh để báo cáo hội đồng đồ án (PBL) hoặc cho giảng viên đánh giá trực tuyến:

#### 1. Đăng ký Domain miễn phí:
- **DuckDNS (Dynamic DNS hoàn toàn miễn phí):**
  - Truy cập `https://www.duckdns.org`, đăng nhập qua GitHub.
  - Tạo một subdomain miễn phí: ví dụ `omamx.duckdns.org`.
  - DuckDNS hỗ trợ cập nhật IP công khai của bạn thông qua Script hoặc Docker Container:
    ```bash
    curl "https://www.duckdns.org/update?domains=omamx&token=YOUR_TOKEN&ip="
    ```
- **Gói GitHub Student Developer Pack:**
  - Sinh viên trường Đại học Bách Khoa (sở hữu email `@sv.dut.udn.vn` hoặc thẻ sinh viên) được tặng:
    - **1 Tên miền `.me` miễn phí 1 năm** từ Namecheap.
    - Tài khoản GitHub Pro, voucher Cloudflare, DigitalOcean credit.
  - Đây là phương án chính quy nhất để sở hữu domain đẹp như `omamx-hue.me`.

#### 2. Dịch vụ phân giải DNS miễn phí tốt nhất thế giới: **Cloudflare DNS**
- Cho dù sở hữu domain từ Namecheap hay bất kỳ nhà đăng ký nào, hãy trỏ NameServer về **Cloudflare (Gói Free)**.
- Lợi ích của Cloudflare DNS:
  - Phân giải cực nhanh (Anycast DNS 1.1.1.1).
  - Tự động cấp phát chứng chỉ SSL/TLS miễn phí (Universal SSL).
  - Tích hợp sẵn Web Application Firewall (WAF) và chống DDoS tầng 3/4/7.
  - Hỗ trợ gán bản ghi CNAME, A Record không giới hạn.

---

### 2.5. Ma Trận Đánh Giá Và Khuyến Nghị Lựa Chọn

| Kịch Bản Sử Dụng | Giải Pháp Tối Ưu | Chi Phí | Có HTTPS? | Yêu Cầu Setup | Đánh Giá Độ Ổn Định |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Local Dev máy cá nhân** | `hosts` file (`omamx.local`) | 0 VNĐ | Tự ký (Self-signed) | 1 phút | ⭐⭐⭐⭐⭐ (Tuyệt đối độc lập) |
| **Test Mobile App qua Wi-Fi** | Wildcard `nip.io` (`api.<IP>.nip.io`) | 0 VNĐ | HTTP / Optional | 0 phút (Chỉ cần kết nối Wi-Fi) | ⭐⭐⭐⭐⭐ (Rất tiện lợi cho nhóm) |
| **Nhận Webhook VietQR / Shopee** | Cloudflare Quick Tunnel (`trycloudflare`) | 0 VNĐ | HTTPS chuẩn quốc tế | 1 phút (Chạy 1 dòng lệnh) | ⭐⭐⭐⭐⭐ (Không cần mở port) |
| **Demo đồ án trực tuyến** | GitHub Student Domain + Cloudflare DNS | 0 VNĐ | HTTPS tự động | 15 phút | ⭐⭐⭐⭐⭐ (Chuyên nghiệp nhất) |

---

## 3. HƯỚNG DẪN CÀI ĐẶT KONG GATEWAY BẰNG DOCKER COMPOSE

### 3.1. So Sánh Kong DB-less Mode vs Kong DB Mode (PostgreSQL)

Kong Gateway có 2 chế độ hoạt động:
1. **DB Mode:** Cần một database PostgreSQL riêng để lưu trữ bảng cấu hình Services, Routes, Plugins. Thay đổi cấu hình qua Kong Admin REST API (`POST http://localhost:8001/services`).
2. **DB-less Mode (Declarative Configuration):** Toàn bộ cấu hình Services, Routes, Upstreams, Plugins được định nghĩa trong một file YAML duy nhất (`kong.yml`) nạp thẳng vào bộ nhớ RAM của Kong khi khởi động.

#### Tại sao dự án Mè Xửng O Mạ bắt buộc chọn DB-less Mode?
- **Triết lý GitOps & Code-as-Configuration:** File `kong.yml` được lưu trữ trực tiếp trong Git repository (`services/api-gateway/kong.yml`). Toàn bộ lịch sử thay đổi cấu hình gateway được theo dõi, review qua Pull Request, loại bỏ hoàn toàn nguy cơ cấu hình sai lệch giữa các thành viên.
- **Tiết kiệm tài nguyên phần cứng:** DB Mode tiêu tốn thêm một container PostgreSQL chạy ngầm (tiêu hao từ 300MB - 500MB RAM chỉ để lưu vài chục dòng route). DB-less mode loại bỏ hoàn toàn container DB này, giúp Kong khởi động chỉ mất **dưới 1 giây** và chiếm chưa tới **80MB RAM**.
- **Tính Bất Biến (Immutability):** Tránh hiện tượng một cá nhân gọi Admin API sửa tạm thời trên môi trường live làm hỏng hệ thống mà không ai hay biết.
- **Tốc độ phục hồi sự cố (Disaster Recovery):** Khi container bị crash hoặc cần scale ra nhiều pod trên Kubernetes, Kong chỉ việc đọc file YAML và sẵn sàng phục vụ tức thì mà không cần migrate schema DB.

---

### 3.2. Cấu Trúc Thư Mục Triển Khai Chuẩn Trong `services/api-gateway`

```text
services/api-gateway/
├── docker-compose.yml       # Khởi chạy Kong Gateway container
├── config/
│   └── kong.yml             # Cấu hình Declarative (Services, Routes, Plugins)
├── certs/                   # (Tùy chọn) Chứng chỉ SSL cục bộ nếu cần test HTTPS
└── README.md                # Hướng dẫn nhanh cho thành viên nhóm
```

---

### 3.3. File `docker-compose.yml` Triển Khai Kong Gateway DB-less

Tạo file `services/api-gateway/docker-compose.yml` với nội dung chuẩn hóa sau:

```yaml
version: '3.8'

networks:
  omamx-network:
    name: omamx-network
    external: true

services:
  kong-gateway:
    image: kong:3.9
    container_name: omamx-kong-gateway
    restart: unless-stopped
    networks:
      - omamx-network
    ports:
      # North-South Proxy Ports (Cổng giao tiếp Client)
      - "8000:8000"     # HTTP Proxy (Mặc định cho Local/Dev)
      - "8443:8443"     # HTTPS Proxy
      # Admin Management Ports (Chỉ bind vào localhost 127.0.0.1 để bảo mật)
      - "127.0.0.1:8001:8001" # Admin API (Chế độ Read-only khi chạy DB-less)
      - "127.0.0.1:8444:8444" # Admin API HTTPS
      - "127.0.0.1:8002:8002" # Kong Manager GUI (Giao diện xem trực quan)
    environment:
      # 1. Kích hoạt chế độ DB-less
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: "/etc/kong/kong.yml"

      # 2. Cấu hình cổng lắng nghe
      KONG_PROXY_LISTEN: "0.0.0.0:8000, 0.0.0.0:8443 ssl"
      KONG_ADMIN_LISTEN: "0.0.0.0:8001, 0.0.0.0:8444 ssl"
      KONG_ADMIN_GUI_LISTEN: "0.0.0.0:8002"

      # 3. Kích hoạt Kong Manager giao diện web (Xem cấu hình miễn phí)
      KONG_ADMIN_GUI_URL: "http://localhost:8002"

      # 4. Tối ưu hóa ghi log cho việc Debug & Observability
      KONG_LOG_LEVEL: "info" # Khi cần debug đổi thành 'debug'
      KONG_PROXY_ACCESS_LOG: "/dev/stdout"
      KONG_PROXY_ERROR_LOG: "/dev/stderr"
      KONG_ADMIN_ACCESS_LOG: "/dev/stdout"
      KONG_ADMIN_ERROR_LOG: "/dev/stderr"

      # 5. Phân giải DNS nội bộ của Docker (Bắt buộc để gọi tên container)
      KONG_DNS_RESOLVER: "127.0.0.11"
      KONG_DNS_ORDER: "LAST,A,CNAME"

      # 6. Tối ưu bộ đệm và kết nối
      KONG_UPSTREAM_KEEPALIVE_POOL_SIZE: 64
      KONG_UPSTREAM_KEEPALIVE_MAX_REQUESTS: 1000
      KONG_UPSTREAM_KEEPALIVE_IDLE_TIMEOUT: 60
    volumes:
      - ./config/kong.yml:/etc/kong/kong.yml:ro
    healthcheck:
      test: ["CMD", "kong", "health"]
      interval: 5s
      timeout: 3s
      retries: 5
```

> [!IMPORTANT]
> **Quy tắc mạng Docker:** Mọi microservice (như `omamx-profile-service`, `omamx-order-service`) và Kong Gateway phải cùng tham gia vào mạng Docker `omamx-network`. Trước khi khởi chạy, tạo mạng bằng lệnh:
> ```powershell
> docker network create omamx-network
> ```

---

## 4. ĐẶC TẢ CẤU HÌNH KONG GATEWAY (`kong.yml` DECLARATIVE CONFIGURATION)

### 4.1. File Cấu Hình Mẫu Đầy Đủ

Tạo file `services/api-gateway/config/kong.yml`:

```yaml
_format_version: "3.0"
_transform: true

# ==============================================================================
# 1. DANH MỤC DỊCH VỤ NỘI BỘ (SERVICES) & ĐỊNH TUYẾN (ROUTES)
# ==============================================================================
services:
  # ----------------------------------------------------------------------------
  # MS-16: IDENTITY SERVICE (Xác thực, Cấp Token, JWKS)
  # ----------------------------------------------------------------------------
  - name: identity-service
    url: http://identity-service:8016
    connect_timeout: 3000
    read_timeout: 5000
    write_timeout: 5000
    retries: 2
    routes:
      - name: auth-public-routes
        paths:
          - /api/v1/auth/login
          - /api/v1/auth/register
          - /api/v1/auth/refresh
          - /api/v1/auth/forgot-password
          - /.well-known/jwks.json
        strip_path: false
        methods:
          - POST
          - GET
          - OPTIONS

  # ----------------------------------------------------------------------------
  # MS-15: PROFILE SERVICE (Hồ sơ người dùng, Sổ địa chỉ, Nhân sự thợ xưởng)
  # ----------------------------------------------------------------------------
  - name: profile-service
    url: http://omamx-profile-service:8080
    connect_timeout: 2000
    read_timeout: 5000
    write_timeout: 5000
    retries: 1
    routes:
      - name: profile-customer-routes
        paths:
          - /api/v1/profile
          - /api/v1/profiles
        strip_path: false
        methods:
          - GET
          - POST
          - PUT
          - PATCH
          - DELETE
          - OPTIONS
        plugins:
          - name: jwt
            config:
              claims_to_verify:
                - exp
              secret_is_base64: false
      - name: profile-admin-routes
        paths:
          - /api/v1/admin/employees
          - /api/v1/admin/customers
        strip_path: false
        methods:
          - GET
          - POST
          - PUT
          - DELETE
          - OPTIONS
        plugins:
          - name: jwt
            config:
              claims_to_verify:
                - exp

  # ----------------------------------------------------------------------------
  # MS-05: CATALOG SERVICE (Danh mục kẹo mè xửng, biến thể, giá niêm yết OCOP)
  # ----------------------------------------------------------------------------
  - name: catalog-service
    url: http://catalog-service:8005
    connect_timeout: 2000
    read_timeout: 3000
    write_timeout: 3000
    retries: 2
    routes:
      - name: catalog-public-routes
        paths:
          - /api/v1/products
          - /api/v1/categories
          - /api/v1/search
        strip_path: false
        methods:
          - GET
          - OPTIONS

  # ----------------------------------------------------------------------------
  # MS-04: ORDER SERVICE (Giỏ hàng, Checkout Critical Path Saga, Tra cứu đơn)
  # ----------------------------------------------------------------------------
  - name: order-service
    url: http://order-service:8004
    connect_timeout: 2000
    read_timeout: 8000 # Cho phép thời gian thực thi Saga
    write_timeout: 8000
    retries: 0        # CẤM RETRY TRÊN CHECKOUT TRÁNH LẶP TẠO ĐƠN
    routes:
      - name: cart-routes
        paths:
          - /api/v1/cart
        strip_path: false
        methods:
          - GET
          - POST
          - PUT
          - DELETE
          - OPTIONS
      - name: order-checkout-routes
        paths:
          - /api/v1/orders
        strip_path: false
        methods:
          - GET
          - POST
          - OPTIONS
        plugins:
          - name: jwt
            config:
              claims_to_verify:
                - exp
          - name: rate-limiting
            config:
              minute: 30 # Giới hạn 30 đơn/phút mỗi user/IP
              policy: local

  # ----------------------------------------------------------------------------
  # MS-07: PROMOTION SERVICE (Mã giảm giá, Voucher O Mạ, Loyalty)
  # ----------------------------------------------------------------------------
  - name: promotion-service
    url: http://promotion-service:8007
    connect_timeout: 2000
    read_timeout: 3000
    write_timeout: 3000
    retries: 1
    routes:
      - name: promotion-routes
        paths:
          - /api/v1/promotions
          - /api/v1/vouchers
        strip_path: false
        methods:
          - GET
          - POST
          - OPTIONS

  # ----------------------------------------------------------------------------
  # MS-13: CHANNEL SERVICE (POS Quầy Xưởng & Webhook Sàn TMĐT Shopee/TikTok)
  # ----------------------------------------------------------------------------
  - name: channel-service
    url: http://channel-service:8013
    connect_timeout: 3000
    read_timeout: 5000
    write_timeout: 5000
    retries: 1
    routes:
      - name: marketplace-webhooks
        paths:
          - /webhooks/marketplace
        strip_path: false
        methods:
          - POST
          - OPTIONS
        plugins:
          - name: rate-limiting
            config:
              minute: 1200 # Cho phép burst lớn khi có flash sale sàn
              policy: local

  # ----------------------------------------------------------------------------
  # PARTNER WEBHOOKS: VIETQR / PAYMENT GATEWAY CALLBACK
  # ----------------------------------------------------------------------------
  - name: payment-webhook-handler
    url: http://order-service:8004
    connect_timeout: 3000
    read_timeout: 5000
    write_timeout: 5000
    retries: 1
    routes:
      - name: vietqr-callback-route
        paths:
          - /payments/vietqr/callback
        strip_path: false
        methods:
          - POST
        plugins:
          - name: rate-limiting
            config:
              minute: 600
              policy: local

  # ----------------------------------------------------------------------------
  # MOBILE BFF: GRAPHQL ENDPOINT DÀNH RIÊNG CHO MOBILE APP
  # ----------------------------------------------------------------------------
  - name: mobile-bff-service
    url: http://mobile-bff:4000
    connect_timeout: 2000
    read_timeout: 5000
    write_timeout: 5000
    retries: 1
    routes:
      - name: mobile-graphql-route
        paths:
          - /graphql
        strip_path: false
        methods:
          - POST
          - OPTIONS

# ==============================================================================
# 2. CÁC PLUGINS TOÀN CỤC (GLOBAL PLUGINS APPLIED TO ALL TRAFFIC)
# ==============================================================================
plugins:
  # ----------------------------------------------------------------------------
  # PLUGIN 1: CORS TOÀN CỤC (Hỗ trợ gọi từ Web Next.js, Web Admin, POS)
  # ----------------------------------------------------------------------------
  - name: cors
    config:
      origins:
        - "*" # Môi trường dev mở rộng; Production giới hạn domain omamx.vn
      methods:
        - GET
        - POST
        - PUT
        - PATCH
        - DELETE
        - OPTIONS
      headers:
        - Accept
        - Accept-Version
        - Content-Length
        - Content-Type
        - Authorization
        - X-Request-Id
        - X-Idempotency-Key
      exposed_headers:
        - X-Request-Id
        - Content-Range
      credentials: true
      max_age: 3600

  # ----------------------------------------------------------------------------
  # PLUGIN 2: CORRELATION ID (Tạo mã vết Trace-ID xuyên suốt toàn hệ thống)
  # ----------------------------------------------------------------------------
  - name: correlation-id
    config:
      header_name: X-Request-Id
      generator: uuid#counter
      echo_downstream: true

  # ----------------------------------------------------------------------------
  # PLUGIN 3: REQUEST TRANSFORMER (BẢO MẬT ZERO-TRUST: XÓA SẠCH HEADER GIẢ MẠO)
  # ----------------------------------------------------------------------------
  - name: request-transformer
    config:
      remove:
        headers:
          - X-User-Id
          - X-User-Role
          - X-User-Permissions
          - X-Internal-Caller

  # ----------------------------------------------------------------------------
  # PLUGIN 4: GLOBAL RATE LIMITING (Bảo vệ toàn hệ thống chống DDoS/Scraping)
  # ----------------------------------------------------------------------------
  - name: rate-limiting
    config:
      minute: 300 # Trung bình 5 requests/giây mỗi IP cho các luồng thông thường
      limit_by: ip
      policy: local
      fault_tolerant: true
      hide_client_headers: false

  # ----------------------------------------------------------------------------
  # PLUGIN 5: PROMETHEUS METRICS (Mặt phẳng Giám sát Observability Plane)
  # ----------------------------------------------------------------------------
  - name: prometheus
    config:
      per_consumer: false
      status_code_metrics: true
      latency_metrics: true
      bandwidth_metrics: true
      upstream_health_metrics: true

# ==============================================================================
# 3. ĐỊNH NGHĨA CONSUMERS & THÔNG TIN XÁC THỰC MẪU CHO DEV/TEST
# ==============================================================================
consumers:
  - username: omamx-test-customer
    custom_id: "customer_001"

jwt_secrets:
  - consumer: omamx-test-customer
    key: "omamx-identity-issuer"
    secret: "omamx-secret-jwt-key-for-local-development-2026-very-secure"
    algorithm: "HS256"
```

---

## 5. GIẢI THÍCH CHI TIẾT KIẾN TRÚC: TẠI SAO CẤU HÌNH NHƯ VẬY?

Phần này phân tích sâu các quyết định thiết kế kỹ thuật (Architectural Decisions) nhằm trả lời câu hỏi: *Vì sao không làm cách khác đơn giản hơn mà phải cấu hình như trên?*

### 5.1. Triết Lý DB-less Và GitOps Trong Quản Lý Hạ Tầng Gateway

Trong hệ thống gồm 18 microservices, việc cấu hình định tuyến thông qua giao diện Web UI (Kong Manager) hoặc gọi Admin API bằng tay (`curl POST /routes`) tiềm ẩn rủi ro rất lớn:
1. **Lệch cấu hình môi trường (Configuration Drift):** Thành viên A sửa timeout trên máy cá nhân để chạy được, nhưng khi chuyển code sang máy thành viên B hoặc đẩy lên Staging server thì bị lỗi `504 Gateway Timeout` do thiếu cấu hình đó.
2. **Không có lịch sử vết (Audit Trail of Infrastructure):** Ai đã thêm route? Ai đã sửa rate-limit? Ai đã tắt JWT plugin?
3. **Phục hồi thảm họa (Disaster Recovery):** Nếu container sập hoặc ổ cứng hỏng, với DB-less, ta chỉ mất đúng **1 lệnh `docker compose up -d`** là toàn bộ hạ tầng Gateway được khôi phục 100% chính xác từ file `kong.yml`.

---

### 5.2. Chống Giả Mạo Danh Tính (Header Spoofing Prevention) & Zero Trust

> [!CAUTION]
> **Lỗ hổng bảo mật chết người: Header Injection từ phía Client**  
> Giả sử một hacker gửi request mua hàng kèm HTTP Header:  
> `POST /api/v1/orders HTTP/1.1`  
> `X-User-Id: 9999`  
> `X-User-Role: SYSTEM_ADMIN`  
> 
> Nếu API Gateway không có cơ chế thanh lọc mà chuyển tiếp thẳng (Forward nguyên vẹn) các header này vào `order-service`, thì backend microservice có thể lầm tưởng request này đến từ Quản trị viên tối cao!

Để triệt tiêu hoàn toàn rủi ro này, cấu hình sử dụng plugin **`request-transformer`** ở phạm vi toàn cục (`global`):
```yaml
plugins:
  - name: request-transformer
    config:
      remove:
        headers:
          - X-User-Id
          - X-User-Role
          - X-User-Permissions
          - X-Internal-Caller
```
**Nguyên tắc vận hành:**
1. Mọi header bắt đầu bằng `X-User-*` do Client tự ý gửi lên đều bị **XÓA SẠCH NGAY TẠI CỔNG VÀO**.
2. Chỉ sau khi plugin `jwt` xác thực chữ ký mật mã thành công, Gateway mới giải mã payload của token do chính `identity-service` cấp phát và **tự tay tiêm (inject)** lại các header tin cậy.
3. Nhờ đó, 18 microservices bên trong tuyệt đối an tâm khi đọc `X-User-Id` từ context mà không sợ bị lừa dối.

---

### 5.3. Xác Thực JWT Phi Tập Trung (Decentralized JWT Validation) Qua JWKS

Trong các kiến trúc Microservices truyền thống kém tối ưu:
```text
Client ──Request──► Gateway ──gRPC CheckToken()──► Identity Service ──OK──► Service Đích
```
Mô hình trên biến `Identity Service` thành "nút cổ chai" nguy hiểm:
- Nếu Identity Service bị chậm 500ms, toàn bộ 18 dịch vụ khác bị chậm 500ms.
- Nếu Identity Service khởi động lại hoặc gặp sự cố, toàn bộ luồng mua sắm kẹo mè xửng của khách hàng trên Website và App bị tê liệt hoàn toàn!

**Giải pháp của hệ thống Mè Xửng O Mạ:**
- Áp dụng **Xác thực phi tập trung (Decentralized Verification)** bằng thuật toán bất đối xứng (Asymmetric Cryptography RSA/ECDSA) hoặc đối xứng (HMAC-SHA256):
  - `identity-service` giữ Private Key để ký phát hành Token khi người dùng Login.
  - Kong Gateway nạp Public Key (qua chuẩn `JWKS`) và lưu vào RAM.
  - Khi request ùa vào, Kong chỉ dùng Public Key giải toán học để kiểm tra chữ ký. Thao tác này diễn ra thuần túy trên CPU trong RAM, **mất dưới 0.1ms**, không phát sinh bất kỳ một gói tin mạng nào sang `identity-service`!

---

### 5.4. Chiến Lược Phân Tầng Rate Limiting Theo Nhóm Tác Nhân

Mỗi endpoint có đặc thù nghiệp vụ hoàn toàn khác biệt, do đó không thể áp chung một mức Rate Limit:
1. **Public Catalog/Search (`300 requests/phút`):** Cho phép người dùng duyệt kẹo, xem ảnh, tìm kiếm mượt mà mà không lo bị chặn.
2. **Checkout & Order Creation (`30 requests/phút`):** Giới hạn chặt chẽ để chống kịch bản bot tự động spam đặt hàng ảo làm cạn kiệt tồn kho tạm giữ (Stock Reservation Denial-of-Service).
3. **Marketplace & Payment Webhooks (`600 - 1200 requests/phút`):** Vào các ngày hội Sale như 11/11 hoặc Tết Nguyên Đán, Shopee/TikTok có thể bắn hàng trăm webhook đơn hàng mỗi giây. Nếu để rate-limit quá thấp, sàn TMĐT sẽ bị lỗi 429 và tạm khóa kết nối API của gian hàng O Mạ!

---

### 5.5. Cơ Chế Cách Ly Lỗi & Timeout Bảo Vệ Dịch Vụ Hạ Nguồn (Resilience & Upstream Timeout)

Chú ý các thông số cấu hình upstream trong `kong.yml`:
```yaml
  - name: order-service
    connect_timeout: 2000
    read_timeout: 8000
    retries: 0 # CẤM RETRY TRÊN CHECKOUT!
```
- **Tại sao `retries: 0` đối với `order-service`?**  
  Nếu mạng bị chập chờn (Network Glitch), request tạo đơn đã đến được Order Service và đang khóa kho, nhưng phản hồi trả về bị trễ. Nếu Kong Gateway tự động retry lần 2, hệ thống sẽ thực hiện tạo đơn lần nữa! Dù có `Idempotency-Key`, việc retry ở tầng Gateway dễ gây tranh chấp luồng và làm tăng tải cho database. Vì vậy, các lệnh ghi thay đổi trạng thái (Mutations) phải cấm retry tự động từ tầng Gateway.
- **Tại sao `connect_timeout` chỉ 2000ms?**  
  Nếu một container service bị chết, Kong không nên chờ đợi lâu. Hết 2 giây không bắt tay được TCP, Kong lập tức ngắt và trả mã lỗi `502 Bad Gateway` cho client để kích hoạt kịch bản phòng thủ (Fallback).

---

## 6. CẨM NANG DEBUG & VẬN HÀNH KONG GATEWAY

Khi làm việc với Kong Gateway, các sự cố về định tuyến, phân giải tên miền hoặc chặn nhầm request là điều thường gặp. Dưới đây là quy trình chẩn đoán từng bước.

### 6.1. Kiểm Tra Tính Đúng Đắn Của Tệp Cấu Hình (Config Validation)

Trước khi khởi động hoặc sau khi chỉnh sửa `kong.yml`, hãy kiểm tra xem cú pháp YAML có hợp lệ hay không:

```powershell
# Chạy lệnh kiểm tra cú pháp trực tiếp bằng container Kong 3.9
docker run --rm -e "KONG_DATABASE=off" -v "e:\DUT.K1N4\PBL\services\api-gateway\config\kong.yml:/etc/kong/kong.yml:ro" kong:3.9 kong config parse /etc/kong/kong.yml
```
- Nếu cấu hình đúng: Trả về `configuration is valid`.
- Nếu cấu hình sai (sai thụt lề YAML, sai tên plugin, thiếu trường bắt buộc): Kong sẽ chỉ rõ số dòng bị lỗi.

---

### 6.2. Kiểm Tra Trạng Thái & Truy Vấn Kong Admin API

Kong Admin API chạy tại cổng `8001` (chỉ mở cho localhost):

```powershell
# 1. Kiểm tra tình trạng sức khỏe của Kong
curl http://localhost:8001/status

# 2. Liệt kê toàn bộ các Services đang hoạt động
curl http://localhost:8001/services

# 3. Liệt kê toàn bộ các Routes đã được đăng ký
curl http://localhost:8001/routes

# 4. Kiểm tra danh sách Plugins đang bật
curl http://localhost:8001/plugins
```

---

### 6.3. Đọc Hiểu Và Phân Tích Header Chẩn Đoán Của Kong (`X-Kong-*`)

Mỗi khi gửi một request qua Kong Gateway bằng lệnh `curl -i` hoặc mở tab Network của DevTools (F12), Kong luôn đính kèm các Header chẩn đoán quý giá:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Connection: keep-alive
X-Kong-Response-Latency: 12
X-Kong-Upstream-Latency: 10
X-Kong-Proxy-Latency: 2
Via: kong/3.6.0
X-Request-Id: 7e3b5e40-8f92-4f81-a67b-f11a8b0d8792
```

**Cách phân tích độ trễ:**
- `X-Kong-Upstream-Latency: 10`: Thời gian mà microservice nội bộ (ví dụ: `profile-service`) xử lý trong DB và trả về cho Kong (ở đây mất 10ms).
- `X-Kong-Proxy-Latency: 2`: Thời gian bản thân Kong Gateway xử lý các plugin (CORS, JWT, Rate-limit, Header rewrite). Chỉ mất **2ms**, chứng minh Kong cực kỳ nhẹ và nhanh.
- `X-Kong-Response-Latency: 12`: Tổng thời gian từ lúc Kong nhận request đến lúc bắn trả về cho Client ($10ms + 2ms = 12ms$).

> [!TIP]
> Nếu thấy request bị chậm:
> - Nếu `X-Kong-Upstream-Latency` cao $\rightarrow$ Lỗi do **Microservice hoặc Database** bị chậm (cần tối ưu SQL query, redis cache).
> - Nếu `X-Kong-Proxy-Latency` cao $\rightarrow$ Lỗi do **Kong Gateway** (quá nhiều plugin nặng hoặc CPU máy host bị quá tải).

---

### 6.4. Tra Cứu Và Xử Lý Các Mã Lỗi Kinh Điển

#### Lỗi 1: `HTTP 502 Bad Gateway`
- **Nguyên nhân 1: Container đích chưa khởi động.**  
  Kong cố gắng gửi gói tin tới `http://omamx-profile-service:8080` nhưng container này chưa `RUNNING`.  
  *Cách sửa:* Chạy `docker ps` kiểm tra xem container backend đã chạy chưa.
- **Nguyên nhân 2: Khác mạng Docker (Network Mismatch).**  
  Container Kong nằm trong mạng `omamx-network`, nhưng `profile-service` lại chạy trong mạng `profile-service_default`. Do đó DNS nội bộ của Docker không tìm thấy hostname `omamx-profile-service`.  
  *Cách sửa:* Đảm bảo cả 2 file `docker-compose.yml` đều khai báo:
  ```yaml
  networks:
    omamx-network:
      name: omamx-network
      external: true
  ```
- **Nguyên nhân 3: Sai Port nội bộ.**  
  Trong `kong.yml` trỏ tới port `8080`, nhưng bên trong container service ứng dụng lại lắng nghe ở port `8000` hoặc `3000`.

---

#### Lỗi 2: `HTTP 504 Gateway Timeout`
- **Nguyên nhân:** Microservice backend mất nhiều thời gian xử lý hơn giá trị `read_timeout` được khai báo trong `kong.yml`.
- **Cách sửa:**
  - Nếu là tác vụ nặng hợp lệ (như Saga Checkout hoặc xuất báo cáo Excel Tết): Tăng `read_timeout: 10000` (10 giây).
  - Nếu do database bị deadlock hoặc nghẽn thread: Kiểm tra log của microservice backend.

---

#### Lỗi 3: `HTTP 401 Unauthorized` / `"Invalid token"`
- **Nguyên nhân 1: Sai Secret hoặc Algorithm.**  
  Token được ký bằng thuật toán `HS256` nhưng Kong cấu hình `RS256`, hoặc chuỗi `secret` trong `kong.yml` không khớp 100% với `secret` trong mã nguồn của `identity-service`.
- **Nguyên nhân 2: Token hết hạn (`Token expired`).**  
  Kiểm tra trường `exp` trong JWT payload bằng trang web `jwt.io`.
- **Nguyên nhân 3: Lệch giờ hệ thống (Clock Skew).**  
  Đồng hồ máy ảo Docker bị lệch so với giờ thực tế, khiến token vừa tạo đã bị Kong coi là đã hết hạn. Chạy lệnh đồng bộ giờ máy ảo.

---

#### Lỗi 4: `HTTP 404 Not Found` - `{"message":"no Route matched with those values"}`
- **Nguyên nhân 1: Sai đường dẫn Path.**  
  Request gửi lên là `GET /api/v1/profile` (số ít), nhưng trong `kong.yml` lại khai báo `paths: [/api/v1/profiles]` (số nhiều).
- **Nguyên nhân 2: Nhầm lẫn về `strip_path`.**  
  - Nếu `strip_path: true`: Khi client gọi `http://localhost:8000/api/v1/profiles/me`, Kong sẽ cắt bỏ `/api/v1/profiles` và chỉ chuyển tiếp `/me` tới backend service. Nếu backend của bạn viết route là `/api/v1/profiles/me`, backend sẽ báo 404!
  - **Khuyến nghị cho dự án Mè Xửng O Mạ:** Đặt `strip_path: false` cho toàn bộ các route để giữ nguyên vẹn đường dẫn API từ ngoài vào trong, tránh nhầm lẫn giữa frontend và backend.
- **Nguyên nhân 3: Sai HTTP Method.**  
  Route chỉ cho phép `GET, POST`, nhưng client gửi `DELETE`.

---

#### Lỗi 5: Lỗi CORS khi gọi từ trình duyệt (`Access-Control-Allow-Origin`)
- **Triệu chứng:** Trình duyệt Chrome báo lỗi đỏ: `Response to preflight request doesn't pass access control check: No 'Access-Control-Allow-Origin' header is present`.
- **Nguyên nhân sâu xa:** Trước khi gửi request thật (POST, PUT), trình duyệt luôn tự động bắn một request thăm dò gọi là **Preflight Request (Method `OPTIONS`)**. Nếu route yêu cầu xác thực JWT mà không cho phép `OPTIONS` đi qua tự do, plugin JWT sẽ chặn đứng request `OPTIONS` với mã lỗi `401 Unauthorized`! Khi đó trình duyệt không nhận được header CORS và hủy toàn bộ cuộc gọi.
- **Cách sửa chuẩn:**
  1. Trong mọi Route của `kong.yml`, luôn thêm method `OPTIONS`:
     ```yaml
     methods:
       - GET
       - POST
       - OPTIONS # Bắt buộc phải có
     ```
  2. Bật plugin `cors` ở cấp độ Global. Plugin CORS của Kong sẽ tự động xử lý request `OPTIONS` và trả về `200 OK` ngay lập tức kèm đầy đủ các header cho phép.

---

### 6.5. Kịch Bản Kiểm Thử Tự Động Bằng cURL Và PowerShell

Bạn có thể chạy các kịch bản sau trong PowerShell để kiểm tra hoạt động của Gateway:

```powershell
# Kịch bản 1: Kiểm tra Route công khai (Catalog)
curl -i http://localhost:8000/api/v1/products

# Kịch bản 2: Kiểm tra Route bảo vệ (Profile) khi CHƯA có Token -> Bắt buộc phải trả về 401
curl -i http://localhost:8000/api/v1/profiles/me

# Kịch bản 3: Kiểm tra Route bảo vệ VỚI Token hợp lệ (Bearer Token)
$TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." # Token lấy từ Identity Service
curl -i -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/profiles/me

# Kịch bản 4: Kiểm tra tính năng Header Sanitization (Gửi thử header giả mạo xem có bị Kong nuốt không)
curl -i -H "X-User-Id: 99999" -H "X-User-Role: FAKE_ADMIN" http://localhost:8000/api/v1/products

# Kịch bản 5: Kiểm tra Rate Limit (Bắn liên tục 35 request xem có bị 429 không)
1..35 | ForEach-Object {
    $res = curl -s -o /dev/null -w "%{http_code}`n" http://localhost:8000/api/v1/orders
    Write-Host "Request $_: Status $res"
}
```

---

## 7. LỘ TRÌNH TRIỂN KHAI THỰC TẾ CHO DỰ ÁN

Để đưa Kong Gateway vào vận hành trơn tru cùng hệ thống, nhóm phát triển nên thực hiện theo 4 bước:

1. **Bước 1 (Hạ tầng mạng):** Tạo mạng Docker chung `omamx-network` để các service nhìn thấy nhau bằng tên container.
2. **Bước 2 (Chạy Kong):** Đặt file `docker-compose.yml` và `kong.yml` vào thư mục `services/api-gateway/`, khởi chạy bằng lệnh `docker compose up -d`.
3. **Bước 3 (Đấu nối từng Service):**
   - Đấu nối `profile-service` (đã có sẵn trong repo tại cổng 8080).
   - Kiểm tra việc gọi API lấy địa chỉ và hồ sơ qua `http://localhost:8000/api/v1/profiles`.
4. **Bước 4 (Cấu hình Domain & Webhook):**
   - Dùng Magic DNS `nip.io` để cả team dev và điện thoại chạy thử.
   - Khi cần test Webhook thanh toán VietQR thật, bật `cloudflared tunnel --url http://localhost:8000` để lấy link HTTPS công khai.
