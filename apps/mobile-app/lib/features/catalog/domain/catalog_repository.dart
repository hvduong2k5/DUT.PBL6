import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/core/models/page_result.dart';

abstract interface class CatalogRepository {
  Future<List<Category>> categories();
  Future<List<Product>> products({String? categoryId, String query = ''});
  Future<Product> product(String id);
  Future<PageResult<Product>> productPage({
    String query = '',
    int page = 1,
    int pageSize = 20,
  });
}
