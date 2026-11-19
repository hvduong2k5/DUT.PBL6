package com.hvduong.identity.application.registration;

import com.hvduong.identity.application.dto.request.RegistrationRequest;
import com.hvduong.identity.application.dto.response.RegistrationResponse;

public interface RegisterPrincipalUseCase {
    RegistrationResponse registerCustomer(RegistrationRequest request);
}
