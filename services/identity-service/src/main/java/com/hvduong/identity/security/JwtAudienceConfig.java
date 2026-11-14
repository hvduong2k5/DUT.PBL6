package com.hvduong.identity.security;

import java.time.Duration;
import java.util.List;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.oauth2.core.OAuth2Error;
import org.springframework.security.oauth2.core.OAuth2TokenValidator;
import org.springframework.security.oauth2.core.OAuth2TokenValidatorResult;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtIssuerValidator;
import org.springframework.security.oauth2.jwt.JwtTimestampValidator;
import org.springframework.security.oauth2.jwt.JwtValidators;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;

@Configuration
public class JwtAudienceConfig {
    static final String ROLES_AUDIENCE = "identity-roles-api";
    static final String ROLES_READER_CLIENT = "identity-service-admin";
    static final String ROLES_READER_AUTHORITY = "IDENTITY_SERVICE_ROLE_LIST";

    static boolean isRolesReader(Jwt jwt) {
        return ROLES_READER_CLIENT.equals(jwt.getClaimAsString("azp"))
                && jwt.getAudience() != null && jwt.getAudience().contains(ROLES_AUDIENCE);
    }

    @Bean
    JwtDecoder jwtDecoder(
            @Value("${spring.security.oauth2.resourceserver.jwt.issuer-uri}") String issuer,
            @Value("${spring.security.oauth2.resourceserver.jwt.jwk-set-uri:}") String jwks,
            @Value("${IDENTITY_OIDC_AUDIENCE:omama-api}") String audience,
            @Value("${app.auth.jwt-clock-skew}") Duration clockSkew) {
        if (audience.isBlank()) throw new IllegalArgumentException("Identity audience is required");
        var decoder = NimbusJwtDecoder.withJwkSetUri(jwks.isBlank()
                ? issuer.replaceAll("/$", "") + "/protocol/openid-connect/certs" : jwks).build();
        decoder.setJwtValidator(tokenValidator(issuer, audience, clockSkew));
        return decoder;
    }

    static OAuth2TokenValidator<Jwt> tokenValidator(String issuer, String audience, Duration clockSkew) {
        return JwtValidators.createDefaultWithValidators(
                new JwtTimestampValidator(clockSkew), new JwtIssuerValidator(issuer), audienceValidator(audience));
    }

    static OAuth2TokenValidator<Jwt> audienceValidator(String audience) {
        return jwt -> {
            if (isRolesReader(jwt)) return OAuth2TokenValidatorResult.success();
            List<String> audiences = jwt.getAudience();
            return audiences != null && audiences.contains(audience)
                    ? OAuth2TokenValidatorResult.success()
                    : OAuth2TokenValidatorResult.failure(new OAuth2Error("invalid_token", "Invalid Identity audience", null));
        };
    }
}
