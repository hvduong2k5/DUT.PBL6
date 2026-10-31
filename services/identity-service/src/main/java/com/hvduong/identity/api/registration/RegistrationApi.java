package com.hvduong.identity.api.registration;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.RegistrationRequest;
import com.hvduong.identity.dto.response.RegistrationResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;

@Tag(name = "Public Registration", description = "Đăng ký tài khoản Khách hàng công khai theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping({"/api/v1/public/customer-registrations", "/api/v1/public/registrations/customers", "/api/v1/public/registration"})
public interface RegistrationApi {

    @Operation(summary = "Đăng ký tài khoản Khách hàng [Chuẩn RESTful 3.3.1]", description = "Đăng ký tài khoản D2C với Idempotency Key, đồng bộ Keycloak và Outbox Event qua POST /api/v1/public/customer-registrations.")
    @PostMapping({"", "/customer", "/customers"})
    ResponseEntity<ApiResponse<RegistrationResponse>> registerCustomer(@RequestBody RegistrationRequest request);
}
