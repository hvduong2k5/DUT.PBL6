import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';

class MockCheckoutRepository implements CheckoutRepository {
  int submitCount = 0;
  @override
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  }) async {
    submitCount++;
    await Future<void>.delayed(const Duration(milliseconds: 350));
    return OrderConfirmation(
      orderId:
          'OM-${DateTime.now().millisecondsSinceEpoch.toString().substring(5)}',
      status: 'PENDING_PAYMENT',
      total: draft.cart.subtotal,
    );
  }
}
