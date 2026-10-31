package com.hvduong.identity.application.account;

import com.hvduong.identity.application.account.impl.EmployeeInvitationServiceImpl;
import com.hvduong.identity.dto.response.InviteEmployeeResponse;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.EmployeeInvitationMapper;
import com.hvduong.identity.infrastructure.persistence.entity.EmployeeInvitationRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.OffsetDateTime;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class EmployeeInvitationServiceTest {

    @Mock
    private EmployeeInvitationMapper invitationMapper;

    @Mock
    private IdentityPrincipalMapper principalMapper;

    @Mock
    private AuditService auditService;

    private EmployeeInvitationService invitationService;

    @BeforeEach
    void setUp() {
        invitationService = new EmployeeInvitationServiceImpl(invitationMapper, principalMapper, auditService);
    }

    @Test
    void shouldCreateInvitationSuccessfully() {
        // Arrange
        UUID invitedBy = UUID.randomUUID();

        // Act
        InviteEmployeeResponse result = invitationService.inviteEmployee(invitedBy, 14);

        // Assert
        assertThat(result.getInvitationId()).isNotNull();
        assertThat(result.getPrincipalId()).isNotNull();
        assertThat(result.getStatus()).isEqualTo("PENDING");
        assertThat(result.getExpiresAt()).isAfter(OffsetDateTime.now());

        verify(principalMapper, times(1)).insert(any());
        verify(invitationMapper, times(1)).insert(any());
        verify(auditService, times(1)).recordAudit(any(), any(), any(), any(), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void shouldAcceptInvitationSuccessfully() {
        // Arrange
        UUID invitationId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        EmployeeInvitationRow row = new EmployeeInvitationRow();
        row.setInvitationId(invitationId);
        row.setPrincipalId(principalId);
        row.setInvitationStatus("PENDING");
        row.setExpiresAt(OffsetDateTime.now().plusDays(5));

        when(invitationMapper.findById(invitationId)).thenReturn(Optional.of(row));

        // Act
        boolean accepted = invitationService.acceptInvitation(invitationId, "keycloak-sub-123");

        // Assert
        assertThat(accepted).isTrue();
        verify(invitationMapper, times(1)).updateStatus(invitationId, "ACCEPTED", null);
        verify(principalMapper, times(1)).updateProvisioningStatus(principalId, "ACTIVE", "keycloak-sub-123", 0L);
    }

    @Test
    void shouldRevokeInvitationSuccessfully() {
        // Arrange
        UUID invitationId = UUID.randomUUID();
        UUID revokedBy = UUID.randomUUID();
        EmployeeInvitationRow row = new EmployeeInvitationRow();
        row.setInvitationId(invitationId);
        row.setInvitationStatus("PENDING");

        when(invitationMapper.findById(invitationId)).thenReturn(Optional.of(row));

        // Act
        boolean revoked = invitationService.revokeInvitation(invitationId, revokedBy);

        // Assert
        assertThat(revoked).isTrue();
        verify(invitationMapper, times(1)).updateStatus(invitationId, "REVOKED", null);
    }
}

