-- =============================================================================
-- MS-15 PROFILE SERVICE DATABASE MIGRATION 000002
-- NATIONAL MULTI-CITY ADMINISTRATIVE UNITS & STREET-WARD MAPPINGS
-- =============================================================================

-- 1. STREET TO WARD MAPPINGS (BẢNG ÁNH XẠ TUYẾN ĐƯỜNG VỚI XÃ/PHƯỜNG VÀ TỈNH THÀNH)
CREATE TABLE IF NOT EXISTS street_ward_mappings (
    id UUID PRIMARY KEY,
    street_name VARCHAR(150) NOT NULL,
    street_name_unaccented VARCHAR(150) NOT NULL,
    ward_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    province_code VARCHAR(20) NOT NULL REFERENCES administrative_units(code),
    latitude DECIMAL(10, 7),
    longitude DECIMAL(10, 7),
    is_primary BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_street_province_unaccented 
ON street_ward_mappings (province_code, street_name_unaccented);

CREATE INDEX IF NOT EXISTS idx_street_ward 
ON street_ward_mappings (ward_code);

CREATE INDEX IF NOT EXISTS idx_street_unaccented_trgm 
ON street_ward_mappings USING GIN (street_name_unaccented gin_trgm_ops);

-- 2. SEED CÁC TỈNH / THÀNH PHỐ TRỌNG ĐIỂM TOÀN QUỐC (PROVINCES)
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('01', 'Thành phố Hà Nội', 'Hanoi City', 'Thành phố Hà Nội', NULL, 'PROVINCE'),
('79', 'Thành phố Hồ Chí Minh', 'Ho Chi Minh City', 'Thành phố Hồ Chí Minh', NULL, 'PROVINCE'),
('48', 'Thành phố Đà Nẵng', 'Da Nang City', 'Thành phố Đà Nẵng', NULL, 'PROVINCE')
ON CONFLICT (code) DO NOTHING;

-- 3. SEED CÁC PHƯỜNG / XÃ TIÊU BIỂU TẠI CÁC THÀNH PHỐ
-- 3.1. THÀNH PHỐ HÀ NỘI ('01')
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('WARD-HN-001', 'Tràng Tiền', 'Trang Tien', 'Phường Tràng Tiền, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-002', 'Hàng Bạc', 'Hang Bac', 'Phường Hàng Bạc, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-003', 'Cửa Đông', 'Cua Dong', 'Phường Cửa Đông, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-004', 'Dịch Vọng', 'Dich Vong', 'Phường Dịch Vọng, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-005', 'Nghĩa Tân', 'Nghia Tan', 'Phường Nghĩa Tân, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-006', 'Láng Thượng', 'Lang Thuong', 'Phường Láng Thượng, Thành phố Hà Nội', '01', 'WARD'),
('WARD-HN-007', 'Bách Khoa', 'Bach Khoa', 'Phường Bách Khoa, Thành phố Hà Nội', '01', 'WARD')
ON CONFLICT (code) DO NOTHING;

-- 3.2. THÀNH PHỐ HỒ CHÍ MINH ('79')
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('WARD-HCM-001', 'Bến Nghé', 'Ben Nghe', 'Phường Bến Nghé, Thành phố Hồ Chí Minh', '79', 'WARD'),
('WARD-HCM-002', 'Bến Thành', 'Ben Thanh', 'Phường Bến Thành, Thành phố Hồ Chí Minh', '79', 'WARD'),
('WARD-HCM-003', 'Đa Kao', 'Da Kao', 'Phường Đa Kao, Thành phố Hồ Chí Minh', '79', 'WARD'),
('WARD-HCM-004', 'Tân Định', 'Tan Dinh', 'Phường Tân Định, Thành phố Hồ Chí Minh', '79', 'WARD'),
('WARD-HCM-005', 'Thảo Điền', 'Thao Dien', 'Phường Thảo Điền, Thành phố Hồ Chí Minh', '79', 'WARD'),
('WARD-HCM-006', 'Võ Thị Sáu', 'Vo Thi Sau', 'Phường Võ Thị Sáu, Thành phố Hồ Chí Minh', '79', 'WARD')
ON CONFLICT (code) DO NOTHING;

-- 3.3. THÀNH PHỐ ĐÀ NẴNG ('48')
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('WARD-DN-001', 'Thạch Thang', 'Thach Thang', 'Phường Thạch Thang, Thành phố Đà Nẵng', '48', 'WARD'),
('WARD-DN-002', 'Hải Châu 1', 'Hai Chau 1', 'Phường Hải Châu 1, Thành phố Đà Nẵng', '48', 'WARD'),
('WARD-DN-003', 'Hải Châu 2', 'Hai Chau 2', 'Phường Hải Châu 2, Thành phố Đà Nẵng', '48', 'WARD'),
('WARD-DN-004', 'Phước Mỹ', 'Phuoc My', 'Phường Phước Mỹ, Thành phố Đà Nẵng', '48', 'WARD'),
('WARD-DN-005', 'Hòa Cường Bắc', 'Hoa Cuong Bac', 'Phường Hòa Cường Bắc, Thành phố Đà Nẵng', '48', 'WARD')
ON CONFLICT (code) DO NOTHING;

-- 3.4. BỔ SUNG THÊM CÁC PHƯỜNG TRỌNG ĐIỂM TẠI THÀNH PHỐ HUẾ ('75')
INSERT INTO administrative_units (code, name, name_en, full_name, parent_code, level) VALUES
('WARD-TL-013', 'Thuận Lộc', 'Thuan Loc', 'Phường Thuận Lộc, Thành phố Huế', '75', 'WARD'),
('WARD-VN-014', 'Vĩnh Ninh', 'Vinh Ninh', 'Phường Vĩnh Ninh, Thành phố Huế', '75', 'WARD'),
('WARD-PH-015', 'Phú Hội', 'Phu Hoi', 'Phường Phú Hội, Thành phố Huế', '75', 'WARD'),
('WARD-PN-016', 'Phú Nhuận', 'Phu Nhuan', 'Phường Phú Nhuận, Thành phố Huế', '75', 'WARD'),
('WARD-GH-017', 'Gia Hội', 'Gia Hoi', 'Phường Gia Hội, Thành phố Huế', '75', 'WARD')
ON CONFLICT (code) DO NOTHING;

-- 4. SEED DỮ LIỆU TUYẾN ĐƯỜNG MẪU (STREET-WARD MAPPINGS)
-- 4.1. THÀNH PHỐ HUẾ ('75')
INSERT INTO street_ward_mappings (id, street_name, street_name_unaccented, ward_code, province_code, latitude, longitude, is_primary) VALUES
('01923400-0001-7000-8000-000000000001', 'Mai Lão Bạng', 'mai lao bang', 'WARD-TL-013', '75', 16.47892, 107.57815, true),
('01923400-0001-7000-8000-000000000002', 'Lê Lợi', 'le loi', 'WARD-VN-014', '75', 16.46320, 107.58540, true),
('01923400-0001-7000-8000-000000000003', 'Lê Lợi', 'le loi', 'WARD-PN-016', '75', 16.46780, 107.59210, false),
('01923400-0001-7000-8000-000000000004', 'Nguyễn Huệ', 'nguyen hue', 'WARD-VN-014', '75', 16.45890, 107.58920, true),
('01923400-0001-7000-8000-000000000005', 'Nguyễn Huệ', 'nguyen hue', 'WARD-AC-006', '75', 16.45230, 107.59450, false),
('01923400-0001-7000-8000-000000000006', 'Bến Nghé', 'ben nghe', 'WARD-PH-015', '75', 16.46510, 107.59320, true),
('01923400-0001-7000-8000-000000000007', 'Nguyễn Tri Phương', 'nguyen tri phuong', 'WARD-PH-015', '75', 16.46670, 107.59180, true),
('01923400-0001-7000-8000-000000000008', 'Trần Hưng Đạo', 'tran hung dao', 'WARD-DB-012', '75', 16.47120, 107.58430, true),
('01923400-0001-7000-8000-000000000009', 'Chi Lăng', 'chi lang', 'WARD-GH-017', '75', 16.47650, 107.59560, true),
('01923400-0001-7000-8000-000000000010', 'Nguyễn Sinh Cung', 'nguyen sinh cung', 'WARD-VD-005', '75', 16.46820, 107.60450, true)
ON CONFLICT (id) DO NOTHING;

-- 4.2. THÀNH PHỐ HÀ NỘI ('01')
INSERT INTO street_ward_mappings (id, street_name, street_name_unaccented, ward_code, province_code, latitude, longitude, is_primary) VALUES
('01923400-0002-7000-8000-000000000001', 'Tràng Tiền', 'trang tien', 'WARD-HN-001', '01', 21.02530, 105.85520, true),
('01923400-0002-7000-8000-000000000002', 'Đinh Tiên Hoàng', 'dinh tien hoang', 'WARD-HN-001', '01', 21.02980, 105.85240, true),
('01923400-0002-7000-8000-000000000003', 'Hàng Bạc', 'hang bac', 'WARD-HN-002', '01', 21.03410, 105.85290, true),
('01923400-0002-7000-8000-000000000004', 'Cầu Giấy', 'cau giay', 'WARD-HN-004', '01', 21.03320, 105.79810, true),
('01923400-0002-7000-8000-000000000005', 'Chùa Láng', 'chua lang', 'WARD-HN-006', '01', 21.02250, 105.80350, true),
('01923400-0002-7000-8000-000000000006', 'Giải Phóng', 'giai phong', 'WARD-HN-007', '01', 20.99840, 105.84230, true)
ON CONFLICT (id) DO NOTHING;

-- 4.3. THÀNH PHỐ HỒ CHÍ MINH ('79')
INSERT INTO street_ward_mappings (id, street_name, street_name_unaccented, ward_code, province_code, latitude, longitude, is_primary) VALUES
('01923400-0003-7000-8000-000000000001', 'Nguyễn Huệ', 'nguyen hue', 'WARD-HCM-001', '79', 10.77410, 106.70320, true),
('01923400-0003-7000-8000-000000000002', 'Đồng Khởi', 'dong khoi', 'WARD-HCM-001', '79', 10.77650, 106.70240, true),
('01923400-0003-7000-8000-000000000003', 'Lê Lợi', 'le loi', 'WARD-HCM-002', '79', 10.77320, 106.69850, true),
('01923400-0003-7000-8000-000000000004', 'Hai Bà Trưng', 'hai ba trung', 'WARD-HCM-004', '79', 10.79120, 106.69140, true),
('01923400-0003-7000-8000-000000000005', 'Hai Bà Trưng', 'hai ba trung', 'WARD-HCM-003', '79', 10.78530, 106.69670, false),
('01923400-0003-7000-8000-000000000006', 'Xuân Thủy', 'xuan thuy', 'WARD-HCM-005', '79', 10.80420, 106.73510, true)
ON CONFLICT (id) DO NOTHING;

-- 4.4. THÀNH PHỐ ĐÀ NẴNG ('48')
INSERT INTO street_ward_mappings (id, street_name, street_name_unaccented, ward_code, province_code, latitude, longitude, is_primary) VALUES
('01923400-0004-7000-8000-000000000001', 'Bạch Đằng', 'bach dang', 'WARD-DN-001', '48', 16.07120, 108.22350, true),
('01923400-0004-7000-8000-000000000002', 'Nguyễn Văn Linh', 'nguyen van linh', 'WARD-DN-002', '48', 16.06150, 108.21680, true),
('01923400-0004-7000-8000-000000000003', 'Trần Phú', 'tran phu', 'WARD-DN-002', '48', 16.06640, 108.22120, true),
('01923400-0004-7000-8000-000000000004', 'Võ Nguyên Giáp', 'vo nguyen giap', 'WARD-DN-004', '48', 16.06280, 108.24670, true),
('01923400-0004-7000-8000-000000000005', '2 Tháng 9', '2 thang 9', 'WARD-DN-005', '48', 16.04520, 108.22290, true)
ON CONFLICT (id) DO NOTHING;
