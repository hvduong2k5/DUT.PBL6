import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/data/rest_auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';

class _Auth extends Mock implements AuthRepository {}

class _Capabilities extends Mock implements CustomerCapabilitiesRepository {}

const _user = AuthUser(id: 'C1', phoneNumber: '0905123456', fullName: 'An');
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'restore verifies identity; expired access refreshes once using ZIP payload',
    () async {
      final tokens = TokenStore();
      await tokens.save(access: 'old', refresh: 'refresh');
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      final client = ApiClient(dio: dio, tokenStore: tokens);
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            if (requests.length == 1) {
              handler.reject(
                DioException(
                  requestOptions: options,
                  type: DioExceptionType.badResponse,
                  response: Response(
                    requestOptions: options,
                    statusCode: 401,
                    data: {
                      'code': 'UNAUTHENTICATED',
                      'message': 'Expired',
                      'requestId': 'r1',
                    },
                  ),
                ),
              );
            } else {
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: options.path.endsWith('/refresh')
                      ? {
                          'access_token': 'new',
                          'refresh_token': 'rotated',
                          'token_type': 'Bearer',
                          'expires_in': 900,
                        }
                      : {
                          'customer_id': 'C1',
                          'phone_number': '0905123456',
                          'full_name': 'An',
                        },
                ),
              );
            }
          },
        ),
      );
      final repository = RestAuthRepository(client, tokens);
      expect((await repository.restoreSession())?.id, 'C1');
      expect(requests.map((r) => r.path), [
        '/api/v1/auth/me',
        '/api/v1/auth/refresh',
        '/api/v1/auth/me',
      ]);
      expect(requests[1].data, {'refresh_token': 'refresh'});
      expect(requests[1].headers['Idempotency-Key'], isNotEmpty);
      expect(requests.last.headers['Authorization'], 'Bearer new');
      expect(await tokens.readRefreshToken(), 'rotated');
    },
  );

  test(
    'offline restore retains tokens; invalid refresh clears them; offline logout clears local session',
    () async {
      final tokens = TokenStore();
      await tokens.save(access: 'old', refresh: 'refresh');
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      var offline = true;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            handler.reject(
              offline
                  ? DioException(
                      requestOptions: options,
                      type: DioExceptionType.connectionError,
                    )
                  : DioException(
                      requestOptions: options,
                      type: DioExceptionType.badResponse,
                      response: Response(
                        requestOptions: options,
                        statusCode: 401,
                        data: {
                          'code': 'UNAUTHENTICATED',
                          'message': 'Expired',
                          'requestId': 'r1',
                        },
                      ),
                    ),
            );
          },
        ),
      );
      final repository = RestAuthRepository(
        ApiClient(dio: dio, tokenStore: tokens),
        tokens,
      );
      await expectLater(repository.restoreSession(), throwsException);
      expect(await tokens.readRefreshToken(), 'refresh');
      offline = false;
      expect(await repository.restoreSession(), isNull);
      expect(await tokens.readAccessToken(), isNull);
      await tokens.save(access: 'again', refresh: 'again');
      offline = true;
      await expectLater(repository.logout(), throwsException);
      expect(await tokens.readRefreshToken(), isNull);
    },
  );

  test(
    'stale identity restore cannot sign a logged-out actor back in',
    () async {
      final repository = _Auth(), pending = Completer<AuthUser?>();
      when(repository.restoreSession).thenAnswer((_) => pending.future);
      when(repository.logout).thenAnswer((_) async {});
      final auth = AuthCubit(repository);
      // Keep the restore future independent from the logout transition.
      final restoring = auth.restore();
      expect(auth.state.pending, isTrue);
      await auth.logout();
      pending.complete(_user);
      await restoring;
      expect(auth.state.status, AuthStatus.guest);
      await auth.close();
    },
  );

  test(
    'capabilities reload on actor change and discard stale privilege response',
    () async {
      final authRepository = _Auth(), repository = _Capabilities();
      final first = Completer<CustomerCapabilities>();
      var calls = 0;
      when(repository.load).thenAnswer((_) async {
        if (++calls == 1) return first.future;
        return CustomerCapabilities(
          actor: calls == 2 ? 'REGISTERED' : 'GUEST',
          values: {'CATALOG_VIEW', if (calls == 2) 'PROFILE_MANAGE'},
          version: 1,
        );
      });
      when(
        () => authRepository.login(phoneNumber: '0905123456', password: 'old'),
      ).thenAnswer(
        (_) async =>
            const AuthSession(user: _user, accessToken: 'a', refreshToken: 'r'),
      );
      when(authRepository.logout).thenAnswer((_) async {});
      final auth = AuthCubit(authRepository);
      final caps = CapabilitiesCubit(repository, auth);
      await auth.login('0905123456', 'old');
      await Future<void>.delayed(Duration.zero);
      expect(caps.state.allows('PROFILE_MANAGE'), isTrue);
      first.complete(
        const CustomerCapabilities(
          actor: 'B2B',
          values: {'B2B_QUOTE_ACCEPT'},
          version: 1,
        ),
      );
      await Future<void>.delayed(Duration.zero);
      expect(caps.state.projection?.actor, 'REGISTERED');
      expect(caps.state.allows('B2B_QUOTE_ACCEPT'), isFalse);
      await auth.logout();
      await Future<void>.delayed(Duration.zero);
      expect(caps.state.allows('PROFILE_MANAGE'), isFalse);
      await caps.close();
      await auth.close();
    },
  );
}
