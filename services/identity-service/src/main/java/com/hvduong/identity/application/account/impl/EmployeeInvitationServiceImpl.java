package com.hvduong.identity.application.account.impl;

import com.hvduong.identity.application.account.EmployeeInvitationService;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.persistence.entity.EmployeeInvitationRow;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.EmployeeInvitationMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class EmployeeInvitationServiceImpl implements EmployeeInvitationService {

    private final EmployeeInvitationMapper invitationMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;

    @Override
    @Transactional
    public InviteResult inviteEmployee(UUID invitedBy, int validDays) {
        UUID principalId = UUID.randomUUID();

        // 1. Create employee principal in PENDING status
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpIssuer("https://keycloak.omama.vn/realms/omama");
        principal.setPrincipalType("EMPLOYEE");
        principal.setProvisioningStatus("PENDING");
        principal.setVersion(0L);
        principalMapper.insert(principal);

        // 2. Create invitation record
        UUID invitationId = UUID.randomUUID();
        OffsetDateTime expiresAt = OffsetDateTime.now().plusDays(validDays > 0 ? validDays : 7);

        EmployeeInvitationRow invitation = new EmployeeInvitationRow();
        invitation.setInvitationId(invitationId);
        invitation.setPrincipalId(principalId);
        invitation.setInvitedByPrincipalId(invitedBy);
        invitation.setInvitationStatus("PENDING");
        invitation.setExpiresAt(expiresAt);
        invitationMapper.insert(invitation);

        auditService.recordAudit(
                "EMPLOYEE",
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

        return InviteResult.builder()
                .invitationId(invitationId)
                .principalId(principalId)
                .status("PENDING")
                .expiresAt(expiresAt)
                .build();
    }

    @Override
    @Transactional
    public boolean acceptInvitation(UUID invitationId, String idpSubject) {
        Optional<EmployeeInvitationRow> invOpt = invitationMapper.findById(invitationId);
        if (invOpt.isEmpty() || !"PENDING".equals(invOpt.get().getInvitationStatus())) {
            return false;
        }

        EmployeeInvitationRow inv = invOpt.get();
        if (inv.getExpiresAt().isBefore(OffsetDateTime.now())) {
            invitationMapper.updateStatus(invitationId, "EXPIRED", null);
            return false;
        }

        invitationMapper.updateStatus(invitationId, "ACCEPTED", null);
        principalMapper.updateProvisioningStatus(inv.getPrincipalId(), "ACTIVE", idpSubject, 0L);

        auditService.recordAudit(
                "EMPLOYEE",
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
        if (invOpt.isEmpty() || !"PENDING".equals(invOpt.get().getInvitationStatus())) {
            return false;
        }

        invitationMapper.updateStatus(invitationId, "REVOKED", null);

        auditService.recordAudit(
                "EMPLOYEE",
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

