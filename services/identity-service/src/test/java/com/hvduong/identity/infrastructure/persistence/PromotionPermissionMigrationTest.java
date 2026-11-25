package com.hvduong.identity.infrastructure.persistence;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.Statement;
import java.util.UUID;

import org.junit.jupiter.api.Test;
import org.springframework.core.io.ClassPathResource;
import org.springframework.jdbc.datasource.init.ScriptUtils;

class PromotionPermissionMigrationTest {

    private static final String MIGRATION = "db/migration/V27__seed_promotion_permissions.sql";

    @Test
    void promotionSeedIsPackagedAsANewMigration() throws Exception {
        String sql = new ClassPathResource(MIGRATION).getContentAsString(StandardCharsets.UTF_8);
        assertThat(sql).contains("'PROMOTION_VIEW'", "'PROMOTION_MANAGE'", "INSERT INTO iam_role_permissions");
        assertThat(sql).contains("r.is_enabled = TRUE", "r.archived_at IS NULL");
        assertThat(sql).doesNotContain("INSERT INTO identity_principals", "INSERT INTO iam_role_assignments");
    }

    @Test
    void promotionSeedIsIdempotentAndOnlyGrantsTheActiveAdminRole() throws Exception {
        String jdbcUrl = System.getenv("IDENTITY_TEST_JDBC_URL");
        assumeTrue(jdbcUrl != null && !jdbcUrl.isBlank(), "IDENTITY_TEST_JDBC_URL is required for PostgreSQL seed coverage");
        try (Connection connection = DriverManager.getConnection(jdbcUrl)) {
            connection.setAutoCommit(false);
            try (Statement statement = connection.createStatement()) {
                String schema = "promotion_seed_" + UUID.randomUUID().toString().replace("-", "");
                statement.execute("CREATE SCHEMA " + schema);
                statement.execute("SET LOCAL search_path TO " + schema);
                statement.execute("CREATE TABLE iam_permissions(permission_code text PRIMARY KEY, resource_code text, action_code text, risk_level text, scope_required boolean, description text, UNIQUE(resource_code,action_code))");
                statement.execute("CREATE TABLE iam_roles(role_id uuid PRIMARY KEY, role_code text, is_enabled boolean, archived_at timestamptz)");
                statement.execute("CREATE TABLE iam_role_permissions(role_id uuid REFERENCES iam_roles, permission_code text REFERENCES iam_permissions, PRIMARY KEY(role_id,permission_code))");
                UUID admin = UUID.randomUUID();
                statement.execute("INSERT INTO iam_roles VALUES ('" + admin + "','ADMIN',true,NULL),('" + UUID.randomUUID() + "','ADMIN',false,NULL),('" + UUID.randomUUID() + "','ADMIN',true,now()),('" + UUID.randomUUID() + "','FINANCE',true,NULL)");
                ScriptUtils.executeSqlScript(connection, new ClassPathResource(MIGRATION));
                ScriptUtils.executeSqlScript(connection, new ClassPathResource(MIGRATION));
                try (ResultSet rows = statement.executeQuery("SELECT count(*) FROM iam_permissions WHERE resource_code='PROMOTION' AND scope_required=true")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(2);
                }
                try (ResultSet rows = statement.executeQuery("SELECT count(*),count(*) FILTER(WHERE role_id='" + admin + "') FROM iam_role_permissions")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(2);
                    assertThat(rows.getInt(2)).isEqualTo(2);
                }
            } finally {
                // Schema and all test data are transaction-local and never persist.
                connection.rollback();
            }
        }
    }
}
