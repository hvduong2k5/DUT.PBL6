CREATE DATABASE IF NOT EXISTS analytics_db;
CREATE USER IF NOT EXISTS analytics_worker IDENTIFIED WITH sha256_password BY 'local-worker';
CREATE USER IF NOT EXISTS analytics_api IDENTIFIED WITH sha256_password BY 'local-api';
GRANT SELECT, INSERT, ALTER DELETE ON analytics_db.* TO analytics_worker;
GRANT SELECT ON analytics_db.* TO analytics_api;
