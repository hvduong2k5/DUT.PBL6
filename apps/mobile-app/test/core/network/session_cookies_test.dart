import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';

class CookieTestAdapter implements HttpClientAdapter {
  final List<RequestOptions> requests = [];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return ResponseBody.fromString(
      '{}',
      200,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
        if (options.path.endsWith('/verify'))
          'set-cookie': [
            'guest_order_grant=server-issued; Path=/api/v1/orders; Secure; HttpOnly; Max-Age=60',
          ],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test('server cookies follow domain/path scope and can be cleared', () async {
    final adapter = CookieTestAdapter();
    final dio = Dio(BaseOptions(baseUrl: 'https://api.example.test'))
      ..httpClientAdapter = adapter;
    final client = ApiClient(dio: dio);
    await client.postJson('/api/v1/orders/guest-access/challenges/c/verify');
    await client.getJson('/api/v1/orders/ORD-1');
    expect(
      adapter.requests.last.headers['cookie'],
      contains('guest_order_grant=server-issued'),
    );
    await client.getJson('/api/v1/products');
    expect(
      adapter.requests.last.headers['cookie'] ?? '',
      isNot(contains('server-issued')),
    );
    await client.getJson('https://other.example.test/api/v1/orders/ORD-1');
    expect(
      adapter.requests.last.headers['cookie'] ?? '',
      isNot(contains('server-issued')),
    );
    await client.clearSessionCookies();
    await client.getJson('/api/v1/orders/ORD-1');
    expect(
      adapter.requests.last.headers['cookie'] ?? '',
      isNot(contains('server-issued')),
    );
    expect(dio.options.followRedirects, isFalse);
  });
}
