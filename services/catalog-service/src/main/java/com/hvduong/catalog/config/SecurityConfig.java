package com.hvduong.catalog.config;

import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;
import org.springframework.security.web.SecurityFilterChain;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

  @Bean
  public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
    http.csrf(AbstractHttpConfigurer::disable)
        .sessionManagement(
            session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
        .authorizeHttpRequests(
            authorize ->
                authorize
                    // Public REST APIs
                    .requestMatchers("/api/v1/categories", "/api/v1/categories/**")
                    .permitAll()
                    .requestMatchers("/api/v1/products", "/api/v1/products/**")
                    .permitAll()
                    .requestMatchers("/api/v1/search", "/api/v1/search/**")
                    .permitAll()
                    .requestMatchers("/api/v1/internal/**")
                    .permitAll()
                    // Infrastructure & Documentation
                    .requestMatchers("/actuator/**")
                    .permitAll()
                    .requestMatchers("/swagger-ui/**", "/v3/api-docs/**", "/swagger-ui.html")
                    .permitAll()
                    // Admin Catalog APIs require authentication
                    .requestMatchers("/api/v1/admin/**")
                    .authenticated()
                    .anyRequest()
                    .authenticated())
        .oauth2ResourceServer(
            oauth2 ->
                oauth2.jwt(jwt -> jwt.jwtAuthenticationConverter(jwtAuthenticationConverter())));

    return http.build();
  }

  @Bean
  public JwtAuthenticationConverter jwtAuthenticationConverter() {
    JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
    converter.setJwtGrantedAuthoritiesConverter(
        jwt -> {
          Set<GrantedAuthority> authorities = new HashSet<>();

          // 1. Map Keycloak Realm Roles (e.g., ADMIN -> ROLE_ADMIN)
          Map<String, Object> realmAccess = jwt.getClaim("realm_access");
          if (realmAccess != null && realmAccess.containsKey("roles")) {
            @SuppressWarnings("unchecked")
            List<String> kcRoles = (List<String>) realmAccess.get("roles");
            for (String r : kcRoles) {
              authorities.add(new SimpleGrantedAuthority("ROLE_" + r));
            }
          }

          // 2. Map scoped permission grants, the Identity sync contract
          //    (e.g., "CATALOG_VIEW#GLOBAL",
          //    "CATALOG_MANAGE#WAREHOUSE#WH-HCM-01"). The authority is the
          //    permission code alone: the flat "permissions" claim this
          //    replaces carried no scope, so code-only mapping preserves
          //    the previous authorization semantics.
          List<String> grants = jwt.getClaimAsStringList("permission_grants");
          if (grants != null) {
            for (String grant : grants) {
              authorities.add(new SimpleGrantedAuthority(parseGrantPermissionCode(grant)));
            }
          }

          return authorities;
        });
    return converter;
  }

  private static final java.util.regex.Pattern PERMISSION_CODE =
      java.util.regex.Pattern.compile("^[A-Za-z][A-Za-z0-9_.:-]{0,159}$");

  /**
   * Extracts the permission code from one {@code permission_grants} entry:
   * {@code PERMISSION#SCOPE_TYPE} for GLOBAL grants,
   * {@code PERMISSION#SCOPE_TYPE#SCOPE_ID} for WAREHOUSE and DEPARTMENT
   * grants. The grammar mirrors the Order/Profile staff-claims parser so
   * every consumer reads the Identity contract the same way; a malformed
   * entry fails the conversion closed instead of silently widening or
   * narrowing access.
   */
  static String parseGrantPermissionCode(String grant) {
    String[] parts = grant.split("#", -1);
    if (parts.length < 2 || parts.length > 3 || !PERMISSION_CODE.matcher(parts[0]).matches()) {
      throw new IllegalArgumentException("INVALID_STAFF_AUTHORIZATION");
    }
    switch (parts[1]) {
      case "GLOBAL" -> {
        if (parts.length != 2) {
          throw new IllegalArgumentException("INVALID_STAFF_AUTHORIZATION");
        }
      }
      case "WAREHOUSE", "DEPARTMENT" -> {
        if (parts.length != 3 || !isValidScopeId(parts[2])) {
          throw new IllegalArgumentException("INVALID_STAFF_AUTHORIZATION");
        }
      }
      default -> throw new IllegalArgumentException("INVALID_STAFF_AUTHORIZATION");
    }
    return parts[0];
  }

  private static boolean isValidScopeId(String value) {
    if (value.isEmpty() || value.length() > 160 || !value.trim().equals(value)) {
      return false;
    }
    for (int i = 0; i < value.length(); i++) {
      if (Character.isISOControl(value.charAt(i))) {
        return false;
      }
    }
    return true;
  }
}
