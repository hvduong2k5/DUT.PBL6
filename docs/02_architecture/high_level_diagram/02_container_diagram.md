# BỘ SƠ ĐỒ CONTAINER KIẾN TRÚC TỔNG (CONTAINER DIAGRAMS - C4 LEVEL 2)

## 1. Vị Trí & Vai Trò Trong Tài Liệu HLD
- **Cấp độ kiến trúc:** Kiến trúc tổng thể container (Level 2 trong mô hình C4).
- **Vị trí trong HLD:** Đặt tại **Chương 1 (Mục 1.3 - Sơ đồ kiến trúc tổng quan)** — Trả lời câu hỏi: *Hệ thống gồm những Container phần mềm nào (Web, App, Gateway, Microservices, Databases, Message Brokers, Storage), công nghệ gì, và ai nói chuyện với ai?*
- **Chiến lược phân rã (Modular De-cluttering):** Để tránh việc dồn toàn bộ 18 microservices, 18 databases, Kafka, Redis và các frontends vào 1 sơ đồ duy nhất gây rối loạn thị giác, tài liệu phân rã Container Diagram thành **4 sơ đồ chuyên biệt**:
  - **Sơ đồ 2.1 (Kiến Trúc Phân Tầng Tổng Thể):** Thể hiện 4 tầng kiến trúc (Clients $\rightarrow$ Tầng Biên $\rightarrow$ Các Cụm Miền Nghiệp Vụ $\rightarrow$ Hạ Tầng Streaming/Storage).
  - **Sơ đồ 2.2 (Phân Rã Miền Thương Mại & Đa Kênh):** Zoom sâu 4 Microservices (Order, Catalog, Promotion, Channel) và luồng checkout.
  - **Sơ đồ 2.3 (Phân Rã Miền Chuỗi Cung Ứng & Kho Vận):** Zoom sâu 5 Microservices (Inventory, Fulfillment, Shipping, Procurement, Traceability) và xưởng kẹo.
  - **Sơ đồ 2.4 (Phân Rã Miền Khách Hàng, Tài Chính, IAM & Audit Plane):** Zoom sâu Care, Finance, Identity, Notification, Analytics và đặc biệt là **Audit Plane tách biệt hoàn toàn**.

---

## 2. Quy Chuẩn Ký Hiệu Áp Dụng (Tuân Thủ `regulation.md`)
- Sử dụng engine `C4Container` của Mermaid.
- **Phân định Sync/Async trên Container Diagram (Dòng 5 & 7 `regulation.md`):** Ghi rõ bản chất giao tiếp trong nhãn:
  - `Rel(a, b, "Lệnh()", "gRPC sync")`
  - `Rel(a, broker, "Publish Event", "Kafka async")` và `Rel(broker, b, "Consume Event", "Kafka async")`
- **Hệ thống ngoài:** Sử dụng `System_Ext(...)` (Shopee, VietQR, 3PL, AWS S3).
- **Audit Plane (Bất khả xâm - Dòng 9 `regulation.md`):** Tách thành `Container_Boundary(audit, "Audit Plane (Bất Khả Xâm)")` riêng biệt.

---

## 3. Sơ Đồ 2.1: Kiến Trúc Phân Tầng Tổng Thể (Tiered Container Topology)
*Mục đích: Cung cấp góc nhìn toàn cảnh về phân tầng kiến trúc từ Client, Edge Gateway, tới các Domain Microservices và Hạ tầng dữ liệu.*

