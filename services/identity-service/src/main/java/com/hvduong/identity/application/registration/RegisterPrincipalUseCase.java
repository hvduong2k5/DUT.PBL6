package com.hvduong.identity.application.registration;

import com.hvduong.identity.dto.request.RegistrationRequest;
import com.hvduong.identity.dto.response.RegistrationResponse;

public interface RegisterPrincipalUseCase {
    RegistrationResponse registerCustomer(RegistrationRequest request);
}
