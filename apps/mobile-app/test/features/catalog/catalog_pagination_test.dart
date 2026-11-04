import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/core/models/page_result.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_remote_data_source.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';

class _Catalog extends Mock implements CatalogRepository {}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'catalog REST uses pageSize/q only and preserves unknown stock',
    () async {
      RequestOptions? request;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            request = options;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'total': 3,
                  'items': [
                    {
                      'product_id': 'P1',
                      'name': 'Mè xửng',
                      'slug': 'me-xung',
                      'summary': '',
                      'thumbnail_url': '',
                      'base_price': {
                        'currency_code': 'VND',
                        'units': 10000,
                        'nanos': 0,
                      },
                      'ocop_star': 4,
                      'category_name': 'Mè xửng',
                      'in_stock': null,
                      'stock_available': null,
                    },
                  ],
                },
              ),
            );
          },
        ),
      );
      final result = await RestCatalogRemoteDataSource(
        ApiClient(dio: dio),
      ).productPage(query: 'mè xửng', page: 2, pageSize: 1);
      expect(request!.path, '/api/v1/products/search');
      expect(request!.queryParameters, {
        'q': 'mè xửng',
        'page': 2,
        'pageSize': 1,
      });
      expect(result.hasMore, isTrue);
      expect(result.items.single.stockKnown, isFalse);
      expect(result.items.single.availabilityLabel, 'Đang cập nhật tồn kho');
    },
  );
  test(
    'loaded facets/sort expand with next page and do not duplicate pending page',
    () async {
      final fixtures = await MockCatalogRepository().products(),
          repository = _Catalog();
      final pending = Completer<PageResult<Product>>();
      var nextCalls = 0;
      when(repository.categories).thenAnswer((_) async => []);
      when(
        () => repository.productPage(query: '', page: 1, pageSize: 20),
      ).thenAnswer(
        (_) async =>
            PageResult(items: [fixtures[0]], total: 2, page: 1, pageSize: 1),
      );
      when(
        () => repository.productPage(query: '', page: 2, pageSize: 1),
      ).thenAnswer((_) {
        nextCalls++;
        return pending.future;
      });
      final cubit = CatalogCubit(repository);
      await cubit.load();
      expect(cubit.state.complete, isFalse);
      cubit.setFilters(const CatalogFilters(sort: CatalogSort.priceAscending));
      final load = cubit.loadMore();
      await cubit.loadMore();
      expect(nextCalls, 1);
      pending.complete(
        PageResult(items: [fixtures[1]], total: 2, page: 2, pageSize: 1),
      );
      await load;
      expect(cubit.state.complete, isTrue);
      expect(cubit.state.products.first.id, fixtures[1].id);
      expect(cubit.state.loadedProducts.length, 2);
      cubit.setFilters(const CatalogFilters(stars: 3));
      expect(cubit.state.products, isEmpty);
      await cubit.close();
    },
  );
  test(
    'stale next-page response cannot replace new search and no-progress page stays retryable',
    () async {
      final product = (await MockCatalogRepository().products()).first,
          repository = _Catalog();
      when(repository.categories).thenAnswer((_) async => []);
      when(
        () => repository.productPage(query: '', page: 1, pageSize: 20),
      ).thenAnswer(
        (_) async =>
            PageResult(items: [product], total: 2, page: 1, pageSize: 1),
      );
      when(
        () => repository.productPage(query: '', page: 2, pageSize: 1),
      ).thenAnswer(
        (_) async =>
            PageResult(items: [product], total: 2, page: 2, pageSize: 1),
      );
      final cubit = CatalogCubit(repository);
      await cubit.load();
      await cubit.loadMore();
      expect(cubit.state.pageMessage, isNotNull);
      expect(cubit.state.page, 1);
      expect(cubit.state.loadedProducts.length, 1);
      final pending = Completer<PageResult<Product>>();
      when(
        () => repository.productPage(query: '', page: 2, pageSize: 1),
      ).thenAnswer((_) => pending.future);
      when(
        () => repository.productPage(query: 'new', page: 1, pageSize: 20),
      ).thenAnswer(
        (_) async => PageResult(items: [], total: 0, page: 1, pageSize: 20),
      );
      final old = cubit.loadMore();
      await cubit.search('new');
      pending.complete(
        PageResult(items: [product], total: 2, page: 2, pageSize: 1),
      );
      await old;
      expect(cubit.state.query, 'new');
      expect(cubit.state.products, isEmpty);
      expect(cubit.state.loadingMore, isFalse);
      await cubit.close();
    },
  );
  testWidgets(
    'unknown stock still allows detail entry and never says sold out',
    (tester) async {
      final product = Product.fromListJson({
        'product_id': 'P',
        'name': 'Mẫu',
        'slug': 'mau',
        'base_price': {'currency_code': 'VND', 'units': 1, 'nanos': 0},
        'stock_available': null,
        'in_stock': null,
      });
      var opened = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 180,
              height: 340,
              child: ProductCard(
                product: product,
                onOpen: () => opened++,
                onAdd: () => opened++,
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Chọn quy cách'));
      expect(opened, 1);
      expect(find.text('Tạm hết hàng'), findsNothing);
      expect(find.text('Đang cập nhật tồn kho'), findsOneWidget);
    },
  );
  testWidgets('filter sheet yields empty recovery and clear restores results', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp(initialLocation: '/catalog'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bộ lọc'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Trên 500.000đ'));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('Áp dụng bộ lọc'),
      200,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.tap(find.text('Áp dụng bộ lọc'));
    await tester.pumpAndSettle();
    expect(find.text('Không tìm thấy sản phẩm'), findsOneWidget);
    await tester.tap(find.text('Xóa tất cả'));
    await tester.pumpAndSettle();
    expect(find.text('Không tìm thấy sản phẩm'), findsNothing);
    expect(find.text('4 sản phẩm phù hợp'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
