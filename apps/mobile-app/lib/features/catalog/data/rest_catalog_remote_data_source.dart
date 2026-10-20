import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/core/models/page_result.dart';

class RestCatalogRemoteDataSource {
  const RestCatalogRemoteDataSource(this._client);
  final ApiClient _client;
  Future<List<Category>> categories() async {
    final json = await _client.getJson('/api/v1/categories');
    final list = json['categories'] is List
        ? json['categories'] as List
        : json['items'] is List
        ? json['items'] as List
        : json['data'] as List? ?? const [];
    return list
        .map(
          (item) => Category.fromJson(Map<String, dynamic>.from(item as Map)),
        )
        .toList();
  }

  Future<List<Product>> products({
    String? categoryId,
    String query = '',
  }) async {
    final result = await productPage(query: query);
    if (categoryId == null || categoryId == 'all') return result.items;
    final category = (await categories())
        .where((c) => c.id == categoryId)
        .firstOrNull;
    return result.items.where((p) => p.category == category?.name).toList();
  }

  Future<PageResult<Product>> productPage({
    String query = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    final searching = query.trim().isNotEmpty;
    final json = await _client.getJson(
      searching ? '/api/v1/products/search' : '/api/v1/products',
      query: {if (searching) 'q': query, 'page': page, 'pageSize': pageSize},
    );
    final items = json['items'] as List? ?? const [];
    final products = items
        .map(
          (item) =>
              Product.fromListJson(Map<String, dynamic>.from(item as Map)),
        )
        .toList();
    return PageResult(
      items: products,
      total: json['total'] as int,
      page: searching ? page : json['page'] as int,
      pageSize: searching ? pageSize : json['page_size'] as int,
    );
  }

  Future<Product> product(String id) async {
    final json = await _client.getJson(
      '/api/v1/products/${Uri.encodeComponent(id)}',
    );
    final variants = (json['variants'] as List? ?? const []).map((item) {
      final value = Map<String, dynamic>.from(item as Map);
      return ProductVariant(
        id: value['variant_id'].toString(),
        sku: value['sku_code'].toString(),
        name: value['name'].toString(),
        price: Money.fromJson(Map<String, dynamic>.from(value['price'] as Map)),
        stock: (value['stock_available'] as num?)?.toInt() ?? 0,
        stockKnown: value['stock_available'] != null,
      );
    }).toList();
    return Product(
      id: json['product_id'].toString(),
      name: json['name'].toString(),
      slug: json['slug'].toString(),
      summary: json['description']?.toString() ?? '',
      imageUrl: (json['images'] as List?)?.firstOrNull?.toString() ?? '',
      price: variants.isEmpty
          ? Money.vnd(0)
          : variants
                .map((item) => item.price)
                .reduce((a, b) => a.compareTo(b) < 0 ? a : b),
      ocopStars: (json['ocop_star'] as num?)?.toInt() ?? 0,
      category: '',
      inStock: variants.any((v) => v.stock > 0),
      stockKnown:
          variants.any((v) => v.stock > 0) ||
          variants.every((v) => v.stockKnown),
      variants: variants,
      description: json['description']?.toString() ?? '',
      story: json['story']?.toString() ?? '',
    );
  }
}
