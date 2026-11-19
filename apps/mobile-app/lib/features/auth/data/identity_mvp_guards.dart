import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/data/identity_purchase_guards.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';

void _customer(AuthUser? user) {
  if (user == null) {
    throw const ApiException(
      Failure(
        kind: FailureKind.unauthenticated,
        message: 'Vui lòng đăng nhập để mua hàng.',
      ),
    );
  }
  requireCustomerPurchase(user);
}

class IdentityMvpPurchaseGuard implements MvpPurchaseRepository {
  IdentityMvpPurchaseGuard(this.inner, this.user);
  final MvpPurchaseRepository inner;
  final AuthUser? Function() user;
  @override
  Future<MvpCart> cart() {
    _customer(user());
    return inner.cart();
  }

  @override
  Future<List<MvpSavedAddress>> savedAddresses() {
    _customer(user());
    return inner.savedAddresses();
  }

  @override
  Future<MvpQuote> quote({
    required int cartRevision,
    required String addressId,
  }) {
    _customer(user());
    return inner.quote(cartRevision: cartRevision, addressId: addressId);
  }
}

class IdentityMvpCheckoutGuard implements MvpCheckoutRepository {
  IdentityMvpCheckoutGuard(this.inner, this.user);
  final MvpCheckoutRepository inner;
  final AuthUser? Function() user;
  @override
  Future<MvpCheckoutOperation> accept(
    MvpCheckoutInput input, {
    required String key,
  }) {
    _customer(user());
    return inner.accept(input, key: key);
  }

  @override
  Future<MvpCheckoutOperation> operation(String id) {
    _customer(user());
    return inner.operation(id);
  }
}

class IdentityMvpOrdersGuard implements MvpOrdersRepository {
  IdentityMvpOrdersGuard(this.inner, this.user);
  final MvpOrdersRepository inner;
  final AuthUser? Function() user;
  @override
  Future<MvpOrderPageData> firstPage() {
    _customer(user());
    return inner.firstPage();
  }

  @override
  Future<MvpOrder> order(String id) {
    _customer(user());
    return inner.order(id);
  }

  @override
  Future<MvpPaymentStatus> payment(String id) {
    _customer(user());
    return inner.payment(id);
  }

  @override
  Future<MvpPaymentInstructions> instructions(String id) {
    _customer(user());
    return inner.instructions(id);
  }

  @override
  Future<MvpCancellation> cancel(MvpCancellationCommand command) {
    _customer(user());
    return inner.cancel(command);
  }
}
