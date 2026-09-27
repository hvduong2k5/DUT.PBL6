import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';

class RestCheckoutRepository implements CheckoutRepository {
  const RestCheckoutRepository(this._client);

  final ApiClient _client;

  @override
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/checkout',
      headers: {'X-Idempotency-Key': idempotencyKey},
      data: {
        'channel': 'MOBILE_APP',
        'shipping_address': {
          'recipient_name': draft.fullName,
          'phone_number': draft.phone,
          'street_address': draft.streetAddress,
          'ward': draft.ward,
          'district': draft.district,
          'province': draft.province,
        },
        'payment_method': draft.paymentMethod.apiValue,
        'is_gift': false,
        'items': draft.cart.items
            .map(
              (line) => {
                'sku_code': line.variant.sku,
                'quantity': line.quantity,
                'price': {
                  'currency_code': line.variant.price.currencyCode,
                  'units': line.variant.price.units.toInt(),
                  'nanos': line.variant.price.nanos,
                },
              },
            )
            .toList(),
      },
    );
    return OrderConfirmation(
      orderId: json['order_id'].toString(),
      status: json['status'].toString(),
      total: Money.fromJson(
        Map<String, dynamic>.from(json['final_amount'] as Map),
      ),
    );
  }
}
