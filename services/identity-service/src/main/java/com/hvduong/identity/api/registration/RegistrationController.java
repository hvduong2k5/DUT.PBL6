package com.hvduong.identity.api.registration;

import com.hvduong.identity.api.registration.dto.RegistrationRequest;
import com.hvduong.identity.api.registration.dto.RegistrationResponse;
import com.hvduong.identity.application.registration.RegisterPrincipalUseCase;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/public/registration")
@RequiredArgsConstructor
public class RegistrationController {

    private final RegisterPrincipalUseCase registerPrincipalUseCase;

    @PostMapping("/customer")
    public ResponseEntity<RegistrationResponse> registerCustomer(@RequestBody RegistrationRequest request) {
        RegistrationResponse response = registerPrincipalUseCase.registerCustomer(request);
        
        if ("CONFLICT".equals(response.getStatus())) {
            return ResponseEntity.status(409).body(response);
        } else if ("FAILED".equals(response.getStatus())) {
            return ResponseEntity.status(500).body(response);
        }
        
        return ResponseEntity.ok(response);
    }
}
