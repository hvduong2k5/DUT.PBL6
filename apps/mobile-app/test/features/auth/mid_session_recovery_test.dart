import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/data/rest_auth_repository.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'protected 401 refreshes identity but never replays mutation; invalid session removes actor',
    () async {
      final tokens = TokenStore();
      await tokens.save(access: 'old', refresh: 'r');
      var expired = false, invalid = false, writes = 0, refreshes = 0;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      final client = ApiClient(dio: dio, tokenStore: tokens);
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            if (o.path == '/api/v1/orders/O/cancel') writes++;
            if (o.path.endsWith('/refresh')) refreshes++;
            if (o.path.endsWith('/cancel') ||
                o.path.endsWith('/me') && expired ||
                o.path.endsWith('/refresh') && invalid) {
              h.reject(
                DioException(
                  requestOptions: o,
                  type: DioExceptionType.badResponse,
                  response: Response(
                    requestOptions: o,
                    statusCode: 401,
                    data: {'code': 'UNAUTHENTICATED', 'message': 'Expired'},
                  ),
                ),
              );
              return;
            }
            if (o.path.endsWith('/refresh')) expired = false;
            h.resolve(
              Response(
                requestOptions: o,
                statusCode: 200,
                data: o.path.endsWith('/refresh')
                    ? {'access_token': 'new', 'refresh_token': 'rotated'}
                    : {
                        'customer_id': 'C1',
                        'phone_number': '0905123456',
                        'full_name': 'An',
                      },
              ),
            );
          },
        ),
      );
      final auth = AuthCubit(RestAuthRepository(client, tokens));
      await auth.restore();
      expect(auth.state.status, AuthStatus.authenticated);
      expired = true;
      await expectLater(
        client.postJson(
          '/api/v1/orders/O/cancel',
          data: {'reason': 'Kiểm thử'},
          headers: {'Idempotency-Key': 'same'},
        ),
        throwsA(
          isA<ApiException>().having(
            (e) => e.failure.message,
            'manual retry',
            contains('thử lại'),
          ),
        ),
      );
      expect(writes, 1);
      expect(refreshes, 1);
      expect(auth.state.user?.id, 'C1');
      expect(await tokens.readAccessToken(), 'new');
      expired = true;
      invalid = true;
      await expectLater(
        client.postJson('/api/v1/orders/O/cancel'),
        throwsA(isA<ApiException>()),
      );
      expect(auth.state.status, AuthStatus.guest);
      expect(auth.state.user, isNull);
      expect(await tokens.readRefreshToken(), isNull);
      expect(writes, 2);
      await auth.close();
    },
  );
  test(
    'concurrent protected failures share recovery; 403 and network failure never trigger it',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost')),
          client = ApiClient(dio: null);
      // Use a separate client with the same controlled transport below.
      final controlled = ApiClient(dio: dio);
      var recoveries = 0, code = 401;
      controlled.recoverSession = () async {
        recoveries++;
        await Future<void>.delayed(const Duration(milliseconds: 20));
        return true;
      };
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            h.reject(
              DioException(
                requestOptions: o,
                type: code == 0
                    ? DioExceptionType.connectionError
                    : DioExceptionType.badResponse,
                response: code == 0
                    ? null
                    : Response(
                        requestOptions: o,
                        statusCode: code,
                        data: {
                          'code': code == 401 ? 'UNAUTHENTICATED' : 'FORBIDDEN',
                          'message': 'Denied',
                        },
                      ),
              ),
            );
          },
        ),
      );
      await Future.wait([
        for (var i = 0; i < 2; i++)
          expectLater(
            controlled.getJson('/api/v1/profile'),
            throwsA(isA<ApiException>()),
          ),
      ]);
      expect(recoveries, 1);
      for (final next in [403, 0]) {
        code = next;
        await expectLater(
          controlled.getJson('/api/v1/profile'),
          throwsA(isA<ApiException>()),
        );
      }
      expect(recoveries, 1);
      // Keep the default client constructible with no configured session recovery.
      expect(client.recoverSession, isNull);
    },
  );
}
