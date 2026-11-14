package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.MarketingApiProxyController;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verifyNoInteractions;

class MarketingApiProxyControllerTest {
    @Test
    void rejectsRequestsWithoutAnAuthenticatedStaffToken() throws Exception {
        var principals = mock(IdentityPrincipalMapper.class);
        var controller = new MarketingApiProxyController(principals);
        var request = new MockHttpServletRequest("POST", "/api/v1/admin/marketing-api/campaigns");

        assertThat(controller.forward(null, request).getStatusCode().value()).isEqualTo(401);
        verifyNoInteractions(principals);
    }
}
