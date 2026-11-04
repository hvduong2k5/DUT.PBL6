import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/auth/data/identity_api.dart';
import 'package:mobile_app/features/auth/data/identity_session_store.dart';

void main() {
  test(
    'native callback carries app state and refresh uses its BFF cookie',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio();
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (request, handler) {
            requests.add(request);
            handler.resolve(
              Response(
                requestOptions: request,
                statusCode: 200,
                data: {'success': true, 'data': <String, dynamic>{}},
                headers: Headers.fromMap({
                  if (request.path.endsWith('/login'))
                    'set-cookie': [
                      'OMAMA_AUTH_STATE=fixture-state; HttpOnly; Path=/',
                    ],
                }),
              ),
            );
          },
        ),
      );
      final api = IdentityApi(
        baseUri: Uri.parse('https://identity.test/api/v1'),
        dio: dio,
      );
      const session = IdentitySessionRecord(
        accessToken: 'expired.fixture',
        refreshToken: 'refresh.fixture',
        sessionId: 'session.fixture',
      );
      await api.request('GET', '/public/auth/login');
      await api.request('GET', '/public/auth/callback');
      expect(requests.last.headers['Cookie'], 'OMAMA_AUTH_STATE=fixture-state');
      await api.request('GET', '/users/me', session: session);
      expect(requests.last.headers['Authorization'], 'Bearer expired.fixture');
      await api.request(
        'POST',
        '/public/auth/tokens/refresh',
        session: session,
      );
      expect(requests.last.headers['Cookie'], 'OMAMA_SESSION=session.fixture');
      expect(requests.last.headers.containsKey('Authorization'), isFalse);
      await api.request('GET', '/public/auth/callback');
      expect(requests.last.headers.containsKey('Cookie'), isFalse);
    },
  );
}
