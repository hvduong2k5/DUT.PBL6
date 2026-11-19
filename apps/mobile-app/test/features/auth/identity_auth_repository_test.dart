import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/identity_api.dart';
import 'package:mobile_app/features/auth/data/identity_auth_repository.dart';
import 'package:mobile_app/features/auth/data/identity_session_store.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'identity_foundations_test.dart' show profile;

const session = IdentitySessionRecord(
  accessToken: 'access.fixture',
  refreshToken: 'refresh.fixture',
  sessionId: 'cookie.fixture',
);

class Fixture {
  Fixture({IdentitySessionStore? sessionStore})
    : store = sessionStore ?? MemoryIdentitySessionStore() {
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (o, h) async {
          requests.add(o);
          final path = o.uri.path;
          if (path.endsWith('/tokens/refresh') && !refreshEntered.isCompleted) {
            refreshEntered.complete();
          }
          final status = statuses[path] ?? 200;
          if (status != 200) {
            h.reject(
              DioException(
                requestOptions: o,
                type: DioExceptionType.badResponse,
                response: Response(requestOptions: o, statusCode: status),
              ),
            );
            return;
          }
          if (path.endsWith('/tokens/refresh') && refreshGate != null) {
            await refreshGate!.future;
          }
          if (path.endsWith('/callback') && callbackGate != null) {
            await callbackGate!.future;
          }
          final data =
              overrides[path] ??
              switch (path) {
                '/api/v1/public/auth/login' => {
                  'authorizationUrl': authorizationUrl,
                },
                '/api/v1/public/auth/callback' => session.toJson(),
                '/api/v1/public/auth/session' => {
                  'authenticated': authenticated,
                },
                '/api/v1/users/me' => profile(type: principalType),
                '/api/v1/public/auth/tokens/refresh' => {
                  'accessToken': 'rotated.access',
                  'refreshToken': 'rotated.refresh',
                },
                _ => <String, dynamic>{},
              };
          h.resolve(
            Response(
              requestOptions: o,
              statusCode: 200,
              data: {'success': true, 'data': data},
              headers: Headers.fromMap({
                if (path.endsWith('/tokens/refresh'))
                  'set-cookie': [
                    'OMAMA_SESSION=rotated.cookie; HttpOnly; Secure',
                  ],
              }),
            ),
          );
        },
      ),
    );
    repo = IdentityAuthRepository(
      api: IdentityApi(
        baseUri: Uri.parse('https://identity.invalid/api/v1'),
        dio: dio,
      ),
      store: store,
      authorizationOrigins: {'https://sso.invalid'},
      providers: {'google'},
      openBrowser: (url) async {
        browserCalls++;
        return await browser?.call(url) ??
            Uri.parse(
              'omama://auth/callback?code=code.fixture&state=state.fixture',
            );
      },
    );
  }
  final IdentitySessionStore store;
  final dio = Dio();
  final requests = <RequestOptions>[];
  final statuses = <String, int>{};
  final overrides = <String, Map<String, dynamic>>{};
  late final IdentityAuthRepository repo;
  int browserCalls = 0;
  bool authenticated = true;
  String principalType = 'CUSTOMER';
  String authorizationUrl = Uri.https('sso.invalid', '/authorize', {
    'state': 'state.fixture',
    'response_type': 'code',
    'redirect_uri': IdentityAuthRepository.redirectUri,
    'code_challenge_method': 'S256',
    'code_challenge': List.filled(43, 'a').join(),
  }).toString();
  Future<Uri> Function(Uri)? browser;
  Completer<void>? callbackGate, refreshGate;
  final refreshEntered = Completer<void>();
}

class FailingStore extends MemoryIdentitySessionStore {
  bool fail = true;
  @override
  Future<void> write(IdentitySessionRecord? record) async {
    if (fail) throw StateError('Storage unavailable');
    await super.write(record);
  }
}

class DelayedReadStore extends MemoryIdentitySessionStore {
  Completer<IdentitySessionRecord?>? nextRead;
  @override
  Future<IdentitySessionRecord?> read() {
    final pending = nextRead;
    nextRead = null;
    return pending?.future ?? super.read();
  }
}

