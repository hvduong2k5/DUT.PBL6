package com.hvduong.identity.application.account.impl;

import com.hvduong.identity.application.account.EmployeeInvitationService;
import com.hvduong.identity.application.audit.AuditService;
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
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

@Slf4j
@Service
public class EmployeeInvitationServiceImpl implements EmployeeInvitationService {

    private final EmployeeInvitationMapper invitationMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;
    private final KeycloakAdminClient keycloakAdminClient;

    public EmployeeInvitationServiceImpl(
            EmployeeInvitationMapper invitationMapper,
            IdentityPrincipalMapper principalMapper,
            AuditService auditService
    ) {
        this(invitationMapper, principalMapper, auditService, null);
    }

    @Autowired
    public EmployeeInvitationServiceImpl(
            EmployeeInvitationMapper invitationMapper,
            IdentityPrincipalMapper principalMapper,
            AuditService auditService,
            @Autowired(required = false) KeycloakAdminClient keycloakAdminClient
    ) {
        this.invitationMapper = invitationMapper;
        this.principalMapper = principalMapper;
        this.auditService = auditService;
        this.keycloakAdminClient = keycloakAdminClient;
    }

    @Value("${keycloak.public-url:${keycloak.server-url:https://keycloak.omama.vn}}")
    private String publicUrl = "https://keycloak.omama.vn";

    @Value("${keycloak.realm:omama}")
    private String realm = "omama";

    @Override
    @Transactional
    public InviteEmployeeResponse inviteEmployee(UUID invitedBy, int validDays) {
        UUID principalId = UUID.randomUUID();

        // 1. Create employee principal in PENDING status
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpIssuer(publicUrl + "/realms/" + realm);
        principal.setPrincipalType(PrincipalType.EMPLOYEE.name());
        principal.setProvisioningStatus(ProvisioningStatus.PENDING.name());
        principal.setVersion(0L);
        principalMapper.insert(principal);

        // 2. Create invitation record
        UUID invitationId = UUID.randomUUID();
        OffsetDateTime expiresAt = OffsetDateTime.now().plusDays(validDays > 0 ? validDays : 7);

        EmployeeInvitationRow invitation = new EmployeeInvitationRow();
        invitation.setInvitationId(invitationId);
        invitation.setPrincipalId(principalId);
        invitation.setInvitedByPrincipalId(invitedBy);
        invitation.setInvitationStatus(InvitationStatus.PENDING.name());
        invitation.setExpiresAt(expiresAt);
        invitationMapper.insert(invitation);

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
    public boolean acceptInvitation(UUID invitationId, String idpSubject) {
        Optional<EmployeeInvitationRow> invOpt = invitationMapper.findById(invitationId);
        if (invOpt.isEmpty() || !InvitationStatus.PENDING.matches(invOpt.get().getInvitationStatus())) {
            return false;
        }

        EmployeeInvitationRow inv = invOpt.get();
        if (inv.getExpiresAt().isBefore(OffsetDateTime.now())) {
            invitationMapper.updateStatus(invitationId, InvitationStatus.EXPIRED.name(), null);
            return false;
        }

        invitationMapper.updateStatus(invitationId, InvitationStatus.ACCEPTED.name(), null);
        principalMapper.updateProvisioningStatus(inv.getPrincipalId(), ProvisioningStatus.ACTIVE.name(), idpSubject, 0L);

        if (keycloakAdminClient != null && idpSubject != null && !idpSubject.isBlank()) {
            try {
                keycloakAdminClient.setUserAttribute(idpSubject, IdentityConstants.ATTR_PRINCIPAL_TYPE, PrincipalType.EMPLOYEE.name());
                keycloakAdminClient.setUserAttribute(idpSubject, IdentityConstants.ATTR_APP_USER_ID, inv.getPrincipalId().toString());
            } catch (Exception ex) {
                log.warn("Failed to set Keycloak attributes for accepted employee invitation [principalId={}, idpSubject={}]",
                        inv.getPrincipalId(), idpSubject, ex);
            }
        }

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

        return true;
    }

    @Override
    @Transactional
    public boolean revokeInvitation(UUID invitationId, UUID revokedBy) {
        Optional<EmployeeInvitationRow> invOpt = invitationMapper.findById(invitationId);
        if (invOpt.isEmpty() || !InvitationStatus.PENDING.matches(invOpt.get().getInvitationStatus())) {
            return false;
        }

        invitationMapper.updateStatus(invitationId, InvitationStatus.REVOKED.name(), null);

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

