package com.hvduong.catalog.config;

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

import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf(AbstractHttpConfigurer::disable)
            .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(authorize -> authorize
                // Public REST APIs
                .requestMatchers("/api/v1/categories/**").permitAll()
                .requestMatchers("/api/v1/products/**").permitAll()
                .requestMatchers("/api/v1/search/**").permitAll()
                .requestMatchers("/api/v1/internal/**").permitAll()
                // Infrastructure & Documentation
                .requestMatchers("/actuator/**").permitAll()
                .requestMatchers("/swagger-ui/**", "/v3/api-docs/**", "/swagger-ui.html").permitAll()
                // Admin Catalog APIs require authentication
                .requestMatchers("/api/v1/admin/**").authenticated()
                .anyRequest().authenticated()
            )
            .oauth2ResourceServer(oauth2 -> oauth2.jwt(jwt ->
                jwt.jwtAuthenticationConverter(jwtAuthenticationConverter())
            ));

        return http.build();
    }

    @Bean
    public JwtAuthenticationConverter jwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(jwt -> {
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

            // 2. Map Stateless Permissions Claim (e.g., CATALOG_VIEW, CATALOG_MANAGE)
            List<String> perms = jwt.getClaimAsStringList("permissions");
            if (perms != null) {
                for (String p : perms) {
                    authorities.add(new SimpleGrantedAuthority(p));
                }
            }

            return authorities;
        });
        return converter;
    }
}

