import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';

void main() {
  test('ZIP Error retains field validation, correlation and cooldown', () {
    final request = RequestOptions(path: '/api/v1/auth/recovery-requests');
    final failure = ApiErrorMapper.fromDio(
      DioException(
        requestOptions: request,
        response: Response(
          requestOptions: request,
          statusCode: 429,
          data: {
            'code': 'RECOVERY_RATE_LIMITED',
            'message': 'Vui lòng chờ.',
            'requestId': 'req-1',
            'retryAfterSeconds': 30,
            'errors': [
              {
                'field': 'email',
                'code': 'INVALID',
                'message': 'Email chưa hợp lệ.',
              },
            ],
          },
        ),
      ),
    );
    expect(failure.kind, FailureKind.rateLimited);
    expect(failure.code, 'RECOVERY_RATE_LIMITED');
    expect(failure.requestId, 'req-1');
    expect(failure.retryAfterSeconds, 30);
    expect(failure.fieldErrors['email'], 'Email chưa hợp lệ.');
  });
  test('maps unified OpenAPI conflict response', () {
    final request = RequestOptions(path: '/api/v1/checkout');
    final failure = ApiErrorMapper.fromDio(
      DioException(
        requestOptions: request,
        response: Response(
          requestOptions: request,
          statusCode: 409,
          data: {
            'error_code': 'ERR_INVENTORY_INSUFFICIENT_STOCK',
            'user_message': 'Sản phẩm không đủ tồn kho.',
            'domain': 'inventory',
          },
        ),
      ),
    );
    expect(failure.kind, FailureKind.conflict);
    expect(failure.code, 'ERR_INVENTORY_INSUFFICIENT_STOCK');
  });
  test(
    'maps transport error separately',
    () => expect(
      ApiErrorMapper.fromDio(
        DioException(
          requestOptions: RequestOptions(path: '/'),
          type: DioExceptionType.connectionError,
        ),
      ).kind,
      FailureKind.network,
    ),
  );
}
