package com.hvduong.identity.api.account;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;

import java.nio.charset.StandardCharsets;
import java.sql.DriverManager;
import java.util.Objects;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;

/** Actual PostgreSQL seed execution in a rollback-only private schema, never live IAM data. */
class InternalShippingPermissionSeedTest {
    @Test
    @EnabledIfEnvironmentVariable(named = "IDENTITY_TEST_JDBC_URL", matches = ".+")
    void seedIsIdempotentAndDoesNotGrantDisabledArchivedOrOtherRoles() throws Exception {
        String schema = "identity_shipping_seed_" + UUID.randomUUID().toString().replace("-", "");
        String sql;
        try (var source = Objects.requireNonNull(getClass().getResourceAsStream("/db/migration/V24__seed_internal_shipping_permissions.sql"))) {
            sql = new String(source.readAllBytes(), StandardCharsets.UTF_8);
        }
        try (var connection = DriverManager.getConnection(System.getenv("IDENTITY_TEST_JDBC_URL"), System.getenv("IDENTITY_TEST_DB_USER"), System.getenv("IDENTITY_TEST_DB_PASSWORD"))) {
            connection.setAutoCommit(false);
            try (var statement = connection.createStatement()) {
                statement.execute("CREATE SCHEMA " + schema);
                statement.execute("SET LOCAL search_path TO " + schema);
                statement.execute("""
                    CREATE TABLE iam_permissions(permission_code TEXT PRIMARY KEY, resource_code TEXT, action_code TEXT, risk_level TEXT CHECK (risk_level IN ('LOW','MEDIUM','HIGH','CRITICAL')), scope_required BOOLEAN, description TEXT, UNIQUE(resource_code,action_code));
                    CREATE TABLE iam_roles(role_id TEXT PRIMARY KEY, role_code TEXT, is_enabled BOOLEAN, archived_at TIMESTAMPTZ);
                    CREATE TABLE iam_role_permissions(role_id TEXT REFERENCES iam_roles(role_id), permission_code TEXT REFERENCES iam_permissions(permission_code), PRIMARY KEY(role_id,permission_code));
                    INSERT INTO iam_roles VALUES ('active','ADMIN',TRUE,NULL),('disabled','ADMIN',FALSE,NULL),('archived','ADMIN',TRUE,now()),('other','FINANCE',TRUE,NULL);
                    """);
                statement.execute(sql); statement.execute(sql);
                try (var rows = statement.executeQuery("SELECT role_id,permission_code FROM iam_role_permissions ORDER BY permission_code")) {
                    assertThat(rows.next()).isTrue(); assertThat(rows.getString(1)).isEqualTo("active"); assertThat(rows.getString(2)).isEqualTo("SHIPMENT_ASSIGN");
                    assertThat(rows.next()).isTrue(); assertThat(rows.getString(1)).isEqualTo("active"); assertThat(rows.getString(2)).isEqualTo("SHIPMENT_EXECUTE"); assertThat(rows.next()).isFalse();
                }
                try (var rows = statement.executeQuery("SELECT count(*) FROM iam_permissions WHERE scope_required=TRUE")) {
                    assertThat(rows.next()).isTrue(); assertThat(rows.getInt(1)).isEqualTo(2);
                }
            } finally { connection.rollback(); }
        }
    }
}