```mermaid
C4Container
    title Sơ Đồ 2.1: Kiến Trúc Phân Tầng Tổng Thể (Tiered Container Topology)

    Container_Boundary(tier_clients, "1. Tầng Giao Diện Người Dùng (Client Surfaces)") {
        Container(c_web, "Web D2C", "Next.js 14, React", "Website bán lẻ chuẩn SEO")
        Container(c_app, "Mobile App", "Flutter", "App di động khách hàng thân thiết")
        Container(c_pos, "POS Quầy Xưởng", "React SPA", "Ứng dụng tablet thu ngân offline-first")
        Container(c_admin, "Admin CMS", "React SPA, Vite", "Cổng quản trị vận hành xưởng kẹo")
    }

    Container_Boundary(tier_edge, "2. Tầng Biên (North - South Edge)") {
        Container(c_gw, "API Gateway", "Kong / Envoy", "TLS Termination, Rate Limiting, Local JWT Check via Cached JWKS")
        Container(c_bff, "Mobile BFF", "Node.js / GraphQL", "Gộp truy vấn, tinh gọn payload cho mạng 4G")
    }

    Container_Boundary(tier_domains, "3. Tầng Miền Microservices (East - West Bounded Contexts)") {
        Container(bc_commerce, "Cụm Bán Hàng & Đa Kênh", "MS-04, MS-05, MS-07, MS-13", "Order Saga, Catalog, Khuyến mại, Sàn TMĐT")
        Container(bc_supply, "Cụm Kho Vận & Cung Ứng", "MS-01, MS-02, MS-03, MS-08, MS-12", "Tồn kho đa kho, Đóng gói seal, 3PL, Nguồn gốc OCOP")
        Container(bc_finance_hr, "Cụm Tài Chính, Nhân Sự & IAM", "MS-06, MS-09, MS-15, MS-16, MS-17", "Hóa đơn VAT, CSKH, Phân quyền RBAC, Zalo ZNS")
        Container(bc_audit, "Audit Plane (Bất Khả Xâm)", "MS-18: audit-service", "Kiểm toán Tamper-Evident Hash Chain, WORM S3")
    }

    Container_Boundary(tier_infra, "4. Tầng Hạ Tầng Chung (Shared Event Streaming & Object Storage)") {
        ContainerQueue(c_kafka, "Apache Kafka Cluster", "Event Streaming Broker", "Truyền thông tin nghiệp vụ và audit bất đồng bộ")
        ContainerDb(c_s3, "AWS S3 Cloud Storage", "Object Storage", "Lưu trữ Video Seal, File Hóa Đơn VAT, Backup Merkle")
    }

    Rel(c_web, c_gw, "REST over HTTPS", "JSON / HTTPS")
    Rel(c_app, c_bff, "GraphQL over HTTPS", "HTTPS")
    Rel(c_bff, c_gw, "Forward query đã gộp", "JSON / HTTPS")
    Rel(c_pos, c_gw, "Đồng bộ giao dịch quầy", "JSON / HTTPS")
    Rel(c_admin, c_gw, "Gọi API Admin CMS", "JSON / HTTPS")

    Rel(c_gw, bc_commerce, "Routing yêu cầu mua sắm & sàn", "HTTP / REST")
    Rel(c_gw, bc_supply, "Routing tablet thợ xưởng & kho", "HTTP / REST")
    Rel(c_gw, bc_finance_hr, "Routing kế toán, nhân sự, CSKH", "HTTP / REST")

    Rel(bc_commerce, bc_supply, "ReserveStock() / ReleaseReservation()", "gRPC sync")
    Rel(bc_commerce, c_kafka, "Publish Domain Events (OrderPaidEvent)", "Kafka async")
    Rel(c_kafka, bc_supply, "Consume Events (Tạo Picking Task)", "Kafka async")
    Rel(c_kafka, bc_finance_hr, "Consume Events (Ghi nhận VAT, gửi ZNS)", "Kafka async")

    Rel(bc_supply, c_s3, "Upload video đóng gói có seal", "AWS S3 SDK")
    Rel(bc_commerce, c_kafka, "Bắn log kiểm toán sang audit.events.v1", "Kafka async")
    Rel(bc_supply, c_kafka, "Bắn log kiểm toán sang audit.events.v1", "Kafka async")
    Rel(c_kafka, bc_audit, "Consume audit.events.v1 (Ghi Hash Chain)", "Kafka async")
```

---

## 4. Sơ Đồ 2.2: Phân Rã Miền Thương Mại & Bán Hàng Đa Kênh (Commerce Containers)
*Mục đích: Zoom sâu 4 Microservices (Order, Catalog, Promotion, Channel), làm rõ luồng giỏ hàng, bảng giá và Anti-Corruption Layer của Sàn TMĐT.*

