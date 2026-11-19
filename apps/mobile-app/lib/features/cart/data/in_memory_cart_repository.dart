import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

class InMemoryCartRepository implements CartRepository {
  InMemoryCartRepository({this._principal});
  final String? Function()? _principal;
  final _carts = <String?, Cart>{};
  String? _lastOwner;
  Cart get _cart {
    final owner = _principal?.call();
    if (_lastOwner == null && owner != null) {
      final guest = _carts.remove(null);
      if (guest != null && guest.items.isNotEmpty) {
        final items = [...?_carts[owner]?.items];
        for (final line in guest.items) {
          final index = items.indexWhere(
            (i) => i.variant.sku == line.variant.sku,
          );
          if (index < 0) {
            items.add(line);
          } else {
            final quantity = (items[index].quantity + line.quantity).clamp(
              1,
              maxCartItemQuantity,
            );
            items[index] = items[index].copyWith(quantity: quantity);
          }
        }
        _carts[owner] = Cart(items: items);
      }
    }
    _lastOwner = owner;
    return _carts[owner] ?? const Cart();
  }

  set _cart(Cart value) {
    _carts[_principal?.call()] = value;
  }

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
