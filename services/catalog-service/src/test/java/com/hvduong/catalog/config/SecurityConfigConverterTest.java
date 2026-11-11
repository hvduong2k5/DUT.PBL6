package com.hvduong.catalog.config;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

import org.junit.jupiter.api.Test;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;

class SecurityConfigConverterTest {

  private final JwtAuthenticationConverter converter =
      new SecurityConfig().jwtAuthenticationConverter();

  private static Jwt.Builder baseJwt() {
    return Jwt.withTokenValue("token")
        .header("alg", "RS256")
        .issuer("https://identity.test/realm")
        .subject("staff");
  }

  private Set<String> authoritiesOf(Jwt jwt) {
    return converter.convert(jwt).getAuthorities().stream()
        .map(GrantedAuthority::getAuthority)
        .collect(Collectors.toSet());
  }

  @Test
  void realmRolesStillMapToRoleAuthorities() {
    Jwt jwt = baseJwt().claim("realm_access", Map.of("roles", List.of("ADMIN"))).build();
    assertThat(authoritiesOf(jwt)).containsExactly("ROLE_ADMIN");
  }

  @Test
  void grantsMapToPermissionCodesAcrossScopes() {
    Jwt jwt =
        baseJwt()
            .claim(
                "permission_grants",
                List.of(
                    "CATALOG_VIEW#GLOBAL",
                    "CATALOG_MANAGE#WAREHOUSE#WH-HCM-01",
                    "CATALOG_APPROVE#DEPARTMENT#SALES"))
            .build();
    assertThat(authoritiesOf(jwt))
        .containsExactlyInAnyOrder("CATALOG_VIEW", "CATALOG_MANAGE", "CATALOG_APPROVE");
  }

  @Test
  void flatPermissionsClaimIsNoLongerHonored() {
    Jwt jwt = baseJwt().claim("permissions", List.of("CATALOG_VIEW", "CATALOG_MANAGE")).build();
    assertThat(authoritiesOf(jwt)).isEmpty();
  }

  @Test
  void missingOrEmptyGrantsClaimYieldsNoPermissionAuthorities() {
    assertThat(authoritiesOf(baseJwt().build())).isEmpty();
    assertThat(authoritiesOf(baseJwt().claim("permission_grants", List.of()).build())).isEmpty();
  }

  @Test
  void malformedGrantsFailClosed() {
    List<String> malformed =
        List.of(
            "CATALOG_VIEW",
            "CATALOG_VIEW#GLOBAL#EXTRA",
            "CATALOG_VIEW#STORE#S1",
            "#GLOBAL",
            "CATALOG_VIEW#WAREHOUSE#",
            "CATALOG_VIEW#WAREHOUSE#A#B",
            "1BAD#GLOBAL");
    for (String grant : malformed) {
      Jwt jwt = baseJwt().claim("permission_grants", List.of(grant)).build();
      assertThatThrownBy(() -> converter.convert(jwt))
          .isInstanceOf(IllegalArgumentException.class)
          .hasMessage("INVALID_STAFF_AUTHORIZATION");
    }
  }

  @Test
  void parserAcceptsTheSettledGrantGrammar() {
    assertThat(SecurityConfig.parseGrantPermissionCode("ORDER_VIEW#GLOBAL"))
        .isEqualTo("ORDER_VIEW");
    assertThat(SecurityConfig.parseGrantPermissionCode("ORDER_VIEW#WAREHOUSE#WH-01"))
        .isEqualTo("ORDER_VIEW");
    assertThat(SecurityConfig.parseGrantPermissionCode("EMPLOYEE_PROFILE_VIEW#DEPARTMENT#HR"))
        .isEqualTo("EMPLOYEE_PROFILE_VIEW");
  }
}
