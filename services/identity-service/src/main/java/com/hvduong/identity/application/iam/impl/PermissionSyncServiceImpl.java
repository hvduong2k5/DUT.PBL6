package com.hvduong.identity.application.iam.impl;

import java.time.OffsetDateTime;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.BffSessionStore;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
@RequiredArgsConstructor
public class PermissionSyncServiceImpl implements PermissionSyncService {

    static final String ACTION_SYNC_FAILED = "IAM_PERMISSION_SYNC_FAILED";

    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IamRolePermissionMapper rolePermissionMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final AuditService auditService;

    @Autowired(required = false)
    private BffSessionStore bffSessionStore;

    @Override
    @Transactional(readOnly = true)
    public List<String> resolveEffectivePermissions(UUID principalId) {
        if (principalId == null) {
            return List.of();
        }
        List<String> permissions = rolePermissionMapper.findEffectivePermissionCodesByPrincipal(
                principalId, OffsetDateTime.now());
        if (permissions == null || permissions.isEmpty()) {
            return List.of();
        }
        return new ArrayList<>(permissions);
    }

    @Override
    @Transactional(readOnly = true)
    public List<PermissionSyncService.PermissionGrant> resolveEffectivePermissionGrants(UUID principalId) {
        if (principalId == null) {
            return List.of();
        }
        List<IamRolePermissionMapper.PermissionGrantRow> rows =
                rolePermissionMapper.findEffectivePermissionGrantsByPrincipal(
                        principalId, OffsetDateTime.now());
        if (rows == null || rows.isEmpty()) {
            return List.of();
        }
        return rows.stream()
                .map(r -> new PermissionSyncService.PermissionGrant(
                        r.getPermissionCode(), r.getScopeType(), r.getScopeId()))
                .toList();
    }

    @Override
    public SyncResult syncPrincipal(UUID principalId, boolean forceRelogin, Trigger trigger, UUID actorId) {
        if (principalId == null) {
            return SyncResult.failed(null, "PRINCIPAL_ID_REQUIRED");
        }

        IdentityPrincipalRow principal;
        try {
            principal = principalMapper.findById(principalId);
        } catch (Exception ex) {
            log.error("Failed to load principal while syncing permissions [principalId={}, trigger={}]",
                    principalId, trigger, ex);
            auditFailure(principalId, trigger, actorId, "PRINCIPAL_LOOKUP_FAILED", ex.getMessage());
            return SyncResult.failed(principalId, "PRINCIPAL_LOOKUP_FAILED");
        }

        if (principal == null) {
            auditFailure(principalId, trigger, actorId, "PRINCIPAL_NOT_FOUND", "Principal does not exist");
            return SyncResult.failed(principalId, "PRINCIPAL_NOT_FOUND");
        }
        if (principal.getIdpSubject() == null || principal.getIdpSubject().isBlank()) {
            auditFailure(principalId, trigger, actorId, "PRINCIPAL_NOT_LINKED",
                    "Principal is not linked to a Keycloak subject");
            return SyncResult.failed(principalId, "PRINCIPAL_NOT_LINKED");
        }

        List<String> flatPermissions;
        List<PermissionSyncService.PermissionGrant> grants;
        try {
            flatPermissions = resolveEffectivePermissions(principalId);
            grants = resolveEffectivePermissionGrants(principalId);
            List<String> grantStrings = grants.stream()
                    .map(g -> g.permission() + "#" + (g.scopeType() != null ? g.scopeType() : "GLOBAL")
                            + (g.scopeId() != null && !g.scopeId().isBlank() ? "#" + g.scopeId() : ""))
                    .toList();
            keycloakAdminClient.updateUserPermissionsWithGrants(
                    principal.getIdpSubject(), flatPermissions, grantStrings);
        } catch (Exception ex) {
            log.error("Keycloak permission sync failed [principalId={}, idpSubject={}, trigger={}]",
                    principalId, principal.getIdpSubject(), trigger, ex);
            auditFailure(principalId, trigger, actorId, "KEYCLOAK_SYNC_FAILED", ex.getMessage());
            return SyncResult.failed(principalId, "KEYCLOAK_SYNC_FAILED");
        }

        if (forceRelogin) {
            revokeSessions(principalId, principal.getIdpSubject(), trigger, actorId);
        }

        log.info("Synced {} permission(s) / {} grant(s) to Keycloak [principalId={}, trigger={}, forceRelogin={}]",
                flatPermissions.size(), grants.size(), principalId, trigger, forceRelogin);
        return SyncResult.ok(principalId, flatPermissions.size(), grants.size());
    }

    @Override
    public int syncAllPrincipalsForRole(UUID roleId, boolean forceRelogin, Trigger trigger, UUID actorId) {
        if (roleId == null) {
            return 0;
        }
        List<IamRoleAssignmentRow> active = roleAssignmentMapper.findActiveByRoleId(roleId, OffsetDateTime.now());
        if (active == null || active.isEmpty()) {
            return 0;
        }

        Set<UUID> principalIds = new LinkedHashSet<>();
        for (IamRoleAssignmentRow assignment : active) {
            if (assignment.getPrincipalId() != null) {
                principalIds.add(assignment.getPrincipalId());
            }
        }

        int synced = 0;
        for (UUID principalId : principalIds) {
            if (syncPrincipal(principalId, forceRelogin, trigger, actorId).success()) {
                synced++;
            }
        }
        return synced;
    }

    private void revokeSessions(UUID principalId, String idpSubject, Trigger trigger, UUID actorId) {
        if (bffSessionStore != null) {
            try {
                bffSessionStore.removeSessionsByPrincipalId(principalId);
            } catch (Exception ex) {
                log.error("Failed to remove BFF sessions [principalId={}, trigger={}]", principalId, trigger, ex);
                auditFailure(principalId, trigger, actorId, "BFF_SESSION_REVOKE_FAILED", ex.getMessage());
            }
        } else {
            log.warn("BffSessionStore unavailable; skipped BFF session revocation [principalId={}]", principalId);
        }

        try {
            keycloakAdminClient.logoutUserStrict(idpSubject);
        } catch (Exception ex) {
            log.error("Failed to revoke Keycloak sessions [principalId={}, idpSubject={}, trigger={}]",
                    principalId, idpSubject, trigger, ex);
            auditFailure(principalId, trigger, actorId, "KEYCLOAK_LOGOUT_FAILED", ex.getMessage());
        }
    }

    private void auditFailure(UUID principalId, Trigger trigger, UUID actorId, String reasonCode, String message) {
        auditService.recordAudit(
                actorId != null ? "USER" : "SYSTEM",
                actorId,
                null,
                ACTION_SYNC_FAILED,
                "PRINCIPAL",
                principalId != null ? principalId.toString() : null,
                "FAILED",
                reasonCode,
                null,
                null,
                Map.of(
                        "trigger", trigger != null ? trigger.name() : "UNKNOWN",
                        "message", message != null ? message : ""
                )
        );
    }
}
