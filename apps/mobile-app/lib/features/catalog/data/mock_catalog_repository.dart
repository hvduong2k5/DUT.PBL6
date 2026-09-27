import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';

class MockCatalogRepository implements CatalogRepository {
  static const _delay = Duration(milliseconds: 180);
  static final _categories = [
    const Category(id: 'all', name: 'Tất Cả', slug: 'tat-ca'),
    const Category(id: 'gion', name: 'Mè Xửng Giòn', slug: 'me-xung-gion'),
    const Category(id: 'deo', name: 'Mè Xửng Dẻo', slug: 'me-xung-deo'),
    const Category(id: 'qua', name: 'Quà Huế', slug: 'qua-hue'),
  ];
  static final _products = [
    Product(
      id: 'PROD-MX-GION-500',
      name: 'Mè Xửng Giòn Ngự Tiến',
      slug: 'me-xung-gion-ngu-tien',
      summary: 'Mè rang vàng, mạch nha thủ công',
      imageUrl: '',
      price: Money.vnd(85000),
      ocopStars: 4,
      category: 'gion',
      inStock: true,
      variants: [
        ProductVariant(
          id: 'VAR-GION-500',
          sku: 'MX-GION-500G',
          name: 'Hộp 500g',
          price: Money.vnd(85000),
          stock: 150,
        ),
      ],
      description:
          'Mè xửng giòn là tuyệt tác ẩm thực Cố Đô Huế, nấu thủ công từ đường mạch nha, mè trắng bùi ngậy và đậu phụng tuyển chọn.',
      story:
          'Sinh ra từ làng nghề Hương Thủy ven sông Hương, công thức được gìn giữ qua ba thế hệ.',
    ),
    Product(
      id: 'PROD-MX-DEO-300',
      name: 'Mè Xửng Dẻo Cung Đình',
      slug: 'me-xung-deo-cung-dinh',
      summary: 'Dẻo thơm, thanh ngọt, gói giấy truyền thống',
      imageUrl: '',
      price: Money.vnd(72000),
      ocopStars: 4,
      category: 'deo',
      inStock: true,
      variants: [
        ProductVariant(
          id: 'VAR-DEO-300',
          sku: 'MX-DEO-300G',
          name: 'Hộp 300g',
          price: Money.vnd(72000),
          stock: 80,
        ),
      ],
      description: 'Vị ngọt thanh, dẻo mềm, thoảng hương mè rang.',
      story: 'Một món quà nhỏ mang ký ức xứ Huế.',
    ),
    Product(
      id: 'PROD-SEN-250',
      name: 'Mứt Hạt Sen Tịnh Tâm',
      slug: 'mut-hat-sen-tinh-tam',
      summary: 'Hạt sen Huế tuyển chọn, sên đường phèn',
      imageUrl: '',
      price: Money.vnd(170000),
      ocopStars: 4,
      category: 'qua',
      inStock: true,
      variants: [
        ProductVariant(
          id: 'VAR-SEN-250',
          sku: 'SEN-250G',
          name: 'Hũ 250g',
          price: Money.vnd(170000),
          stock: 40,
        ),
      ],
      description: 'Hạt sen bùi, lớp đường mỏng, vị ngọt dịu.',
      story: 'Tinh túy hồ Tịnh Tâm trong từng hạt sen.',
    ),
    Product(
      id: 'PROD-COMBO',
      name: 'Hộp Quà Di Sản Cố Đô',
      slug: 'hop-qua-di-san',
      summary: 'Tuyển tập đặc sản cho dịp sum vầy',
      imageUrl: '',
      price: Money.vnd(295000),
      ocopStars: 4,
      category: 'qua',
      inStock: true,
      variants: [
        ProductVariant(
          id: 'VAR-COMBO',
          sku: 'COMBO-DISAN',
          name: 'Hộp tiêu chuẩn',
          price: Money.vnd(295000),
          stock: 25,
        ),
      ],
      description: 'Hộp quà kết hợp mè xửng và mứt sen.',
      story: 'Một lời chào trang trọng từ Cố Đô.',
    ),
  ];
  @override
  Future<List<Category>> categories() async {
    await Future<void>.delayed(_delay);
    return _categories;
  }

  @override
  Future<List<Product>> products({
    String? categoryId,
    String query = '',
  }) async {
    await Future<void>.delayed(_delay);
    final normalized = query.trim().toLowerCase();
    return _products
        .where(
          (p) =>
              (categoryId == null ||
                  categoryId == 'all' ||
                  p.category == categoryId) &&
              (normalized.isEmpty ||
                  p.name.toLowerCase().contains(normalized) ||
                  p.summary.toLowerCase().contains(normalized)),
        )
        .toList();
  }

  @override
  Future<Product> product(String id) async {
    await Future<void>.delayed(_delay);
    return _products.firstWhere((item) => item.id == id || item.slug == id);
  }
}
