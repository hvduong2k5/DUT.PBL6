import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';

class CartRepositoryMock extends Mock implements CartRepository {}

void main() {
  test(
    'only selected lines are purchased, unselected quantities survive',
    () async {
      final repository = InMemoryCartRepository();
      final catalog = MockCatalogRepository();
      final first = await catalog.product('PROD-MX-GION-500');
      final second = await catalog.product('PROD-MX-DEO-300');
      await repository.add(first, first.variants.first, 2);
      await repository.add(second, second.variants.first, 3);
      final cubit = CartCubit(repository);
      await cubit.load();
      expect(cubit.state.selectedIds.length, 2);
      final unselected = cubit.state.cart.items.last.itemId;
      cubit.toggle(unselected);
      expect(cubit.state.selectedCart.totalItems, 2);
      await cubit.update(unselected, 4);
      expect(cubit.state.selectedIds, isNot(contains(unselected)));
      await cubit.consume(cubit.state.selectedIds);
      expect(cubit.state.cart.items.single.itemId, unselected);
      expect(cubit.state.cart.items.single.quantity, 4);
      cubit.toggleAll();
      expect(cubit.state.selectedIds, {unselected});
      cubit.toggleAll();
      expect(cubit.state.selectedIds, isEmpty);
      await cubit.close();
    },
  );

  test(
    'pending mutation prevents duplicate and failure preserves selections',
    () async {
      final product = await MockCatalogRepository().product('PROD-MX-GION-500');
      final cart = Cart(
        items: [
          CartLine(
            itemId: 'line',
            product: product,
            variant: product.variants.first,
            quantity: 1,
          ),
        ],
      );
      final repository = CartRepositoryMock();
      when(repository.getCart).thenAnswer((_) async => cart);
      final pending = Completer<Cart>();
      when(
        () => repository.update('line', 2),
      ).thenAnswer((_) => pending.future);
      final cubit = CartCubit(repository);
      await cubit.load();
      final first = cubit.update('line', 2);
      expect(await cubit.update('line', 2), isFalse);
      pending.completeError(StateError('offline'));
      expect(await first, isFalse);
      expect(cubit.state.selectedIds, {'line'});
      expect(cubit.state.cart, cart);
      expect(cubit.state.pending, isFalse);
      expect(cubit.state.message, isNotNull);
      verify(() => repository.update('line', 2)).called(1);
      await cubit.close();
    },
  );
}
