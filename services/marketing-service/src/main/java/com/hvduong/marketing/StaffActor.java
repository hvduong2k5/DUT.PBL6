package com.hvduong.marketing;

import java.util.Set;
import java.util.UUID;

record StaffActor(UUID id, Set<Grant> grants) {
    record Grant(String permissionCode, String scopeType, String scopeId) {}

    StaffActor {
        grants = Set.copyOf(grants);
    }

    boolean can(String permission, String departmentId) {
        return grants.stream().anyMatch(grant -> permission.equals(grant.permissionCode())
                && (("GLOBAL".equals(grant.scopeType()) && grant.scopeId() == null)
                || ("DEPARTMENT".equals(grant.scopeType()) && departmentId.equals(grant.scopeId()))));
    }
}
