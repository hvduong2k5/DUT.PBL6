package com.hvduong.catalog.common.handler;

import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import com.hvduong.catalog.common.response.ApiResponse;

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
  public ResponseEntity<ApiResponse<Void>> handleCatalogException(CatalogException ex) {
    log.warn("CatalogException occurred: code={}, message={}", ex.getErrorCode(), ex.getMessage());
    return ResponseEntity.status(ex.getErrorCode().getHttpStatus())
        .body(ApiResponse.error(ex.getErrorCode().name(), ex.getMessage()));
  }

  @ExceptionHandler(AccessDeniedException.class)
  public ResponseEntity<ApiResponse<Void>> handleAccessDeniedException(AccessDeniedException ex) {
    log.warn("AccessDeniedException: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.FORBIDDEN)
        .body(ApiResponse.error("FORBIDDEN", "Bạn không có quyền thực hiện thao tác này."));
  }

  @ExceptionHandler(MethodArgumentNotValidException.class)
  public ResponseEntity<ApiResponse<Void>> handleValidationException(
      MethodArgumentNotValidException ex) {
    List<ApiResponse.ValidationError> errors = new ArrayList<>();
    for (FieldError fe : ex.getBindingResult().getFieldErrors()) {
      errors.add(
          ApiResponse.ValidationError.builder()
              .field(fe.getField())
              .code(fe.getCode())
              .message(fe.getDefaultMessage())
              .build());
    }
    return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(ApiResponse.validationError(errors));
  }

  @ExceptionHandler(IllegalArgumentException.class)
  public ResponseEntity<ApiResponse<Void>> handleIllegalArgumentException(
      IllegalArgumentException ex) {
    log.warn("IllegalArgumentException: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.BAD_REQUEST)
        .body(ApiResponse.error("INVALID_ARGUMENT", ex.getMessage()));
  }

  @ExceptionHandler(org.springframework.web.bind.MissingRequestHeaderException.class)
  public ResponseEntity<ApiResponse<Void>> handleMissingHeader() {
    return handleCatalogException(new CatalogException(ErrorCode.INVALID_ARGUMENT));
  }

  @ExceptionHandler(Exception.class)
  public ResponseEntity<ApiResponse<Void>> handleGeneralException(Exception ex) {
    log.error("Unhandled exception occurred", ex);
    return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
        .body(
            ApiResponse.error(
                "INTERNAL_SERVER_ERROR", "Hệ thống gặp sự cố, vui lòng thử lại sau."));
  }
}
