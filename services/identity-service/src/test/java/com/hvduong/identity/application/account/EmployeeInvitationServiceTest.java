package com.hvduong.identity.application.account;

import com.hvduong.identity.application.account.impl.EmployeeInvitationServiceImpl;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.exception.ErrorCode;
import com.hvduong.identity.common.exception.IdentityException;
import com.hvduong.identity.domain.entity.EmployeeInvitationRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.EmployeeInvitationMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.keycloak.representations.idm.UserRepresentation;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.OffsetDateTime;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class EmployeeInvitationServiceTest {
    private static final String ISSUER = "https://keycloak.omama.vn/realms/omama";
    private static final String SUBJECT = "invited-keycloak-subject";

    @Mock private EmployeeInvitationMapper invitations;
    @Mock private IdentityPrincipalMapper principals;
    @Mock private AuditService audit;
    @Mock private KeycloakAdminClient keycloak;
    @Mock private PermissionSyncService permissions;

    private final UUID invitationId = UUID.randomUUID();
    private final UUID principalId = UUID.randomUUID();
    private EmployeeInvitationService service;
    private EmployeeInvitationRow invitation;

    @BeforeEach
    void setUp() {
        service = new EmployeeInvitationServiceImpl(invitations, principals, audit, keycloak, permissions);
        invitation = new EmployeeInvitationRow();
        invitation.setInvitationId(invitationId);
        invitation.setPrincipalId(principalId);
        invitation.setInvitationStatus("PENDING");
        invitation.setExpiresAt(OffsetDateTime.now().plusDays(5));
    }

    @Test
    void createsInvitationBoundToExistingKeycloakSubject() {
        when(keycloak.getUserById(SUBJECT)).thenReturn(Optional.of(new UserRepresentation()));
        var response = service.inviteEmployee(UUID.randomUUID(), 14, SUBJECT);

        var principal = ArgumentCaptor.forClass(IdentityPrincipalRow.class);
        verify(principals).insert(principal.capture());
        assertThat(principal.getValue().getIdpSubject()).isEqualTo(SUBJECT);
        assertThat(principal.getValue().getPrincipalId()).isEqualTo(response.getPrincipalId());
        assertThat(response.getStatus()).isEqualTo("PENDING");
        assertThat(response.getExpiresAt()).isAfter(OffsetDateTime.now().plusDays(13));
        verify(invitations).insert(any());
    }

    @Test
    void acceptsForOwnerAndActivatesWithCurrentVersionAndGrantSync() {
        stubOwner(SUBJECT);
        when(principals.updateProvisioningStatus(principalId, "ACTIVE", null, 4L)).thenReturn(1);
        when(permissions.syncPrincipal(principalId, false, PermissionSyncService.Trigger.ACCOUNT_STATUS_CHANGE, principalId))
                .thenReturn(PermissionSyncService.SyncResult.ok(principalId, 0, 0));

        service.acceptInvitation(invitationId, ISSUER, SUBJECT);

        var order = inOrder(principals, keycloak, permissions, invitations);
        order.verify(principals).updateProvisioningStatus(principalId, "ACTIVE", null, 4L);
        order.verify(keycloak).setUserAttribute(SUBJECT, IdentityConstants.ATTR_APP_USER_ID, principalId.toString());
        order.verify(permissions).syncPrincipal(principalId, false, PermissionSyncService.Trigger.ACCOUNT_STATUS_CHANGE, principalId);
        order.verify(invitations).updateStatus(invitationId, "ACCEPTED");
    }

    @Test
    void rejectsAnotherSubjectOrIssuerAndAnUnboundLegacyInvitation() {
        stubOwner(SUBJECT);
        assertError(ISSUER, "wrong-subject", ErrorCode.PERMISSION_DENIED);
        assertError("https://different-issuer.test", SUBJECT, ErrorCode.PERMISSION_DENIED);
        stubOwner(null);
        assertError(ISSUER, SUBJECT, ErrorCode.PERMISSION_DENIED);
        verify(invitations, never()).updateStatus(any(), any());
        verifyNoInteractions(permissions, keycloak);
    }

    @ParameterizedTest
    @ValueSource(strings = {"REVOKED", "ACCEPTED"})
    void rejectsRevokedInvitationAndReplay(String status) {
        invitation.setInvitationStatus(status);
        stubOwner(SUBJECT);
        assertError(ISSUER, SUBJECT, ErrorCode.INVITATION_NOT_PENDING);
        verify(invitations, never()).updateStatus(any(), any());
    }

    @Test
    void rejectsExpiredInvitation() {
        invitation.setExpiresAt(OffsetDateTime.now().minusSeconds(1));
        stubOwner(SUBJECT);
        assertError(ISSUER, SUBJECT, ErrorCode.INVITATION_EXPIRED);
        verify(invitations, never()).updateStatus(any(), any());
    }

    @Test
    void revokesPendingInvitation() {
        when(invitations.findByIdForUpdate(invitationId)).thenReturn(Optional.of(invitation));
        assertThat(service.revokeInvitation(invitationId, UUID.randomUUID())).isTrue();
        verify(invitations).updateStatus(invitationId, "REVOKED");
    }

    private void stubOwner(String subject) {
        var principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpIssuer(ISSUER);
        principal.setIdpSubject(subject);
        principal.setProvisioningStatus("PENDING");
        principal.setVersion(4L);
        when(invitations.findByIdForUpdate(invitationId)).thenReturn(Optional.of(invitation));
        when(principals.findById(principalId)).thenReturn(principal);
    }

    private void assertError(String issuer, String subject, ErrorCode code) {
        assertThatThrownBy(() -> service.acceptInvitation(invitationId, issuer, subject))
                .isInstanceOfSatisfying(IdentityException.class, ex -> assertThat(ex.getErrorCode()).isEqualTo(code));
    }
}
