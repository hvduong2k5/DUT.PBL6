package com.hvduong.identity.common.exception;

import lombok.Getter;

/**
 * Base domain exception for Identity Service.
 */
@Getter
public class IdentityException extends RuntimeException {

    private final ErrorCode errorCode;
    private final String devMessage;

    public IdentityException(ErrorCode errorCode) {
        super(errorCode.getUserMessage());
        this.errorCode = errorCode;
        this.devMessage = errorCode.getUserMessage();
    }

    public IdentityException(ErrorCode errorCode, String devMessage) {
        super(devMessage != null ? devMessage : errorCode.getUserMessage());
        this.errorCode = errorCode;
        this.devMessage = devMessage;
    }

    public IdentityException(ErrorCode errorCode, String devMessage, Throwable cause) {
        super(devMessage != null ? devMessage : errorCode.getUserMessage(), cause);
        this.errorCode = errorCode;
        this.devMessage = devMessage;
    }
}
