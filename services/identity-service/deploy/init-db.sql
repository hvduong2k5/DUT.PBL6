-- Automatically create databases on first container startup
SELECT 'CREATE DATABASE keycloak_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'keycloak_db')\gexec

SELECT 'CREATE DATABASE identity_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'identity_db')\gexec

