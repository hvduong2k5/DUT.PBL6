import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';

class RestMvpCheckoutRepository implements MvpCheckoutRepository {
  const RestMvpCheckoutRepository(this._client);
  final ApiClient _client;
  @override
  Future<MvpCheckoutOperation> accept(
    MvpCheckoutInput input, {
    required String key,
  }) async {
    if (!RegExp(r'^[!-~]{16,128}$').hasMatch(key)) {
      throw ArgumentError('Invalid checkout key');
    }
    return MvpCheckoutOperation.fromJson(
      await _client.postJson(
        '/api/v1/checkout',
        data: input.toJson(),
        headers: {'Idempotency-Key': key},
      ),
    );
  }

  @override
  Future<MvpCheckoutOperation> operation(String id) async {
    checkoutId(id);
    final value = MvpCheckoutOperation.fromJson(
      await _client.getJson('/api/v1/checkout-operations/$id'),
    );
    if (value.id != id) {
      throw const FormatException('Checkout operation identity mismatch');
    }
    return value;
  }
}
