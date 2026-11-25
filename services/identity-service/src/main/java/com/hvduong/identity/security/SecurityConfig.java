package com.hvduong.identity.security;

import com.hvduong.identity.api.account.impl.EmployeeInvitationAcceptanceController;
import lombok.RequiredArgsConstructor;
import jakarta.servlet.http.HttpServletResponse;
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
import org.springframework.security.oauth2.server.resource.web.BearerTokenAuthenticationEntryPoint;
import org.springframework.security.oauth2.server.resource.web.access.BearerTokenAccessDeniedHandler;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.security.web.SecurityFilterChain;

import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.io.IOException;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
@RequiredArgsConstructor
public class SecurityConfig {

    private final BffSessionAuthFilter bffSessionAuthFilter;
    private final PrincipalAccess principalAccess;

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        var bearerEntry = new BearerTokenAuthenticationEntryPoint();
        var bearerDenied = new BearerTokenAccessDeniedHandler();
        AuthenticationEntryPoint entry = (request, response, cause) -> {
            if (request.getRequestURI().startsWith("/api/v1/admin/shipping-api/")) shippingSecurityError(response, 401);
            else bearerEntry.commence(request, response, cause);
        };
        http
            .csrf(AbstractHttpConfigurer::disable)
            .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .exceptionHandling(errors -> errors.defaultAuthenticationEntryPointFor(entry,
                    request -> request.getRequestURI().startsWith("/api/v1/admin/shipping-api/")))
            .authorizeHttpRequests(authorize -> authorize
                .requestMatchers(HttpMethod.POST, EmployeeInvitationAcceptanceController.ACCEPT_PATH).authenticated()
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
            .oauth2ResourceServer(oauth2 -> oauth2.authenticationEntryPoint(entry).accessDeniedHandler((request, response, cause) -> {
                if (request.getRequestURI().startsWith("/api/v1/admin/shipping-api/")) shippingSecurityError(response, 403);
                else bearerDenied.handle(request, response, cause);
            }).jwt(jwt ->
                jwt.jwtAuthenticationConverter(jwtAuthenticationConverter())
            ))
            .addFilterBefore(bffSessionAuthFilter, BearerTokenAuthenticationFilter.class);

        return http.build();
    }

    private static void shippingSecurityError(HttpServletResponse response, int status) throws IOException {
        response.setStatus(status);
        response.setContentType("application/json");
        response.setHeader("Cache-Control", "private, no-store");
        if (status == 401) response.setHeader("WWW-Authenticate", "Bearer");
        response.getWriter().write("{\"code\":\"" + (status == 401 ? "UNAUTHORIZED" : "FORBIDDEN")
                + "\",\"message\":\"Employee Shipping authorization required.\",\"requestId\":\"" + UUID.randomUUID() + "\"}");
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
