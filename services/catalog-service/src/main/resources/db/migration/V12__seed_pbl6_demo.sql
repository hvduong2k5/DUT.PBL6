-- PBL6 Omama 40-SKU demo seed. Source: DanhSach40SanPham_Seed_v1.md.
-- Catalog Flyway seed. Version V12 was free on dev when prepared; recheck before applying.
-- Re-running preserves existing demo prices, stock and dates. Conflicting SKUs abort.
-- Database: om_catalog_db. Apply current Catalog migrations first.
SELECT pg_advisory_xact_lock(2026112040);
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000001','Mè Xửng','me-xung') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000002','Mứt Huế','mut-hue') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000003','Trà Huế','tra-hue') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000004','Bánh Huế','banh-hue') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000005','Kẹo & Hạt Sen','keo-hat-sen') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000006','Hộp Quà & Combo','hop-qua') ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories(id,name,slug) VALUES ('9b200001-2026-4020-8020-000000000007','Đặc Sản Khô Khác','dac-san-kho') ON CONFLICT (slug) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000001' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-01' AND (id<>'9b200003-2026-4020-8020-000000000001' OR product_id<>'9b200002-2026-4020-8020-000000000001')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000001',id,'Mè xửng giòn Omama','me-xung-gion-omama-om-mx-01','Mè xửng giòn Omama. Quy cách: 150g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000001','9b200002-2026-4020-8020-000000000001','OM-MX-01','150g',150,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000001','9b200003-2026-4020-8020-000000000001','D2C_WEB',25000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000002' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-02' AND (id<>'9b200003-2026-4020-8020-000000000002' OR product_id<>'9b200002-2026-4020-8020-000000000002')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000002',id,'Mè xửng giòn Omama','me-xung-gion-omama-om-mx-02','Mè xửng giòn Omama. Quy cách: 300g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000002','9b200002-2026-4020-8020-000000000002','OM-MX-02','300g',300,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000002','9b200003-2026-4020-8020-000000000002','D2C_WEB',50000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000003' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-03' AND (id<>'9b200003-2026-4020-8020-000000000003' OR product_id<>'9b200002-2026-4020-8020-000000000003')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000003',id,'Mè xửng dẻo Omama','me-xung-deo-omama-om-mx-03','Mè xửng dẻo Omama. Quy cách: 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000003','9b200002-2026-4020-8020-000000000003','OM-MX-03','250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000003','9b200003-2026-4020-8020-000000000003','D2C_WEB',38000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000004' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-04' AND (id<>'9b200003-2026-4020-8020-000000000004' OR product_id<>'9b200002-2026-4020-8020-000000000004')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000004',id,'Mè xửng dẻo Omama','me-xung-deo-omama-om-mx-04','Mè xửng dẻo Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000004','9b200002-2026-4020-8020-000000000004','OM-MX-04','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000004','9b200003-2026-4020-8020-000000000004','D2C_WEB',65000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000005' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-05' AND (id<>'9b200003-2026-4020-8020-000000000005' OR product_id<>'9b200002-2026-4020-8020-000000000005')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-05';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000005',id,'Mè xửng mè đen Omama','me-xung-me-den-omama-om-mx-05','Mè xửng mè đen Omama. Quy cách: 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000005','9b200002-2026-4020-8020-000000000005','OM-MX-05','250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000005','9b200003-2026-4020-8020-000000000005','D2C_WEB',38000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000006' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-06' AND (id<>'9b200003-2026-4020-8020-000000000006' OR product_id<>'9b200002-2026-4020-8020-000000000006')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-06';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000006',id,'Kẹo đậu phộng Omama','keo-dau-phong-omama-om-mx-06','Kẹo đậu phộng Omama. Quy cách: 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000006','9b200002-2026-4020-8020-000000000006','OM-MX-06','250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000006','9b200003-2026-4020-8020-000000000006','D2C_WEB',38000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000007' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-07' AND (id<>'9b200003-2026-4020-8020-000000000007' OR product_id<>'9b200002-2026-4020-8020-000000000007')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-07';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000007',id,'Kẹo gương Omama','keo-guong-omama-om-mx-07','Kẹo gương Omama. Quy cách: 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000007','9b200002-2026-4020-8020-000000000007','OM-MX-07','250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000007','9b200003-2026-4020-8020-000000000007','D2C_WEB',45000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000008' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MX-08' AND (id<>'9b200003-2026-4020-8020-000000000008' OR product_id<>'9b200002-2026-4020-8020-000000000008')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MX-08';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000008',id,'Mè xửng dẻo 2 lớp Omama','me-xung-deo-2-lop-omama-om-mx-08','Mè xửng dẻo 2 lớp Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='me-xung'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000008','9b200002-2026-4020-8020-000000000008','OM-MX-08','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000008','9b200003-2026-4020-8020-000000000008','D2C_WEB',55000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000009' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-01' AND (id<>'9b200003-2026-4020-8020-000000000009' OR product_id<>'9b200002-2026-4020-8020-000000000009')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000009',id,'Mứt hạt sen Omama','mut-hat-sen-omama-om-mu-01','Mứt hạt sen Omama. Quy cách: 300g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000009','9b200002-2026-4020-8020-000000000009','OM-MU-01','300g',300,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000009','9b200003-2026-4020-8020-000000000009','D2C_WEB',109000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000010' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-02' AND (id<>'9b200003-2026-4020-8020-000000000010' OR product_id<>'9b200002-2026-4020-8020-000000000010')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000010',id,'Mứt hạt sen Omama','mut-hat-sen-omama-om-mu-02','Mứt hạt sen Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000010','9b200002-2026-4020-8020-000000000010','OM-MU-02','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000010','9b200003-2026-4020-8020-000000000010','D2C_WEB',160000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000011' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-03' AND (id<>'9b200003-2026-4020-8020-000000000011' OR product_id<>'9b200002-2026-4020-8020-000000000011')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000011',id,'Mứt gừng Omama','mut-gung-omama-om-mu-03','Mứt gừng Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000011','9b200002-2026-4020-8020-000000000011','OM-MU-03','200g',200,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000011','9b200003-2026-4020-8020-000000000011','D2C_WEB',35000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000012' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-04' AND (id<>'9b200003-2026-4020-8020-000000000012' OR product_id<>'9b200002-2026-4020-8020-000000000012')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000012',id,'Mứt gừng Omama','mut-gung-omama-om-mu-04','Mứt gừng Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000012','9b200002-2026-4020-8020-000000000012','OM-MU-04','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000012','9b200003-2026-4020-8020-000000000012','D2C_WEB',65000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000013' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-05' AND (id<>'9b200003-2026-4020-8020-000000000013' OR product_id<>'9b200002-2026-4020-8020-000000000013')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-05';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000013',id,'Mứt dừa non Omama','mut-dua-non-omama-om-mu-05','Mứt dừa non Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000013','9b200002-2026-4020-8020-000000000013','OM-MU-05','200g',200,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000013','9b200003-2026-4020-8020-000000000013','D2C_WEB',85000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000014' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-06' AND (id<>'9b200003-2026-4020-8020-000000000014' OR product_id<>'9b200002-2026-4020-8020-000000000014')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-06';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000014',id,'Mứt dừa non Omama','mut-dua-non-omama-om-mu-06','Mứt dừa non Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000014','9b200002-2026-4020-8020-000000000014','OM-MU-06','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000014','9b200003-2026-4020-8020-000000000014','D2C_WEB',170000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000015' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-MU-07' AND (id<>'9b200003-2026-4020-8020-000000000015' OR product_id<>'9b200002-2026-4020-8020-000000000015')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-MU-07';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000015',id,'Mứt gừng mật ong Omama','mut-gung-mat-ong-omama-om-mu-07','Mứt gừng mật ong Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='mut-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000015','9b200002-2026-4020-8020-000000000015','OM-MU-07','200g',200,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000015','9b200003-2026-4020-8020-000000000015','D2C_WEB',75000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000016' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-01' AND (id<>'9b200003-2026-4020-8020-000000000016' OR product_id<>'9b200002-2026-4020-8020-000000000016')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000016',id,'Trà cung đình G8 Omama','tra-cung-dinh-g8-omama-om-tr-01','Trà cung đình G8 Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000016','9b200002-2026-4020-8020-000000000016','OM-TR-01','500g',500,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000016','9b200003-2026-4020-8020-000000000016','D2C_WEB',88000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-TR-02: synthetic weight 200g; original specification Hộp 20 gói.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000017' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-02' AND (id<>'9b200003-2026-4020-8020-000000000017' OR product_id<>'9b200002-2026-4020-8020-000000000017')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000017',id,'Trà cung đình G10 Omama','tra-cung-dinh-g10-omama-om-tr-02','Trà cung đình G10 Omama. Quy cách: Hộp 20 gói. Khối lượng 200g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000017','9b200002-2026-4020-8020-000000000017','OM-TR-02','Hộp 20 gói',200,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000017','9b200003-2026-4020-8020-000000000017','D2C_WEB',155000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-TR-03: synthetic weight 150g; original specification Hộp 30 túi.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000018' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-03' AND (id<>'9b200003-2026-4020-8020-000000000018' OR product_id<>'9b200002-2026-4020-8020-000000000018')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000018',id,'Trà cung đình túi lọc Omama','tra-cung-dinh-tui-loc-omama-om-tr-03','Trà cung đình túi lọc Omama. Quy cách: Hộp 30 túi. Khối lượng 150g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000018','9b200002-2026-4020-8020-000000000018','OM-TR-03','Hộp 30 túi',150,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000018','9b200003-2026-4020-8020-000000000018','D2C_WEB',65000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000019' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-04' AND (id<>'9b200003-2026-4020-8020-000000000019' OR product_id<>'9b200002-2026-4020-8020-000000000019')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000019',id,'Trà lá sen Omama','tra-la-sen-omama-om-tr-04','Trà lá sen Omama. Quy cách: 100g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000019','9b200002-2026-4020-8020-000000000019','OM-TR-04','100g',100,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000019','9b200003-2026-4020-8020-000000000019','D2C_WEB',105000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000020' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-05' AND (id<>'9b200003-2026-4020-8020-000000000020' OR product_id<>'9b200002-2026-4020-8020-000000000020')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-05';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000020',id,'Trà gừng Omama','tra-gung-omama-om-tr-05','Trà gừng Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000020','9b200002-2026-4020-8020-000000000020','OM-TR-05','200g',200,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000020','9b200003-2026-4020-8020-000000000020','D2C_WEB',80000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000021' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-06' AND (id<>'9b200003-2026-4020-8020-000000000021' OR product_id<>'9b200002-2026-4020-8020-000000000021')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-06';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000021',id,'Trà atiso đỏ Omama','tra-atiso-do-omama-om-tr-06','Trà atiso đỏ Omama. Quy cách: 100g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000021','9b200002-2026-4020-8020-000000000021','OM-TR-06','100g',100,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000021','9b200003-2026-4020-8020-000000000021','D2C_WEB',135000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000022' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-TR-07' AND (id<>'9b200003-2026-4020-8020-000000000022' OR product_id<>'9b200002-2026-4020-8020-000000000022')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-TR-07';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000022',id,'Trà tim sen Omama','tra-tim-sen-omama-om-tr-07','Trà tim sen Omama. Quy cách: 100g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='tra-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000022','9b200002-2026-4020-8020-000000000022','OM-TR-07','100g',100,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000022','9b200003-2026-4020-8020-000000000022','D2C_WEB',45000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-BA-01: synthetic weight 200g; original specification Hộp 10 cái.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000023' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-BA-01' AND (id<>'9b200003-2026-4020-8020-000000000023' OR product_id<>'9b200002-2026-4020-8020-000000000023')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-BA-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000023',id,'Bánh đậu xanh trái cây Omama','banh-dau-xanh-trai-cay-omama-om-ba-01','Bánh đậu xanh trái cây Omama. Quy cách: Hộp 10 cái. Khối lượng 200g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='banh-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000023','9b200002-2026-4020-8020-000000000023','OM-BA-01','Hộp 10 cái',200,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000023','9b200003-2026-4020-8020-000000000023','D2C_WEB',35000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-BA-02: synthetic weight 500g; original specification Gói 50 cái.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000024' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-BA-02' AND (id<>'9b200003-2026-4020-8020-000000000024' OR product_id<>'9b200002-2026-4020-8020-000000000024')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-BA-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000024',id,'Bánh in / bánh phục linh Omama','banh-in-banh-phuc-linh-omama-om-ba-02','Bánh in / bánh phục linh Omama. Quy cách: Gói 50 cái. Khối lượng 500g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='banh-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000024','9b200002-2026-4020-8020-000000000024','OM-BA-02','Gói 50 cái',500,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000024','9b200003-2026-4020-8020-000000000024','D2C_WEB',79000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-BA-03: synthetic weight 200g; original specification Hộp.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000025' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-BA-03' AND (id<>'9b200003-2026-4020-8020-000000000025' OR product_id<>'9b200002-2026-4020-8020-000000000025')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-BA-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000025',id,'Bánh ép khô Omama','banh-ep-kho-omama-om-ba-03','Bánh ép khô Omama. Quy cách: Hộp. Khối lượng 200g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='banh-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000025','9b200002-2026-4020-8020-000000000025','OM-BA-03','Hộp',200,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000025','9b200003-2026-4020-8020-000000000025','D2C_WEB',25000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000026' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-BA-04' AND (id<>'9b200003-2026-4020-8020-000000000026' OR product_id<>'9b200002-2026-4020-8020-000000000026')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-BA-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000026',id,'Bánh tráng dừa non Omama','banh-trang-dua-non-omama-om-ba-04','Bánh tráng dừa non Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='banh-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000026','9b200002-2026-4020-8020-000000000026','OM-BA-04','200g',200,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000026','9b200003-2026-4020-8020-000000000026','D2C_WEB',37000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-BA-05: synthetic weight 400g; original specification Hộp 20 cái.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000027' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-BA-05' AND (id<>'9b200003-2026-4020-8020-000000000027' OR product_id<>'9b200002-2026-4020-8020-000000000027')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-BA-05';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000027',id,'Bánh đậu xanh trái cây Omama','banh-dau-xanh-trai-cay-omama-om-ba-05','Bánh đậu xanh trái cây Omama. Quy cách: Hộp 20 cái. Khối lượng 400g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='banh-hue'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000027','9b200002-2026-4020-8020-000000000027','OM-BA-05','Hộp 20 cái',400,'GRAM','HOP_GIAY_KRAFT')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000027','9b200003-2026-4020-8020-000000000027','D2C_WEB',65000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000028' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-01' AND (id<>'9b200003-2026-4020-8020-000000000028' OR product_id<>'9b200002-2026-4020-8020-000000000028')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000028',id,'Kẹo mè xửng Omama','keo-me-xung-omama-om-ks-01','Kẹo mè xửng Omama. Quy cách: Hũ 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000028','9b200002-2026-4020-8020-000000000028','OM-KS-01','Hũ 250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000028','9b200003-2026-4020-8020-000000000028','D2C_WEB',45000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000029' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-02' AND (id<>'9b200003-2026-4020-8020-000000000029' OR product_id<>'9b200002-2026-4020-8020-000000000029')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000029',id,'Kẹo mè xửng chuối gừng Omama','keo-me-xung-chuoi-gung-omama-om-ks-02','Kẹo mè xửng chuối gừng Omama. Quy cách: Hũ 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000029','9b200002-2026-4020-8020-000000000029','OM-KS-02','Hũ 250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000029','9b200003-2026-4020-8020-000000000029','D2C_WEB',45000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000030' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-03' AND (id<>'9b200003-2026-4020-8020-000000000030' OR product_id<>'9b200002-2026-4020-8020-000000000030')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000030',id,'Kẹo gừng Omama','keo-gung-omama-om-ks-03','Kẹo gừng Omama. Quy cách: Hũ 100g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000030','9b200002-2026-4020-8020-000000000030','OM-KS-03','Hũ 100g',100,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000030','9b200003-2026-4020-8020-000000000030','D2C_WEB',22000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000031' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-04' AND (id<>'9b200003-2026-4020-8020-000000000031' OR product_id<>'9b200002-2026-4020-8020-000000000031')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000031',id,'Hạt sen khô Omama','hat-sen-kho-omama-om-ks-04','Hạt sen khô Omama. Quy cách: 500g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000031','9b200002-2026-4020-8020-000000000031','OM-KS-04','500g',500,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000031','9b200003-2026-4020-8020-000000000031','D2C_WEB',350000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000032' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-05' AND (id<>'9b200003-2026-4020-8020-000000000032' OR product_id<>'9b200002-2026-4020-8020-000000000032')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-05';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000032',id,'Hạt sen sấy lạnh Omama','hat-sen-say-lanh-omama-om-ks-05','Hạt sen sấy lạnh Omama. Quy cách: 100g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000032','9b200002-2026-4020-8020-000000000032','OM-KS-05','100g',100,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000032','9b200003-2026-4020-8020-000000000032','D2C_WEB',75000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000033' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-KS-06' AND (id<>'9b200003-2026-4020-8020-000000000033' OR product_id<>'9b200002-2026-4020-8020-000000000033')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-KS-06';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000033',id,'Bột hạt sen Omama','bot-hat-sen-omama-om-ks-06','Bột hạt sen Omama. Quy cách: 200g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='keo-hat-sen'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000033','9b200002-2026-4020-8020-000000000033','OM-KS-06','200g',200,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000033','9b200003-2026-4020-8020-000000000033','D2C_WEB',150000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000034' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-HQ-01' AND (id<>'9b200003-2026-4020-8020-000000000034' OR product_id<>'9b200002-2026-4020-8020-000000000034')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-HQ-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000034',id,'Combo 4 mè xửng Omama','combo-4-me-xung-omama-om-hq-01','Combo Omama đóng sẵn gồm 2 gói mè xửng giòn 150g và 2 gói mè xửng dẻo 250g. Tổng 800g; tổng giá lẻ 126.000đ, giá combo 119.000đ.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='hop-qua'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000034','9b200002-2026-4020-8020-000000000034','OM-HQ-01','2 giòn 150g + 2 dẻo 250g',800,'GRAM','HOP_QUA_TANG')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000034','9b200003-2026-4020-8020-000000000034','D2C_WEB',119000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000035' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-HQ-02' AND (id<>'9b200003-2026-4020-8020-000000000035' OR product_id<>'9b200002-2026-4020-8020-000000000035')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-HQ-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000035',id,'Hộp quà Đoàn Viên Omama','hop-qua-doan-vien-omama-om-hq-02','Hộp quà Đoàn Viên Omama đóng sẵn gồm mè xửng giòn 150g, mứt hạt sen 150g và trà cung đình 100g trong hộp giấy cứng màu kem. Tổng 400g. Mứt sen 150g chỉ có trong hộp, chưa bán lẻ. Tổng giá tham chiếu 125.000đ; giá hộp 119.000đ. Một hộp mua riêng chưa đạt ngưỡng đơn 150.000đ của GOIHUE10.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='hop-qua'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000035','9b200002-2026-4020-8020-000000000035','OM-HQ-02','Giòn 150g + sen 150g + trà 100g (400g)',400,'GRAM','HOP_QUA_TANG')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000035','9b200003-2026-4020-8020-000000000035','D2C_WEB',119000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000036' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-HQ-03' AND (id<>'9b200003-2026-4020-8020-000000000036' OR product_id<>'9b200002-2026-4020-8020-000000000036')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-HQ-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000036',id,'Hộp quà Di sản Cố Đô Omama','hop-qua-di-san-co-do-omama-om-hq-03','Hộp quà Di sản Cố Đô Omama đóng sẵn gồm mè xửng giòn 300g, mè xửng dẻo 300g và mứt hạt sen 300g. Tổng 900g. Tổng giá tham chiếu 209.000đ; giá hộp 199.000đ.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='hop-qua'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000036','9b200002-2026-4020-8020-000000000036','OM-HQ-03','Giòn 300g + dẻo 300g + sen 300g',900,'GRAM','HOP_QUA_TANG')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000036','9b200003-2026-4020-8020-000000000036','D2C_WEB',199000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-HQ-04: synthetic weight 600g; original specification Trà sen + bánh sen + mè xửng sen.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000037' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-HQ-04' AND (id<>'9b200003-2026-4020-8020-000000000037' OR product_id<>'9b200002-2026-4020-8020-000000000037')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-HQ-04';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000037',id,'Set quà trà sen – bánh sen – mè xửng Omama','set-qua-tra-sen-banh-sen-me-xung-omama-om-hq-04','Set quà Omama đóng sẵn gồm trà sen, bánh sen và mè xửng sen. Khối lượng 600g là giả lập cho demo, quy cách từng món cần bổ sung.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='hop-qua'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000037','9b200002-2026-4020-8020-000000000037','OM-HQ-04','Trà sen + bánh sen + mè xửng sen',600,'GRAM','HOP_QUA_TANG')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000037','9b200003-2026-4020-8020-000000000037','D2C_WEB',289000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-DS-01: synthetic weight 100g; original specification 100ml.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000038' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-DS-01' AND (id<>'9b200003-2026-4020-8020-000000000038' OR product_id<>'9b200002-2026-4020-8020-000000000038')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-DS-01';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000038',id,'Tinh dầu tràm Omama','tinh-dau-tram-omama-om-ds-01','Tinh dầu tràm Omama. Quy cách: 100ml. Khối lượng 100g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='dac-san-kho'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000038','9b200002-2026-4020-8020-000000000038','OM-DS-01','100ml',100,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000038','9b200003-2026-4020-8020-000000000038','D2C_WEB',195000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
-- OM-DS-02: synthetic weight 50g; original specification 50ml.
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000039' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-DS-02' AND (id<>'9b200003-2026-4020-8020-000000000039' OR product_id<>'9b200002-2026-4020-8020-000000000039')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-DS-02';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000039',id,'Tinh dầu tràm Omama','tinh-dau-tram-omama-om-ds-02','Tinh dầu tràm Omama. Quy cách: 50ml. Khối lượng 50g là giá trị giả lập phục vụ demo.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='dac-san-kho'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000039','9b200002-2026-4020-8020-000000000039','OM-DS-02','50ml',50,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000039','9b200003-2026-4020-8020-000000000039','D2C_WEB',139000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM products WHERE id='9b200002-2026-4020-8020-000000000040' AND created_by<>'pbl6-40-demo') OR
EXISTS(SELECT 1 FROM product_variants WHERE sku_code='OM-DS-03' AND (id<>'9b200003-2026-4020-8020-000000000040' OR product_id<>'9b200002-2026-4020-8020-000000000040')) THEN
RAISE EXCEPTION 'Conflicting pre-existing product or SKU: OM-DS-03';
END IF;
END $$;
INSERT INTO products(id,category_id,name,slug,description,approval_status,listing_status,created_by,approved_by,approved_at)
SELECT '9b200002-2026-4020-8020-000000000040',id,'Mắm ruốc Omama','mam-ruoc-omama-om-ds-03','Mắm ruốc Omama. Quy cách: Hũ 250g.','APPROVED','ACTIVE','pbl6-40-demo','pbl6-40-demo',CURRENT_TIMESTAMP
FROM categories WHERE slug='dac-san-kho'
ON CONFLICT (id) DO NOTHING;
INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit,packaging_type)
VALUES ('9b200003-2026-4020-8020-000000000040','9b200002-2026-4020-8020-000000000040','OM-DS-03','Hũ 250g',250,'GRAM','DANG_ROI')
ON CONFLICT (id) DO NOTHING;
INSERT INTO channel_prices(id,variant_id,channel,amount_units,amount_nanos,created_by)
VALUES ('9b200004-2026-4020-8020-000000000040','9b200003-2026-4020-8020-000000000040','D2C_WEB',38000,0,'pbl6-40-demo')
ON CONFLICT (id) DO NOTHING;
