package com.hvduong.identity.api.registration;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.RegistrationRequest;
import com.hvduong.identity.dto.response.RegistrationResponse;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

@Tag(name = "Public Customer Registration", description = "Đăng ký tài khoản Khách hàng công khai theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping("/api/v1/public/customer-registrations")
public interface RegistrationApi {

    @Operation(summary = "Đăng ký tài khoản Khách hàng [Chuẩn RESTful 3.3.1]", description = "Đăng ký tài khoản D2C với Idempotency Key, đồng bộ Keycloak và Outbox Event qua POST /api/v1/public/customer-registrations.")
    @PostMapping
    ResponseEntity<ApiResponse<RegistrationResponse>> registerCustomer(@jakarta.validation.Valid @RequestBody RegistrationRequest request);
}
