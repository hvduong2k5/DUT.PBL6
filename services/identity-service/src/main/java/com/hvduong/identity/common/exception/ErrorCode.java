package com.hvduong.identity.common.exception;

import lombok.Getter;

/**
 * Business error codes for Identity Service.
 */
@Getter
public enum ErrorCode {

    // Shared system error codes (1 - 99)
    INTERNAL_SERVER_ERROR(1, 500, "Đã xảy ra lỗi hệ thống, vui lòng thử lại sau."),
    INVALID_ARGUMENT(2, 400, "Dữ liệu đầu vào không hợp lệ."),
    UNAUTHENTICATED(3, 401, "Bạn chưa đăng nhập hoặc phiên làm việc đã hết hạn."),
    PERMISSION_DENIED(4, 403, "Bạn không có quyền thực hiện thao tác này."),
    RESOURCE_NOT_FOUND(5, 404, "Không tìm thấy tài nguyên yêu cầu."),
    STATE_CONFLICT(6, 409, "Xung đột trạng thái dữ liệu."),

    // Identity & IAM domain error codes (100 - 199)
    PRINCIPAL_NOT_FOUND(100, 404, "Không tìm thấy tài khoản người dùng."),
    ROLE_NOT_FOUND(101, 404, "Không tìm thấy vai trò hệ thống hoặc vai trò đã bị xóa."),
    ROLE_CODE_DUPLICATE(102, 409, "Mã vai trò này đã tồn tại trong hệ thống."),
    ROLE_IN_USE(103, 409, "Không thể xóa vai trò đang có nhân viên đảm nhận hiệu lực."),
    ROLE_ALREADY_DELETED(104, 409, "Vai trò đã bị xóa mềm và không thể kích hoạt lại."),
    PROTECTED_ROLE_MODIFICATION(105, 400, "Không thể sửa đổi hoặc xóa vai trò hệ thống được bảo vệ."),
    SELF_ASSIGNMENT_FORBIDDEN(106, 403, "Không được phép tự gán vai trò cho chính mình."),
    LAST_ADMIN_PROTECTION(107, 409, "Không thể thu hồi hoặc khóa quản trị viên hoạt động cuối cùng của hệ thống."),
    CUSTOMER_ADMIN_ROLE_FORBIDDEN(108, 400, "Không thể gán vai trò quản trị cho tài khoản khách hàng."),
    VERSION_CONFLICT(109, 409, "Dữ liệu đã bị thay đổi bởi người khác. Vui lòng tải lại trang và thử lại."),
    PERMISSION_SYNC_FAILED(110, 502, "Không thể đồng bộ quyền sang Keycloak. Vui lòng thử lại hoặc chạy lại đồng bộ cho tài khoản này.");

    private final int protoCode;
    private final int httpStatus;
    private final String userMessage;

    ErrorCode(int protoCode, int httpStatus, String userMessage) {
        this.protoCode = protoCode;
        this.httpStatus = httpStatus;
        this.userMessage = userMessage;
    }
}
