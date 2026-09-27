import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

abstract interface class CatalogRepository {
  Future<List<Category>> categories();
  Future<List<Product>> products({String? categoryId, String query = ''});
  Future<Product> product(String id);
}
