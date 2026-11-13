package com.hvduong.identity.security;

import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.PrincipalType;
import com.hvduong.identity.common.enums.ProvisioningStatus;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;

/** Account type is resolved from Identity's own record, independently of role names. */
@Component
@RequiredArgsConstructor
public class PrincipalAccess {
    private final IdentityPrincipalMapper principals;
    public boolean employee(Authentication authentication) {
        if (!(authentication instanceof JwtAuthenticationToken token) || !authentication.isAuthenticated()) return false;
        var jwt = token.getToken();
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        return principal != null
                && PrincipalType.EMPLOYEE.matches(principal.getPrincipalType())
                && ProvisioningStatus.ACTIVE.matches(principal.getProvisioningStatus())
                && principal.getPrincipalId().toString().equals(jwt.getClaimAsString(IdentityConstants.CLAIM_USER_ID));
    }
}
