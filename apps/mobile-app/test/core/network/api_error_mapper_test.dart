import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';

void main() {
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
