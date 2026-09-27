import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';

void main() {
  late InMemoryCartRepository repository;
  setUp(() => repository = InMemoryCartRepository());

  test('add, update, and remove keep guest cart totals correct', () async {
    final product = await MockCatalogRepository().product('PROD-MX-GION-500');
    final cubit = CartCubit(repository);
    await cubit.add(product, product.variants.first, quantity: 2);
    expect(cubit.state.cart.totalItems, 2);
    final line = cubit.state.cart.items.single;
    await cubit.update(line.itemId, 3);
    expect(cubit.state.cart.subtotal, product.price.times(3));
    await cubit.remove(line.itemId);
    expect(cubit.state.cart.items, isEmpty);
    await cubit.close();
  });

  blocTest<CartCubit, CartState>(
    'loads an empty cart',
    build: () => CartCubit(repository),
    act: (cubit) => cubit.load(),
    expect: () => [
      isA<CartState>().having(
        (state) => state.status,
        'loading',
        CartStatus.loading,
      ),
      isA<CartState>().having(
        (state) => state.status,
        'success',
        CartStatus.success,
      ),
    ],
  );
}
