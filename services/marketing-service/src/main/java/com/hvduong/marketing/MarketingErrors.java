package com.hvduong.marketing;

import jakarta.servlet.http.HttpServletRequest;
import java.util.Map;
import org.springframework.http.ResponseEntity;
import org.springframework.dao.DataAccessException;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.bind.ServletRequestBindingException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;

@RestControllerAdvice
class MarketingErrors {
    private static final org.slf4j.Logger LOG = org.slf4j.LoggerFactory.getLogger(MarketingErrors.class);

    @ExceptionHandler(DataAccessException.class)
    ResponseEntity<Map<String, String>> database(DataAccessException error, HttpServletRequest request) {
        LOG.error("Marketing database operation failed, requestId={}", CampaignController.requestId(request), error);
        return response(500, "INTERNAL_SERVER_ERROR", request);
    }

    @ExceptionHandler(MarketingFailure.class)
    ResponseEntity<Map<String, String>> domain(MarketingFailure error, HttpServletRequest request) {
        return response(error.status(), error.getMessage(), request);
    }

    @ExceptionHandler(ReviewWorkflow.Rejected.class)
    ResponseEntity<Map<String, String>> review(ReviewWorkflow.Rejected error, HttpServletRequest request) {
        int status = switch (error.error()) {
            case FORBIDDEN -> 403;
            case VALIDATION_FAILED -> 422;
            case STALE_REVISION, STATE_CONFLICT -> 409;
        };
        return response(status, error.error().name(), request);
    }

    @ExceptionHandler({HttpMessageNotReadableException.class, ServletRequestBindingException.class,
            MethodArgumentTypeMismatchException.class})
    ResponseEntity<Map<String, String>> invalid(Exception error, HttpServletRequest request) {
        Throwable cause = error;
        while (cause != null) {
            if (cause instanceof IllegalArgumentException && cause.getStackTrace().length != 0
                    && cause.getStackTrace()[0].getClassName().equals(CampaignPlan.class.getName())) {
                return validation((IllegalArgumentException) cause, request);
            }
            cause = cause.getCause();
        }
        return response(400, "BAD_REQUEST", request);
    }

    @ExceptionHandler(IllegalArgumentException.class)
    ResponseEntity<Map<String, String>> validation(IllegalArgumentException error, HttpServletRequest request) {
        return response(422, "VALIDATION_FAILED", request);
    }

    private ResponseEntity<Map<String, String>> response(int status, String code, HttpServletRequest request) {
        String message = switch (status) {
            case 400 -> "Dữ liệu yêu cầu sai định dạng.";
            case 401 -> "Vui lòng đăng nhập lại.";
            case 403 -> "Bạn không có quyền thực hiện thao tác này.";
            case 404 -> "Không tìm thấy tài nguyên.";
            case 409 -> "Dữ liệu đã thay đổi hoặc thao tác bị trùng.";
            case 422 -> "Kế hoạch hoặc thao tác không hợp lệ.";
            default -> "Dịch vụ tạm thời không khả dụng.";
        };
        return ResponseEntity.status(status).body(Map.of(
                "code", code, "message", message, "requestId", CampaignController.requestId(request)));
    }
}
