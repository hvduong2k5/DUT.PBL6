import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';

void main() {
  blocTest<CatalogCubit, CatalogState>(
    'loads categories and products',
    build: () => CatalogCubit(MockCatalogRepository()),
    act: (cubit) => cubit.load(),
    wait: const Duration(milliseconds: 500),
    expect: () => [
      isA<CatalogState>().having(
        (state) => state.status,
        'status',
        CatalogStatus.loading,
      ),
      isA<CatalogState>()
          .having((state) => state.products.isNotEmpty, 'has products', true)
          .having((state) => state.status, 'status', CatalogStatus.success),
    ],
  );

  blocTest<CatalogCubit, CatalogState>(
    'search emits empty success for unknown term',
    build: () => CatalogCubit(MockCatalogRepository()),
    act: (cubit) => cubit.search('không tồn tại'),
    wait: const Duration(milliseconds: 500),
    verify: (cubit) {
      expect(cubit.state.status, CatalogStatus.success);
      expect(cubit.state.products, isEmpty);
    },
  );
}
