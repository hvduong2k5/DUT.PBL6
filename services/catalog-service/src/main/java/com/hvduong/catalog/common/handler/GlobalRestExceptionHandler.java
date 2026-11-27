package com.hvduong.catalog.common.handler;

import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import com.hvduong.catalog.common.response.BaseResponse;

import java.util.ArrayList;
import java.util.List;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@RestControllerAdvice
public class GlobalRestExceptionHandler {

  @ExceptionHandler(CatalogException.class)
  public ResponseEntity<BaseResponse<Void>> handleCatalogException(CatalogException ex) {
    log.warn("CatalogException occurred: code={}, message={}", ex.getErrorCode(), ex.getMessage());
    return ResponseEntity.status(ex.getErrorCode().getHttpStatus())
        .body(BaseResponse.error(ex.getErrorCode().name(), ex.getMessage()));
  }

  @ExceptionHandler(AccessDeniedException.class)
  public ResponseEntity<BaseResponse<Void>> handleAccessDeniedException(AccessDeniedException ex) {
    log.warn("AccessDeniedException: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.FORBIDDEN)
        .body(BaseResponse.error("FORBIDDEN", "Bạn không có quyền thực hiện thao tác này."));
  }

  @ExceptionHandler(MethodArgumentNotValidException.class)
  public ResponseEntity<BaseResponse<Void>> handleValidationException(
      MethodArgumentNotValidException ex) {
    List<BaseResponse.ValidationError> errors = new ArrayList<>();
    for (FieldError fe : ex.getBindingResult().getFieldErrors()) {
      errors.add(
          BaseResponse.ValidationError.builder()
              .field(fe.getField())
              .code(fe.getCode())
              .message(fe.getDefaultMessage())
              .build());
    }
    return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(BaseResponse.validationError(errors));
  }

  @ExceptionHandler(IllegalArgumentException.class)
  public ResponseEntity<BaseResponse<Void>> handleIllegalArgumentException(
      IllegalArgumentException ex) {
    log.warn("IllegalArgumentException: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.BAD_REQUEST)
        .body(BaseResponse.error("INVALID_ARGUMENT", ex.getMessage()));
  }

  @ExceptionHandler(org.springframework.dao.DataIntegrityViolationException.class)
  public ResponseEntity<BaseResponse<Void>> handleDataIntegrityViolation(
      org.springframework.dao.DataIntegrityViolationException ex) {
    log.warn("DataIntegrityViolationException: {}", ex.getMessage());
    String msg = ex.getMessage();
    if (msg != null && (msg.contains("excl_channel_prices_no_overlap") || msg.contains("23P01"))) {
      return ResponseEntity.status(HttpStatus.CONFLICT)
          .body(
              BaseResponse.error(
                  "PRICE_OVERLAP_CONFLICT",
                  "Khoảng thời gian hiệu lực bị trùng lặp với bản giá đã tồn tại trên kênh này."));
    }
    return ResponseEntity.status(HttpStatus.CONFLICT)
        .body(
            BaseResponse.error(
                "DATA_INTEGRITY_VIOLATION",
                "Dữ liệu bị trùng lặp hoặc vi phạm ràng buộc toàn vẹn."));
  }

  @ExceptionHandler(org.springframework.web.bind.MissingRequestHeaderException.class)
  public ResponseEntity<BaseResponse<Void>> handleMissingHeader() {
    return handleCatalogException(new CatalogException(ErrorCode.INVALID_ARGUMENT));
  }

  @ExceptionHandler(Exception.class)
  public ResponseEntity<BaseResponse<Void>> handleGeneralException(Exception ex) {
    log.error("Unhandled exception occurred", ex);
    return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
        .body(
            BaseResponse.error(
                "INTERNAL_SERVER_ERROR", "Hệ thống gặp sự cố, vui lòng thử lại sau."));
  }
}
