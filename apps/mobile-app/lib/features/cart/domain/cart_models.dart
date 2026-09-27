import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

class CartLine extends Equatable {
  const CartLine({
    required this.itemId,
    required this.product,
    required this.variant,
    required this.quantity,
  });
  final String itemId;
  final Product product;
  final ProductVariant variant;
  final int quantity;
  Money get subtotal => variant.price.times(quantity);
  CartLine copyWith({int? quantity}) => CartLine(
    itemId: itemId,
    product: product,
    variant: variant,
    quantity: quantity ?? this.quantity,
  );
  @override
  List<Object?> get props => [itemId, product, variant, quantity];
}

class Cart extends Equatable {
  const Cart({this.id = 'CART-LOCAL-GUEST', this.items = const []});
  final String id;
  final List<CartLine> items;
  int get totalItems => items.fold(0, (sum, item) => sum + item.quantity);
  Money get subtotal =>
      items.fold(Money.vnd(0), (sum, item) => sum + item.subtotal);
  @override
  List<Object?> get props => [id, items];
}
