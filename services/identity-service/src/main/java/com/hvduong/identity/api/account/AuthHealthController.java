package com.hvduong.identity.api.account;

import com.hvduong.identity.common.response.ApiResponse;
import java.util.Map;
import java.util.UUID;
import org.springframework.boot.actuate.health.HealthEndpoint;
import org.springframework.boot.actuate.health.HttpCodeStatusMapper;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuthHealthController {
    private final HealthEndpoint healthEndpoint;

    public AuthHealthController(HealthEndpoint healthEndpoint) {
        this.healthEndpoint = healthEndpoint;
    }

    @GetMapping("/api/v1/public/auth/health")
    public ResponseEntity<?> health() {
        var status = healthEndpoint.health().getStatus();
        int httpStatus = HttpCodeStatusMapper.DEFAULT.getStatusCode(status);
        if (httpStatus == 503) {
            return ResponseEntity.status(httpStatus).body(ApiResponse.error(
                    "IDENTITY_UNAVAILABLE", "Dịch vụ tài khoản tạm thời không khả dụng.",
                    "req-" + UUID.randomUUID()));
        }
        return ResponseEntity.status(httpStatus).body(Map.of("status", status.getCode()));
    }
}
