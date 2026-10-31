package com.hvduong.identity.application.account.impl;

import com.hvduong.identity.application.account.UserLifecycleService;
import com.hvduong.identity.dto.response.UserProfileResponse;
import com.hvduong.identity.dto.response.UserRoleScopeResponse;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.entity.*;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.*;

@Slf4j
@Service
@RequiredArgsConstructor
public class UserLifecycleServiceImpl implements UserLifecycleService {

    private final IdentityPrincipalMapper principalMapper;
    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IamRoleMapper roleMapper;
    private final IamRolePermissionMapper rolePermissionMapper;
    private final OutboxEventMapper outboxEventMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final AuditService auditService;
    private final ObjectMapper objectMapper;

    @Override
    @Transactional(readOnly = true)
    public Optional<UserProfileResponse> getProfileBySubject(String idpIssuer, String idpSubject, String email) {
        IdentityPrincipalRow principal = principalMapper.findByIssuerAndSubject(idpIssuer, idpSubject);
        if (principal == null) {
            return Optional.empty();
        }

        List<IamRoleAssignmentRow> assignments = roleAssignmentMapper.findActiveByPrincipal(
                principal.getPrincipalId(), 
                OffsetDateTime.now()
        );

        Set<String> effectivePermissions = new LinkedHashSet<>();
        List<UserRoleScopeResponse> roleDtos = new ArrayList<>();

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
                roleDtos.add(UserRoleScopeResponse.builder()
                        .roleCode(roleCode)
                        .roleName(roleName)
                        .scopeType(a.getScopeType())
                        .scopeId(a.getScopeId())
                        .build());
            }
        }

        if (roleDtos.isEmpty() && "CUSTOMER".equalsIgnoreCase(principal.getPrincipalType())) {
            roleDtos.add(UserRoleScopeResponse.builder()
                    .roleCode("CUSTOMER")
                    .roleName("Khách hàng")
                    .scopeType("GLOBAL")
                    .scopeId(null)
                    .build());
        }

        return Optional.of(UserProfileResponse.builder()
                .principalId(principal.getPrincipalId())
                .idpSubject(principal.getIdpSubject())
                .email(email)
                .principalType(principal.getPrincipalType())
                .provisioningStatus(principal.getProvisioningStatus())
                .roles(roleDtos)
                .permissions(new ArrayList<>(effectivePermissions))
                .build());
    }

    @Override
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

        if (principal.getIdpSubject() != null) {
            try {
                keycloakAdminClient.setUserEnabled(principal.getIdpSubject(), false);
            } catch (Exception ex) {
                log.warn("Failed to disable user {} in Keycloak: {}", principal.getIdpSubject(), ex.getMessage());
            }
        }

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

        enqueueOutboxEvent(principalId, "identity.events.v1.UserSuspended", Map.of(
                "principalId", principalId.toString(),
                "reason", reason != null ? reason : "Administrative suspension",
                "suspendedAt", OffsetDateTime.now().toString()
        ));
    }

    @Override
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

        if (principal.getIdpSubject() != null) {
            try {
                keycloakAdminClient.setUserEnabled(principal.getIdpSubject(), true);
            } catch (Exception ex) {
                log.warn("Failed to enable user {} in Keycloak: {}", principal.getIdpSubject(), ex.getMessage());
            }
        }

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

