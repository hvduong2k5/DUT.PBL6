package com.hvduong.identity.application.account;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import lombok.Builder;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.*;

@Slf4j
@Service
@RequiredArgsConstructor
public class UserLifecycleService {

    private final IdentityPrincipalMapper principalMapper;
    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IamRoleMapper roleMapper;
    private final IamRolePermissionMapper rolePermissionMapper;
    private final OutboxEventMapper outboxEventMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final AuditService auditService;
    private final ObjectMapper objectMapper;

    @Data
    @Builder
    public static class UserRoleScopeDto {
        private String roleCode;
        private String roleName;
        private String scopeType;
        private String scopeId;
    }

    @Data
    @Builder
    public static class UserProfileDto {
        private UUID principalId;
        private String idpSubject;
        private String email;
        private String principalType;
        private String provisioningStatus;
        private List<UserRoleScopeDto> roles;
        private List<String> permissions;
    }

    @Transactional(readOnly = true)
    public Optional<UserProfileDto> getProfileBySubject(String idpSubject, String email) {
        IdentityPrincipalRow principal = principalMapper.findBySubject(idpSubject);
        if (principal == null) {
            return Optional.empty();
        }

        List<IamRoleAssignmentRow> assignments = roleAssignmentMapper.findActiveByPrincipal(
                principal.getPrincipalId(), 
                OffsetDateTime.now()
        );

        Set<String> effectivePermissions = new LinkedHashSet<>();
        List<UserRoleScopeDto> roleDtos = new ArrayList<>();

        if (assignments != null) {
            for (IamRoleAssignmentRow a : assignments) {
                String roleCode = a.getRoleId() != null ? a.getRoleId().toString() : "UNKNOWN";
                String roleName = null;
                if (roleMapper != null && a.getRoleId() != null) {
                    IamRoleRow role = roleMapper.findById(a.getRoleId());
                    if (role != null) {
                        roleCode = role.getRoleCode();
                        roleName = role.getRoleName();
                    }
                }
                if (rolePermissionMapper != null && a.getRoleId() != null) {
                    List<IamRolePermissionRow> permRows = rolePermissionMapper.findByRoleId(a.getRoleId());
                    if (permRows != null) {
                        for (IamRolePermissionRow p : permRows) {
                            if (p.getPermissionCode() != null && !p.getPermissionCode().isBlank()) {
                                effectivePermissions.add(p.getPermissionCode());
                            }
                        }
                    }
                }
                roleDtos.add(UserRoleScopeDto.builder()
                        .roleCode(roleCode)
                        .roleName(roleName)
                        .scopeType(a.getScopeType())
                        .scopeId(a.getScopeId())
                        .build());
            }
        }

        if (roleDtos.isEmpty() && "CUSTOMER".equalsIgnoreCase(principal.getPrincipalType())) {
            roleDtos.add(UserRoleScopeDto.builder()
                    .roleCode("CUSTOMER")
                    .roleName("Khách hàng")
                    .scopeType("GLOBAL")
                    .build());
        }

        return Optional.of(UserProfileDto.builder()
                .principalId(principal.getPrincipalId())
                .idpSubject(principal.getIdpSubject())
                .email(email)
                .principalType(principal.getPrincipalType())
                .provisioningStatus(principal.getProvisioningStatus())
                .roles(roleDtos)
                .permissions(new ArrayList<>(effectivePermissions))
                .build());
    }

    @Transactional
    public void suspendAccount(UUID principalId, String reason, UUID actorPrincipalId) {
        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Principal not found: " + principalId);
        }

        int updated = principalMapper.updateProvisioningStatus(
                principalId, 
                "SUSPENDED", 
                principal.getIdpSubject(), 
                principal.getVersion()
        );

        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict when suspending principal: " + principalId);
        }

        // Disable in Keycloak if synced
        if (principal.getIdpSubject() != null) {
            try {
                keycloakAdminClient.setUserEnabled(principal.getIdpSubject(), false);
            } catch (Exception ex) {
                log.warn("Failed to disable user {} in Keycloak: {}", principal.getIdpSubject(), ex.getMessage());
            }
        }

        // Write Audit Log
        auditService.recordAudit(
                "USER",
                actorPrincipalId,
                null,
                "SUSPEND_ACCOUNT",
                "PRINCIPAL",
                principalId.toString(),
                "SUCCESS",
                reason,
                null,
                null,
                Map.of("reason", reason != null ? reason : "Administrative suspension")
        );

        // Queue Outbox Event
        enqueueOutboxEvent(principalId, "identity.events.v1.UserSuspended", Map.of(
                "principalId", principalId.toString(),
                "reason", reason != null ? reason : "Administrative suspension",
                "suspendedAt", OffsetDateTime.now().toString()
        ));
    }

    @Transactional
    public void reactivateAccount(UUID principalId, UUID actorPrincipalId) {
        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Principal not found: " + principalId);
        }

        int updated = principalMapper.updateProvisioningStatus(
                principalId, 
                "ACTIVE", 
                principal.getIdpSubject(), 
                principal.getVersion()
        );

        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict when reactivating principal: " + principalId);
        }

        // Enable in Keycloak if synced
        if (principal.getIdpSubject() != null) {
            try {
                keycloakAdminClient.setUserEnabled(principal.getIdpSubject(), true);
            } catch (Exception ex) {
                log.warn("Failed to enable user {} in Keycloak: {}", principal.getIdpSubject(), ex.getMessage());
            }
        }

        // Write Audit Log
        auditService.recordAudit(
                "USER",
                actorPrincipalId,
                null,
                "REACTIVATE_ACCOUNT",
                "PRINCIPAL",
                principalId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Collections.emptyMap()
        );

        // Queue Outbox Event
        enqueueOutboxEvent(principalId, "identity.events.v1.UserReactivated", Map.of(
                "principalId", principalId.toString(),
                "reactivatedAt", OffsetDateTime.now().toString()
        ));
    }

    private void enqueueOutboxEvent(UUID principalId, String eventType, Object payload) {
        try {
            OutboxEventRow outboxEvent = new OutboxEventRow();
            outboxEvent.setEventId(UUID.randomUUID());
            outboxEvent.setAggregateType("IdentityPrincipal");
            outboxEvent.setAggregateId(principalId.toString());
            outboxEvent.setEventType(eventType);
            outboxEvent.setSchemaVersion(1);
            outboxEvent.setPartitionKey(principalId.toString());
            outboxEvent.setPayload(objectMapper.writeValueAsString(payload));
            outboxEvent.setDeliveryStatus("PENDING");
            outboxEvent.setAttemptCount(0);
            outboxEvent.setNextAttemptAt(OffsetDateTime.now());
            outboxEvent.setOccurredAt(OffsetDateTime.now());
            outboxEventMapper.insert(outboxEvent);
        } catch (Exception ex) {
            log.error("Failed to enqueue outbox event {} for principal {}", eventType, principalId, ex);
        }
    }
}

