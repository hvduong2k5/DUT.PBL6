import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_remote_data_source.dart';

void main() {
  test('categories maps the OpenAPI categories envelope', () async {
    final dio = Dio(BaseOptions(baseUrl: 'http://mockoon.test'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          handler.resolve(
            Response<Map<String, dynamic>>(
              requestOptions: options,
              statusCode: 200,
              data: {
                'categories': [
                  {
                    'category_id': 'CAT-ME-XUNG-GION',
                    'name': 'Mè Xửng Giòn Truyền Thống',
                    'slug': 'me-xung-gion',
                  },
                ],
              },
            ),
          );
        },
      ),
    );

    final dataSource = RestCatalogRemoteDataSource(ApiClient(dio: dio));
    final categories = await dataSource.categories();

    expect(categories, hasLength(1));
    expect(categories.single.id, 'CAT-ME-XUNG-GION');
    expect(categories.single.slug, 'me-xung-gion');
  });
}
