package com.hvduong.identity.security;

import lombok.RequiredArgsConstructor;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpMethod;
import org.springframework.security.authorization.AuthorizationDecision;
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
                .requestMatchers(HttpMethod.GET, "/api/v1/admin/roles")
                    .access((authentication, context) -> new AuthorizationDecision(
                            isRolesReader(authentication.get()) || principalAccess.employee(authentication.get())))
                .requestMatchers("/api/v1/admin/**").access((authentication,context) ->
                        new org.springframework.security.authorization.AuthorizationDecision(principalAccess.employee(authentication.get())))
                .requestMatchers("/actuator/**").permitAll()
                .requestMatchers("/swagger-ui/**", "/v3/api-docs/**", "/swagger-ui.html").permitAll()
                .anyRequest().access((authentication, context) -> {
                    var actor = authentication.get();
                    return new AuthorizationDecision(actor.isAuthenticated()
                            && !(actor instanceof org.springframework.security.authentication.AnonymousAuthenticationToken)
                            && !isRolesReader(actor));
                })
            )
            .oauth2ResourceServer(oauth2 -> oauth2.jwt(jwt -> 
                jwt.jwtAuthenticationConverter(jwtAuthenticationConverter())
            ))
            .addFilterBefore(bffSessionAuthFilter, BearerTokenAuthenticationFilter.class);

        return http.build();
    }

    private static boolean isRolesReader(org.springframework.security.core.Authentication authentication) {
        return authentication.getAuthorities().stream()
                .anyMatch(authority -> JwtAudienceConfig.ROLES_READER_AUTHORITY.equals(authority.getAuthority()));
    }

    @Bean
    public JwtAuthenticationConverter jwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(jwt -> {
            if (JwtAudienceConfig.isRolesReader(jwt)) {
                return Set.of(new SimpleGrantedAuthority(JwtAudienceConfig.ROLES_READER_AUTHORITY));
            }
            Set<GrantedAuthority> authorities = new HashSet<>();

            // Lấy Permissions scoped từ claim 'permission_grants' trong JWT
            // Mỗi phần tử lưu dưới dạng chuỗi: permission#scope_type[#scope_id]
            // Sinh cả authority có scope (PERMISSION#SCOPE_TYPE#SCOPE_ID) lẫn authority mã quyền gốc (PERMISSION)
            // để các @PreAuthorize("hasAuthority('...')") tự xử lý trực tiếp mà không cần claim 'permissions' rời.
            @SuppressWarnings("unchecked")
            List<String> grantStrings = (List<String>) jwt.getClaim("permission_grants");
            if (grantStrings != null) {
                for (String gs : grantStrings) {
                    if (gs != null && !gs.isBlank()) {
                        String trimmed = gs.trim();
                        if (trimmed.equals(JwtAudienceConfig.ROLES_READER_AUTHORITY)
                                || trimmed.startsWith(JwtAudienceConfig.ROLES_READER_AUTHORITY + "#")) continue;
                        authorities.add(new SimpleGrantedAuthority(trimmed));
                        int sepIdx = trimmed.indexOf('#');
                        if (sepIdx > 0) {
                            String basePermission = trimmed.substring(0, sepIdx).trim();
                            if (!basePermission.isEmpty()) {
                                authorities.add(new SimpleGrantedAuthority(basePermission));
                            }
                        }
                    }
                }
            }

            return authorities;
        });
        return converter;
    }
}
