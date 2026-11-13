import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_location.dart';

abstract interface class CheckoutRepository {
  Future<List<CheckoutProvince>> locations();
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  });
}
