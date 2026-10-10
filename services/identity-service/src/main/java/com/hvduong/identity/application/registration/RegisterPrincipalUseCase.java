package com.hvduong.identity.application.registration;

import com.hvduong.identity.api.registration.dto.RegistrationRequest;
import com.hvduong.identity.api.registration.dto.RegistrationResponse;

public interface RegisterPrincipalUseCase {
    RegistrationResponse registerCustomer(RegistrationRequest request);
}
