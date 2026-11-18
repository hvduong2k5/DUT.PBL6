package com.hvduong.identity.security;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.Test;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import java.util.*;
import static org.mockito.Mockito.*;
import static org.assertj.core.api.Assertions.assertThat;

class PrincipalAccessTest {
    @Test void administratorRoleDoesNotConvertCustomerIntoEmployee() {
        var principals=mock(IdentityPrincipalMapper.class);
        var access=new PrincipalAccess(principals);
        UUID id=UUID.randomUUID();
        var jwt=Jwt.withTokenValue("fixture").header("alg","RS256").issuer("https://identity.test/realms/test").subject("external-subject").claim("user_id",id.toString()).build();
        var authentication=new JwtAuthenticationToken(jwt,List.of(new SimpleGrantedAuthority("ROLE_ADMIN")));
        var row=new IdentityPrincipalRow();row.setPrincipalId(id);row.setPrincipalType("CUSTOMER");row.setProvisioningStatus("ACTIVE");
        when(principals.findByIssuerAndSubject(jwt.getIssuer().toString(),jwt.getSubject())).thenReturn(row);
        assertThat(access.employee(authentication)).isFalse();
        row.setPrincipalType("EMPLOYEE");assertThat(access.employee(authentication)).isTrue();
        row.setProvisioningStatus("SUSPENDED");assertThat(access.employee(authentication)).isFalse();
        row.setProvisioningStatus("ACTIVE");row.setPrincipalId(UUID.randomUUID());assertThat(access.employee(authentication)).isFalse();
        assertThat(access.employee(null)).isFalse();
    }

    @Test
    void limitedGrantTokenRequiresAnActiveMatchingEmployeeRecord() {
        var principals = mock(IdentityPrincipalMapper.class);
        var access = new PrincipalAccess(principals);
        UUID id = UUID.randomUUID();
        String issuer = "https://identity.test/realms/test";
        var jwt = Jwt.withTokenValue("fixture").header("alg", "RS256")
                .issuer(issuer).subject("external-subject")
                .claim("user_id", id.toString()).claim("principal_type", "EMPLOYEE")
                .claim("permission_grants", List.of("ROLE_VIEW#GLOBAL")).build();
        var auth = new SecurityConfig(null, access).jwtAuthenticationConverter().convert(jwt);
        assertThat(auth.getAuthorities()).extracting("authority").contains("ROLE_VIEW");
        assertThat(access.employee(auth)).isFalse();

        var row = new IdentityPrincipalRow();
        row.setPrincipalId(id);
        row.setPrincipalType("EMPLOYEE");
        row.setProvisioningStatus("ACTIVE");
        when(principals.findByIssuerAndSubject(issuer, "external-subject")).thenReturn(row);
        assertThat(access.employee(auth)).isTrue();
    }
}
