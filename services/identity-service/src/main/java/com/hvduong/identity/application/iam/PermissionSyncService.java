package com.hvduong.identity.application.iam;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Reconciles the Keycloak projection of a principal's grants with the IAM database.
 *
 * <p>The IAM database ({@code iam_roles}, {@code iam_role_permissions},
 * {@code iam_role_assignments}) is the source of truth. Keycloak user attributes are
 * only a projection used to mint tokens. Every operation that changes effective grants
 * must call this service so the next issued token reflects the new state.</p>
 *
 * <p>Issued access tokens are immutable. Revoking a grant cannot rewrite a token that
 * was already signed; it only takes effect after the next refresh, which is why
 * revocation paths also force a re-login.</p>
 */
public interface PermissionSyncService {

    /** Why a sync is running. Recorded in the audit trail on failure. */
    enum Trigger {
        ROLE_ASSIGN,
        ROLE_REVOKE,
        ROLE_PERMISSION_CHANGE,
        ROLE_DISABLE,
        ROLE_ENABLE,
        ACCOUNT_STATUS_CHANGE,
        MANUAL_RESYNC
    }

    /**
     * Outcome of syncing a single principal. Failures never abort the caller's
     * transaction: they are logged and written to the audit trail.
     */
    record SyncResult(UUID principalId, boolean success, int permissionCount, int grantCount, String errorCode) {

        public static SyncResult ok(UUID principalId, int permissionCount, int grantCount) {
            return new SyncResult(principalId, true, permissionCount, grantCount, null);
        }

        public static SyncResult failed(UUID principalId, String errorCode) {
            return new SyncResult(principalId, false, 0, 0, errorCode);
        }
    }

    /**
     * One permission grant bound to its scope. Never cross-product between elements.
     */
    record PermissionGrant(String permission, String scopeType, String scopeId) {}

    /**
     * Recompute the effective permission codes for a principal from the IAM database.
     * Only active assignments of enabled roles are counted.
     */
    List<String> resolveEffectivePermissions(UUID principalId);

    /**
     * Recompute the effective permission grants (with scope) for a principal from the IAM database.
     * Each element binds a permission to its assignment scope ({@code scope_type} + {@code scope_id}).
     */
    List<PermissionGrant> resolveEffectivePermissionGrants(UUID principalId);

    /**
     * Push the recomputed permissions of one principal to Keycloak.
     *
     * @param forceRelogin when true (any reduction of grants) also drop BFF sessions and
     *                     Keycloak sessions so the principal must authenticate again
     */
    SyncResult syncPrincipal(UUID principalId, boolean forceRelogin, Trigger trigger, UUID actorId);

    /**
     * Sync every principal that currently holds an active assignment of {@code roleId}.
     *
     * @return number of principals synced successfully
     */
    int syncAllPrincipalsForRole(UUID roleId, boolean forceRelogin, Trigger trigger, UUID actorId);
}
