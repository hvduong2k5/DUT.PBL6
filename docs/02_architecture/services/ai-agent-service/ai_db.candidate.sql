-- MS-19 ai_db — DDL tham chiếu.
-- Nguồn thực thi: services/ai-agent-service/migrations/postgres/001_ai_agent.sql
-- File này là bản chụp để tra cứu cùng bộ tài liệu; khi lệch nhau, migration thắng.

-- ai-agent-service (MS-19) — own database ai_db. Strict database-per-service:
-- this schema holds only AI-owned data (conversations, knowledge projection,
-- execution records). Business facts (price, stock, orders) stay in their
-- owning services and are read through the Tool Gateway.
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type TEXT NOT NULL CHECK (owner_type IN ('guest', 'customer')),
    customer_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages (conversation_id, created_at);

CREATE TABLE IF NOT EXISTS conversation_contexts (
    conversation_id UUID PRIMARY KEY REFERENCES conversations (id) ON DELETE CASCADE,
    slots JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS consent_records (
    customer_id TEXT PRIMARY KEY,
    history_use BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS knowledge_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    source TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'public',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retracted')),
    source_version TEXT NOT NULL DEFAULT '1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES knowledge_documents (id) ON DELETE CASCADE,
    product_id TEXT,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL,
    search_text TEXT NOT NULL DEFAULT '',
    tsv TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', search_text)) STORED,
    embedding VECTOR(1024) NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'public',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    effective_from DATE,
    effective_until DATE,
    source_version TEXT NOT NULL DEFAULT '1',
    content_hash TEXT NOT NULL,
    embedding_model TEXT NOT NULL,
    embedding_version TEXT NOT NULL DEFAULT '1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chunks_embedding_active
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_chunks_tsv ON knowledge_chunks USING gin (tsv);
CREATE INDEX IF NOT EXISTS idx_chunks_product ON knowledge_chunks (product_id) WHERE is_active;

CREATE TABLE IF NOT EXISTS agent_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations (id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'running',
    steps INTEGER NOT NULL DEFAULT 0,
    tool_calls INTEGER NOT NULL DEFAULT 0,
    tokens_in INTEGER NOT NULL DEFAULT 0,
    tokens_out INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tool_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES agent_runs (id) ON DELETE SET NULL,
    tool_name TEXT NOT NULL,
    args JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_summary TEXT,
    status TEXT NOT NULL,
    latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendation_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations (id) ON DELETE SET NULL,
    customer_id TEXT,
    product_id TEXT NOT NULL,
    signal TEXT NOT NULL CHECK (signal IN ('helpful', 'not_helpful', 'clicked', 'dismissed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
