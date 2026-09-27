import 'package:mobile_app/features/checkout/domain/checkout_models.dart';

abstract interface class CheckoutRepository {
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  });
}
