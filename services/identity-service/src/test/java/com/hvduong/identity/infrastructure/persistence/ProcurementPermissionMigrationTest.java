package com.hvduong.identity.infrastructure.persistence;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;
import java.util.UUID;

import org.junit.jupiter.api.Test;
import org.springframework.core.io.ClassPathResource;

class ProcurementPermissionMigrationTest {
    private static final String MIGRATION = "db/migration/V28__seed_procurement_permissions.sql";

    private static void executeScript(Connection connection, String resource) throws Exception {
        // PostgreSQL must parse PL/pgSQL dollar quotes; splitting on semicolons
        // with Spring ScriptUtils corrupts the real V1/V9 function bodies.
        try (Statement statement = connection.createStatement()) {
            statement.execute(new ClassPathResource(resource).getContentAsString(StandardCharsets.UTF_8));
        }
    }

    @Test
    void packagesFiveScopedPermissionsWithoutCreatingAssignments() throws Exception {
        String sql = new ClassPathResource(MIGRATION).getContentAsString(StandardCharsets.UTF_8);
        assertThat(sql).contains("'PROCUREMENT_VIEW'", "'PROCUREMENT_MANAGE'", "'PROCUREMENT_APPROVE'",
                "'PROCUREMENT_RECEIVE'", "'PROCUREMENT_VARIANCE_RESOLVE'",
                "r.is_enabled = TRUE", "r.archived_at IS NULL");
        assertThat(sql).doesNotContain("INSERT INTO identity_principals", "INSERT INTO iam_role_assignments",
                "INSERT INTO iam_roles", "UPDATE iam_permissions", "DELETE FROM");
    }

    @Test
    void seedsOnlyActiveAdminsPreservesExistingDataAndCanRunTwice() throws Exception {
        String url = System.getenv("IDENTITY_TEST_JDBC_URL");
        assumeTrue(url != null && !url.isBlank(), "Set IDENTITY_TEST_JDBC_URL for PostgreSQL migration coverage");
        try (Connection connection = DriverManager.getConnection(url)) {
            connection.setAutoCommit(false);
            try (Statement statement = connection.createStatement()) {
                String schema = "procurement_seed_" + UUID.randomUUID().toString().replace("-", "");
                statement.execute("CREATE SCHEMA " + schema);
                statement.execute("SET LOCAL search_path TO " + schema);
                // Use real IAM DDL, including constraints, not an in-memory approximation.
                executeScript(connection, "db/migration/V1__create_principals_and_provisioning.sql");
                executeScript(connection, "db/migration/V2__create_iam_role_permission_model.sql");
                executeScript(connection, "db/migration/V9__partial_unique_role_code_on_iam_roles.sql");
                executeScript(connection, "db/migration/V13__separate_disabled_and_archived_roles.sql");
                UUID admin = UUID.randomUUID();
                statement.execute("INSERT INTO iam_roles(role_id,role_code,role_name,is_enabled,archived_at) VALUES "
                        + "('" + admin + "','ADMIN','Active admin',true,NULL),"
                        + "('" + UUID.randomUUID() + "','ADMIN','Archived admin',false,now()),"
                        + "('" + UUID.randomUUID() + "','FINANCE','Finance',true,NULL)");
                statement.execute("INSERT INTO iam_permissions(permission_code,resource_code,action_code,description,scope_required) "
                        + "VALUES ('PROCUREMENT_VIEW','PROCUREMENT','VIEW','Existing owner description',true),"
                        + "('UNRELATED_VIEW','UNRELATED','VIEW','Unrelated permission',false)");
                statement.execute("INSERT INTO iam_role_permissions(role_id,permission_code) VALUES ('" + admin + "','UNRELATED_VIEW')");
                executeScript(connection, MIGRATION);
                executeScript(connection, MIGRATION);
                try (var rows = statement.executeQuery("SELECT count(*) FROM iam_permissions WHERE resource_code='PROCUREMENT' AND scope_required=true")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(5);
                }
                try (var rows = statement.executeQuery("SELECT count(*),count(*) FILTER(WHERE role_id='" + admin + "') FROM iam_role_permissions WHERE permission_code LIKE 'PROCUREMENT_%'")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(5);
                    assertThat(rows.getInt(2)).isEqualTo(5);
                }
                try (var rows = statement.executeQuery("SELECT description FROM iam_permissions WHERE permission_code='PROCUREMENT_VIEW'")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getString(1)).isEqualTo("Existing owner description");
                }
                try (var rows = statement.executeQuery("SELECT (SELECT count(*) FROM iam_roles), (SELECT count(*) FROM iam_role_assignments), (SELECT count(*) FROM identity_principals), (SELECT count(*) FROM iam_role_permissions WHERE permission_code='UNRELATED_VIEW')")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(3);
                    assertThat(rows.getInt(2)).isZero();
                    assertThat(rows.getInt(3)).isZero();
                    assertThat(rows.getInt(4)).isEqualTo(1);
                }
                // An unarchived but disabled ADMIN must not receive new grants.
                statement.execute("DELETE FROM iam_role_permissions WHERE permission_code LIKE 'PROCUREMENT_%'");
                statement.execute("UPDATE iam_roles SET is_enabled=false WHERE role_id='" + admin + "'");
                executeScript(connection, MIGRATION);
                try (var rows = statement.executeQuery("SELECT count(*) FROM iam_role_permissions WHERE permission_code LIKE 'PROCUREMENT_%'")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isZero();
                }
            } finally {
                connection.rollback();
            }
        }
    }

    @Test
    void emptyRoleCatalogDoesNotBootstrapAnAdmin() throws Exception {
        String url = System.getenv("IDENTITY_TEST_JDBC_URL");
        assumeTrue(url != null && !url.isBlank(), "Set IDENTITY_TEST_JDBC_URL for PostgreSQL migration coverage");
        try (Connection connection = DriverManager.getConnection(url)) {
            connection.setAutoCommit(false);
            try (Statement statement = connection.createStatement()) {
                String schema = "procurement_empty_" + UUID.randomUUID().toString().replace("-", "");
                statement.execute("CREATE SCHEMA " + schema);
                statement.execute("SET LOCAL search_path TO " + schema);
                executeScript(connection, "db/migration/V1__create_principals_and_provisioning.sql");
                executeScript(connection, "db/migration/V2__create_iam_role_permission_model.sql");
                executeScript(connection, "db/migration/V9__partial_unique_role_code_on_iam_roles.sql");
                executeScript(connection, "db/migration/V13__separate_disabled_and_archived_roles.sql");
                executeScript(connection, MIGRATION);
                try (var rows = statement.executeQuery("SELECT (SELECT count(*) FROM iam_permissions), (SELECT count(*) FROM iam_role_permissions), (SELECT count(*) FROM iam_roles)")) {
                    assertThat(rows.next()).isTrue();
                    assertThat(rows.getInt(1)).isEqualTo(5);
                    assertThat(rows.getInt(2)).isZero();
                    assertThat(rows.getInt(3)).isZero();
                }
            } finally {
                connection.rollback();
            }
        }
    }
}
