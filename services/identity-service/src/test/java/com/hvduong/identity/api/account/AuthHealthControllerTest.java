package com.hvduong.identity.api.account;

import org.junit.jupiter.api.Test;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthEndpoint;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class AuthHealthControllerTest {
    @Test
    void mirrorsActuatorStatusWithoutExposingDetails() throws Exception {
        var health = mock(HealthEndpoint.class);
        var mvc = MockMvcBuilders.standaloneSetup(new AuthHealthController(health)).build();

        when(health.health()).thenReturn(Health.up().build());
        mvc.perform(get("/api/v1/public/auth/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("UP"));

        when(health.health()).thenReturn(Health.down().withDetail("database", "internal").build());
        mvc.perform(get("/api/v1/public/auth/health"))
                .andExpect(status().isServiceUnavailable())
                .andExpect(jsonPath("$.code").value("IDENTITY_UNAVAILABLE"))
                .andExpect(jsonPath("$.requestId").isString())
                .andExpect(jsonPath("$.details").doesNotExist());
    }
}
