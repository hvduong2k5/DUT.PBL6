package com.hvduong.identity.security;

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import org.junit.jupiter.api.Test;
import org.springframework.security.oauth2.jwt.Jwt;
import static org.junit.jupiter.api.Assertions.*;

class JwtAudienceConfigTest {
    @Test
    void requiresIntendedApiAudienceIncludingMultiAudienceTokens() {
        var validator = JwtAudienceConfig.audienceValidator("omama-api");
        var builder = Jwt.withTokenValue("test").header("alg", "RS256").subject("subject");
        assertFalse(validator.validate(builder.audience(List.of("order-api", "omama-api")).build()).hasErrors());
        assertTrue(validator.validate(builder.audience(List.of("other-service")).build()).hasErrors());
        assertTrue(validator.validate(builder.claims(claims -> claims.remove("aud")).build()).hasErrors());
    }

    @Test
    void enforcesThirtySecondWindowForExpiryAndNotBefore() {
        String issuer = "https://issuer.example";
        var validator = JwtAudienceConfig.tokenValidator(issuer, "omama-api", Duration.ofSeconds(30));
        var now = Instant.now();
        var builder = Jwt.withTokenValue("test").header("alg", "RS256").subject("subject")
                .issuer(issuer).audience(List.of("omama-api")).issuedAt(now.minusSeconds(120));
        assertFalse(validator.validate(builder.expiresAt(now.minusSeconds(29)).build()).hasErrors());
        assertTrue(validator.validate(builder.expiresAt(now.minusSeconds(31)).build()).hasErrors());
        assertFalse(validator.validate(builder.expiresAt(now.plusSeconds(120)).notBefore(now.plusSeconds(29)).build()).hasErrors());
        assertTrue(validator.validate(builder.notBefore(now.plusSeconds(31)).build()).hasErrors());
    }
}