void main() {
  test(
    'a delayed gateway credential read cannot release bearer after logout',
    () async {
      final store = DelayedReadStore();
      // Use one actual store for restore, delayed read and logout.
      final scoped = Fixture(sessionStore: store);
      await store.write(session);
      await scoped.repo.restoreSession();
      final gate = Completer<IdentitySessionRecord?>();
      store.nextRead = gate;
      final token = scoped.repo.accessToken();
      await scoped.repo.logout();
      gate.complete(session);
      expect(await token, isNull);
      expect(await store.read(), isNull);
    },
  );
  test(
    'gateway receives no saved bearer until a current persona has been verified',
    () async {
      final f = Fixture();
      await f.store.write(session);
      expect(await f.repo.accessToken(), isNull);
      f.overrides['/api/v1/users/me'] = profile()..remove('provisioningStatus');
      await expectLater(f.repo.restoreSession(), throwsFormatException);
      expect((await f.store.read())!.accessToken, session.accessToken);
      expect(await f.repo.accessToken(), isNull);
      f.overrides.clear();
      await f.repo.restoreSession();
      expect(await f.repo.accessToken(), session.accessToken);
    },
  );
  test(
    'malformed session status retains credentials rather than treating schema drift as logout',
    () async {
      final f = Fixture();
      await f.store.write(session);
      f.overrides['/api/v1/public/auth/session'] = {};
      await expectLater(f.repo.restoreSession(), throwsFormatException);
      expect((await f.store.read())!.accessToken, session.accessToken);
      expect(f.repo.currentUser, isNull);
    },
  );
  test(
    'logout while refresh is pending never restores rotated credentials or actor',
    () async {
      final f = Fixture()..refreshGate = Completer<void>();
      await f.store.write(session);
      final refreshing = f.repo.refreshSession();
      await f.refreshEntered.future;
      expect(
        f.requests.where((r) => r.uri.path.endsWith('/tokens/refresh')),
        hasLength(1),
      );
      await f.repo.logout();
      final rejected = expectLater(refreshing, throwsA(isA<ApiException>()));
      f.refreshGate!.complete();
      await rejected;
      expect(await f.store.read(), isNull);
      expect(f.repo.currentUser, isNull);
    },
  );
  test(
    'malformed persona never publishes or saves callback credentials',
    () async {
      final f = Fixture();
      f.overrides['/api/v1/users/me'] = profile()..remove('provisioningStatus');
      await expectLater(f.repo.signIn(), throwsFormatException);
      expect(await f.store.read(), isNull);
      expect(f.repo.currentUser, isNull);
    },
  );
  test(
    'legacy profile update cannot erase Employee purchase restrictions',
    () async {
      final f = Fixture()..principalType = 'EMPLOYEE';
      await f.store.write(session);
      final c = AuthCubit(f.repo);
      await c.restore();
      c.updateUser(
        const AuthUser(
          id: 'principal-test',
          phoneNumber: '',
          fullName: 'Legacy projection',
        ),
      );
      expect(c.state.user!.persona!.isEmployee, isTrue);
      expect(c.state.canPurchase, isFalse);
      await c.close();
    },
  );
  test(
    'native login target uses JSON callback, validates persona and persists complete session',
    () async {
      final f = Fixture();
      final result = await f.repo.signIn(provider: 'google');
      expect(f.requests.first.uri.queryParameters, {
        'redirect_uri': IdentityAuthRepository.redirectUri,
        'format': 'json',
        'provider': 'google',
      });
      final callback = f.requests.firstWhere(
        (r) => r.uri.path.endsWith('/callback'),
      );
      expect(callback.uri.queryParameters, {
        'code': 'code.fixture',
        'state': 'state.fixture',
        'format': 'json',
      });
      expect((await f.store.read())!.accessToken, session.accessToken);
      expect(result.user.id, 'principal-test');
      expect(result.user.phoneNumber, isEmpty);
      expect(result.user.canPurchase, isTrue);
      expect(f.requests.last.headers['Cookie'], 'OMAMA_SESSION=cookie.fixture');
    },
  );
  test(
    'wrong callback host/path/state, duplicate parameters, token fragment and error never exchange a code',
    () async {
      for (final callback in [
        'omama://evil/callback?code=x&state=state.fixture',
        'omama://auth/callback/extra?code=x&state=state.fixture',
        'omama://auth/callback?code=x&state=wrong',
        'omama://auth/callback?code=x&state=state.fixture&state=state.fixture',
        'omama://auth/callback?code=x&state=state.fixture#token=evil',
        'omama://auth/callback?error=access_denied&state=state.fixture',
      ]) {
        final f = Fixture()..browser = (_) async => Uri.parse(callback);
        await expectLater(f.repo.signIn(), throwsA(isA<ApiException>()));
        expect(
          f.requests.where((r) => r.uri.path.endsWith('/callback')),
          isEmpty,
        );
        expect(await f.store.read(), isNull);
      }
    },
  );
  test(
    'authorization origin and PKCE S256 are checked before browser handoff',
    () async {
      for (final change in [
        (String s) => s.replaceFirst('sso.invalid', 'evil.invalid'),
        (String s) => s.replaceFirst('S256', 'plain'),
        (String s) => s.replaceFirst('state.fixture', ''),
      ]) {
        final f = Fixture();
        f.authorizationUrl = change(f.authorizationUrl);
        await expectLater(f.repo.signIn(), throwsFormatException);
        expect(f.browserCalls, 0);
      }
    },
  );
  test(
    'browser cancellation saves no session and can retry; duplicate login does not launch twice',
    () async {
      final f = Fixture(), gate = Completer<Uri>();
      final entered = Completer<void>();
      f.browser = (_) {
        entered.complete();
        return gate.future;
      };
      final first = f.repo.signIn();
      await entered.future;
      await expectLater(f.repo.signIn(), throwsA(isA<ApiException>()));
      final rejected = expectLater(first, throwsStateError);
      gate.completeError(StateError('Synthetic browser cancelled'));
      await rejected;
      expect(f.browserCalls, 1);
      expect(await f.store.read(), isNull);
      f.browser = null;
      await f.repo.signIn();
      expect(f.browserCalls, 2);
    },
  );
  test(
    'logout during browser handoff prevents late session resurrection',
    () async {
      final f = Fixture(), gate = Completer<Uri>();
      f.browser = (_) => gate.future;
      final login = f.repo.signIn();
      await Future<void>.delayed(Duration.zero);
      await f.repo.logout();
      final rejected = expectLater(login, throwsA(isA<ApiException>()));
      gate.complete(
        Uri.parse('omama://auth/callback?code=x&state=state.fixture'),
      );
      await rejected;
      expect(await f.store.read(), isNull);
      expect(f.repo.currentUser, isNull);
    },
  );
  test('failed vault write never publishes an authenticated user', () async {
    final f = Fixture(sessionStore: FailingStore());
    await expectLater(f.repo.signIn(), throwsStateError);
    expect(f.repo.currentUser, isNull);
    expect(await f.store.read(), isNull);
  });
  test(
    'cold restoration reads session then persona; suspended/employee remains denied',
    () async {
      for (final type in ['CUSTOMER', 'EMPLOYEE']) {
        final f = Fixture()..principalType = type;
        await f.store.write(session);
        final user = await f.repo.restoreSession();
        expect(user!.canPurchase, type == 'CUSTOMER');
        expect(f.requests.first.uri.path, '/api/v1/public/auth/session');
        expect(
          f.requests.first.headers['Authorization'],
          isNull,
        );
        expect(f.requests.first.headers['Cookie'], 'OMAMA_SESSION=cookie.fixture');
        expect(f.requests.last.headers['Authorization'], 'Bearer access.fixture');
      }
    },
  );
  test(
    'refresh is single-flight; rotates bearer/cookie atomically; has no raw refresh-token body',
    () async {
      final f = Fixture()..refreshGate = Completer<void>();
      await f.store.write(session);
      final first = f.repo.refreshSession(), second = f.repo.refreshSession();
      await Future<void>.delayed(Duration.zero);
      f.refreshGate!.complete();
      expect(await first, isTrue);
      expect(await second, isTrue);
      final requests = f.requests
          .where((r) => r.uri.path.endsWith('/tokens/refresh'))
          .toList();
      expect(requests.length, 1);
      expect(requests.single.data, isNull);
      expect((await f.store.read())!.sessionId, 'rotated.cookie');
      expect((await f.store.read())!.refreshToken, 'rotated.refresh');
    },
  );
  test(
    'refresh transient failure retains recovery credentials; definitive 401 clears them',
    () async {
      final f = Fixture();
      await f.store.write(session);
      f.statuses['/api/v1/public/auth/tokens/refresh'] = 503;
      await expectLater(f.repo.refreshSession(), throwsA(isA<ApiException>()));
      expect((await f.store.read())!.accessToken, session.accessToken);
      f.statuses['/api/v1/public/auth/tokens/refresh'] = 401;
      expect(await f.repo.refreshSession(), isFalse);
      expect(await f.store.read(), isNull);
    },
  );
  test(
    'register target preserves same idempotency identity and never implies a login',
    () async {
      final f = Fixture();
      final cubit = AuthCubit(f.repo);
      for (var i = 0; i < 2; i++) {
        expect(
          await cubit.registerIdentity(
            firstName: ' A ',
            lastName: ' B ',
            email: 'QA@EXAMPLE.INVALID',
            password: 'SyntheticTest!2026',
            idempotencyKey: 'stable-key',
          ),
          isTrue,
        );
      }
      expect(cubit.state.user, isNull);
      expect(f.browserCalls, 0);
      expect(f.requests.last.data, {
        'firstName': 'A',
        'lastName': 'B',
        'email': 'qa@example.invalid',
        'rawPassword': 'SyntheticTest!2026',
        'idempotencyKey': 'stable-key',
        'requestFingerprint': 'qa@example.invalid',
      });
      expect(f.requests.last.headers['Idempotency-Key'], 'stable-key');
      await cubit.close();
    },
  );
  test(
    'password change and logout use target verbs; local logout survives server failure',
    () async {
      final f = Fixture();
      await f.repo.signIn();
      await f.repo.changePassword(
        currentPassword: 'OldSynthetic!2026',
        newPassword: 'NewSynthetic!2026',
      );
      expect(f.requests.last.method, 'PUT');
      expect(f.requests.last.uri.path, '/api/v1/users/me/password');
      f.statuses['/api/v1/public/auth/logout'] = 503;
      await expectLater(f.repo.logout(), throwsA(isA<ApiException>()));
      expect(await f.store.read(), isNull);
      expect(await f.repo.accessToken(), isNull);
    },
  );
  test(
    'gateway bearer is origin restricted, sends no BFF cookie and never replays a failed mutation',
    () async {
      final f = Fixture();
      await f.repo.signIn();
      final dio = Dio(BaseOptions(baseUrl: 'https://gateway.invalid'));
      final client = ApiClient(
        dio: dio,
        baseUrl: 'https://gateway.invalid',
        manageCookies: false,
        accessTokenReader: f.repo.accessToken,
      );
      client.recoverSession = f.repo.refreshSession;
      var writes = 0;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            writes++;
            expect(o.headers['Authorization'], 'Bearer access.fixture');
            expect(o.headers['Cookie'], isNull);
            h.reject(
              DioException(
                requestOptions: o,
                type: DioExceptionType.badResponse,
                response: Response(requestOptions: o, statusCode: 401),
              ),
            );
          },
        ),
      );
      await expectLater(
        client.postJson('/api/v1/checkout'),
        throwsA(isA<ApiException>()),
      );
      expect(writes, 1);
      expect((await f.store.read())!.accessToken, 'rotated.access');
      await expectLater(
        client.getJson('https://evil.invalid/api/v1/cart'),
        throwsA(isA<ApiException>()),
      );
      expect(writes, 1);
    },
  );
}
