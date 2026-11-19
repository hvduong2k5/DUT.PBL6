import 'dart:convert';
import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/cart/data/mvp_cart_repository.dart';

class ScriptedAdapter implements HttpClientAdapter {
  ScriptedAdapter(this.respond);
  final ResponseBody Function(RequestOptions request) respond;
  final requests = <RequestOptions>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return respond(options);
  }

  @override
  void close({bool force = false}) {}
}

ResponseBody cart(int revision, List<String> skus) => ResponseBody.fromString(
  jsonEncode({
    'revision': revision,
    'items': [
      for (final sku in skus) {'item_id': sku, 'sku_code': sku, 'quantity': 1},
    ],
  }),
  200,
  headers: {
    Headers.contentTypeHeader: [Headers.jsonContentType],
  },
);

MvpCartRepository repository(ScriptedAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'https://mvp.test'))
    ..httpClientAdapter = adapter;
  return MvpCartRepository(
    ApiClient(
      dio: dio,
      manageCookies: false,
      accessTokenReader: () async => 'synthetic-fixture',
    ),
  );
}

void main() {
  test('line deletion accepts 204 and reloads the real cart', () async {
    var deleted = false;
    final adapter = ScriptedAdapter((request) {
      if (request.method == 'GET') {
        return cart(deleted ? 9 : 8, deleted ? [] : ['SKU-A']);
      }
      expect(request.path, '/api/v1/cart/items/SKU-A');
      expect(request.headers['If-Match'], '"8"');
      deleted = true;
      return ResponseBody.fromString(
        '',
        204,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    });
    expect((await repository(adapter).remove('SKU-A')).items, isEmpty);
    expect(adapter.requests.map((request) => request.method), [
      'GET',
      'DELETE',
      'GET',
    ]);
  });

  test('clear uses line deletion and each current CAS revision', () async {
    var deleted = 0;
    final adapter = ScriptedAdapter((request) {
      if (request.method == 'GET') {
        return cart(8 + deleted, ['SKU-A', 'SKU-B'].skip(deleted).toList());
      }
      expect(request.method, 'DELETE');
      expect(
        request.path,
        '/api/v1/cart/items/${deleted == 0 ? 'SKU-A' : 'SKU-B'}',
      );
      expect(request.headers['If-Match'], '"${8 + deleted}"');
      deleted++;
      return ResponseBody.fromString('', 204);
    });
    expect((await repository(adapter).clear()).items, isEmpty);
    expect(deleted, 2);
    expect(adapter.requests, hasLength(5));
  });

  test(
    'a partial clear stops at conflict without replay or a fake empty cart',
    () async {
      var deleted = 0;
      final adapter = ScriptedAdapter((request) {
        if (request.method == 'GET') {
          return cart(8 + deleted, ['SKU-A', 'SKU-B'].skip(deleted).toList());
        }
        if (deleted == 0) {
          deleted++;
          return ResponseBody.fromString('', 204);
        }
        return ResponseBody.fromString(
          '{"code":"CART_REVISION_MISMATCH"}',
          409,
          headers: {
            Headers.contentTypeHeader: [Headers.jsonContentType],
          },
        );
      });
      await expectLater(
        repository(adapter).clear(),
        throwsA(
          isA<ApiException>().having(
            (error) => error.failure.kind,
            'kind',
            FailureKind.conflict,
          ),
        ),
      );
      expect(adapter.requests, hasLength(4));
      expect(deleted, 1);
    },
  );
}
