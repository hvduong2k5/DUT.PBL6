import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

abstract interface class MvpPurchaseRepository {
  Future<MvpCart> cart();
  Future<List<MvpSavedAddress>> savedAddresses();
  Future<MvpQuote> quote({
    required int cartRevision,
    required String addressId,
  });
}
