import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

class InMemoryCartRepository implements CartRepository {
  Cart _cart = const Cart();
  @override
  Future<Cart> getCart() async => _cart;
  @override
  Future<Cart> add(
    Product product,
    ProductVariant variant,
    int quantity,
  ) async {
    final items = [..._cart.items];
    final index = items.indexWhere((item) => item.variant.sku == variant.sku);
    if (index < 0) {
      items.add(
        CartLine(
          itemId: 'ITEM-${variant.sku}',
          product: product,
          variant: variant,
          quantity: quantity,
        ),
      );
    } else {
      items[index] = items[index].copyWith(
        quantity: items[index].quantity + quantity,
      );
    }
    return _cart = Cart(items: items);
  }

  @override
  Future<Cart> update(String itemId, int quantity) async {
    if (quantity <= 0) return remove(itemId);
    _cart = Cart(
      items: _cart.items
          .map(
            (item) => item.itemId == itemId
                ? item.copyWith(quantity: quantity)
                : item,
          )
          .toList(),
    );
    return _cart;
  }

  @override
  Future<Cart> remove(String itemId) async {
    _cart = Cart(
      items: _cart.items.where((item) => item.itemId != itemId).toList(),
    );
    return _cart;
  }

  @override
  Future<Cart> clear() async => _cart = const Cart();
}