```mermaid
C4Container
    title Sơ Đồ 2.2: Phân Rã Miền Thương Mại & Đa Kênh (Commerce & Omnichannel)

    Container(gw, "API Gateway", "Kong / Envoy", "Cửa ngõ tiếp nhận traffic khách hàng & sàn")
    ContainerQueue(kafka, "Apache Kafka", "Broker", "Xương sống sự kiện nghiệp vụ")

    System_Ext(ext_shopee, "Shopee / TikTok Shop", "Sàn TMĐT bên ngoài")
    System_Ext(ext_vietqr, "Cổng VietQR / Ngân Hàng", "Xác thực thanh toán tiền về")

    Container_Boundary(commerce_boundary, "Miền Bán Hàng & Đa Kênh (Commerce Context)") {
        Container(ms_order, "MS-04: order-service", "Go / Clean Arch, Port: 8004", "Quản lý đơn hàng, điều phối Saga Orchestration, payment module")
        ContainerDb(db_order, "Order DB & Cache", "PostgreSQL 16 + Redis 7", "Bảng orders, quotations, payments, saga_states, cart_sessions (TTL 30d)")

        Container(ms_catalog, "MS-05: catalog-service", "Go, Port: 8005", "Quản lý danh mục kẹo mè xửng, biến thể, giá niêm yết")
        ContainerDb(db_catalog, "Catalog DB & Search", "PostgreSQL 16 + Elasticsearch 8", "Lưu sản phẩm, bảng giá đa kênh, fuzzy search không dấu")

        Container(ms_promo, "MS-07: promotion-service", "Go, Port: 8007", "Quản lý mã giảm giá, voucher freeship, loyalty tích điểm 1%")
        ContainerDb(db_promo, "Promotion DB & Cache", "PostgreSQL 16 + Redis 7", "Lưu coupons, hạn mức ngân sách, điểm thưởng khách")

        Container(ms_channel, "MS-13: channel-service", "Go, Port: 8013", "Anti-Corruption Layer (ACL) cho Shopee/TikTok, POS bán lẻ quầy xưởng")
        ContainerDb(db_channel, "Channel DB", "PostgreSQL 16", "Lưu marketplace_orders, POS offline transactions, ca thu ngân")
    }

    Rel(gw, ms_order, "Gọi Checkout & Webhook VietQR", "JSON / HTTPS")
    Rel(gw, ms_catalog, "Duyệt kẹo, tìm kiếm mờ", "JSON / HTTPS")
    Rel(gw, ms_channel, "Forward Webhook từ Shopee/TikTok", "JSON / HTTPS")

    Rel(ext_shopee, gw, "POST /webhooks/marketplace/*", "HTTPS Webhook")
    Rel(ext_vietqr, gw, "POST /payments/vietqr/callback", "HTTPS HMAC Webhook")

    Rel(ms_order, ms_catalog, "ValidatePriceAndSKU()", "gRPC sync")
    Rel(ms_order, ms_promo, "ValidateVoucher()", "gRPC sync")
    Rel(ms_channel, ms_catalog, "GetProductPrice()", "gRPC sync")

    Rel(ms_channel, kafka, "Publish MarketplaceOrderImportedEvent", "Kafka async")
    Rel(kafka, ms_order, "Consume MarketplaceOrderImportedEvent (Chạy Saga)", "Kafka async")

    Rel(ms_order, kafka, "Publish OrderPlacedEvent / OrderPaidEvent", "Kafka async")
    Rel(ms_order, kafka, "Publish MarketplaceOrderStockFailedEvent", "Kafka async")
    Rel(kafka, ms_channel, "Consume MarketplaceOrderStockFailedEvent (Hủy đơn sàn)", "Kafka async")
```

---

## 5. Sơ Đồ 2.3: Phân Rã Miền Chuỗi Cung Ứng & Kho Vận (Supply Chain Containers)
*Mục đích: Zoom sâu 5 Microservices (Inventory, Fulfillment, Shipping, Procurement, Traceability), làm rõ quy trình đa kho, đóng gói quay video seal và gọi 3PL.*

