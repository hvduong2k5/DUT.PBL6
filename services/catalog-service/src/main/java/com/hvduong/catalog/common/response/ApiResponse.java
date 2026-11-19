package com.hvduong.catalog.common.response;

import java.time.OffsetDateTime;
import java.util.List;

import com.fasterxml.jackson.annotation.JsonInclude;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class ApiResponse<T> {

  private boolean success;
  private String code;
  private String message;
  private T data;
  private List<ValidationError> errors;
  private String requestId;
  private OffsetDateTime timestamp;

  @Data
  @Builder
  @NoArgsConstructor
  @AllArgsConstructor
  public static class ValidationError {
    private String field;
    private String code;
    private String message;
  }

  public static <T> ApiResponse<T> success(T data) {
    return ApiResponse.<T>builder()
        .success(true)
        .code("SUCCESS")
        .message("Thao tác thành công")
        .data(data)
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> ApiResponse<T> success(String message, T data) {
    return ApiResponse.<T>builder()
        .success(true)
        .code("SUCCESS")
        .message(message)
        .data(data)
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> ApiResponse<T> error(String code, String message) {
    return ApiResponse.<T>builder()
        .success(false)
        .code(code)
        .message(message)
        .requestId(java.util.UUID.randomUUID().toString())
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> ApiResponse<T> validationError(List<ValidationError> errors) {
    return ApiResponse.<T>builder()
        .success(false)
        .code("VALIDATION_ERROR")
        .message("Dữ liệu đầu vào không hợp lệ")
        .errors(errors)
        .requestId(java.util.UUID.randomUUID().toString())
        .timestamp(OffsetDateTime.now())
        .build();
  }
}
