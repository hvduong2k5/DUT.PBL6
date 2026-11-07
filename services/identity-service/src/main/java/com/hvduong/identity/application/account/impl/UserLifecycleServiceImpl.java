package com.hvduong.identity.application.account.impl;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.account.UserLifecycleService;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.dto.common.PageResponse;
import com.hvduong.identity.dto.response.AdminUserDetailResponse;
import com.hvduong.identity.dto.response.AdminUserItemResponse;
import com.hvduong.identity.dto.response.UserProfileResponse;
import com.hvduong.identity.dto.response.UserRoleScopeResponse;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.entity.*;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.keycloak.representations.idm.UserRepresentation;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

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
    private final RestTemplate restTemplate;

    @Value("${keycloak.server-url}")
    private String serverUrl;

    @Value("${keycloak.realm}")
    private String realm;

    @Value("${keycloak.public-client-id:frontend-web}")
    private String publicClientId;

    @Override
    @Transactional(readOnly = true)
    public Optional<UserProfileResponse> getProfileBySubject(String idpIssuer, String idpSubject, String email) {
        IdentityPrincipalRow principal = principalMapper.findByIssuerAndSubject(idpIssuer, idpSubject);
        if (principal == null) {
            return Optional.empty();
        }

        // The owner realm uses openid without an email claim. Resolve display
        // data through the already-bound IdP subject rather than treating a
        // username or caller header as an email address.
        if (email == null || email.isBlank()) {
            email = keycloakAdminClient.getUserById(principal.getIdpSubject())
                    .map(UserRepresentation::getEmail).orElse(null);
        }

        List<IamRoleAssignmentRow> assignments = roleAssignmentMapper.findActiveByPrincipal(
                principal.getPrincipalId(), 
                OffsetDateTime.now()
        );

        Set<String> effectivePermissions = new LinkedHashSet<>();
        List<UserRoleScopeResponse> roleDtos = new ArrayList<>();

        if (assignments != null) {
            for (IamRoleAssignmentRow a : assignments) {
                List<String> scopedPermissions = new ArrayList<>();
                boolean enabled = false;
                String roleCode = a.getRoleId() != null ? a.getRoleId().toString() : "UNKNOWN";
                String roleName = null;
                if (roleMapper != null && a.getRoleId() != null) {
                    IamRoleRow role = roleMapper.findById(a.getRoleId());
                    if (role != null) {
                        roleCode = role.getRoleCode();
                        roleName = role.getRoleName();
                        enabled = Boolean.TRUE.equals(role.getIsEnabled());
                    }
                }
                if (rolePermissionMapper != null && a.getRoleId() != null) {
                    List<IamRolePermissionRow> permRows = rolePermissionMapper.findByRoleId(a.getRoleId());
                    if (permRows != null) {
                        for (IamRolePermissionRow p : permRows) {
                            if (p.getPermissionCode() != null && !p.getPermissionCode().isBlank()) {
                                effectivePermissions.add(p.getPermissionCode());
                                if (enabled) scopedPermissions.add(p.getPermissionCode());
                            }
                        }
                    }
                }
                roleDtos.add(UserRoleScopeResponse.builder()
                        .roleCode(roleCode)
                        .roleName(roleName)
                        .scopeType(a.getScopeType())
                        .scopeId(a.getScopeId())
                        .permissions(scopedPermissions)
                        .build());
            }
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
        // RULE 1: ADMIN không được phép tự khóa tài khoản của chính mình
        if (actorPrincipalId != null && actorPrincipalId.equals(principalId)) {
            throw new IllegalArgumentException("ADMIN không được phép tự khóa tài khoản của chính mình.");
        }

        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Principal not found: " + principalId);
        }

        if (principal.getIdpSubject() != null && actorPrincipalId != null 
                && principal.getIdpSubject().equals(actorPrincipalId.toString())) {
            throw new IllegalArgumentException("ADMIN không được phép tự khóa tài khoản của chính mình.");
        }

        // RULE 2: Chặn khóa tài khoản có vai trò ADMIN
        if (isAdmin(principalId)) {
            throw new IllegalStateException("Không thể khóa tài khoản có vai trò ADMIN.");
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
                keycloakAdminClient.logoutUser(principal.getIdpSubject());
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

    @Override
    @Transactional(readOnly = true)
    public PageResponse<AdminUserItemResponse> listUsers(String type, String status, int page, int size, UUID actorId) {

        int validPage = Math.max(0, page);
        int validSize = size <= 0 ? 20 : Math.min(size, 100);
        int offset = validPage * validSize;

        List<IdentityPrincipalRow> rows = principalMapper.findPage(type, status, offset, validSize);
        long totalItems = principalMapper.countTotal(type, status);
        int totalPages = (int) Math.ceil((double) totalItems / validSize);

        List<AdminUserItemResponse> items = new ArrayList<>();
        OffsetDateTime now = OffsetDateTime.now();

        for (IdentityPrincipalRow row : rows) {
            String email = null;
            String firstName = null;
            String lastName = null;

            if (row.getIdpSubject() != null) {
                Optional<UserRepresentation> kcUserOpt = keycloakAdminClient.getUserById(row.getIdpSubject());
                if (kcUserOpt.isPresent()) {
                    UserRepresentation kcUser = kcUserOpt.get();
                    email = kcUser.getEmail();
                    firstName = kcUser.getFirstName();
                    lastName = kcUser.getLastName();
                }
            }

            List<String> roleCodes = new ArrayList<>();
            List<IamRoleAssignmentRow> assignments = roleAssignmentMapper.findActiveByPrincipal(row.getPrincipalId(), now);
            if (assignments != null) {
                for (IamRoleAssignmentRow a : assignments) {
                    if (a.getRoleId() != null) {
                        IamRoleRow role = roleMapper.findById(a.getRoleId());
                        if (role != null) {
                            roleCodes.add(role.getRoleCode());
                        }
                    }
                }
            }


            items.add(AdminUserItemResponse.builder()
                    .principalId(row.getPrincipalId())
                    .idpSubject(row.getIdpSubject())
                    .email(email)
                    .firstName(firstName)
                    .lastName(lastName)
                    .principalType(row.getPrincipalType())
                    .provisioningStatus(row.getProvisioningStatus())
                    .createdAt(row.getCreatedAt())
                    .roles(roleCodes)
                    .build());
        }

        return PageResponse.<AdminUserItemResponse>builder()
                .items(items)
                .page(validPage)
                .size(validSize)
                .totalItems(totalItems)
                .totalPages(totalPages)
                .build();
    }

    @Override
    @Transactional(readOnly = true)
    public AdminUserDetailResponse getUserDetail(UUID principalId, UUID actorId) {

        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Không tìm thấy người dùng với ID: " + principalId);
        }

        String email = null;
        String firstName = null;
        String lastName = null;

        if (principal.getIdpSubject() != null) {
            Optional<UserRepresentation> kcUserOpt = keycloakAdminClient.getUserById(principal.getIdpSubject());
            if (kcUserOpt.isPresent()) {
                UserRepresentation kcUser = kcUserOpt.get();
                email = kcUser.getEmail();
                firstName = kcUser.getFirstName();
                lastName = kcUser.getLastName();
            }
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



        return AdminUserDetailResponse.builder()
                .principalId(principal.getPrincipalId())
                .idpSubject(principal.getIdpSubject())
                .email(email)
                .firstName(firstName)
                .lastName(lastName)
                .principalType(principal.getPrincipalType())
                .provisioningStatus(principal.getProvisioningStatus())
                .createdAt(principal.getCreatedAt())
                .updatedAt(principal.getUpdatedAt())
                .roles(roleDtos)
                .permissions(new ArrayList<>(effectivePermissions))
                .build();
    }

    @Override
    @Transactional
    public void adminResetPassword(UUID principalId, String newPassword, boolean temporary, UUID actorId) {

        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Không tìm thấy người dùng với ID: " + principalId);
        }

        if (principal.getIdpSubject() == null || principal.getIdpSubject().isBlank()) {
            throw new IllegalStateException("Tài khoản chưa được liên kết với Identity Provider.");
        }

        keycloakAdminClient.resetUserPassword(principal.getIdpSubject(), newPassword, temporary);

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "ADMIN_RESET_PASSWORD",
                "PRINCIPAL",
                principalId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("temporary", temporary)
        );
    }

    @Override
    @Transactional
    public void revokeUserSessions(UUID principalId, UUID actorId) {

        IdentityPrincipalRow principal = principalMapper.findById(principalId);
        if (principal == null) {
            throw new IllegalArgumentException("Không tìm thấy người dùng với ID: " + principalId);
        }

        if (principal.getIdpSubject() != null) {
            keycloakAdminClient.logoutUser(principal.getIdpSubject());
        }

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "REVOKE_USER_SESSIONS",
                "PRINCIPAL",
                principalId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Collections.emptyMap()
        );
    }

    @Override
    public void changeOwnPassword(UUID actorId, String currentPassword, String newPassword) {
        if (actorId == null) {
            throw new IllegalArgumentException("Không xác định được danh tính người dùng.");
        }

        IdentityPrincipalRow principal = principalMapper.findById(actorId);
        if (principal == null) {
            principal = principalMapper.findBySubject(actorId.toString());
        }
        if (principal == null || principal.getIdpSubject() == null) {
            throw new IllegalArgumentException("Tài khoản không tồn tại trên hệ thống.");
        }

        Optional<UserRepresentation> kcUserOpt = keycloakAdminClient.getUserById(principal.getIdpSubject());
        if (kcUserOpt.isEmpty() || kcUserOpt.get().getEmail() == null) {
            throw new IllegalStateException("Không tìm thấy thông tin xác thực trên Identity Provider.");
        }

        String email = kcUserOpt.get().getEmail();

        // Xác thực mật khẩu cũ bằng cách gọi token endpoint
        String tokenUrl = buildTokenEndpoint();
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

        MultiValueMap<String, String> body = new LinkedMultiValueMap<>();
        body.add("grant_type", "password");
        body.add("client_id", publicClientId);
        body.add("username", email);
        body.add("password", currentPassword);

        try {
            restTemplate.postForEntity(tokenUrl, new HttpEntity<>(body, headers), String.class);
        } catch (Exception ex) {
            throw new IllegalArgumentException("Mật khẩu hiện tại không chính xác.");
        }

        // Cập nhật mật khẩu mới
        keycloakAdminClient.resetUserPassword(principal.getIdpSubject(), newPassword, false);

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "CHANGE_OWN_PASSWORD",
                "PRINCIPAL",
                actorId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Collections.emptyMap()
        );
    }

    private UUID resolvePrincipalId(UUID actorId) {
        if (actorId == null) {
            return null;
        }
        IdentityPrincipalRow row = principalMapper.findById(actorId);
        if (row != null) {
            return row.getPrincipalId();
        }
        row = principalMapper.findBySubject(actorId.toString());
        if (row != null) {
            return row.getPrincipalId();
        }
        return actorId;
    }

    private boolean isAdmin(UUID principalId) {
        if (principalId == null) return false;
        return principalMapper.hasRoleCode(principalId, "ADMIN");
    }

    private void requireAdmin(UUID actorId) {
        UUID effectiveActorId = resolvePrincipalId(actorId);
        if (effectiveActorId == null || !isAdmin(effectiveActorId)) {
            throw new SecurityException("Thao tác này chỉ dành riêng cho tài khoản có vai trò ADMIN.");
        }
    }

    private String buildTokenEndpoint() {
        String baseUrl = serverUrl.endsWith("/") ? serverUrl.substring(0, serverUrl.length() - 1) : serverUrl;
        return baseUrl + "/realms/" + realm + "/protocol/openid-connect/token";
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
