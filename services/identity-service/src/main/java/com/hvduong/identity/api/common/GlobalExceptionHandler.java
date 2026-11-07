package com.hvduong.identity.api.common;

import com.hvduong.identity.dto.common.ApiResponse;

/**
 * @deprecated Moved to {@link com.hvduong.identity.common.handler.GlobalExceptionHandler}.
 * Kept for backward compatibility with tests that reference this FQCN.
 */
@Deprecated
public class GlobalExceptionHandler extends com.hvduong.identity.common.handler.GlobalExceptionHandler {
    // Delegates all logic to the canonical handler in common.handler package.
    // The parent class uses com.hvduong.identity.common.response.ApiResponse;
    // existing tests that use this FQCN will continue to work.
}

