package com.hvduong.catalog.common.exception;

import lombok.Getter;

/**
 * Mã lỗi nghiệp vụ của Catalog Service.
 * Mỗi code gồm: protoCode, HTTP status, thông điệp người dùng (tiếng Việt).
 */
@Getter
public enum ErrorCode {

    // ── Lỗi hệ thống dùng chung (1 - 99) ─────────────────────────────────────
    INTERNAL_SERVER_ERROR(1, 500, "Hệ thống gặp sự cố, vui lòng thử lại sau."),
    INVALID_ARGUMENT(2, 400, "Dữ liệu đầu vào không hợp lệ."),
    UNAUTHENTICATED(3, 401, "Bạn chưa đăng nhập hoặc phiên làm việc đã hết hạn."),
    PERMISSION_DENIED(4, 403, "Bạn không có quyền thực hiện thao tác này."),
    RESOURCE_NOT_FOUND(5, 404, "Không tìm thấy tài nguyên yêu cầu."),
    IDEMPOTENCY_CONFLICT(6, 409, "Yêu cầu đang được xử lý hoặc đã hoàn tất trước đó."),
    CIRCUIT_BREAKER_OPEN(7, 503, "Hệ thống đang quá tải, vui lòng thử lại sau giây lát."),
    DEADLINE_EXCEEDED(8, 504, "Yêu cầu đã quá thời gian chờ phản hồi."),

    // ── Lỗi miền Catalog (200 - 299) ─────────────────────────────────────────
    CATALOG_PRICE_TAMPERED(200, 400,
            "Giá sản phẩm đã thay đổi. Vui lòng kiểm tra lại giỏ hàng."),
    CATALOG_PRODUCT_OUT_OF_SEASON(201, 400,
            "Sản phẩm này chưa mở bán trong thời điểm hiện tại."),
    CATALOG_VARIANT_MISMATCH(202, 400,
            "Biến thể sản phẩm không khớp với sản phẩm cha. Vui lòng chọn lại."),
    CATALOG_PRODUCT_NOT_FOUND(203, 404,
            "Không tìm thấy sản phẩm."),
    CATALOG_VARIANT_NOT_FOUND(204, 404,
            "Không tìm thấy biến thể sản phẩm với mã SKU này."),
    CATALOG_CATEGORY_NOT_FOUND(205, 404,
            "Không tìm thấy danh mục sản phẩm."),
    CATALOG_PRODUCT_NOT_PUBLIC(206, 403,
            "Sản phẩm chưa được phê duyệt hoặc đang tạm ngừng bán."),
    CATALOG_SKU_DUPLICATE(207, 409,
            "Mã SKU này đã tồn tại trong hệ thống."),
    CATALOG_PRODUCT_INVALID_STATE(208, 422,
            "Không thể thực hiện thao tác này với trạng thái hiện tại của sản phẩm."),
    CATALOG_PRICE_NOT_FOUND(209, 404,
            "Không tìm thấy giá niêm yết cho kênh bán này."),
    CATALOG_SLUG_DUPLICATE(210, 409,
            "Đường dẫn (slug) này đã được sử dụng bởi sản phẩm khác."),
    CATALOG_VERSION_CONFLICT(211, 409,
            "Sản phẩm đã được người khác cập nhật. Vui lòng tải lại và thử lại.");

    private final int protoCode;
    private final int httpStatus;
    private final String userMessage;

    ErrorCode(int protoCode, int httpStatus, String userMessage) {
        this.protoCode = protoCode;
        this.httpStatus = httpStatus;
        this.userMessage = userMessage;
    }
}
