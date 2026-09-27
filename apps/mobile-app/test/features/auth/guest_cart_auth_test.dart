import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';

void main() {
  test('guest cart contents survive successful authentication', () async {
    final cart = CartCubit(InMemoryCartRepository());
    final auth = AuthCubit(MockAuthRepository());
    final product = await MockCatalogRepository().product('PROD-MX-GION-500');

    await cart.add(product, product.variants.first, quantity: 2);
    final itemBeforeLogin = cart.state.cart.items.single;
    final loggedIn = await auth.login('0905123456', 'matkhau123');

    expect(loggedIn, isTrue);
    expect(auth.state.status, AuthStatus.authenticated);
    expect(cart.state.cart.totalItems, 2);
    expect(cart.state.cart.items.single.itemId, itemBeforeLogin.itemId);
    expect(cart.state.cart.items.single.quantity, 2);

    await cart.close();
    await auth.close();
  });
}
