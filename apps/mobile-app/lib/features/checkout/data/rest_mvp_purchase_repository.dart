import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';

class RestMvpPurchaseRepository implements MvpPurchaseRepository {
  const RestMvpPurchaseRepository(this._client);
  final ApiClient _client;

  @override
  Future<MvpCart> cart() async =>
      MvpCart.fromJson(await _client.getJson('/api/v1/cart'));

  @override
  Future<List<MvpSavedAddress>> savedAddresses() async => List.unmodifiable(
    (await _client.getJsonList(
      '/api/v1/profile/addresses',
    )).map(MvpSavedAddress.fromJson),
  );

  @override
  Future<MvpQuote> quote({
    required int cartRevision,
    required String addressId,
  }) async {
    if (cartRevision < 0 || addressId.trim().isEmpty) {
      throw ArgumentError('Quote requires cart revision and saved address');
    }
    final quote = MvpQuote.fromJson(
      await _client.postJson(
        '/api/v1/checkout/quote',
        data: {
          'cart_revision': cartRevision,
          'payment_method': 'VIETQR',
          'shipping_address_id': addressId,
        },
      ),
    );
    if (quote.cartRevision != cartRevision ||
        quote.paymentMethod != 'VIETQR' ||
        quote.sourceAddressId != null && quote.sourceAddressId != addressId) {
      throw const FormatException('Quote does not match requested context');
    }
    return quote;
  }
}
