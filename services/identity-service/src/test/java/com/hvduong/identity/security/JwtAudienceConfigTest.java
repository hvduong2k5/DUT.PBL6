package com.hvduong.identity.security;

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
}
