package com.hvduong.identity.application.account.impl;

import com.hvduong.identity.application.account.EmployeeInvitationService;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.common.exception.ErrorCode;
import com.hvduong.identity.common.exception.IdentityException;
import com.hvduong.identity.application.dto.response.InviteEmployeeResponse;
import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.InvitationStatus;
import com.hvduong.identity.common.enums.PrincipalType;
import com.hvduong.identity.common.enums.ProvisioningStatus;
import com.hvduong.identity.domain.entity.EmployeeInvitationRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.EmployeeInvitationMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class EmployeeInvitationServiceImpl implements EmployeeInvitationService {

    private final EmployeeInvitationMapper invitationMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;
    private final KeycloakAdminClient keycloakAdminClient;
    private final PermissionSyncService permissionSyncService;

    @Value("${spring.security.oauth2.resourceserver.jwt.issuer-uri}")
    private String issuer = "https://keycloak.omama.vn/realms/omama";

    @Override
    @Transactional
    public InviteEmployeeResponse inviteEmployee(UUID invitedBy, int validDays, String idpSubject) {
        if (idpSubject != null && (idpSubject.isBlank() || idpSubject.length() > 255
                || keycloakAdminClient.getUserById(idpSubject).isEmpty())) {
            throw new IdentityException(ErrorCode.INVALID_ARGUMENT);
        }

        IdentityPrincipalRow principal = idpSubject == null ? null : principalMapper.findByIssuerAndSubject(issuer, idpSubject);
        if (principal == null) {
            principal = new IdentityPrincipalRow();
            principal.setPrincipalId(UUID.randomUUID());
            principal.setIdpIssuer(issuer);
            principal.setIdpSubject(idpSubject);
            principal.setPrincipalType(PrincipalType.EMPLOYEE.name());
            principal.setProvisioningStatus(ProvisioningStatus.PENDING.name());
            principal.setVersion(0L);
            try {
                principalMapper.insert(principal);
            } catch (DuplicateKeyException ex) {
                throw new IdentityException(ErrorCode.INVITATION_ACCOUNT_ALREADY_LINKED);
            }
        } else {
            if (!PrincipalType.EMPLOYEE.matches(principal.getPrincipalType())
                    || !ProvisioningStatus.PENDING.matches(principal.getProvisioningStatus())) {
                throw new IdentityException(ErrorCode.INVITATION_ACCOUNT_ALREADY_LINKED);
            }
            var pending = invitationMapper.findPendingByPrincipalIdForUpdate(principal.getPrincipalId());
            if (pending.isPresent()) {
                if (pending.get().getExpiresAt().isAfter(OffsetDateTime.now())) {
                    throw new IdentityException(ErrorCode.INVITATION_ACCOUNT_ALREADY_LINKED);
                }
                invitationMapper.updateStatus(pending.get().getInvitationId(), InvitationStatus.EXPIRED.name());
            }
            if (!ProvisioningStatus.PENDING.matches(principalMapper.findById(principal.getPrincipalId()).getProvisioningStatus())) {
                throw new IdentityException(ErrorCode.INVITATION_ACCOUNT_ALREADY_LINKED);
            }
        }
        UUID principalId = principal.getPrincipalId();

        // 2. Create invitation record
        UUID invitationId = UUID.randomUUID();
        OffsetDateTime expiresAt = OffsetDateTime.now().plusDays(validDays > 0 ? validDays : 7);

        EmployeeInvitationRow invitation = new EmployeeInvitationRow();
        invitation.setInvitationId(invitationId);
        invitation.setPrincipalId(principalId);
        invitation.setInvitedByPrincipalId(invitedBy);
        invitation.setInvitationStatus(InvitationStatus.PENDING.name());
        invitation.setExpiresAt(expiresAt);
        try {
            invitationMapper.insert(invitation);
        } catch (DuplicateKeyException ex) {
            throw new IdentityException(ErrorCode.INVITATION_ACCOUNT_ALREADY_LINKED);
        }

        auditService.recordAudit(
                PrincipalType.EMPLOYEE.name(),
                invitedBy,
                null,
                "EMPLOYEE_INVITE_CREATE",
                "EMPLOYEE_INVITATION",
                invitationId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("principalId", principalId.toString(), "expiresAt", expiresAt.toString())
        );

        return InviteEmployeeResponse.builder()
                .invitationId(invitationId)
                .principalId(principalId)
                .status(InvitationStatus.PENDING.name())
                .expiresAt(expiresAt)
                .build();
    }

    @Override
    @Transactional
    public void acceptInvitation(UUID invitationId, String issuer, String idpSubject) {
        EmployeeInvitationRow inv = invitationMapper.findByIdForUpdate(invitationId)
                .orElseThrow(() -> new IdentityException(ErrorCode.INVITATION_NOT_FOUND));
        IdentityPrincipalRow principal = principalMapper.findById(inv.getPrincipalId());
        if (principal == null || idpSubject == null || idpSubject.isBlank()
                || !idpSubject.equals(principal.getIdpSubject()) || !principal.getIdpIssuer().equals(issuer)) {
            throw new IdentityException(ErrorCode.PERMISSION_DENIED);
        }
        if (InvitationStatus.EXPIRED.matches(inv.getInvitationStatus())) {
            throw new IdentityException(ErrorCode.INVITATION_EXPIRED);
        }
        if (!InvitationStatus.PENDING.matches(inv.getInvitationStatus())
                || !ProvisioningStatus.PENDING.matches(principal.getProvisioningStatus())) {
            throw new IdentityException(ErrorCode.INVITATION_NOT_PENDING);
        }
        if (!inv.getExpiresAt().isAfter(OffsetDateTime.now())) {
            throw new IdentityException(ErrorCode.INVITATION_EXPIRED);
        }

        int activated = principalMapper.updateProvisioningStatus(
                principal.getPrincipalId(), ProvisioningStatus.ACTIVE.name(), null, principal.getVersion());
        if (activated != 1) {
            throw new IdentityException(ErrorCode.VERSION_CONFLICT);
        }
        keycloakAdminClient.setUserAttribute(idpSubject, IdentityConstants.ATTR_APP_USER_ID, inv.getPrincipalId().toString());
        var sync = permissionSyncService.syncPrincipal(
                inv.getPrincipalId(), false, PermissionSyncService.Trigger.ACCOUNT_STATUS_CHANGE, inv.getPrincipalId());
        if (!sync.success()) {
            throw new IdentityException(ErrorCode.PERMISSION_SYNC_FAILED);
        }
        invitationMapper.updateStatus(invitationId, InvitationStatus.ACCEPTED.name());

        auditService.recordAudit(
                PrincipalType.EMPLOYEE.name(),
                inv.getPrincipalId(),
                null,
                "EMPLOYEE_INVITE_ACCEPT",
                "EMPLOYEE_INVITATION",
                invitationId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("idpSubject", idpSubject)
        );
    }

    @Override
    @Transactional
    public boolean revokeInvitation(UUID invitationId, UUID revokedBy) {
        Optional<EmployeeInvitationRow> invOpt = invitationMapper.findByIdForUpdate(invitationId);
        if (invOpt.isEmpty() || !InvitationStatus.PENDING.matches(invOpt.get().getInvitationStatus())) {
            return false;
        }

        invitationMapper.updateStatus(invitationId, InvitationStatus.REVOKED.name());

        auditService.recordAudit(
                PrincipalType.EMPLOYEE.name(),
                revokedBy,
                null,
                "EMPLOYEE_INVITE_REVOKE",
                "EMPLOYEE_INVITATION",
                invitationId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("invitationId", invitationId.toString())
        );

        return true;
    }
}

