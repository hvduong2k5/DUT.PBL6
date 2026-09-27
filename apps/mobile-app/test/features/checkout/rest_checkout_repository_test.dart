import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/checkout/data/rest_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';

void main() {
  test('REST checkout sends the structured OpenAPI address contract', () async {
    RequestOptions? request;
    final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          request = options;
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 201,
              data: {
                'order_id': 'ORD-1',
                'status': 'PENDING_PAYMENT',
                'final_amount': {
                  'currency_code': 'VND',
                  'units': 345000,
                  'nanos': 0,
                },
              },
            ),
          );
        },
      ),
    );
    final product = await MockCatalogRepository().product('PROD-MX-GION-500');
    final cart = await InMemoryCartRepository().add(
      product,
      product.variants.first,
      2,
    );
    final repository = RestCheckoutRepository(ApiClient(dio: dio));

    final result = await repository.submit(
      CheckoutDraft(
        fullName: 'Nguyễn Văn An',
        phone: '0905123456',
        streetAddress: '123 Lê Duẩn',
        ward: 'Phường Thuận Hòa',
        district: 'Thành phố Huế',
        province: 'Thừa Thiên Huế',
        cart: cart,
      ),
      idempotencyKey: '9a8b7c6d-5e4f-4a2b-8c0d-e9f8a7b6c5d4',
    );

    expect(request!.path, '/api/v1/checkout');
    expect(
      request!.headers['X-Idempotency-Key'],
      '9a8b7c6d-5e4f-4a2b-8c0d-e9f8a7b6c5d4',
    );
    final body = Map<String, dynamic>.from(request!.data as Map);
    expect(body['channel'], 'MOBILE_APP');
    expect(body['payment_method'], 'COD');
    expect(body['shipping_address'], {
      'recipient_name': 'Nguyễn Văn An',
      'phone_number': '0905123456',
      'street_address': '123 Lê Duẩn',
      'ward': 'Phường Thuận Hòa',
      'district': 'Thành phố Huế',
      'province': 'Thừa Thiên Huế',
    });
    expect((body['items'] as List).single['quantity'], 2);
    expect(result.orderId, 'ORD-1');
  });
}
