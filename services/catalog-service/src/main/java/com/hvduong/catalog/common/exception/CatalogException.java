package com.hvduong.catalog.common.exception;

import lombok.Getter;

/**
 * Base exception cho mọi lỗi nghiệp vụ của Catalog Service.
 */
@Getter
public class CatalogException extends RuntimeException {

    private final ErrorCode errorCode;
    private final String devMessage;

    public CatalogException(ErrorCode errorCode) {
        super(errorCode.getUserMessage());
        this.errorCode = errorCode;
        this.devMessage = errorCode.getUserMessage();
    }

    public CatalogException(ErrorCode errorCode, String devMessage) {
        super(errorCode.getUserMessage());
        this.errorCode = errorCode;
        this.devMessage = devMessage;
    }

    public CatalogException(ErrorCode errorCode, String devMessage, Throwable cause) {
        super(errorCode.getUserMessage(), cause);
        this.errorCode = errorCode;
        this.devMessage = devMessage;
    }

    public static CatalogException notFound(String resourceType, String identifier) {
        return new CatalogException(
                ErrorCode.RESOURCE_NOT_FOUND,
                resourceType + " không tồn tại: " + identifier
        );
    }

    public static CatalogException versionConflict(String productId) {
        return new CatalogException(
                ErrorCode.CATALOG_VERSION_CONFLICT,
                "Conflict khi cập nhật product: " + productId
        );
    }
}
