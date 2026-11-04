import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

/// The MVP cart uses integer VND and an explicit revision on every mutation.
class MvpCartRepository implements CartRepository {
  MvpCartRepository(this.client);
  final ApiClient client;
  Future<int> _revision() async =>
      (await client.getJson('/api/v1/cart'))['revision'] as int;
  @override
  Future<Cart> getCart() async => _cart(await client.getJson('/api/v1/cart'));
  @override
  Future<Cart> add(
    Product product,
    ProductVariant variant,
    int quantity,
  ) async {
    final revision = await _revision();
    return _cart(
      await client.postJson(
        '/api/v1/cart/items',
        data: {
          'sku_code': variant.sku,
          'quantity': quantity,
          'revision': revision,
        },
      ),
    );
  }

  @override
  Future<Cart> update(String itemId, int quantity) async {
    if (quantity <= 0) return remove(itemId);
    final revision = await _revision();
    return _cart(
      await client.putJson(
        '/api/v1/cart/items/${Uri.encodeComponent(itemId)}',
        data: {'quantity': quantity, 'revision': revision},
      ),
    );
  }

  @override
  Future<Cart> remove(String itemId) async {
    final revision = await _revision();
    await client.deleteVoid(
      '/api/v1/cart/items/${Uri.encodeComponent(itemId)}',
      headers: {'If-Match': '"$revision"'},
    );
    return getCart();
  }

  @override
  Future<Cart> clear() async {
    var current = await client.getJson('/api/v1/cart');
    final original = List<Map<String, dynamic>>.from(
      (current['items'] as List).map(
        (item) => Map<String, dynamic>.from(item as Map),
      ),
    );
    // Order only exposes line deletion (204), not DELETE /cart. Stop on a
    // conflict; never replay a write or pretend a partial clear was atomic.
    for (final item in original) {
      final id = (item['item_id'] as String?) ?? item['sku_code'] as String;
      final revision = current['revision'] as int;
      await client.deleteVoid(
        '/api/v1/cart/items/${Uri.encodeComponent(id)}',
        headers: {'If-Match': '"$revision"'},
      );
      current = await client.getJson('/api/v1/cart');
    }
    return _cart(current);
  }

  Future<Cart> _cart(Map<String, dynamic> data) async {
    final lines = <CartLine>[];
    for (final raw in data['items'] as List) {
      final item = Map<String, dynamic>.from(raw as Map);
      final sku = item['sku_code'] as String;
      final response = await client.getJson(
        '/api/v1/products/variants/${Uri.encodeComponent(sku)}',
      );
      final view = Map<String, dynamic>.from(response['variant'] as Map);
      final price = Money.fromJson(
        Map<String, dynamic>.from(view['price'] as Map),
      );
      final stock = (view['stock_available'] as num?)?.toInt() ?? 0;
      final stockKnown = view['stock_available'] != null;
      final variant = ProductVariant(
        id: sku,
        sku: sku,
        name: view['name'] as String,
        price: price,
        stock: stock,
        stockKnown: stockKnown,
      );
      final product = Product(
        id: sku,
        name: (item['product_name'] as String?) ?? view['name'] as String,
        slug: sku,
        summary: '',
        imageUrl: '',
        price: price,
        ocopStars: 0,
        category: '',
        inStock: stock > 0,
        stockKnown: stockKnown,
        variants: [variant],
      );
      lines.add(
        CartLine(
          itemId: (item['item_id'] as String?) ?? sku,
          product: product,
          variant: variant,
          quantity: item['quantity'] as int,
        ),
      );
    }
    return Cart(id: 'CART-MVP', items: lines);
  }
}