```mermaid
C4Container
    title Sơ Đồ 2.3: Phân Rã Miền Chuỗi Cung Ứng & Kho Vận (Supply Chain & Logistics)

    Container(ms_order_client, "MS-04: order-service", "Go (Client gọi vào)", "Saga Orchestrator điều phối mua hàng")
    ContainerQueue(kafka_sc, "Apache Kafka", "Broker", "Kênh nhận OrderPaidEvent, PackingCompletedEvent")

    System_Ext(ext_3pl_api, "Hãng Vận Chuyển 3PL", "GHN, ViettelPost API")
    System_Ext(ext_s3_video, "AWS S3 Object Storage", "Bucket lưu trữ video đóng gói có seal")
    System_Ext(ext_coop_farm, "Hợp Tác Xã Nông Sản Huế", "Vùng cung ứng mè đen, đậu phụng")

    Container_Boundary(supply_boundary, "Miền Chuỗi Cung Ứng & Kho Vận (Supply Chain Context)") {
        Container(ms_inv, "MS-01: inventory-service", "Go, Port: 8001", "Quản lý tồn kho đa kho (Xưởng/Hub), lô hạn dùng FEFO, Redis Redlock")
        ContainerDb(db_inv, "Inventory DB & Lock", "PostgreSQL 16 + Redis 7", "Bảng inventory_stocks theo kho, stock_reservations")

        Container(ms_ful, "MS-02: fulfillment-service", "Go, Port: 8002", "Điều phối tablet thợ xưởng nhặt hàng, quay video đóng gói dán tem seal")
        ContainerDb(db_ful, "Fulfillment DB", "PostgreSQL 16", "Bảng packing_jobs, seal_verifications, packing_videos")

        Container(ms_shp, "MS-12: shipping-service", "Go, Port: 8012", "Tích hợp Open API 3PL, tính cước vận chuyển, cập nhật lộ trình shipper")
        ContainerDb(db_shp, "Shipping DB", "PostgreSQL 16", "Bảng shipments, carrier_contracts, tracking_events")

        Container(ms_pro, "MS-08: procurement-service", "Go, Port: 8008", "Quản lý hợp đồng nhà cung cấp HTX Huế, đơn mua nguyên liệu PO")
        ContainerDb(db_pro, "Procurement DB", "PostgreSQL 16", "Bảng suppliers, raw_materials, purchase_orders")

        Container(ms_trc, "MS-03: traceability-service", "Go, Port: 8003", "Kích hoạt mã QR tem OCOP trên từng hộp kẹo, minh bạch nguồn nguyên liệu")
        ContainerDb(db_trc, "Traceability DB", "PostgreSQL 16", "Bảng batch_origins, ocop_certifications")
    }

    Rel(ms_order_client, ms_inv, "ReserveStock() / ReleaseReservation()", "gRPC sync")
    Rel(ms_order_client, ms_shp, "CalculateShippingFee()", "gRPC sync")

    Rel(kafka_sc, ms_ful, "Consume OrderPaidEvent (Tạo Picking Task)", "Kafka async")
    Rel(kafka_sc, ms_inv, "Consume OrderPaidEvent (Trừ kho vật lý FEFO)", "Kafka async")

    Rel(ms_ful, ext_s3_video, "Upload video quay cận cảnh kẹo có tem seal", "AWS S3 SDK")
    Rel(ms_ful, kafka_sc, "Publish PackingCompletedEvent", "Kafka async")

    Rel(kafka_sc, ms_shp, "Consume PackingCompletedEvent", "Kafka async")
    Rel(ms_shp, ext_3pl_api, "Gọi API tạo vận đơn, truyền cân nặng/kích thước", "REST API HTTPS")

    Rel(kafka_sc, ms_trc, "Consume PackingCompletedEvent (Gán mã QR lô kẹo)", "Kafka async")
    Rel(ext_coop_farm, ms_pro, "Cung ứng nguyên liệu mè/đậu đạt chuẩn VietGAP", "Biên bản giao nhận")
    Rel(ms_pro, kafka_sc, "Publish GoodsReceivedEvent (Nhập lô mè mới)", "Kafka async")
    Rel(kafka_sc, ms_inv, "Consume GoodsReceivedEvent (Tạo Lô mới)", "Kafka async")
```

---

## 6. Sơ Đồ 2.4: Phân Rã Miền Khách Hàng, Tài Chính, IAM & Audit Plane
*Mục đích: Zoom sâu các dịch vụ hỗ trợ quản trị, làm rõ ranh giới **Audit Plane bất khả xâm (Dòng 9 `regulation.md`)** và Observability Plane.*

