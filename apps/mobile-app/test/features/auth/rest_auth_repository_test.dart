import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/data/rest_auth_repository.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));

  test(
    'ZIP login identifier and complete recovery HTTP payloads, 204 and local logout',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            final data = switch (options.path) {
              '/api/v1/auth/login' => <String, Object>{
                'user': {
                  'customer_id': 'C1',
                  'full_name': 'An',
                  'phone_number': '0905123456',
                },
                'tokens': {
                  'access_token': 'access',
                  'refresh_token': 'refresh',
                },
              },
              '/api/v1/auth/recovery-requests/verify' => <String, Object>{
                'verified': true,
                'resetProof': 'restricted-proof',
                'expiresAt': DateTime.now()
                    .add(const Duration(minutes: 5))
                    .toIso8601String(),
              },
              '/api/v1/auth/recovery-requests' => <String, Object>{
                'accepted': true,
                'messageCode': 'RECOVERY_REQUEST_ACCEPTED',
              },
              _ => null,
            };
            handler.resolve(
              Response(
                requestOptions: options,
                data: data,
                statusCode: data == null
                    ? 204
                    : options.path.endsWith('recovery-requests')
                    ? 202
                    : 200,
              ),
            );
          },
        ),
      );
      final tokens = TokenStore();
      final repository = RestAuthRepository(
        ApiClient(dio: dio, tokenStore: tokens),
        tokens,
      );
      await repository.login(phoneNumber: '0905123456', password: 'old');
      expect(requests.last.data, {
        'identifier': '0905123456',
        'password': 'old',
      });
      expect(await tokens.readAccessToken(), 'access');
      await repository.requestRecovery(
        'an@example.com',
        idempotencyKey: 'request-key',
      );
      expect(requests.last.data, {'email': 'an@example.com'});
      expect(requests.last.headers['Idempotency-Key'], 'request-key');
      final verification = await repository.verifyRecovery(
        'token',
        idempotencyKey: 'verify-key',
      );
      expect(requests.last.data, {'recoveryToken': 'token'});
      expect(verification.resetProof, 'restricted-proof');
      await repository.resetPassword(
        verification.resetProof,
        'TriKyMatKhau!2026',
        idempotencyKey: 'reset-key',
      );
      expect(requests.last.data, {
        'resetProof': 'restricted-proof',
        'newPassword': 'TriKyMatKhau!2026',
      });
      expect(await tokens.readAccessToken(), isNull);
      expect(await tokens.readRefreshToken(), isNull);
    },
  );
}
