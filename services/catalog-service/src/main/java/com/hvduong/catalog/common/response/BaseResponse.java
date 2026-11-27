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
public class BaseResponse<T> {

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

  public static <T> BaseResponse<T> success(T data) {
    return BaseResponse.<T>builder()
        .success(true)
        .code("SUCCESS")
        .message("Thao tác thành công")
        .data(data)
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> BaseResponse<T> success(String message, T data) {
    return BaseResponse.<T>builder()
        .success(true)
        .code("SUCCESS")
        .message(message)
        .data(data)
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> BaseResponse<T> error(String code, String message) {
    return BaseResponse.<T>builder()
        .success(false)
        .code(code)
        .message(message)
        .requestId(java.util.UUID.randomUUID().toString())
        .timestamp(OffsetDateTime.now())
        .build();
  }

  public static <T> BaseResponse<T> validationError(List<ValidationError> errors) {
    return BaseResponse.<T>builder()
        .success(false)
        .code("VALIDATION_ERROR")
        .message("Dữ liệu đầu vào không hợp lệ")
        .errors(errors)
        .requestId(java.util.UUID.randomUUID().toString())
        .timestamp(OffsetDateTime.now())
        .build();
  }
}
