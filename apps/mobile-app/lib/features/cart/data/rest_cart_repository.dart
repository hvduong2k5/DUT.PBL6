import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:uuid/uuid.dart';

class RestCartRepository implements CartRepository {
  RestCartRepository(this._client, {String? sessionId})
    : _sessionId = sessionId ?? const Uuid().v4();

  final ApiClient _client;
  final String _sessionId;
  Map<String, dynamic> get _headers => {'X-Session-Id': _sessionId};

  @override
  Future<Cart> getCart() async =>
      _cartFromJson(await _client.getJson('/api/v1/cart', headers: _headers));

  @override
  Future<Cart> add(
    Product product,
    ProductVariant variant,
    int quantity,
  ) async => _cartFromJson(
    await _client.postJson(
      '/api/v1/cart/items',
      data: {'sku_code': variant.sku, 'quantity': quantity},
      headers: _headers,
    ),
  );

  @override
  Future<Cart> update(String itemId, int quantity) async {
    if (quantity <= 0) return remove(itemId);
    return _cartFromJson(
      await _client.putJson(
        '/api/v1/cart/items/$itemId',
        data: {'quantity': quantity},
        headers: _headers,
      ),
    );
  }

  @override
  Future<Cart> remove(String itemId) async => _cartFromJson(
    await _client.deleteJson('/api/v1/cart/items/$itemId', headers: _headers),
  );

  @override
  Future<Cart> clear() async {
    await _client.deleteJson('/api/v1/cart', headers: _headers);
    return const Cart(id: 'CART-REMOTE');
  }

  Cart _cartFromJson(Map<String, dynamic> json) {
    final items = (json['items'] as List? ?? const []).map((raw) {
      final item = Map<String, dynamic>.from(raw as Map);
      final price = Money.fromJson(
        Map<String, dynamic>.from(item['unit_price'] as Map),
      );
      final sku = item['sku_code'].toString();
      final variant = ProductVariant(
        id: sku,
        sku: sku,
        name: item['variant_name'].toString(),
        price: price,
        stock: item['in_stock'] == true ? 999999 : 0,
      );
      final product = Product(
        id: sku,
        name: item['product_name'].toString(),
        slug: sku,
        summary: '',
        imageUrl: item['thumbnail_url']?.toString() ?? '',
        price: price,
        ocopStars: 0,
        category: '',
        inStock: item['in_stock'] as bool? ?? false,
        variants: [variant],
      );
      return CartLine(
        itemId: item['item_id'].toString(),
        product: product,
        variant: variant,
        quantity: (item['quantity'] as num).toInt(),
      );
    }).toList();
    return Cart(id: json['cart_id']?.toString() ?? 'CART-REMOTE', items: items);
  }
}
