import 'package:dio/dio.dart';
import 'package:mobile_app/core/error/failures.dart';

abstract final class ApiErrorMapper {
  static Failure fromDio(DioException error) {
    if (error.type == DioExceptionType.connectionTimeout ||
        error.type == DioExceptionType.sendTimeout ||
        error.type == DioExceptionType.receiveTimeout) {
      return const Failure(
        kind: FailureKind.timeout,
        message: 'Kết nối quá lâu. Vui lòng thử lại.',
      );
    }
    if (error.type == DioExceptionType.connectionError) {
      return const Failure(
        kind: FailureKind.network,
        message: 'Không có kết nối mạng. Kiểm tra kết nối và thử lại.',
      );
    }
    final status = error.response?.statusCode;
    final body = error.response?.data;
    final data = body is Map
        ? Map<String, dynamic>.from(body)
        : <String, dynamic>{};
    final fields = <String, String>{};
    if (data['details'] case final Map details) {
      for (final entry in details.entries) {
        fields[entry.key.toString()] = entry.value.toString();
      }
    }
    return Failure(
      kind: _kindForStatus(status),
      message: data['user_message'] as String? ?? _defaultMessage(status),
      code: data['error_code'] as String?,
      domain: data['domain'] as String?,
      fieldErrors: fields,
    );
  }

  static FailureKind _kindForStatus(int? status) => switch (status) {
    400 || 422 => FailureKind.validation,
    401 => FailureKind.unauthenticated,
    403 => FailureKind.forbidden,
    404 => FailureKind.notFound,
    409 => FailureKind.conflict,
    int value when value >= 500 => FailureKind.server,
    _ => FailureKind.unknown,
  };

  static String _defaultMessage(int? status) => switch (status) {
    400 || 422 => 'Thông tin chưa hợp lệ. Vui lòng kiểm tra lại.',
    401 => 'Phiên đăng nhập đã hết hạn.',
    403 => 'Bạn không có quyền thực hiện thao tác này.',
    404 => 'Không tìm thấy dữ liệu yêu cầu.',
    409 => 'Dữ liệu đã thay đổi. Vui lòng tải lại.',
    int value when value >= 500 => 'Hệ thống đang bận. Vui lòng thử lại sau.',
    _ => 'Đã xảy ra lỗi. Vui lòng thử lại.',
  };
}
