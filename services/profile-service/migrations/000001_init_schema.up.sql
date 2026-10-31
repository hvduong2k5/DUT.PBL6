-- =============================================================================
-- MS-15 PROFILE SERVICE DATABASE INITIAL SCHEMA MIGRATION
-- MODEL: 2-TIER ADMINISTRATIVE STRUCTURE (POST-JULY 2025 REFORM)
-- PROVINCE / MUNICIPALITY -> WARD / COMMUNE
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. ADMINISTRATIVE UNITS (Chuẩn hóa 2 cấp: Tỉnh/TP TW -> Xã/Phường)
CREATE TABLE IF NOT EXISTS administrative_units (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),
    full_name VARCHAR(150) NOT NULL,
    parent_code VARCHAR(20) REFERENCES administrative_units(code),
    level VARCHAR(20) NOT NULL CHECK (level IN ('PROVINCE', 'WARD')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_units_parent ON administrative_units(parent_code);
CREATE INDEX IF NOT EXISTS idx_admin_units_name_trgm ON administrative_units USING GIN (name gin_trgm_ops);

-- 2. CUSTOMER PROFILES
CREATE TABLE IF NOT EXISTS customer_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    full_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    email VARCHAR(255),
    date_of_birth DATE,
    gender VARCHAR(10) CHECK (gender IN ('MALE', 'FEMALE', 'OTHER', 'UNSPECIFIED')),
    avatar_url VARCHAR(500),
    preferences JSONB DEFAULT '{"favorite_products": [], "dietary_preference": "NORMAL", "allergy_alert": []}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'PENDING_VERIFICATION')),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_customer_phone ON customer_profiles(phone_number);
CREATE INDEX IF NOT EXISTS idx_customer_email ON customer_profiles(email);
CREATE INDEX IF NOT EXISTS idx_customer_user_id ON customer_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_customer_preferences ON customer_profiles USING GIN (preferences);

-- 3. SHIPPING ADDRESSES (2-TIER STREAMLINED: NO DISTRICT)
CREATE TABLE IF NOT EXISTS shipping_addresses (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_profiles(id) ON DELETE CASCADE,
    recipient_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    street_address VARCHAR(255) NOT NULL,
    ward_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    ward_name VARCHAR(100) NOT NULL,
    province_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    province_name VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 7),
    longitude DECIMAL(10, 7),
    label VARCHAR(50) DEFAULT 'HOME' CHECK (label IN ('HOME', 'OFFICE', 'GIFT_RECIPIENT', 'OTHER')),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_customer_default_address 
ON shipping_addresses (customer_id) 
WHERE is_default = TRUE AND is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_shipping_addresses_customer ON shipping_addresses(customer_id) WHERE is_deleted = FALSE;

-- 4. GUEST ORDER CLAIMS
CREATE TABLE IF NOT EXISTS guest_order_claims (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_profiles(id) ON DELETE CASCADE,
    order_id VARCHAR(50) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    claim_status VARCHAR(20) NOT NULL DEFAULT 'VERIFIED' CHECK (claim_status IN ('VERIFIED', 'REVOKED'))
);

CREATE INDEX IF NOT EXISTS idx_guest_claims_phone ON guest_order_claims(phone_number);

CREATE UNIQUE INDEX IF NOT EXISTS uq_order_claim_active 
ON guest_order_claims(order_id) 
WHERE claim_status = 'VERIFIED';

-- 5. DEPARTMENTS
CREATE TABLE IF NOT EXISTS departments (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    manager_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO departments (id, name, description) VALUES
('DEPT-PROD-HUONGTHUY', 'Xưởng Nấu Kẹo Hương Thủy', 'Khu vực nấu mè xửng truyền thống và kiểm tra mẻ kẹo'),
('DEPT-FULFILL-PACK', 'Bộ Phận Đóng Gói & Niêm Phong', 'Khu vực cân kẹo, dán tem QR OCOP, đóng thùng có camera ghi hình'),
('DEPT-WAREHOUSE', 'Kho Nguyên Liệu & Thành Phẩm', 'Bảo quản mè vừng, đậu phụng, đường non và kẹo xuất xưởng'),
('DEPT-RETAIL-HUEMARKET', 'Cửa Hàng Trưng Bày Huế', 'Showroom bán lẻ và tiếp đón khách du lịch dùng thử kẹo')
ON CONFLICT (id) DO NOTHING;

-- 6. EMPLOYEE PROFILES (HR & ENVELOPE ENCRYPTION)
CREATE TABLE IF NOT EXISTS employee_profiles (
    id UUID PRIMARY KEY,
    user_id UUID UNIQUE,
    employee_code VARCHAR(50) NOT NULL UNIQUE,
    full_name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    id_card_encrypted BYTEA NOT NULL,
    id_card_nonce BYTEA NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    kek_version INT NOT NULL DEFAULT 1,
    department_id VARCHAR(50) NOT NULL REFERENCES departments(id),
    position VARCHAR(100) NOT NULL,
    contract_type VARCHAR(20) NOT NULL CHECK (contract_type IN ('FULLTIME', 'PARTTIME', 'SEASONAL')),
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    food_safety_cert_no VARCHAR(100),
    food_safety_cert_expiry DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ON_LEAVE', 'TERMINATED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_emp_code ON employee_profiles(employee_code);
CREATE INDEX IF NOT EXISTS idx_emp_department ON employee_profiles(department_id);
CREATE INDEX IF NOT EXISTS idx_emp_status ON employee_profiles(status);
CREATE INDEX IF NOT EXISTS idx_emp_cert_expiry ON employee_profiles(food_safety_cert_expiry);

-- 7. TRANSACTIONAL OUTBOX EVENTS
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox_events(created_at) WHERE published_at IS NULL;

-- 8. SEED ADMINISTRATIVE UNITS (HUẾ 2-TIER REFORM)
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('75', 'Thành phố Huế', 'Hue City', 'Thành phố Huế', NULL, 'PROVINCE')
ON CONFLICT (code) DO NOTHING;

INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('WARD-TH-001', 'Thuận Hòa', 'Thuan Hoa', 'Phường Thuận Hòa, Thành phố Huế', '75', 'WARD'),
('WARD-VL-002', 'Vĩnh Lộc', 'Vinh Loc', 'Xã Vĩnh Lộc, Thành phố Huế', '75', 'WARD'),
('WARD-PB-003', 'Phú Bài', 'Phu Bai', 'Phường Phú Bài, Thành phố Huế', '75', 'WARD'),
('WARD-KL-004', 'Kim Long', 'Kim Long', 'Phường Kim Long, Thành phố Huế', '75', 'WARD'),
('WARD-VD-005', 'Vỹ Dạ', 'Vy Da', 'Phường Vỹ Dạ, Thành phố Huế', '75', 'WARD'),
('WARD-AC-006', 'An Cựu', 'An Cuu', 'Phường An Cựu, Thành phố Huế', '75', 'WARD'),
('WARD-XP-007', 'Xuân Phú', 'Xuan Phu', 'Phường Xuân Phú, Thành phố Huế', '75', 'WARD'),
('WARD-DN-008', 'Đức Nhuận', 'Duc Nhuan', 'Phường Đức Nhuận, Thành phố Huế', '75', 'WARD'),
('WARD-TX-009', 'Thủy Xuân', 'Thuy Xuan', 'Phường Thủy Xuân, Thành phố Huế', '75', 'WARD'),
('WARD-TB-010', 'Thủy Biều', 'Thuy Bieu', 'Phường Thủy Biều, Thành phố Huế', '75', 'WARD'),
('WARD-TL-011', 'Tây Lộc', 'Tay Loc', 'Phường Tây Lộc, Thành phố Huế', '75', 'WARD'),
('WARD-DB-012', 'Đông Ba', 'Dong Ba', 'Phường Đông Ba, Thành phố Huế', '75', 'WARD')
ON CONFLICT (code) DO NOTHING;

