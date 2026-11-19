package com.hvduong.identity.application.dto.response;

import com.hvduong.identity.infrastructure.session.BffSession;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class BffCallbackResult {
    private BffSession session;
    private String redirectUri;
}

