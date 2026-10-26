package com.hvduong.identity.api.registration;

import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.api.registration.dto.RegistrationRequest;
import com.hvduong.identity.api.registration.dto.RegistrationResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;

@Tag(name = "Public Registration", description = "Đăng ký tài khoản Khách hàng công khai")
@RequestMapping("/api/v1/public/registration")
public interface RegistrationApi {

    @Operation(summary = "Đăng ký tài khoản Khách hàng", description = "Đăng ký tài khoản D2C với Idempotency Key, đồng bộ Keycloak và Outbox Event.")
    @PostMapping("/customer")
    ResponseEntity<ApiResponse<RegistrationResponse>> registerCustomer(@RequestBody RegistrationRequest request);
}

