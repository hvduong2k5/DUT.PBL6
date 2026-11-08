package com.hvduong.identity.security;

import lombok.RequiredArgsConstructor;
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
import org.springframework.security.oauth2.server.resource.web.authentication.BearerTokenAuthenticationFilter;
import org.springframework.security.web.SecurityFilterChain;

import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
@RequiredArgsConstructor
public class SecurityConfig {

    private final BffSessionAuthFilter bffSessionAuthFilter;
    private final PrincipalAccess principalAccess;

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf(AbstractHttpConfigurer::disable)
            .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(authorize -> authorize
                .requestMatchers("/api/v1/public/**").permitAll()
                .requestMatchers("/api/v1/admin/**").access((authentication,context) ->
                        new org.springframework.security.authorization.AuthorizationDecision(principalAccess.employee(authentication.get())))
                .requestMatchers("/actuator/**").permitAll()
                .requestMatchers("/swagger-ui/**", "/v3/api-docs/**", "/swagger-ui.html").permitAll()
                .anyRequest().authenticated()
            )
            .oauth2ResourceServer(oauth2 -> oauth2.jwt(jwt -> 
                jwt.jwtAuthenticationConverter(jwtAuthenticationConverter())
            ))
            .addFilterBefore(bffSessionAuthFilter, BearerTokenAuthenticationFilter.class);

        return http.build();
    }

    @Bean
    public JwtAuthenticationConverter jwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(jwt -> {
            Set<GrantedAuthority> authorities = new HashSet<>();

            // 1. Lấy Realm Roles từ Keycloak (ví dụ: ADMIN -> ROLE_ADMIN)
            Map<String, Object> realmAccess = jwt.getClaim("realm_access");
            if (realmAccess != null && realmAccess.containsKey("roles")) {
                @SuppressWarnings("unchecked")
                List<String> kcRoles = (List<String>) realmAccess.get("roles");
                for (String r : kcRoles) {
                    authorities.add(new SimpleGrantedAuthority("ROLE_" + r));
                }
            }

            // 2. Lấy Permissions scoped từ claim 'permission_grants' trong JWT (Consumer mới)
            // Mỗi phần tử lưu dưới dạng chuỗi: permission#scope_type#scope_id
            // Sinh authority dạng: PERMISSION#SCOPE_TYPE#SCOPE_ID (scope_id có thể null)
            @SuppressWarnings("unchecked")
            List<String> grantStrings = (List<String>) jwt.getClaim("permission_grants");
            if (grantStrings != null) {
                for (String gs : grantStrings) {
                    if (gs != null && !gs.isBlank()) {
                        authorities.add(new SimpleGrantedAuthority(gs));
                    }
                }
            }

            // 3. Flat permissions (consumer cũ - Catalog, Identity tự kiểm) giữ trong giai đoạn chuyển tiếp
            List<String> flatPerms = jwt.getClaimAsStringList("permissions");
            if (flatPerms != null) {
                for (String p : flatPerms) {
                    if (p != null && !p.isBlank()) {
                        authorities.add(new SimpleGrantedAuthority(p));
                    }
                }
            }

            return authorities;
        });
        return converter;
    }
}
