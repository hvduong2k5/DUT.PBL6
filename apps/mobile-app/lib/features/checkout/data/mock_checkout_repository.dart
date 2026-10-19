import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_location.dart';

class MockCheckoutRepository implements CheckoutRepository {
  int submitCount = 0;
  @override
  Future<List<CheckoutProvince>> locations() async => const [
    CheckoutProvince(
      code: 'HUE',
      name: 'Thành phố Huế',
      wards: [
        CheckoutWard(code: 'THUAN-HOA', name: 'Phường Thuận Hòa'),
        CheckoutWard(code: 'PHU-NHUAN', name: 'Phường Phú Nhuận'),
        CheckoutWard(code: 'VY-DA', name: 'Phường Vỹ Dạ'),
        CheckoutWard(code: 'KIM-LONG', name: 'Phường Kim Long'),
      ],
    ),
    CheckoutProvince(
      code: 'DA-NANG',
      name: 'Thành phố Đà Nẵng',
      wards: [
        CheckoutWard(code: 'HAI-CHAU', name: 'Phường Hải Châu'),
        CheckoutWard(code: 'SON-TRA', name: 'Phường Sơn Trà'),
      ],
    ),
  ];
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
      paymentMethod: draft.paymentMethod,
      paymentExpiresAt: draft.paymentMethod == PaymentMethod.vietqr
          ? DateTime.now().add(const Duration(minutes: 15))
          : null,
    );
  }
}
