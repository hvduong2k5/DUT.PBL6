import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_location.dart';

class RestCheckoutRepository implements CheckoutRepository {
  const RestCheckoutRepository(this._client);

  final ApiClient _client;

  @override
  Future<List<CheckoutProvince>> locations() async {
    final json = await _client.getJson('/api/v1/locations/checkout');
    return (json['provinces'] as List)
        .map(
          (raw) =>
              CheckoutProvince.fromJson(Map<String, dynamic>.from(raw as Map)),
        )
        .toList();
  }

  @override
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/checkout',
      headers: {'Idempotency-Key': idempotencyKey},
      data: {
        'channel': 'MOBILE_APP',
        'shipping_address': {
          'recipient_name': draft.fullName,
          'phone_number': draft.phone,
          'street_address': draft.streetAddress,
          'ward': draft.ward,
          'province': draft.province,
        },
        'payment_method': draft.paymentMethod.apiValue,
        'items': draft.cart.items
            .map(
              (line) => {
                'sku_code': line.variant.sku,
                'quantity': line.quantity,
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
      paymentMethod: draft.paymentMethod,
      vietQrUrl: json['vietqr_url'] as String?,
      paymentExpiresAt: DateTime.tryParse(
        json['payment_expires_at'] as String? ?? '',
      ),
      subtotal: _money(json['subtotal_amount']),
      discount: _money(json['discount_amount']),
      shippingFee: _money(json['shipping_fee']),
    );
  }

  Money? _money(Object? json) =>
      json is Map ? Money.fromJson(Map<String, dynamic>.from(json)) : null;
}
