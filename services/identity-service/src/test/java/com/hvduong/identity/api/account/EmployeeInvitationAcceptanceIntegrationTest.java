package com.hvduong.identity.api.account;

import com.hvduong.identity.application.account.EmployeeInvitationService;
import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.Test;
import org.keycloak.representations.idm.UserRepresentation;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest(properties = "grpc.server.port=-1")
@ActiveProfiles("test")
@AutoConfigureMockMvc
class EmployeeInvitationAcceptanceIntegrationTest {
    private static final String ACCEPT = "/api/v1/public/auth/employee-invitations/{invitationId}/accept";

    @Autowired private MockMvc mvc;
    @Autowired private EmployeeInvitationService invitations;
    @Autowired private IdentityPrincipalMapper principals;
    @Autowired private JdbcTemplate jdbc;
    @MockBean private KeycloakAdminClient keycloak;
    @MockBean private JwtDecoder jwtDecoder;

    @Test
    void requiresTheBoundBearerAndActivatesExactlyOnce() throws Exception {
        String subject = UUID.randomUUID().toString();
        when(keycloak.getUserById(subject)).thenReturn(Optional.of(new UserRepresentation()));
        var invitation = invitations.inviteEmployee(null, 7, subject);
        var principal = principals.findById(invitation.getPrincipalId());
        UUID invitationId = invitation.getInvitationId();
        try {
            mvc.perform(post(ACCEPT, invitationId)).andExpect(status().isUnauthorized());
            mvc.perform(post(ACCEPT, invitationId).with(jwt().jwt(jwt ->
                    jwt.issuer(principal.getIdpIssuer()).subject("other-subject"))))
                    .andExpect(status().isForbidden()).andExpect(jsonPath("$.code").value("PERMISSION_DENIED"));
            mvc.perform(post(ACCEPT, invitationId).with(jwt().jwt(jwt ->
                    jwt.issuer(principal.getIdpIssuer()).subject(subject))))
                    .andExpect(status().isOk()).andExpect(jsonPath("$.success").value(true));
            assertThat(principals.findById(invitation.getPrincipalId()).getProvisioningStatus()).isEqualTo("ACTIVE");
            verify(keycloak).setUserAttribute(subject, IdentityConstants.ATTR_APP_USER_ID, invitation.getPrincipalId().toString());
            verify(keycloak).updateUserPermissionsWithGrants(subject, List.of(), "EMPLOYEE");
            mvc.perform(post(ACCEPT, invitationId).with(jwt().jwt(jwt ->
                    jwt.issuer(principal.getIdpIssuer()).subject(subject))))
                    .andExpect(status().isConflict()).andExpect(jsonPath("$.code").value("INVITATION_NOT_PENDING"));
        } finally {
            jdbc.update("DELETE FROM identity_audit_log WHERE actor_principal_id = ? OR target_id = ?", principal.getPrincipalId(), invitationId.toString());
            jdbc.update("DELETE FROM employee_invitations WHERE invitation_id = ?", invitationId);
            jdbc.update("DELETE FROM identity_principals WHERE principal_id = ?", principal.getPrincipalId());
        }
    }
}
