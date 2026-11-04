package com.hvduong.identity.security;
import java.util.UUID;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.security.access.AccessDeniedException;
public final class VerifiedActor {
    private VerifiedActor() {}
    public static UUID id() {
        var authentication=SecurityContextHolder.getContext().getAuthentication();
        if (!(authentication instanceof JwtAuthenticationToken token)) throw new AccessDeniedException("Verified actor required");
        try {return UUID.fromString(token.getToken().getClaimAsString("user_id"));}
        catch(IllegalArgumentException | NullPointerException error){throw new AccessDeniedException("Invalid actor");}
    }
}
