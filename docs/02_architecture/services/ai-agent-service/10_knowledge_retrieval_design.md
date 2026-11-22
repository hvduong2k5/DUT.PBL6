# 10 — Tri thức và retrieval

## Vai trò của RAG trong service này

Kho tri thức cung cấp **ý nghĩa** — nguồn gốc, đặc tính, cách bảo quản, gợi ý theo dịp — và là nền của lớp FAQ tĩnh cho khách. RAG **không bao giờ** là nguồn của giá hay tồn kho: các dữ kiện đó luôn đến từ tool Catalog/Inventory trong cùng lượt chat. Thiết kế này là chủ đích: tài liệu có thể cũ theo giờ, giá/tồn thay đổi theo phút.

## Ingestion

- Đường vào: `KnowledgeStore.ingest(title, source, text, product_id?, visibility?)` — gọi từ endpoint nội bộ `POST /api/v1/ai/knowledge/products/sync` hoặc job gieo dữ liệu.
- **Chunking:** tách theo đoạn văn, ghép tới tối đa **500 ký tự** mỗi chunk, đoạn dài bị cắt với **80 ký tự chồng lấp**; nội dung tìm kiếm (`search_text`) = tiêu đề tài liệu + nội dung chunk, để truy vấn khớp cả theo tên chủ đề.
- Mỗi chunk lưu: vector nhúng, `content_hash` (SHA-256), `embedding_model`/`embedding_version`, `visibility`, `is_active`, cửa sổ hiệu lực theo ngày, `product_id` nếu là tri thức gắn sản phẩm.
- **Thu hồi:** `deactivate_product(product_id)` tắt `is_active` của mọi chunk thuộc sản phẩm; tài liệu có `status` (`draft`/`active`/`retracted`) ở tầng document. Cả hai đều có hiệu lực ngay vì retrieval lọc trong truy vấn.

## Truy hồi hybrid (PostgreSQL)

Một truy vấn SQL duy nhất, **lọc trước — không lọc sau**:

- Điều kiện cứng ngay trong `WHERE`: chunk `is_active`, document `status = 'active'`, `visibility = 'public'` (đợt 1), cửa sổ hiệu lực chứa ngày hiện tại.
- Điểm vector: `1 - (embedding <=> query_vector)` (khoảng cách cosine của pgvector, chạy trên chỉ mục HNSW từng phần).
- Điểm từ khóa: `ts_rank` trên cột `tsv` sinh từ `search_text` với cấu hình `simple` (không stemming — phù hợp văn bản tiếng Việt ngắn).
- Điểm cuối: **0.6 × vector + 0.4 × từ khóa**, sắp xếp và lấy top-N ngay trong SQL.

Hiện thực in-memory (fixture/test) dùng cùng trọng số: cosine trên vector FakeEmbedder + tỉ lệ chồng lấp từ khóa, ngưỡng điểm 0.05. Tính liên quan trong test vì thế do nửa từ khóa gánh — đúng thiết kế đã nêu ở lớp embedding.

## Corpus đợt 1

Ở chế độ fixture, service tự gieo lúc khởi động (`modules/shopping_advisor/seed.py`):

- 5 tài liệu FAQ tĩnh theo đúng các chủ đề bản đề xuất (§5.3) giao cho lớp FAQ: hạn sử dụng và bảo quản; khác nhau dẻo/giòn; làng nghề Kim Long và Nam Phổ; địa chỉ xưởng và cửa hàng; gợi ý chọn quà Tết.
- Một tài liệu “Sản phẩm: …” cho mỗi sản phẩm published của catalog fixture, gắn `product_id`, sinh từ tên + mô tả. **Lưu ý:** văn bản gieo này có chứa giá của fixture để phục vụ dev — corpus production phải được duy trì qua ingestion API từ nguồn nội dung đã duyệt, và giá trong tài liệu không bao giờ được dùng làm căn cứ trả lời (grounding chỉ nhận số từ tool).

## Mở rộng đợt 2

`knowledge_assistant` dùng lại toàn bộ nền này, cộng ba việc: thêm giá trị `visibility` nội bộ và lọc theo `permission_grants` của nhân viên ngay trong `WHERE`; khóa endpoint ingestion bằng grant; và (tùy quyết định nhóm) consumer Kafka cập nhật corpus từ sự kiện Catalog thay cho ingest thủ công.
