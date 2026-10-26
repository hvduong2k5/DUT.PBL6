package com.hvduong.identity.infrastructure.persistence.mybatis;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;

@SpringBootTest
@ActiveProfiles("test")
@Transactional
class IdentityPrincipalMapperIntegrationTest {

    @Autowired
    private IdentityPrincipalMapper identityPrincipalMapper;

    @Test
    void shouldInsertAndFindPrincipal() {
        // Arrange
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpIssuer("https://keycloak.omama.vn/realms/omama");
        row.setIdpSubject("test-subject-123");
        row.setPrincipalType("CUSTOMER");
        row.setProvisioningStatus("PENDING");
        row.setVersion(0L);

        // Act
        identityPrincipalMapper.insert(row);
        IdentityPrincipalRow found = identityPrincipalMapper.findById(principalId);

        // Assert
        assertThat(found).isNotNull();
        assertThat(found.getPrincipalId()).isEqualTo(principalId);
        assertThat(found.getIdpIssuer()).isEqualTo("https://keycloak.omama.vn/realms/omama");
        assertThat(found.getIdpSubject()).isEqualTo("test-subject-123");
        assertThat(found.getPrincipalType()).isEqualTo("CUSTOMER");
        assertThat(found.getProvisioningStatus()).isEqualTo("PENDING");
        assertThat(found.getVersion()).isEqualTo(0L);
        assertThat(found.getCreatedAt()).isNotNull();
        assertThat(found.getUpdatedAt()).isNotNull();
    }

    @Test
    void shouldUpdateProvisioningStatusWithOptimisticLocking() {
        // Arrange
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpIssuer("issuer");
        row.setIdpSubject(null);
        row.setPrincipalType("EMPLOYEE");
        row.setProvisioningStatus("PENDING");
        row.setVersion(0L);
        identityPrincipalMapper.insert(row);

        // Act
        int rowsUpdated = identityPrincipalMapper.updateProvisioningStatus(
            principalId, "ACTIVE", "subject-456", 0L
        );
        IdentityPrincipalRow updated = identityPrincipalMapper.findById(principalId);

        // Assert
        assertThat(rowsUpdated).isEqualTo(1);
        assertThat(updated.getProvisioningStatus()).isEqualTo("ACTIVE");
        assertThat(updated.getIdpSubject()).isEqualTo("subject-456");
        assertThat(updated.getVersion()).isEqualTo(1L);
    }
}
