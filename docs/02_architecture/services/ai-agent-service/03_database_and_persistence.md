# 03 — Persistence

## Nguyên tắc

Database-per-service: MS-19 sở hữu riêng PostgreSQL `ai_db` (image `pgvector/pgvector:pg16`). Schema chỉ chứa dữ liệu AI sở hữu — hội thoại, tri thức chiếu (projection) phục vụ RAG, bản ghi thực thi. Dữ kiện nghiệp vụ (giá, tồn, đơn) không sao chép vào `ai_db`; chúng được đọc trực tiếp qua tool mỗi lượt chat. DDL thực thi: `services/ai-agent-service/migrations/postgres/001_ai_agent.sql` (bản chụp tại [ai_db.candidate.sql](ai_db.candidate.sql)).

## Schema

| Bảng | Nội dung chính |
| --- | --- |
| `conversations` | Phiên chat: `owner_type` (`guest`/`customer`), `customer_id`, `expires_at` (retention chốt tại lúc tạo). |
| `messages` | Tin nhắn theo phiên (`role` user/assistant/system, `metadata` JSONB — ví dụ danh sách product_id đã gợi ý). Xóa theo phiên (CASCADE). |
| `conversation_contexts` | Slot làm việc của workflow theo phiên (JSONB), một dòng/phiên, upsert khi slot đổi. |
| `consent_records` | Consent lưu lịch sử chat theo `customer_id`: `history_use`, `version` tăng dần. |
| `knowledge_documents` | Tài liệu tri thức: `title`, `source`, `visibility`, `status` (`draft`/`active`/`retracted`), `source_version`. |
| `knowledge_chunks` | Chunk để truy hồi: `content`, `search_text` (= tiêu đề + nội dung), cột `tsv` sinh tự động (`to_tsvector('simple', search_text)`), `embedding VECTOR(1024)`, `visibility`, `is_active`, cửa sổ `effective_from/until`, `content_hash`, `embedding_model` + `embedding_version`. |
| `agent_runs` | Mỗi lượt chạy agent: trạng thái, số bước, số tool call, token in/out, lỗi, thời gian. |
| `tool_calls` | Từng lần gọi tool qua gateway: tên, args, tóm tắt kết quả, trạng thái, độ trễ. |
| `recommendation_feedback` | Signal feedback của khách theo sản phẩm (`helpful`/`not_helpful`/`clicked`/`dismissed`). |
| `ai_audit_logs` | Nhật ký audit chung của service (actor, action, detail JSONB). |

Chỉ mục truy hồi: HNSW trên `embedding` (cosine) với điều kiện từng phần `WHERE is_active`, GIN trên `tsv`, và chỉ mục theo `product_id` cho các chunk đang active. Việc gắn `embedding_model` theo từng chunk là chủ đích: đổi model embedding bắt buộc nhúng lại toàn bộ corpus, và schema phải phân biệt được chunk của model cũ/mới trong quá trình đó.

## Hai hiện thực store

Cùng một interface (`ConversationStore`, `KnowledgeStore`), chọn theo chế độ chạy trong `main.py`:

| Chế độ | Store | Khi nào |
| --- | --- | --- |
| `AI_FIXTURES=1` | In-memory | Test, CI, dev không cần database. Knowledge vẫn chạy hybrid đầy đủ (cosine + chồng từ khóa trong bộ nhớ). |
| `AI_FIXTURES=0` | PostgreSQL (psycopg, autocommit) | Production. `/health/ready` ping DB thật; chế độ fixture luôn báo ready. |

## Migration

`python -m ai_agent_service migrate` áp các file `migrations/postgres/*.sql` theo thứ tự tên, ghi nhận vào bảng `schema_migrations` và bỏ qua file đã áp. Không dùng Flyway/Alembic — runner tự quản nhỏ trong `db.py`, nhất quán với cách service Python còn lại của dự án vận hành migration SQL thuần.
