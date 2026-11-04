import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/data/rest_mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_quote_cubit.dart';

class _Repository extends Mock implements MvpPurchaseRepository {}

final _now = DateTime.utc(2026, 10, 31);
Map<String, dynamic> cartJson({int revision = 4}) => {
  'revision': revision,
  'items': [
    {'sku_code': 'SKU-A', 'quantity': 2},
    {'sku_code': 'SKU-B', 'quantity': 1, 'unit_price': 85000},
  ],
};
Map<String, dynamic> addressJson(String id) => {
  'id': id,
  'recipient_name': 'Local fixture',
  'phone_number': '0900000000',
  'street_address': 'Fixture street',
  'province_code': '01',
  'ward_code': '00001',
  'province_name': 'Province',
  'ward_name': 'Ward',
  'version': 2,
  'is_default': true,
};
Map<String, dynamic> quoteJson() => {
  'quote_id': 'quote-1',
  'cart_revision': 4,
  'payment_method': 'VIETQR',
  'items': cartJson()['items'],
  'source_address_id': 'a',
  'source_address_version': 2,
  'address': {
    'recipient_name': 'Snapshot',
    'phone': '0900000000',
    'street': 'Snapshot street',
    'province_code': '01',
    'ward_code': '00001',
  },
  'catalog_snapshot_id': 'catalog-1',
  'shipping_quote_reference': 'shipping-1',
  'subtotal': 100000,
  'shipping_fee': 12000,
  'final_amount': 112000,
  'expires_at': '2026-10-31T00:15:00Z',
  'catalog_expires_at': '2026-10-31T00:10:00Z',
  'shipping_expires_at': '2026-10-31T00:12:00Z',
};
void _seed(_Repository r) {
  when(r.cart).thenAnswer((_) async => MvpCart.fromJson(cartJson()));
  when(r.savedAddresses).thenAnswer(
    (_) async => [
      MvpSavedAddress.fromJson(addressJson('a')),
      MvpSavedAddress.fromJson(addressJson('b')),
    ],
  );
  when(
    () => r.quote(cartRevision: 4, addressId: 'a'),
  ).thenAnswer((_) async => MvpQuote.fromJson(quoteJson()));
}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'MVP cart preserves optional prices, integer VND, revision and immutable lines',
    () {
      final cart = MvpCart.fromJson(cartJson());
      expect(cart.revision, 4);
      expect(cart.items.first.unitPrice, isNull);
      expect(cart.items.last.unitPrice!.units, BigInt.from(85000));
      expect(() => cart.items.clear(), throwsUnsupportedError);
      expect(() => MvpCart.fromJson({'items': []}), throwsFormatException);
      expect(
        () => MvpCartItem.fromJson({
          'sku_code': 'a',
          'quantity': 1,
          'unit_price': 1.5,
        }),
        throwsFormatException,
      );
    },
  );
  test(
    'canonical codes and server quote snapshot/earliest expiry are retained',
    () {
      final address = MvpSavedAddress.fromJson(addressJson('a'));
      expect(address.wardCode, '00001');
      expect(address.provinceCode, '01');
      expect(address.version, 2);
      final quote = MvpQuote.fromJson(quoteJson());
      expect(quote.address.recipient, 'Snapshot');
      expect(quote.total.units, BigInt.from(112000));
      expect(quote.sourceAddressVersion, 2);
      expect(quote.isUsableAt(_now), isTrue);
      expect(quote.isUsableAt(_now.add(const Duration(minutes: 10))), isFalse);
      expect(
        () => MvpQuote.fromJson({
          ...quoteJson(),
          'expires_at': '2026-10-31T00:15:00',
        }),
        throwsFormatException,
      );
    },
  );
  test(
    'REST reads array addresses and exact full-cart quote; no legacy fields or order creation',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            requests.add(o);
            final Object body = switch (o.path) {
              '/api/v1/cart' => cartJson(),
              '/api/v1/profile/addresses' => [addressJson('a')],
              '/api/v1/checkout/quote' => quoteJson(),
              _ => throw StateError('Unexpected endpoint'),
            };
            h.resolve(Response(requestOptions: o, statusCode: 200, data: body));
          },
        ),
      );
      final repo = RestMvpPurchaseRepository(ApiClient(dio: dio));
      expect((await repo.cart()).items.length, 2);
      expect((await repo.savedAddresses()).single.wardCode, '00001');
      final q = await repo.quote(cartRevision: 4, addressId: 'a');
      expect(q.total.units, BigInt.from(112000));
      expect(requests.last.data, {
        'cart_revision': 4,
        'payment_method': 'VIETQR',
        'shipping_address_id': 'a',
      });
      expect(requests.where((o) => o.method == 'POST').length, 1);
    },
  );
  test(
    'array reads reject legacy wrapper and retain mapped auth/network error handling',
    () async {
      var fail = false;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            if (fail) {
              h.reject(
                DioException(
                  requestOptions: o,
                  type: DioExceptionType.connectionError,
                ),
              );
            } else {
              h.resolve(
                Response(
                  requestOptions: o,
                  statusCode: 200,
                  data: {'addresses': []},
                ),
              );
            }
          },
        ),
      );
      final client = ApiClient(dio: dio);
      await expectLater(
        client.getJsonList('/api/v1/profile/addresses'),
        throwsFormatException,
      );
      fail = true;
      await expectLater(
        client.getJsonList('/api/v1/profile/addresses'),
        throwsA(isA<ApiException>()),
      );
    },
  );
  test(
    'REST refuses quote from a different revision or saved address',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) => h.resolve(
            Response(
              requestOptions: o,
              statusCode: 200,
              data: {...quoteJson(), 'cart_revision': 5},
            ),
          ),
        ),
      );
      final repo = RestMvpPurchaseRepository(ApiClient(dio: dio));
      await expectLater(
        repo.quote(cartRevision: 4, addressId: 'a'),
        throwsFormatException,
      );
      await expectLater(
        repo.quote(cartRevision: 4, addressId: ''),
        throwsArgumentError,
      );
    },
  );
  test('empty cart, deleted/missing canonical address block quote', () async {
    final r = _Repository();
    _seed(r);
    when(
      r.cart,
    ).thenAnswer((_) async => MvpCart.fromJson({'revision': 4, 'items': []}));
    final c = MvpQuoteCubit(r, now: () => _now);
    await c.load();
    await c.requestQuote();
    expect(c.state.status, MvpQuoteStatus.emptyCart);
    verifyNever(r.savedAddresses);
    when(r.cart).thenAnswer((_) async => MvpCart.fromJson(cartJson()));
    when(r.savedAddresses).thenAnswer(
      (_) async => [
        MvpSavedAddress.fromJson({...addressJson('a'), 'is_deleted': true}),
        MvpSavedAddress.fromJson({...addressJson('b'), 'ward_code': ''}),
      ],
    );
    await c.load();
    await c.requestQuote();
    expect(c.state.status, MvpQuoteStatus.noAddress);
    await c.close();
  });
  test(
    'duplicate quote tap is ignored; address change discards previous response',
    () async {
      final r = _Repository();
      _seed(r);
      final pending = Completer<MvpQuote>();
      when(
        () => r.quote(cartRevision: 4, addressId: 'a'),
      ).thenAnswer((_) => pending.future);
      final c = MvpQuoteCubit(r, now: () => _now);
      await c.load();
      final request = c.requestQuote();
      await c.requestQuote();
      verify(() => r.quote(cartRevision: 4, addressId: 'a')).called(1);
      c.selectAddress('b');
      pending.complete(MvpQuote.fromJson(quoteJson()));
      await request;
      expect(c.state.addressId, 'b');
      expect(c.state.quote, isNull);
      expect(c.state.status, MvpQuoteStatus.editing);
      await c.close();
    },
  );
  test(
    'ready quote expires without extending deadline and selection invalidates it',
    () async {
      final r = _Repository();
      _seed(r);
      final c = MvpQuoteCubit(r, now: () => _now);
      await c.load();
      await c.requestQuote();
      expect(c.state.usableAt(_now), isTrue);
      expect(c.state.usableAt(_now.add(const Duration(minutes: 20))), isFalse);
      c.selectAddress('b');
      expect(c.state.quote, isNull);
      await c.close();
    },
  );
  test(
    'quote failure clears old authority; retry reloads a new revision',
    () async {
      final r = _Repository();
      _seed(r);
      when(
        () => r.quote(cartRevision: 4, addressId: 'a'),
      ).thenThrow(StateError('offline'));
      final c = MvpQuoteCubit(r, now: () => _now);
      await c.load();
      await c.requestQuote();
      expect(c.state.status, MvpQuoteStatus.failure);
      expect(c.state.cart, isNull);
      await c.requestQuote();
      verify(() => r.quote(cartRevision: 4, addressId: 'a')).called(1);
      when(
        r.cart,
      ).thenAnswer((_) async => MvpCart.fromJson(cartJson(revision: 5)));
      when(() => r.quote(cartRevision: 5, addressId: 'a')).thenAnswer(
        (_) async => MvpQuote.fromJson({...quoteJson(), 'cart_revision': 5}),
      );
      await c.load();
      await c.requestQuote();
      expect(c.state.quote!.cartRevision, 5);
      await c.close();
    },
  );
  test('array transport recovers protected 401 without replay', () async {
    var calls = 0, recoveries = 0;
    final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (o, h) {
          calls++;
          h.reject(
            DioException(
              requestOptions: o,
              type: DioExceptionType.badResponse,
              response: Response(
                requestOptions: o,
                statusCode: 401,
                data: {
                  'code': 'SESSION_EXPIRED',
                  'message': 'Expired',
                  'requestId': 'r1',
                },
              ),
            ),
          );
        },
      ),
    );
    final client = ApiClient(dio: dio);
    client.recoverSession = () async {
      recoveries++;
      return true;
    };
    await expectLater(
      client.getJsonList('/api/v1/profile/addresses'),
      throwsA(isA<ApiException>()),
    );
    expect(calls, 1);
    expect(recoveries, 1);
  });
  test('new load discards an older cart response', () async {
    final r = _Repository();
    _seed(r);
    final old = Completer<MvpCart>();
    when(r.cart).thenAnswer((_) => old.future);
    final c = MvpQuoteCubit(r, now: () => _now);
    final first = c.load();
    when(
      r.cart,
    ).thenAnswer((_) async => MvpCart.fromJson(cartJson(revision: 8)));
    await c.load();
    old.complete(MvpCart.fromJson(cartJson()));
    await first;
    expect(c.state.cart!.revision, 8);
    expect(c.state.status, MvpQuoteStatus.editing);
    await c.close();
  });
  test(
    'expired quote and disposed pending operation cannot become ready',
    () async {
      final r = _Repository();
      _seed(r);
      final c = MvpQuoteCubit(r, now: () => _now.add(const Duration(days: 1)));
      await c.load();
      await c.requestQuote();
      expect(c.state.status, MvpQuoteStatus.failure);
      await c.close();
      final pending = Completer<MvpQuote>();
      when(
        () => r.quote(cartRevision: 4, addressId: 'a'),
      ).thenAnswer((_) => pending.future);
      final d = MvpQuoteCubit(r, now: () => _now);
      await d.load();
      final request = d.requestQuote();
      await d.close();
      pending.complete(MvpQuote.fromJson(quoteJson()));
      await request;
      expect(d.isClosed, isTrue);
    },
  );
}
