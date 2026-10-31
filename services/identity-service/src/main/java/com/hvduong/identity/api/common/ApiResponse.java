package com.hvduong.identity.api.common;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.List;

/**
 * @deprecated Đã chuyển sang {@link com.hvduong.identity.dto.common.ApiResponse}.
 */
@Deprecated
@NoArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class ApiResponse<T> extends com.hvduong.identity.dto.common.ApiResponse<T> {

    public ApiResponse(boolean success, String code, String message, T data, List<ValidationError> errors, String requestId, OffsetDateTime timestamp) {
        super(success, code, message, data, errors, requestId, timestamp);
    }
}