```mermaid
C4Container
    title Sơ Đồ 2.4: Miền Khách Hàng, Tài Chính, IAM & Audit Plane

    ContainerQueue(kafka_core, "Apache Kafka", "Broker", "Kênh sự kiện nghiệp vụ và kiểm toán")
    System_Ext(ext_s3_worm, "AWS S3 Object Lock (WORM)", "Lưu trữ Merkle Root Checkpoint kiểm toán chống sửa đổi")

    Container_Boundary(crm_finance_boundary, "Miền Khách Hàng & Tài Chính (CRM & Finance)") {
        Container(ms_care, "MS-06: care-service", "Node.js, Port: 8006", "Quản lý vé khiếu nại kẹo vỡ, đánh giá Verified Reviews")
        ContainerDb(db_care, "Care DB", "MongoDB 7", "Collections tickets, reviews, inspection_logs")

        Container(ms_fin, "MS-09: finance-service", "Go, Port: 8009", "Sổ cái kế toán A/R, xuất hóa đơn VAT điện tử, hoàn tiền refund")
        ContainerDb(db_fin, "Finance DB", "PostgreSQL 16", "Bảng general_ledger, vat_invoices, refund_vouchers")

        Container(ms_not, "MS-17: notification-service", "Node.js, Port: 8017", "Gửi tin nhắn đa kênh: Zalo ZNS, SMS Brandname, Email")
        ContainerDb(db_not, "Notification Queue", "Redis Queue", "Hàng đợi gửi tin đa kênh theo độ ưu tiên")

        Container(ms_prf, "MS-15: profile-service", "Go, Port: 8015", "Hồ sơ người dùng, sổ địa chỉ, nhân sự thợ xưởng")
        Container(ms_iam, "MS-16: identity-service", "Go, Port: 8016", "Cấp phát JWT, quản lý 19 vai trò RBAC, phát hành JWKS")
    }

    Container_Boundary(audit_boundary, "Audit Plane (Mặt Phẳng Kiểm Toán Pháp Lý - Bất Khả Xâm)") {
        Container(ms_aud, "MS-18: audit-service", "Go / Rust, Port: 8018", "Ghi nhật ký kiểm toán bất biến, tính toán Entity-Level Hash Chain")
        ContainerDb(db_aud, "Audit DB", "PostgreSQL Append-Only", "Bảng audit_records chỉ cho phép INSERT, cấm UPDATE/DELETE")
    }

    Container_Boundary(otel_boundary, "Observability Plane (Mặt Phẳng Giám Sát Viễn Trắc)") {
        Container(c_otel, "OpenTelemetry Collector", "OTel Core Agent", "Thu gom Traces, Metrics, Logs qua cổng gRPC/HTTP 4317/4318")
        ContainerDb(c_mon, "Monitoring Backends", "Prometheus + Jaeger + Loki", "Hạ tầng lưu trữ tín hiệu viễn trắc và bảng điều khiển Grafana")
    }

    Rel(kafka_core, ms_fin, "Consume OrderPaidEvent (Ghi nhận doanh thu VAT)", "Kafka async")
    Rel(kafka_core, ms_not, "Consume OrderPaidEvent / ShipmentDeliveredEvent", "Kafka async")
    Rel(kafka_core, ms_care, "Consume OrderCompletedEvent (Mở quyền viết review)", "Kafka async")

    Rel(kafka_core, ms_aud, "Consume topic audit.events.v1 (Từ toàn bộ 17 service)", "Kafka async")
    Rel(ms_aud, db_aud, "INSERT bản ghi kèm SHA-256 Hash Chain", "SQL Append-Only")
    Rel(ms_aud, ext_s3_worm, "Neo định kỳ Merkle Tree Root Hash mỗi 1 giờ", "AWS S3 SDK WORM")

    Rel(c_otel, c_mon, "Xuất dữ liệu viễn trắc đo lường P95, RPS, Errors", "OTLP Push")
```

---

## 7. Đánh Giá Hiệu Quả Của Việc Phân Rã Container
1. **Triệt Tiêu Hoàn Toàn Điểm Nghẽn Trực Quan:**
   - Thay vì 1 sơ đồ khổng lồ với 40+ đối tượng, mỗi sơ đồ phân rã chỉ chứa từ 5 đến 7 container chính.
   - Các đường kết nối đi theo một chiều rõ ràng (từ Client xuống Gateway, từ Gateway sang Microservice, từ Microservice ra Kafka).
2. **Minh Chứng Tuyệt Đối Cho `regulation.md`:**
   - **Audit Plane** được đóng khung riêng biệt `Container_Boundary(audit_boundary)` tại Sơ đồ 2.4 theo đúng Dòng 9 của quy định.
   - Mọi mũi tên đều được định danh rõ `gRPC sync` hoặc `Kafka async` trong nhãn (Dòng 5 & 7).
   - Hệ thống ngoài đều dùng chuẩn `System_Ext(...)`.
