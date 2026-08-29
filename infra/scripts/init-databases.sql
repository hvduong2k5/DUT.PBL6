-- Script khởi tạo nhiều Cơ sở dữ liệu riêng biệt cho các Microservices (Database-per-Service Pattern)

CREATE DATABASE om_identity_db;
CREATE DATABASE om_catalog_db;
CREATE DATABASE om_order_db;
CREATE DATABASE om_payment_db;
CREATE DATABASE om_inventory_db;
CREATE DATABASE om_shipping_db;
CREATE DATABASE om_analytics_db;

-- Cấp quyền
GRANT ALL PRIVILEGES ON DATABASE om_identity_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_catalog_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_order_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_payment_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_inventory_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_shipping_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE om_analytics_db TO postgres;
