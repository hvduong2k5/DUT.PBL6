package com.hvduong.identity.api.registration;

import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.api.registration.dto.RegistrationRequest;
import com.hvduong.identity.api.registration.dto.RegistrationResponse;
import com.hvduong.identity.application.registration.RegisterPrincipalUseCase;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/public/registration")
@RequiredArgsConstructor
public class RegistrationController {

    private final RegisterPrincipalUseCase registerPrincipalUseCase;

    @PostMapping("/customer")
    public ResponseEntity<ApiResponse<RegistrationResponse>> registerCustomer(@RequestBody RegistrationRequest request) {
        RegistrationResponse response = registerPrincipalUseCase.registerCustomer(request);
        
        if ("CONFLICT".equals(response.getStatus())) {
            return ResponseEntity.status(HttpStatus.CONFLICT)
                    .body(ApiResponse.error("STATE_CONFLICT", response.getMessage()));
        } else if ("FAILED".equals(response.getStatus())) {
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                    .body(ApiResponse.error("REGISTRATION_FAILED", response.getMessage()));
        }
        
        return ResponseEntity.ok(ApiResponse.success(response.getMessage(), response));
    }
}
