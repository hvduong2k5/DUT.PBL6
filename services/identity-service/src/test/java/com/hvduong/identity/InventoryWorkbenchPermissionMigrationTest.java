package com.hvduong.identity;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

@SpringBootTest
class InventoryWorkbenchPermissionMigrationTest {
    @Autowired private JdbcTemplate database;

    @Test void inventoryWorkbenchPermissionsExistAndRequireExplicitScope() {
        assertEquals(2, database.queryForObject("""
                SELECT count(*) FROM iam_permissions
                WHERE permission_code IN ('INVENTORY_VIEW', 'WAREHOUSE_WORKBENCH_VIEW')
                  AND scope_required = TRUE AND action_code = 'VIEW'
                """, Integer.class));
    }
}
