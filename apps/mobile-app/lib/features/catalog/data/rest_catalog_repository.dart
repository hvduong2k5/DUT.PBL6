import 'package:mobile_app/features/catalog/data/rest_catalog_remote_data_source.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/core/models/page_result.dart';

class RestCatalogRepository implements CatalogRepository {
  const RestCatalogRepository(this._remote);
  final RestCatalogRemoteDataSource _remote;

  @override
  Future<List<Category>> categories() => _remote.categories();

  @override
  Future<Product> product(String id) => _remote.product(id);

  @override
  Future<List<Product>> products({String? categoryId, String query = ''}) =>
      _remote.products(categoryId: categoryId, query: query);
  @override
  Future<PageResult<Product>> productPage({
    String query = '',
    int page = 1,
    int pageSize = 20,
  }) => _remote.productPage(query: query, page: page, pageSize: pageSize);
}
