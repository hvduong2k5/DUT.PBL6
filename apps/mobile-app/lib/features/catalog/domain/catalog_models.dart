import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';

class Category extends Equatable {
  const Category({required this.id, required this.name, required this.slug});
  factory Category.fromJson(Map<String, dynamic> json) => Category(
    id: json['category_id'].toString(),
    name: json['name'].toString(),
    slug: json['slug'].toString(),
  );
  final String id;
  final String name;
  final String slug;
  @override
  List<Object?> get props => [id, name, slug];
}

class Product extends Equatable {
  const Product({
    required this.id,
    required this.name,
    required this.slug,
    required this.summary,
    required this.imageUrl,
    required this.price,
    required this.ocopStars,
    required this.category,
    required this.inStock,
    required this.variants,
    this.description = '',
    this.story = '',
  });
  factory Product.fromListJson(Map<String, dynamic> json) => Product(
    id: json['product_id'].toString(),
    name: json['name'].toString(),
    slug: json['slug'].toString(),
    summary: json['summary']?.toString() ?? '',
    imageUrl: json['thumbnail_url']?.toString() ?? '',
    price: Money.fromJson(Map<String, dynamic>.from(json['base_price'] as Map)),
    ocopStars: (json['ocop_star'] as num?)?.toInt() ?? 0,
    category: json['category_name']?.toString() ?? '',
    inStock: json['in_stock'] as bool? ?? false,
    variants: [
      ProductVariant(
        id: json['product_id'].toString(),
        sku: json['product_id'].toString(),
        name: 'Tiêu chuẩn',
        price: Money.fromJson(
          Map<String, dynamic>.from(json['base_price'] as Map),
        ),
        stock: json['in_stock'] == true ? 99 : 0,
      ),
    ],
  );
  final String id;
  final String name;
  final String slug;
  final String summary;
  final String imageUrl;
  final Money price;
  final int ocopStars;
  final String category;
  final bool inStock;
  final List<ProductVariant> variants;
  final String description;
  final String story;
  @override
  List<Object?> get props => [
    id,
    name,
    slug,
    summary,
    imageUrl,
    price,
    ocopStars,
    category,
    inStock,
    variants,
    description,
    story,
  ];
}

class ProductVariant extends Equatable {
  const ProductVariant({
    required this.id,
    required this.sku,
    required this.name,
    required this.price,
    required this.stock,
  });
  final String id;
  final String sku;
  final String name;
  final Money price;
  final int stock;
  @override
  List<Object?> get props => [id, sku, name, price, stock];
}
