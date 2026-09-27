import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

abstract interface class CartRepository {
  Future<Cart> getCart();
  Future<Cart> add(Product product, ProductVariant variant, int quantity);
  Future<Cart> update(String itemId, int quantity);
  Future<Cart> remove(String itemId);
  Future<Cart> clear();
}
