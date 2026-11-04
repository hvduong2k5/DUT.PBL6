import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_location.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';

void requireCustomerPurchase(AuthUser? user) {
  if (user != null && !user.canPurchase) {
    throw ApiException(
      Failure(kind: FailureKind.forbidden, message: user.purchaseRestriction!),
    );
  }
}

class IdentityCartGuard implements CartRepository {
  IdentityCartGuard(this.inner, this.user);
  final CartRepository inner;
  final AuthUser? Function() user;
  @override
  Future<Cart> getCart() => inner.getCart();
  @override
  Future<Cart> add(
    Product product,
    ProductVariant variant,
    int quantity,
  ) async {
    requireCustomerPurchase(user());
    return inner.add(product, variant, quantity);
  }

  @override
  Future<Cart> update(String itemId, int quantity) async {
    requireCustomerPurchase(user());
    return inner.update(itemId, quantity);
  }

  @override
  Future<Cart> remove(String itemId) => inner.remove(itemId);
  @override
  Future<Cart> clear() => inner.clear();
}

class IdentityCheckoutGuard implements CheckoutRepository {
  IdentityCheckoutGuard(this.inner, this.user);
  final CheckoutRepository inner;
  final AuthUser? Function() user;
  @override
  Future<List<CheckoutProvince>> locations() => inner.locations();
  @override
  Future<OrderConfirmation> submit(
    CheckoutDraft draft, {
    required String idempotencyKey,
  }) async {
    requireCustomerPurchase(user());
    return inner.submit(draft, idempotencyKey: idempotencyKey);
  }
}

class IdentityCapabilitiesGuard implements CustomerCapabilitiesRepository {
  IdentityCapabilitiesGuard(this.inner, this.user);
  final CustomerCapabilitiesRepository inner;
  final AuthUser? Function() user;
  @override
  Future<CustomerCapabilities> load() async {
    final projection = await inner.load();
    return CustomerCapabilities(
      actor: projection.actor,
      version: projection.version,
      values: {...projection.values}
        ..removeWhere(
          (v) =>
              user()?.canPurchase == false &&
              {
                'CART_MANAGE',
                'CHECKOUT_CREATE',
                'B2B_QUOTE_CREATE',
                'B2B_QUOTE_ACCEPT',
              }.contains(v),
        ),
    );
  }
}
