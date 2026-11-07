package com.hvduong.identity.application.dto.common;

/**
 * Re-export of {@link com.hvduong.identity.common.response.ApiResponse}.
 * New code should import from {@code com.hvduong.identity.common.response.ApiResponse} directly.
 */
public class ApiResponse<T> extends com.hvduong.identity.common.response.ApiResponse<T> {
    public ApiResponse() { super(); }
    public ApiResponse(boolean success, String code, String message, T data,
                       java.util.List<ValidationError> errors, String requestId,
                       java.time.OffsetDateTime timestamp) {
        super(success, code, message, data, errors, requestId, timestamp);
    }
}
